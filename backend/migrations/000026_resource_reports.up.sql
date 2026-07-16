CREATE TABLE IF NOT EXISTS resource_reports (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  reporter_user_id bigint REFERENCES users(id),
  reason_code varchar(64) NOT NULL,
  reason_text text,
  evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
  status varchar(32) NOT NULL DEFAULT 'pending',
  reviewed_by bigint REFERENCES users(id),
  reviewed_at timestamptz,
  review_action varchar(32),
  review_reason text,
  refund_publish_quota boolean NOT NULL DEFAULT false,
  batch_review_id bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_resource_reports_status CHECK (status IN ('pending', 'valid', 'invalid')),
  CONSTRAINT chk_resource_reports_review_action CHECK (review_action IS NULL OR review_action IN ('valid', 'invalid'))
);

CREATE INDEX IF NOT EXISTS idx_resource_reports_status_created
  ON resource_reports(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_resource_reports_resource_status
  ON resource_reports(resource_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_resource_reports_reporter
  ON resource_reports(reporter_user_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_resource_reports_pending_user
  ON resource_reports(resource_id, reporter_user_id)
  WHERE status = 'pending' AND reporter_user_id IS NOT NULL;
