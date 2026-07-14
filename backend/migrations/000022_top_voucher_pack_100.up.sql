-- 置顶功能重新放开后，默认售卖单张不同天数的置顶券。
-- 已存在旧 1 张/3 张包的环境保留历史订单快照，只停用旧商品，避免用户继续购买旧价格。
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

UPDATE vip_quota_packs
SET status = 'inactive',
    updated_at = now()
WHERE code IN ('top_1', 'top_3');
