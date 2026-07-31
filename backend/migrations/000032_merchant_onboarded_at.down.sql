DROP INDEX IF EXISTS idx_merchants_home_recent;

ALTER TABLE merchants
  DROP COLUMN IF EXISTS onboarded_at;
