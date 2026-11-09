ALTER TABLE job_search_preferences ADD COLUMN ai_enabled boolean NOT NULL DEFAULT true, ADD COLUMN ollama_model text NOT NULL DEFAULT 'qwen3:1.7b', ADD COLUMN ai_context integer NOT NULL DEFAULT 4096, ADD COLUMN ai_keep_alive text NOT NULL DEFAULT '1m';
ALTER TABLE company_sources DROP CONSTRAINT company_sources_provider_check;
ALTER TABLE company_sources ADD CONSTRAINT company_sources_provider_check CHECK(provider IN ('greenhouse','lever','lever-eu','himalayas'));
ALTER TABLE company_sources ADD COLUMN last_fetched_at timestamptz;
ALTER TABLE jobs ADD COLUMN source_name text NOT NULL DEFAULT '', ADD COLUMN source_url text NOT NULL DEFAULT '', ADD COLUMN application_url text NOT NULL DEFAULT '', ADD COLUMN expires_at timestamptz;
CREATE TABLE analysis_tasks (id uuid PRIMARY KEY, resource_id uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('job','company')), status text NOT NULL CHECK(status IN ('queued','running','completed','failed')), error text NOT NULL DEFAULT '', attempts integer NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(kind,resource_id));
CREATE INDEX analysis_tasks_queue_idx ON analysis_tasks(status,created_at);
ALTER TABLE company_research ADD COLUMN interpretation jsonb;
