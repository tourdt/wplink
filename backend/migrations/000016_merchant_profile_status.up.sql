-- profile_status 列及约束由 000002 核心迁移拥有；本迁移只补充独立查询索引。
CREATE INDEX IF NOT EXISTS idx_merchants_profile_status
  ON merchants(profile_status);
