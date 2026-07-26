DROP INDEX IF EXISTS idx_resource_exposure_events_merchant_created;
DROP INDEX IF EXISTS idx_resource_exposure_events_resource_created;
DROP TABLE IF EXISTS resource_content_audit_runs;
DROP TABLE IF EXISTS resource_exposure_events;
DROP INDEX IF EXISTS idx_resources_audit_retry_due;
ALTER TABLE resources
  DROP COLUMN IF EXISTS audit_last_error,
  DROP COLUMN IF EXISTS audit_retry_at,
  DROP COLUMN IF EXISTS audit_retry_count;
