ALTER TABLE merchants
  ADD COLUMN IF NOT EXISTS profile_status varchar(32) NOT NULL DEFAULT 'completed';

UPDATE merchants
SET profile_status = 'completed'
WHERE profile_status IS NULL OR profile_status = '';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'chk_merchants_profile_status'
  ) THEN
    ALTER TABLE merchants
      ADD CONSTRAINT chk_merchants_profile_status
      CHECK (profile_status IN ('incomplete', 'completed'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_merchants_profile_status
  ON merchants(profile_status);
