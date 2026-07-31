ALTER TABLE merchants
  ADD COLUMN IF NOT EXISTS onboarded_at timestamptz;

-- 历史完成资料的商家没有独立入驻时间，使用创建时间回填，保证升级后首页立即有可展示数据。
UPDATE merchants
SET onboarded_at = created_at
WHERE profile_status = 'completed'
  AND onboarded_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_merchants_home_recent
  ON merchants(city_station_id, onboarded_at DESC, id DESC)
  WHERE deleted_at IS NULL
    AND status = 'active'
    AND profile_status = 'completed'
    AND onboarded_at IS NOT NULL;
