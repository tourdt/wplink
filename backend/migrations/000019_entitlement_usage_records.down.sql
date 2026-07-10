CREATE TABLE IF NOT EXISTS top_vouchers (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  entitlement_id bigint REFERENCES merchant_entitlements(id),
  source_type varchar(64) NOT NULL,
  allowed_type_codes jsonb NOT NULL DEFAULT '[]'::jsonb,
  top_duration_hours integer NOT NULL,
  used_resource_id bigint REFERENCES resources(id),
  used_at timestamptz,
  expires_at timestamptz,
  status varchar(32) NOT NULL DEFAULT 'unused',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_top_vouchers_merchant_status
  ON top_vouchers(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_top_vouchers_used_resource
  ON top_vouchers(used_resource_id);

DROP INDEX IF EXISTS idx_entitlement_usage_resource;
DROP INDEX IF EXISTS idx_entitlement_usage_merchant;
DROP INDEX IF EXISTS idx_entitlement_usage_entitlement;
DROP TABLE IF EXISTS merchant_entitlement_usage_records;

ALTER TABLE IF EXISTS merchant_entitlements
  DROP COLUMN IF EXISTS top_duration_hours,
  DROP COLUMN IF EXISTS allowed_type_codes;

DROP INDEX IF EXISTS idx_resources_top_expires_at;

ALTER TABLE IF EXISTS resources
  DROP COLUMN IF EXISTS top_expires_at,
  DROP COLUMN IF EXISTS top_started_at;
