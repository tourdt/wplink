ALTER TABLE IF EXISTS resources
  ADD COLUMN IF NOT EXISTS top_started_at timestamptz,
  ADD COLUMN IF NOT EXISTS top_expires_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_resources_top_expires_at
  ON resources(top_expires_at DESC);

ALTER TABLE IF EXISTS merchant_entitlements
  ADD COLUMN IF NOT EXISTS allowed_type_codes jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS top_duration_hours integer NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS merchant_entitlement_usage_records (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  entitlement_id bigint NOT NULL REFERENCES merchant_entitlements(id),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  entitlement_type varchar(64) NOT NULL,
  action_type varchar(64) NOT NULL,
  amount integer NOT NULL DEFAULT 1,
  resource_id bigint REFERENCES resources(id),
  before_remaining_amount integer NOT NULL,
  after_remaining_amount integer NOT NULL,
  snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  used_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_entitlement_usage_entitlement
  ON merchant_entitlement_usage_records(entitlement_id, used_at DESC);
CREATE INDEX IF NOT EXISTS idx_entitlement_usage_merchant
  ON merchant_entitlement_usage_records(merchant_id, used_at DESC);
CREATE INDEX IF NOT EXISTS idx_entitlement_usage_resource
  ON merchant_entitlement_usage_records(resource_id);

DROP INDEX IF EXISTS idx_top_vouchers_used_resource;
DROP INDEX IF EXISTS idx_top_vouchers_merchant_status;
DROP TABLE IF EXISTS top_vouchers;
