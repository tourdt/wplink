CREATE TABLE IF NOT EXISTS merchant_map_events (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint REFERENCES users(id) ON DELETE SET NULL,
  merchant_id bigint NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
  target_merchant_id bigint REFERENCES merchants(id) ON DELETE SET NULL,
  visitor_key varchar(96) NOT NULL,
  session_id varchar(96) NOT NULL,
  event_type varchar(40) NOT NULL,
  source varchar(30) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_merchant_map_event_type CHECK (event_type IN (
    'location_entry_click', 'location_view', 'navigation_click', 'nearby_drawer_open',
    'nearby_marker_click', 'nearby_list_item_click', 'nearby_merchant_click'
  )),
  CONSTRAINT chk_merchant_map_event_source CHECK (source IN (
    'directory', 'merchant_detail', 'merchant_location'
  ))
);

-- 支持按入口商家、行为和发生时间汇总地图转化漏斗。
CREATE INDEX IF NOT EXISTS idx_merchant_map_events_merchant_event_created
  ON merchant_map_events(merchant_id, event_type, created_at DESC);

-- 周边商家点击需要按目标商家聚合；空目标不进入该索引，减少无效索引项。
CREATE INDEX IF NOT EXISTS idx_merchant_map_events_target_created
  ON merchant_map_events(target_merchant_id, created_at DESC)
  WHERE target_merchant_id IS NOT NULL;

-- 账号软注销需要按用户快速定位并匿名化历史事件；匿名记录不进入索引。
CREATE INDEX IF NOT EXISTS idx_merchant_map_events_user_created
  ON merchant_map_events(user_id, created_at DESC)
  WHERE user_id IS NOT NULL;
