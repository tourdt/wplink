DROP INDEX IF EXISTS idx_verifications_expires_at;

ALTER TABLE verifications
  DROP COLUMN IF EXISTS expires_at;
