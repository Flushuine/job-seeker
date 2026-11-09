# AI Job Hunter

A free, local-first job intelligence MVP for macOS Apple Silicon. Go + Chi + GORM + PostgreSQL, with a React/TypeScript dashboard. Optional native Ollama (`qwen3:1.7b`) adds explanations. Deterministic matching, discovery, evidence and application tracking work without AI. No cloud AI requests, subscriptions, paid APIs, vector databases, or model downloads at startup.

## Quick start

Prerequisites: Go **1.24.6+**, Node **22+**, and PostgreSQL **17+** (Postgres.app, Homebrew, or Docker). PostgreSQL and Ollama run natively when possible to conserve memory. Docker Compose contains PostgreSQL only.

```sh
cp .env.example .env
openssl rand -hex 24
```

Set `APP_TOKEN` in `.env` to the generated value. Do not commit `.env`. The browser uses the same token to unlock the local workspace; it is kept in session storage and sent only to the same-origin local API. `make` loads `.env` for Go commands. The Vite proxy targets `127.0.0.1:8080`.

Start PostgreSQL using **one** of these options. The optional native helper keeps data in ignored `.local/postgres`, binds to loopback, and uses local trust authentication; do not expose it on a network:

```sh
# Docker must already be running; binds only to localhost.
make db

# Or start an isolated native Postgres.app/Homebrew cluster in .local/postgres:
make db-native
# Set DATABASE_URL to the connection URL it prints (default port 55432).
# Stop later with: make db-native-stop

# Alternatively, use an existing native PostgreSQL server:
createdb jobhunter
# Set DATABASE_URL in .env to your local user's connection URL.
# postgres://YOUR_MAC_USERNAME@127.0.0.1:5432/jobhunter?sslmode=disable
```

For Docker, the example URL and password match Compose. If you change `POSTGRES_PASSWORD`, also update the password in `DATABASE_URL`. An existing volume retains its original database password.

```sh
make setup
make migrate
make api         # terminal 1: localhost:8080; includes scheduler + AI queue
make web         # terminal 2: open the Local URL printed by Vite
```

Open the Vite URL (usually `http://127.0.0.1:5173`) and enter your `APP_TOKEN`. All API routes except health require this token. The seeded professional profile is editable; matching logic has no hardcoded personal preferences.

In Settings, add your target companies and board identifiers. For example, the public **Grafana Labs** Greenhouse board is `grafanalabs`; **Palantir** has a Lever board `palantir` (board availability can change). These companies are examples of actual source configuration, not endorsements. No company boards or job listings are seeded by migrations. Click **Discover jobs** to retrieve real opportunities.

The API runs the scheduler and one queue consumer. `make worker` is optional for running background processing separately; you do not need it alongside the API. Database advisory locks prevent overlapping discovery and concurrent inference across processes. You can stop the app while the laptop is unused; it safely runs overdue discovery when restarted or resumed.

## Optional native Ollama

