CREATE TABLE IF NOT EXISTS vip_plans (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  code varchar(64) UNIQUE NOT NULL,
  name varchar(128) NOT NULL,
  duration_months integer NOT NULL,
  standard_price_cent integer NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  display_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_vip_plans_duration CHECK (duration_months > 0),
  CONSTRAINT chk_vip_plans_price CHECK (standard_price_cent >= 0),
  CONSTRAINT chk_vip_plans_status CHECK (status IN ('active', 'inactive'))
);

CREATE TABLE IF NOT EXISTS vip_plan_versions (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  plan_id bigint NOT NULL REFERENCES vip_plans(id),
  version integer NOT NULL,
  benefits jsonb NOT NULL DEFAULT '{}'::jsonb,
  starts_at timestamptz NOT NULL DEFAULT now(),
  ends_at timestamptz,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT uniq_vip_plan_versions_plan_version UNIQUE (plan_id, version),
  CONSTRAINT chk_vip_plan_versions_status CHECK (status IN ('active', 'inactive')),
  CONSTRAINT chk_vip_plan_versions_period CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE TABLE IF NOT EXISTS vip_promotions (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  code varchar(64) UNIQUE NOT NULL,
  plan_id bigint NOT NULL REFERENCES vip_plans(id),
  promotion_type varchar(32) NOT NULL,
  sale_price_cent integer NOT NULL,
  starts_at timestamptz NOT NULL DEFAULT now(),
  ends_at timestamptz,
  quota_limit integer,
  used_count integer NOT NULL DEFAULT 0,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_vip_promotions_price CHECK (sale_price_cent >= 0),
  CONSTRAINT chk_vip_promotions_used_count CHECK (used_count >= 0),
  CONSTRAINT chk_vip_promotions_type CHECK (promotion_type IN ('first_purchase', 'launch')),
  CONSTRAINT chk_vip_promotions_status CHECK (status IN ('active', 'inactive')),
  CONSTRAINT chk_vip_promotions_period CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE TABLE IF NOT EXISTS vip_orders (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  user_id bigint NOT NULL REFERENCES users(id),
  plan_id bigint NOT NULL REFERENCES vip_plans(id),
  plan_version_id bigint NOT NULL REFERENCES vip_plan_versions(id),
  promotion_id bigint REFERENCES vip_promotions(id),
  out_trade_no varchar(64) UNIQUE NOT NULL,
  standard_price_cent integer NOT NULL,
  actual_price_cent integer NOT NULL,
  currency varchar(16) NOT NULL DEFAULT 'CNY',
  benefits_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  status varchar(32) NOT NULL DEFAULT 'pending',
  transaction_id varchar(128),
  notify_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  paid_at timestamptz,
  closed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_vip_orders_price CHECK (standard_price_cent >= 0 AND actual_price_cent >= 0),
  CONSTRAINT chk_vip_orders_status CHECK (status IN ('pending', 'paid', 'closed', 'refunded'))
);

CREATE TABLE IF NOT EXISTS merchant_vip_subscriptions (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  plan_id bigint NOT NULL REFERENCES vip_plans(id),
  order_id bigint NOT NULL REFERENCES vip_orders(id),
  status varchar(32) NOT NULL DEFAULT 'active',
  source_type varchar(32) NOT NULL DEFAULT 'paid',
  starts_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_merchant_vip_subscriptions_status CHECK (status IN ('active', 'expired', 'cancelled')),
  CONSTRAINT chk_merchant_vip_subscriptions_source CHECK (source_type IN ('paid', 'manual', 'promo')),
  CONSTRAINT chk_merchant_vip_subscriptions_period CHECK (expires_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_vip_plan_versions_plan_status
  ON vip_plan_versions(plan_id, status);
CREATE INDEX IF NOT EXISTS idx_vip_promotions_plan_status
  ON vip_promotions(plan_id, status);
CREATE INDEX IF NOT EXISTS idx_vip_orders_merchant_status
  ON vip_orders(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_vip_orders_out_trade_no
  ON vip_orders(out_trade_no);
CREATE INDEX IF NOT EXISTS idx_merchant_vip_subscriptions_merchant_status
  ON merchant_vip_subscriptions(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_merchant_vip_subscriptions_expires_at
  ON merchant_vip_subscriptions(expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_active_merchant_vip_subscription
  ON merchant_vip_subscriptions(merchant_id)
  WHERE status = 'active';

INSERT INTO vip_plans (code, name, duration_months, standard_price_cent, status, display_order)
VALUES
  ('monthly', 'VIP 月卡', 1, 4900, 'active', 10),
  ('half_year', 'VIP 半年卡', 6, 29900, 'active', 20),
  ('yearly', 'VIP 年卡', 12, 49900, 'active', 30)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  duration_months = EXCLUDED.duration_months,
  standard_price_cent = EXCLUDED.standard_price_cent,
  status = EXCLUDED.status,
  display_order = EXCLUDED.display_order,
  updated_at = now();

INSERT INTO vip_plan_versions (plan_id, version, benefits, status)
SELECT
  id,
  1,
  '{"publishPolicy":"quota","publishQuota":80,"refreshQuota":30,"topVoucherCount":3,"topDurationHours":24,"homepageImageLimit":18}'::jsonb,
  'active'
FROM vip_plans
WHERE code IN ('monthly', 'half_year', 'yearly')
ON CONFLICT (plan_id, version) DO UPDATE SET
  benefits = EXCLUDED.benefits,
  status = EXCLUDED.status;

INSERT INTO vip_promotions (code, plan_id, promotion_type, sale_price_cent, starts_at, ends_at, status)
SELECT 'launch_monthly_first', id, 'first_purchase', 1990, now(), now() + interval '90 days', 'active'
FROM vip_plans
WHERE code = 'monthly'
ON CONFLICT (code) DO UPDATE SET
  plan_id = EXCLUDED.plan_id,
  promotion_type = EXCLUDED.promotion_type,
  sale_price_cent = EXCLUDED.sale_price_cent,
  starts_at = EXCLUDED.starts_at,
  ends_at = EXCLUDED.ends_at,
  status = EXCLUDED.status,
  updated_at = now();

INSERT INTO vip_promotions (code, plan_id, promotion_type, sale_price_cent, starts_at, ends_at, status)
SELECT 'launch_half_year', id, 'launch', 19900, now(), now() + interval '90 days', 'active'
FROM vip_plans
WHERE code = 'half_year'
ON CONFLICT (code) DO UPDATE SET
  plan_id = EXCLUDED.plan_id,
  promotion_type = EXCLUDED.promotion_type,
  sale_price_cent = EXCLUDED.sale_price_cent,
  starts_at = EXCLUDED.starts_at,
  ends_at = EXCLUDED.ends_at,
  status = EXCLUDED.status,
  updated_at = now();

INSERT INTO vip_promotions (code, plan_id, promotion_type, sale_price_cent, starts_at, ends_at, status)
SELECT 'launch_yearly', id, 'launch', 29900, now(), now() + interval '90 days', 'active'
FROM vip_plans
WHERE code = 'yearly'
ON CONFLICT (code) DO UPDATE SET
  plan_id = EXCLUDED.plan_id,
  promotion_type = EXCLUDED.promotion_type,
  sale_price_cent = EXCLUDED.sale_price_cent,
  starts_at = EXCLUDED.starts_at,
  ends_at = EXCLUDED.ends_at,
  status = EXCLUDED.status,
  updated_at = now();
