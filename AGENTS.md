# Project instructions

## Required project skills

Before starting a task in this repository, read and apply:

- [Ponytail](.agents/skills/ponytail/SKILL.md): use **full** by default for coding, fixes, reviews, and dependency decisions. Make the smallest complete change; reuse existing code before adding abstractions or dependencies.
- [Caveman](.agents/skills/caveman/SKILL.md): use concise, direct chat replies by default. Preserve technical facts, commands, errors, and necessary context. Keep persisted code, comments, documentation, and product text clear and grammatical.

These are project defaults. Follow the user's current instructions when they request another mode, more detail, or disable a skill. Skills guide engineering and communication; they must not alter application matching rules or user-facing content solely to fit a style. Only the two core skills are installed; do not claim missing upstream modes are available.

## Project guardrails

- Keep the application free and local-first. Use optional native Ollama, with one inference request at a time, bounded context, and deterministic fallback. Do not download models without the user's explicit authorization.
- Preserve evidence sources and unknown information. Never invent jobs, company facts, reviews, salary data, or employment arrangements.
- Keep secrets, the CV, and local database files out of tracked changes. Do not print `.env` values unless the user explicitly requests them.
- Respect user-managed development processes. Do not start another persistent API on port 8080 when it is occupied. Stop temporary verification servers when finished.
- Run checks appropriate to the change. Use `make test` for Go tests and `npm --prefix web run build` for frontend changes. Database integration tests require an isolated `TEST_DATABASE_URL`, never the application database. Documentation-only changes need link and content checks, not application tests.