Install the native macOS application from [ollama.com](https://ollama.com/download/mac). The application never installs Ollama or pulls a model. **Only if you choose to download the model**, run:

```sh
ollama pull qwen3:1.7b
# If Ollama is not already running as the desktop app:
OLLAMA_NUM_PARALLEL=1 OLLAMA_MAX_LOADED_MODELS=1 ollama serve
```

For the desktop app, set these environment variables before restarting Ollama if needed:

```sh
launchctl setenv OLLAMA_NUM_PARALLEL 1
launchctl setenv OLLAMA_MAX_LOADED_MODELS 1
```

Defaults: `OLLAMA_BASE_URL=http://localhost:11434`, `OLLAMA_MODEL=qwen3:1.7b`, `AI_ENABLED=true`, `AI_MAX_CONCURRENT_REQUESTS=1`. The base URL must be a loopback HTTP origin. No remote/cloud Ollama host is supported. Configure an installed local model in Settings; if you want the environment default to replace the initial model, change it in Settings as well. To avoid AI completely, set `AI_ENABLED=false`, restart the API, and/or switch it off in Settings.

Memory controls for an 8 GB M1:

- A durable PostgreSQL queue admits at most **20** queued/running tasks. One consumer processes one inference at a time, including company interpretations. Analysis endpoints return **202** immediately and expose task status.
- Context is restricted to **2048 or 4096** tokens. Job descriptions are deterministically truncated to 500 or 2,600 characters respectively. The model sees only the bounded profile and relevant preferences; output is capped at 600 or 850 tokens. Oversized or invalid outputs are rejected.
- `think:false` avoids a separate reasoning stream. Keep-alive defaults to **1 minute**, with immediate unload or 5 minutes selectable in Settings. Close other model runners to avoid memory pressure.
- Failed/offline inference leaves deterministic matching intact. No AI result replaces the deterministic ranking score. JSON schemas constrain outputs, and Go validates required fields, types, enums, bounds and cited evidence IDs.
- Successful analyses are cached by job content, profile/preferences, model, and prompt version. Previous analyses are retained. At most one retry is attempted for network/5xx failures; schema failures are surfaced without a retry storm.

## Discovery sources and correctness

- [Greenhouse public Job Board API](https://docs.greenhouse.io/job-board): a configured company board; no global search endpoint. Complete snapshot, content enabled.
- [Lever public Postings API](https://github.com/lever/postings-api): company site identifier, paginated at 100 postings, global or EU API selected per source.
- [Himalayas free public API](https://himalayas.app/docs/remote-jobs-api): optional broader search, no key required. Select Himalayas in Settings, use a feed label as company name, and a keyword such as `golang` as the board field. The keyword is fixed per registered source so snapshots remain comparable. It searches the full public result set, not only target companies. Every listing credits Himalayas and preserves its supplied application link unchanged. Re-fetches are limited to **once per 24 hours** per configured search, including manual discovery. Change keyword by adding another source; disable the old one. Do not redistribute these jobs to third-party boards. Follow the [API terms](https://himalayas.app/terms).

Source-specific clients rate-limit requests. Requests time out, transient failures retry with exponential backoff and bounded `Retry-After`, and a malformed or incomplete response prevents snapshot ingestion. Greenhouse/Lever share a 3-board fetch concurrency bound, not inference workers. Original payloads and per-source identities/URLs are stored in `job_observations`.

Ingestion uses source identity, canonical URL, and a best-effort company/title/location key. First-seen is preserved; last-seen advances. A job closes after **two successful complete snapshots** omit its source observation, and stays open if another source still reports it. Provider failures never trigger closure. Known expiry timestamps exclude expired listings immediately. Missing/unknown data remains explicit. Cross-source title matching can merge distinct requisitions with identical title/location; observations preserve provenance for review. This is a documented MVP tradeoff.

Discovery keeps excluded jobs for auditability and labels the exclusion. Hard exclusions and soft preference mismatches run **before AI queuing**. Scheduled auto-analysis is opt-in and skips excluded jobs and soft mismatches. Manual discovery is synchronous for the fetch only; AI inference is always asynchronous. Runs record partial failures and source errors.

## Matching and ranking

The deterministic engine checks title, recognised technology mentions, profile skills, explicitly extractable experience requirements, seniority, work arrangement, structured geography, comparable salary and employment type. It reports **match / mismatch / unknown**, reasons, missing skills and recruiter questions. It does not assume a technology mention is necessarily a mandatory requirement. Technologies not stated are unknown. Remote does not imply eligibility in Indonesia.

Hard exclusions: excluded companies/keywords, confirmed incompatible arrangements when enabled, unavailable jobs and a known original posting date older than your maximum. Other preferences are soft. Provider update dates are not falsely presented as original publication dates; first-seen is displayed separately.

The technical ranking factor combines recognised skill compatibility (70%) with confirmed title relevance (30%) when a role preference is configured; the separate technical skill metric remains visible. Default ranking weights: technical 30%, work-life balance 30%, remote 20%, compensation 15%, stability 5%. Weights must sum to 100. The score is `sum(known score × weight) / sum(known weights)`. Coverage is `known weights / all weights`. With no known factors the score is null. Workplace scores are deliberately unavailable when evidence does not substantiate them. A high score with low coverage is not a complete endorsement.

Salary comparisons annualise monthly/annual values **only in the same currency**. No invented FX rates or hourly-to-annual conversion. Recognised skills and company title matches are conservative heuristics, not semantic proof. Score sorting recalculates matching records in batches using current settings; this is suitable for a personal collection, not a large public search service.

## Company intelligence

Evidence collection and AI interpretation are separate. Automatic collection reads only **explicitly configured public RSS/Atom feed URLs** whose hosts you add to `RESEARCH_ALLOWED_HOSTS` (comma-separated). HTTPS only, no redirects, proxies, credentials, private/link-local IPs, arbitrary page scraping or anti-bot bypass. Only feed entries mentioning the company are stored, as **public source claims** with low confidence and source links. This is a limited collector, not a general web search or Glassdoor/Reddit integration.

When an article or review cannot be retrieved automatically, paste a permitted excerpt and its URL from Company Details. Select employee opinion, company statement, public claim or unverified allegation. Saving an excerpt does not fetch or verify its URL. Manual timestamps represent submission time. Publication dates are optional. Automatic research caches for 24 hours; manual evidence invalidates cached interpretation. A source URL is unique per company; resubmitting it updates that evidence record.

Research explicitly reports **Insufficient Evidence**. A handful of reviews does not establish a pattern or a company reputation rating. Local AI can interpret the supplied records, cite their IDs, and explain limitations; it cannot browse, verify reviews, or manufacture a Glassdoor score. Interpretation is clearly labelled separately from evidence.

## API examples

Set the token in your shell for these examples (do not paste secrets into source files):

```sh
export APP_TOKEN='YOUR_LOCAL_TOKEN'
curl http://127.0.0.1:8080/api/health
curl -H "Authorization: Bearer $APP_TOKEN" http://127.0.0.1:8080/api/profile
curl -H "Authorization: Bearer $APP_TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Grafana Labs","provider":"greenhouse","board":"grafanalabs","research_urls":[]}' \
  http://127.0.0.1:8080/api/companies
curl -X POST -H "Authorization: Bearer $APP_TOKEN" http://127.0.0.1:8080/api/jobs/discover
curl -H "Authorization: Bearer $APP_TOKEN" 'http://127.0.0.1:8080/api/jobs?q=golang&remote=remote&page=1&limit=20'
curl -X POST -H "Authorization: Bearer $APP_TOKEN" http://127.0.0.1:8080/api/jobs/JOB_UUID/analyse
# Response 202 contains a task UUID:
curl -H "Authorization: Bearer $APP_TOKEN" http://127.0.0.1:8080/api/analysis-tasks/TASK_UUID
```

Endpoints: health; GET jobs/job details; POST discovery/analysis; GET/POST companies; GET company details; POST company research/analysis/evidence; GET/PUT profile and preferences; POST `/api/profile/cv` (multipart `file`) and GET `/api/profile/cv` (latest upload/task); GET/POST applications; PATCH applications; GET discovery runs, sources, status and dashboard stats; PATCH source enabled state. Profile/preferences PUT accepts the complete GET representation. Unknown JSON fields, invalid IDs, invalid pagination and oversized bodies return structured errors: `{"error":{"code":"validation","message":"…"}}`.

`/api/jobs` supports `q`, `remote`, `company_id`, `status` (`all`, `closed`, default open), `sort` (`score`, `company`, `title`, default newest), `page`, `limit` (1–100), and `include_excluded=true` to inspect excluded/unconfirmed jobs. Default lists are paginated. Company and application registries are intentionally small, single-user collections. Discovery history is limited to the most recent 50 runs. Application stages are Saved, Applied, Screening, Interview, Offer, Rejected and Withdrawn.

Administrative mutations require a token and are capped at 30 requests/minute in each API process. This is a local single-user MVP, not a public hosted/multi-user authentication system. Keep Go, PostgreSQL, Vite and Ollama bound to localhost.

## Tests and build

```sh
make test                 # unit + mocked providers/AI, no credentials or Ollama
make build                # Go binaries + TypeScript + production frontend
createdb jobhunter_test   # on your chosen PostgreSQL instance
# Add TEST_DATABASE_URL for this isolated DB to .env, then:
make test-integration
```

`TEST_DATABASE_URL` enables real PostgreSQL migration/repository/API tests; otherwise those tests explicitly skip. Use a disposable database, since tests insert clearly labelled fixture companies and jobs. Tests cover normalization, deduplication, salary periods, unknown handling, scoring, source failure isolation, retries, schema rejection, cancellation, research source attribution/private-network rejection, API auth/validation, migration reruns, snapshots, cross-source availability and queue deduplication. `go test -race ./...` enables concurrency checks.

SQL migrations are embedded and applied in filename order within a transaction under a database lock. `schema_migrations` records applied versions. They run automatically on startup or via `make migrate`. No GORM AutoMigrate or destructive database reset is used.

## Troubleshooting and limits

- **Docker cannot connect:** start Docker Desktop, or use native PostgreSQL to avoid Docker memory overhead.
- **Database connection refused:** check PostgreSQL, the port, DB name, role and `DATABASE_URL`. For Postgres.app, use your macOS role instead of the Docker `jobhunter` role.
- **Port already occupied:** Vite prints the next available port. If changing the API port, update `HTTP_ADDR` and the Vite proxy in `web/vite.config.ts`.
- **Unauthorized:** enter the exact `.env` APP_TOKEN; use “Lock workspace” and sign in again after changing it. Restart Go after environment changes.
- **No jobs:** configure a public board identifier (not a career-page URL), enable the source, run discovery and inspect Discovery activity. Greenhouse and Lever are company-specific; neither offers global search.
- **Himalayas rate-limited or changed:** retries are bounded, the run records the error, and existing availability is preserved. Disable that source if its published access terms change. Search pagination has a 100-page safety limit; exceeding it rejects that snapshot instead of closing unseen jobs.
- **Ollama/model unavailable:** jobs still receive deterministic scores. Task failure explains the error; install your chosen model yourself or disable AI. No model download endpoint is called. Local model quality may be limited; invalid JSON never becomes saved analysis.
- **Research empty:** configure permitted public feeds and allowed hosts, or add manual excerpts. Restricted review platforms are not scraped. “Insufficient Evidence” is a valid result.
- **Model memory pressure:** use the default 1.7B model, 2048 context and immediate unload; stop other model runners and Docker if using native PostgreSQL.

Known MVP limits: no scanned résumé OCR, global search-engine integration, automatic verification of employee reviews, workplace sentiment score, currency conversion, public hosting or multi-user login. Company evidence and analysis are aids to investigation. Cross-source deduplication and text rules are heuristic. Settings-driven ranking recomputes on reads; very large collections would need indexed materialised matching results. JSON fixtures occur only in tests; the application has no fake opportunity seed data.

## CV context and local office restrictions

Settings has three tabs: **Professional profile**, **Job preferences**, and **Target companies & sources**. In Professional profile, use **Upload and personalise** to upload a text-based PDF or UTF-8 TXT CV (up to 5 MB). Analysis runs automatically in the existing background queue using your installed local Ollama model. Local AI must be enabled in both server configuration and Job preferences. No model is downloaded.

PDF extraction requires `pdftotext` from Poppler (`brew install poppler` on macOS); TXT uploads need no extra dependency. Scanned PDFs need an OCR/text export first. Extracted text must contain at least 20 words and fit within 200,000 bytes. Long CVs use bounded beginning/end excerpts and show a truncation note. The database stores extracted text with contact email addresses, phone numbers, and HTTP URLs redacted, plus the extraction result, filename, fingerprint, model and prompt version. Uploaded files are not retained, and CV text is never sent to job providers or cloud AI. The supplied CV PDF and local database are ignored by Git.

A successful analysis replaces the professional summary and supported skills. Name, location and years of experience update only when explicitly supported by the CV; unknown fields retain their previous values. Job preferences, including nearby offices and working arrangements, stay unchanged. Profile updates and normalized skills are saved together. The newest upload takes priority, and manual profile edits made during analysis prevent an overwrite. Invalid or failed analysis retains the current profile; fix the reported issue and upload again. Unchanged CV text reuses its analysis when the model, context and prompt version match.

Updated profile data immediately changes deterministic job matching. Its fingerprint invalidates previous AI job interpretations; request new job analysis as needed. The upload panel shows progress, extracted skills, and unknown information. You can still edit the profile manually.

Settings supports nearby office areas, hard location exclusions, and **Show only roles with confirmed location and arrangement**. A hybrid/on-site job must name an allowed nearby area. Remote jobs must advertise an eligible hiring country, a region including it, or worldwide hiring. Structured provider country restrictions take precedence over broad location labels. Explicit foreign-only/residency restrictions and country exclusions prevent a match. These are source claims, not legal/employment verification; recruiter confirmation and time-zone expectations still matter. Nearby-area names are matched against the advertised office label; no distance is fabricated or inferred from coordinates.

When strict location confirmation is enabled, unknown location/arrangement data is held outside the default shortlist without being marked as a confirmed mismatch. **Include excluded or unconfirmed roles** in the explorer reveals these records and their reasons. Eligibility is applied before pagination and matching totals. Discovery retains source records for auditing; distant jobs are not automatically marked closed. Direct job details and application history remain available.

For this profile, remote and hybrid arrangements are allowed, hiring country is Indonesia, and nearby office areas cover Jabodetabek (including Tangerang Selatan). These choices are editable in Settings. On-site jobs are excluded by the working arrangement rule. The Job Explorer defaults to highest deterministic match, prioritising preferred role titles and known technical compatibility before ranking scores. Explicitly required technologies such as Ruby/Rails that are absent from the profile lower technical compatibility rather than being overlooked. Headline discovered count includes all stored source jobs; new opportunities and recommendations use the current shortlist.

Himalayas sources can additionally set a two-letter hiring country, for example `ID`. This is forwarded as the public API's `country` query parameter so discovery can target Indonesia and worldwide roles. Each keyword/country combination is tracked separately and retains its daily refresh limit. The company source form and `POST /api/companies` accept this optional `country`; leave it blank for Greenhouse/Lever.

## Company origin and leadership profiles

Company Details has a **Company profile** panel and source-backed editor for company overview, legal entity, origin/incorporation country, headquarters, CEO/co-CEOs and CEO nationality/citizenship. Origins, legal incorporation and current headquarters can differ; use the context field to identify what a source actually establishes.

Each nonempty field requires its own HTTPS source URL and source name. Unsupported fields display **Unknown**. Nationality is never inferred from a person's name, birthplace, residence or the company's location, and is not used in ranking. For co-CEOs, explicitly associate each reported nationality with the correct person. Leadership changes clear a previous CEO's nationality unless a new, separately sourced claim is supplied.

Use **Edit profile** to add or correct information. `PATCH /api/companies/:id/profile` accepts partial updates, with field keys `summary`, `legal_name`, `origin_country`, `headquarters`, `ceo`, `ceo_nationality`. Each value is `{ "value": "...", "source_url": "https://...", "source_name": "...", "notes": "..." }`. An empty value clears that fact. The server controls each fact's `recorded_at`; unchanged facts keep their original dates. Recorded date means when a claim was saved, not that the link was automatically fetched or its content independently verified.

Company profiles persist separately from discovery observations and workplace research. Automatic job discovery and company feed refreshes do not overwrite them. This feature supports manually recorded sourced research; it does not automatically look up every company's CEO, scrape biographies, or ask the local LLM to invent corporate facts. Prefer current official leadership pages, filings and reliable public biographies, and record dates or limitations for older material.

## Applied jobs in the job list

Job cards show application status and offer **Mark applied** to record an application you already submitted. This does not submit anything to an employer. Existing recruiter notes are preserved when moving a saved job to Applied, and existing later stages are not reset by the card's Save action.

Applied, Screening, Interview, Offer, Rejected and Withdrawn jobs are hidden from default opportunity lists. Saved jobs remain visible. **Show applied jobs** in Job Explorer restores these records with their current application stage. `GET /api/jobs?include_applied=true` provides the same option. Filtering happens before sorting, pagination and result counts. Jobs and application history are retained in PostgreSQL and Application Tracker; they are not deleted or marked closed. Moving a job back to Saved makes it visible again. The applied filter also keeps already-tracked applications out of scheduled automatic matching.


## Working hours in your time zone

Your display time zone is derived from your saved CV/profile location. Tangerang Selatan/Jabodetabek resolves to `Asia/Jakarta` (WIB, GMT+7). CV updates and saved location edits automatically refresh it. Known Indonesian cities resolve to Jakarta, Makassar or Jayapura time; explicit IANA zone names are also supported. Country-only, conflicting or unsupported locations remain unknown rather than defaulting to Jakarta. Settings shows the resolved zone.

Job cards and details show conversions into your resolved time zone for posting time windows with explicit UTC/GMT offsets, common fixed abbreviations (CET/CEST, EST/EDT, PST/PDT), or IANA zones. Overnight windows show the date change. A stated window may describe overlap rather than a complete shift; its excerpt appears in job details.

When only a time zone is mentioned, the UI labels 09:00–17:00 conversions as hypothetical examples. EMEA uses London, Berlin and Dubai examples; Europe uses London and Berlin. These examples do not establish the employer's hours, hiring eligibility or required overlap. Named zones follow the current date's daylight saving rules; explicit offsets and abbreviations stay fixed. Jobs without enough information show unknown hours. Time-zone information does not change job filtering or ranking.


## Excluding a company from job cards

Click **Exclude company** on a job card to add that employer to Settings → Job preferences → Excluded companies. All its jobs disappear from default results, and a top-right notification confirms the change. Application history is preserved. To undo, remove the name from Excluded companies and save preferences. The explorer's **Include excluded or unconfirmed roles** option can still show these jobs with a Company excluded label. This is your personal exclusion, not a verified finding about the company's legitimacy.

The authenticated `POST /api/companies/{id}/exclude` endpoint updates only company exclusions, deduplicates names case-insensitively, and serializes concurrent updates. Existing preference limits apply.
