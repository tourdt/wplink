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

CREATE TABLE IF NOT EXISTS vip_quota_packs (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  code varchar(64) UNIQUE NOT NULL,
  name varchar(128) NOT NULL,
  description varchar(255) NOT NULL DEFAULT '',
  standard_price_cent integer NOT NULL,
  sale_price_cent integer,
  sale_label varchar(64),
  benefits jsonb NOT NULL DEFAULT '{}'::jsonb,
  status varchar(32) NOT NULL DEFAULT 'active',
  display_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_vip_quota_packs_price CHECK (
    standard_price_cent >= 0
    AND (sale_price_cent IS NULL OR sale_price_cent >= 0)
  ),
  CONSTRAINT chk_vip_quota_packs_status CHECK (status IN ('active', 'inactive'))
);

CREATE TABLE IF NOT EXISTS vip_orders (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  user_id bigint NOT NULL REFERENCES users(id),
  product_type varchar(32) NOT NULL DEFAULT 'vip_plan',
  product_code varchar(64) NOT NULL DEFAULT '',
  product_name varchar(128) NOT NULL DEFAULT '',
  plan_id bigint REFERENCES vip_plans(id),
  plan_version_id bigint REFERENCES vip_plan_versions(id),
  promotion_id bigint REFERENCES vip_promotions(id),
  out_trade_no varchar(64) UNIQUE NOT NULL,
  standard_price_cent integer NOT NULL,
  actual_price_cent integer NOT NULL,
  currency varchar(16) NOT NULL DEFAULT 'CNY',
  benefits_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  product_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  status varchar(32) NOT NULL DEFAULT 'pending',
  transaction_id varchar(128),
  notify_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  paid_at timestamptz,
  closed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_vip_orders_price CHECK (standard_price_cent >= 0 AND actual_price_cent >= 0),
  CONSTRAINT chk_vip_orders_product_type CHECK (product_type IN ('vip_plan', 'quota_pack')),
  CONSTRAINT chk_vip_orders_product_ref CHECK (
    (product_type = 'vip_plan' AND plan_id IS NOT NULL AND plan_version_id IS NOT NULL)
    OR (product_type = 'quota_pack' AND product_code <> '')
  ),
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
CREATE INDEX IF NOT EXISTS idx_vip_quota_packs_status
  ON vip_quota_packs(status, display_order);
CREATE INDEX IF NOT EXISTS idx_vip_orders_merchant_status
  ON vip_orders(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_vip_orders_out_trade_no
  ON vip_orders(out_trade_no);
CREATE INDEX IF NOT EXISTS idx_vip_orders_product_type
  ON vip_orders(product_type, product_code, status);
CREATE INDEX IF NOT EXISTS idx_vip_orders_pending_created
  ON vip_orders(created_at)
  WHERE status = 'pending';
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

INSERT INTO vip_quota_packs (
  code,
  name,
  description,
  standard_price_cent,
  sale_price_cent,
  sale_label,
  benefits,
  status,
  display_order
)
VALUES
  ('publish_5', '发布次数包', '临时多发供需', 2500, 2500, '限时特价', '{"publishQuota":5}'::jsonb, 'active', 10),
  ('refresh_10', '刷新次数包', '让信息回到前面', 1900, 1900, '限时特价', '{"refreshQuota":10}'::jsonb, 'active', 20),
  ('top_1d', '1天置顶券', '单张可置顶 1 天', 10000, 10000, '置顶 1 天', '{"topVoucherCount":1,"topDurationHours":24}'::jsonb, 'active', 30),
  ('top_3d', '3天置顶券', '单张可置顶 3 天', 20000, 20000, '置顶 3 天', '{"topVoucherCount":1,"topDurationHours":72}'::jsonb, 'active', 40),
  ('top_5d', '5天置顶券', '单张可置顶 5 天', 30000, 30000, '置顶 5 天', '{"topVoucherCount":1,"topDurationHours":120}'::jsonb, 'active', 50),
  ('top_7d', '7天置顶券', '单张可置顶 7 天', 40000, 40000, '置顶 7 天', '{"topVoucherCount":1,"topDurationHours":168}'::jsonb, 'active', 60),
  ('top_15d', '15天置顶券', '单张可置顶 15 天', 60000, 60000, '置顶 15 天', '{"topVoucherCount":1,"topDurationHours":360}'::jsonb, 'active', 70),
  ('top_30d', '30天置顶券', '单张可置顶 30 天', 90000, 90000, '置顶 30 天', '{"topVoucherCount":1,"topDurationHours":720}'::jsonb, 'active', 80)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  standard_price_cent = EXCLUDED.standard_price_cent,
  sale_price_cent = EXCLUDED.sale_price_cent,
  sale_label = EXCLUDED.sale_label,
  benefits = EXCLUDED.benefits,
  status = EXCLUDED.status,
  display_order = EXCLUDED.display_order,
  updated_at = now();
