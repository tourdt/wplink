UPDATE vip_quota_packs
SET
  name = replace(name, '置顶服务', '置顶券'),
  description = replace(description, '购买后可置顶', '单张可置顶'),
  updated_at = now()
WHERE code IN ('top_1d', 'top_3d', 'top_5d', 'top_7d', 'top_15d', 'top_30d')
  AND benefits ? 'topVoucherCount';
