DROP INDEX IF EXISTS idx_merchants_profile_status;

ALTER TABLE merchants
  DROP CONSTRAINT IF EXISTS chk_merchants_profile_status;

ALTER TABLE merchants
  DROP COLUMN IF EXISTS profile_status;
