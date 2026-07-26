ALTER TABLE resources
  ADD COLUMN IF NOT EXISTS audit_retry_count integer NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS audit_retry_at timestamptz,
  ADD COLUMN IF NOT EXISTS audit_last_error text;

CREATE INDEX IF NOT EXISTS idx_resources_audit_retry_due
  ON resources(audit_retry_at, updated_at)
  WHERE status = 'audit_retry' AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS resource_exposure_events (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  user_id bigint REFERENCES users(id),
  visitor_key varchar(96) NOT NULL,
  session_id varchar(96) NOT NULL,
  source varchar(32) NOT NULL,
  visible_duration_ms integer NOT NULL DEFAULT 0,
  exposure_date date NOT NULL DEFAULT CURRENT_DATE,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_resource_exposure_source CHECK (source IN ('home', 'list', 'search', 'topic', 'merchant')),
  CONSTRAINT chk_resource_exposure_duration CHECK (visible_duration_ms >= 0),
  CONSTRAINT uniq_resource_exposure_daily UNIQUE (resource_id, visitor_key, source, exposure_date)
);

CREATE INDEX IF NOT EXISTS idx_resource_exposure_events_resource_created
  ON resource_exposure_events(resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_resource_exposure_events_merchant_created
  ON resource_exposure_events(merchant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS resource_content_audit_runs (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  attempt_no integer NOT NULL DEFAULT 0,
  audit_action varchar(32) NOT NULL,
  decision varchar(32) NOT NULL,
  reason text,
  labels jsonb NOT NULL DEFAULT '[]'::jsonb,
  trace_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_resource_content_audit_run_action CHECK (audit_action IN ('create_resource', 'submit_resource', 'retry_resource_audit', 'media_callback')),
  CONSTRAINT chk_resource_content_audit_run_decision CHECK (decision IN ('pass', 'review_relaxed', 'risky', 'dependency_error', 'manual_review'))
);

CREATE INDEX IF NOT EXISTS idx_resource_content_audit_runs_resource_created
  ON resource_content_audit_runs(resource_id, created_at DESC);
