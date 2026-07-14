-- 回滚时恢复旧的 3 张置顶券包，并停用新的多天数置顶券商品。
UPDATE vip_quota_packs
SET status = 'inactive',
    updated_at = now()
WHERE code IN ('top_1d', 'top_3d', 'top_5d', 'top_7d', 'top_15d', 'top_30d');

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
