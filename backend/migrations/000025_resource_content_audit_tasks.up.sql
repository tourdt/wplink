CREATE TABLE IF NOT EXISTS resource_content_audit_tasks (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  trace_id varchar(128) UNIQUE NOT NULL,
  audit_type varchar(32) NOT NULL,
  media_url text NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'pending',
  suggest varchar(32),
  label integer,
  reason text,
  raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  CONSTRAINT chk_resource_content_audit_tasks_type CHECK (audit_type IN ('image')),
  CONSTRAINT chk_resource_content_audit_tasks_status CHECK (status IN ('pending', 'pass', 'rejected', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_resource_content_audit_tasks_resource_status
  ON resource_content_audit_tasks(resource_id, status);

CREATE INDEX IF NOT EXISTS idx_resource_content_audit_tasks_trace
  ON resource_content_audit_tasks(trace_id);
