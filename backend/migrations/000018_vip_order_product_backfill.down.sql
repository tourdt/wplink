ALTER TABLE IF EXISTS vip_orders
  DROP CONSTRAINT IF EXISTS chk_vip_orders_product_ref,
  DROP CONSTRAINT IF EXISTS chk_vip_orders_product_type;

DROP INDEX IF EXISTS idx_vip_orders_product_type;
DROP INDEX IF EXISTS idx_vip_quota_packs_status;

ALTER TABLE IF EXISTS vip_orders
  DROP COLUMN IF EXISTS product_snapshot,
  DROP COLUMN IF EXISTS product_name,
  DROP COLUMN IF EXISTS product_code,
  DROP COLUMN IF EXISTS product_type;

DROP TABLE IF EXISTS vip_quota_packs;
