CREATE TABLE IF NOT EXISTS map_object_reports (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  object_id bigint NOT NULL REFERENCES map_object(id),
  scene_code varchar(64) NOT NULL REFERENCES map_scene(code),
  reporter_user_id bigint NOT NULL REFERENCES users(id),
  report_kind varchar(32) NOT NULL,
  reason_code varchar(64) NOT NULL,
  description varchar(500) NOT NULL DEFAULT '',
  object_snapshot jsonb NOT NULL,
  status varchar(20) NOT NULL DEFAULT 'pending',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  CONSTRAINT chk_map_object_report_kind
    CHECK (report_kind IN ('location_correction', 'risk_report')),
  CONSTRAINT chk_map_object_report_status
    CHECK (status IN ('pending', 'resolved', 'dismissed'))
);

-- 同一用户的同类、同原因反馈保持幂等，避免重复点击或恶意刷高聚合人数。
CREATE UNIQUE INDEX IF NOT EXISTS uniq_open_map_object_report_user_reason
  ON map_object_reports(object_id, reporter_user_id, report_kind, reason_code)
  WHERE status = 'pending';

-- 支持按“点位 + 类型 + 原因 + 时间”统计近 30 天不同反馈用户。
CREATE INDEX IF NOT EXISTS idx_map_object_reports_aggregate
  ON map_object_reports(object_id, report_kind, reason_code, created_at DESC)
  WHERE status = 'pending';

-- pending 记录本身就是异常队列，前期无需为普通反馈建立人工审核流程。
CREATE INDEX IF NOT EXISTS idx_map_object_reports_pending
  ON map_object_reports(created_at DESC, object_id)
  WHERE status = 'pending';
