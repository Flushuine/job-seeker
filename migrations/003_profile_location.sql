ALTER TABLE profiles ADD COLUMN summary text NOT NULL DEFAULT '', ADD COLUMN cv_source text NOT NULL DEFAULT '';
ALTER TABLE job_search_preferences ADD COLUMN nearby_locations jsonb NOT NULL DEFAULT '[]', ADD COLUMN hard_location boolean NOT NULL DEFAULT false, ADD COLUMN require_location_match boolean NOT NULL DEFAULT false;
-- NULL means the source did not supply country restrictions. An explicitly
-- empty list is interpreted as worldwide only when the provider confirms it.
ALTER TABLE jobs ADD COLUMN eligible_countries jsonb, ADD COLUMN worldwide boolean;
ALTER TABLE company_sources ADD COLUMN country text NOT NULL DEFAULT '';
ALTER TABLE company_sources DROP CONSTRAINT company_sources_provider_board_key;
ALTER TABLE company_sources ADD CONSTRAINT company_sources_provider_board_country_key UNIQUE(provider,board,country);
