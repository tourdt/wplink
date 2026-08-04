DROP INDEX IF EXISTS idx_resources_audit_retry_due;

ALTER TABLE resources
    DROP COLUMN IF EXISTS audit_processing_by,
    DROP COLUMN IF EXISTS audit_lease_until;

CREATE INDEX idx_resources_audit_retry_due
    ON resources (audit_retry_at, updated_at)
    WHERE status = 'audit_retry'
      AND deleted_at IS NULL;
