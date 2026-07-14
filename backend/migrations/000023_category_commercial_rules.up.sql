ALTER TABLE resource_type_configs
  ADD COLUMN IF NOT EXISTS commercial_rules jsonb NOT NULL DEFAULT '{"publish":{"mode":"consume_quota"},"contactUnlock":{"mode":"login_free","priceCent":0,"currency":"CNY","vipFree":false,"repeatUnlockDays":30}}'::jsonb;

UPDATE resource_type_configs
SET commercial_rules = '{"publish":{"mode":"consume_quota"},"contactUnlock":{"mode":"login_free","priceCent":0,"currency":"CNY","vipFree":false,"repeatUnlockDays":30}}'::jsonb
WHERE commercial_rules IS NULL OR commercial_rules = '{}'::jsonb;

CREATE TABLE IF NOT EXISTS resource_contact_unlock_orders (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  buyer_user_id bigint NOT NULL REFERENCES users(id),
  buyer_merchant_id bigint REFERENCES merchants(id),
  type_code varchar(64) NOT NULL,
  out_trade_no varchar(64) UNIQUE NOT NULL,
  price_cent integer NOT NULL,
  currency varchar(16) NOT NULL DEFAULT 'CNY',
  commercial_rules_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  status varchar(32) NOT NULL DEFAULT 'pending',
  paid_at timestamptz,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_resource_contact_unlock_orders_price CHECK (price_cent >= 0),
  CONSTRAINT chk_resource_contact_unlock_orders_status CHECK (status IN ('pending', 'paid', 'closed', 'refunded')),
  CONSTRAINT chk_resource_contact_unlock_orders_currency CHECK (currency = 'CNY')
);

CREATE INDEX IF NOT EXISTS idx_contact_unlock_orders_resource_status
  ON resource_contact_unlock_orders(resource_id, status);
CREATE INDEX IF NOT EXISTS idx_contact_unlock_orders_buyer_status
  ON resource_contact_unlock_orders(buyer_user_id, status);
CREATE INDEX IF NOT EXISTS idx_contact_unlock_orders_out_trade_no
  ON resource_contact_unlock_orders(out_trade_no);

CREATE TABLE IF NOT EXISTS resource_contact_unlocks (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  user_id bigint NOT NULL REFERENCES users(id),
  viewer_merchant_id bigint REFERENCES merchants(id),
  source_type varchar(32) NOT NULL,
  order_id bigint REFERENCES resource_contact_unlock_orders(id),
  starts_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz,
  CONSTRAINT chk_resource_contact_unlocks_source CHECK (source_type IN ('paid', 'vip', 'login_free', 'owner', 'manual')),
  CONSTRAINT chk_resource_contact_unlocks_period CHECK (expires_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_contact_unlocks_resource_user
  ON resource_contact_unlocks(resource_id, user_id, viewer_merchant_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_contact_unlocks_order
  ON resource_contact_unlocks(order_id);
