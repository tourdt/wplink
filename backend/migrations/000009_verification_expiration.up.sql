ALTER TABLE verifications
  ADD COLUMN IF NOT EXISTS expires_at timestamptz;

UPDATE verifications
SET expires_at = COALESCE(reviewed_at, updated_at, now()) + interval '1 year'
WHERE status = 'verified'
  AND expires_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_verifications_expires_at
  ON verifications(expires_at)
  WHERE status = 'verified';
