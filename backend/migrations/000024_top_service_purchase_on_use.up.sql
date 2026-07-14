-- 单独售卖的置顶商品改为“置顶服务”，购买时必须绑定具体资源，支付后立即生效。
-- 内部仍复用 top_voucher 权益类型作为核销与审计账本，不再把单独购买商品展示成可提前囤积的券。
UPDATE vip_quota_packs
SET
  name = replace(name, '置顶券', '置顶服务'),
  description = replace(description, '单张可置顶', '购买后可置顶'),
  updated_at = now()
WHERE code IN ('top_1d', 'top_3d', 'top_5d', 'top_7d', 'top_15d', 'top_30d')
  AND benefits ? 'topVoucherCount';
