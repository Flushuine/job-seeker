ALTER TABLE companies ADD COLUMN profile jsonb;
ALTER TABLE companies ADD CONSTRAINT companies_profile_object CHECK (profile IS NULL OR jsonb_typeof(profile) = 'object');
