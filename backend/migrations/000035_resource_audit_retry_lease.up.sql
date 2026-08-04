ALTER TABLE resources
    ADD COLUMN audit_lease_until TIMESTAMPTZ,
    ADD COLUMN audit_processing_by VARCHAR(128);

COMMENT ON COLUMN resources.audit_lease_until IS '内容审核重试租约到期时间；到期后允许其他实例重新领取';
COMMENT ON COLUMN resources.audit_processing_by IS '当前内容审核重试实例标识';

DROP INDEX IF EXISTS idx_resources_audit_retry_due;
CREATE INDEX idx_resources_audit_retry_due
    ON resources (audit_retry_at, audit_lease_until, updated_at)
    WHERE status = 'audit_retry'
      AND deleted_at IS NULL;
