CREATE TABLE cv_uploads (
 id uuid PRIMARY KEY,
 profile_id uuid NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 filename text NOT NULL,
 fingerprint text NOT NULL,
 extracted_text text NOT NULL,
 base_profile_fingerprint text NOT NULL,
 model text NOT NULL,
 prompt_version text NOT NULL,
 ai_context integer NOT NULL,
 analysis jsonb,
 applied_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX cv_uploads_cache_idx ON cv_uploads(profile_id,fingerprint,model,prompt_version,ai_context);
ALTER TABLE profiles ADD COLUMN latest_cv_upload_id uuid REFERENCES cv_uploads(id), ADD COLUMN cv_fingerprint text NOT NULL DEFAULT '';
ALTER TABLE analysis_tasks DROP CONSTRAINT analysis_tasks_kind_check;
ALTER TABLE analysis_tasks ADD CONSTRAINT analysis_tasks_kind_check CHECK(kind IN ('job','company','cv'));
