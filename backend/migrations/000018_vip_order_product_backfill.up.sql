-- 旧开发库如果已经执行过早期 000017，vip_orders 会缺少统一权益商品字段。
-- 这里用幂等 ALTER 补齐字段，避免服务启动后查询 product_type 时报错。
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

ALTER TABLE IF EXISTS vip_orders
  ADD COLUMN IF NOT EXISTS product_type varchar(32) NOT NULL DEFAULT 'vip_plan',
  ADD COLUMN IF NOT EXISTS product_code varchar(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS product_name varchar(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS product_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE IF EXISTS vip_orders
  ALTER COLUMN plan_id DROP NOT NULL,
  ALTER COLUMN plan_version_id DROP NOT NULL;

UPDATE vip_orders o
SET
  product_type = 'vip_plan',
  product_code = COALESCE(NULLIF(o.product_code, ''), p.code),
  product_name = COALESCE(NULLIF(o.product_name, ''), p.name),
  product_snapshot = CASE
    WHEN o.product_snapshot = '{}'::jsonb THEN jsonb_build_object('planCode', p.code, 'planName', p.name)
    ELSE o.product_snapshot
  END
FROM vip_plans p
WHERE o.plan_id = p.id
  AND o.product_type = 'vip_plan';

DO $$
BEGIN
  IF to_regclass('vip_orders') IS NOT NULL THEN
    IF NOT EXISTS (
      SELECT 1
      FROM pg_constraint
      WHERE conname = 'chk_vip_orders_product_type'
        AND conrelid = 'vip_orders'::regclass
    ) THEN
      ALTER TABLE vip_orders
        ADD CONSTRAINT chk_vip_orders_product_type CHECK (product_type IN ('vip_plan', 'quota_pack'));
    END IF;

    IF NOT EXISTS (
      SELECT 1
      FROM pg_constraint
      WHERE conname = 'chk_vip_orders_product_ref'
        AND conrelid = 'vip_orders'::regclass
    ) THEN
      ALTER TABLE vip_orders
        ADD CONSTRAINT chk_vip_orders_product_ref CHECK (
          (product_type = 'vip_plan' AND plan_id IS NOT NULL AND plan_version_id IS NOT NULL)
          OR (product_type = 'quota_pack' AND product_code <> '')
        );
    END IF;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_vip_quota_packs_status
  ON vip_quota_packs(status, display_order);
CREATE INDEX IF NOT EXISTS idx_vip_orders_product_type
  ON vip_orders(product_type, product_code, status);

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
  ('top_3', '置顶券包', '单张可置顶 24 小时', 2900, 2900, '限时特价', '{"topVoucherCount":3,"topDurationHours":24}'::jsonb, 'active', 30)
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
