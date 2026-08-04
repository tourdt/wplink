-- 000032 只能回填迁移执行前已经存在的商家；后续导入的旧种子可能仍缺少正式入驻时间。
-- 使用创建时间恢复历史顺序，避免把旧商家误排成修复当天的新入驻商家。
UPDATE merchants
SET onboarded_at = created_at
WHERE profile_status = 'completed'
  AND onboarded_at IS NULL;
