CREATE TABLE IF NOT EXISTS admin_login_attempts (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  operator_id bigint REFERENCES admin_operators(id),
  login_name varchar(64) NOT NULL,
  client_ip varchar(64) NOT NULL,
  user_agent text,
  result varchar(32) NOT NULL,
  failure_reason varchar(64),
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_admin_login_attempts_result CHECK (result IN ('success', 'failed', 'blocked'))
);

CREATE INDEX IF NOT EXISTS idx_admin_login_attempts_ip_created
  ON admin_login_attempts(client_ip, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_login_attempts_login_created
  ON admin_login_attempts(login_name, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_security_alerts (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  alert_type varchar(64) NOT NULL,
  severity varchar(16) NOT NULL DEFAULT 'high',
  event_key varchar(192) NOT NULL UNIQUE,
  status varchar(32) NOT NULL DEFAULT 'pending',
  details jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  CONSTRAINT chk_admin_security_alert_severity CHECK (severity IN ('medium', 'high', 'critical')),
  CONSTRAINT chk_admin_security_alert_status CHECK (status IN ('pending', 'resolved', 'ignored'))
);

CREATE INDEX IF NOT EXISTS idx_admin_security_alerts_pending_created
  ON admin_security_alerts(status, created_at DESC);

CREATE TABLE IF NOT EXISTS user_consents (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint NOT NULL REFERENCES users(id),
  consent_type varchar(32) NOT NULL,
  version varchar(32) NOT NULL,
  source varchar(32) NOT NULL DEFAULT 'wechat_mini_program',
  consented_at timestamptz NOT NULL DEFAULT now(),
  withdrawn_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_user_consents_type CHECK (consent_type IN ('privacy_policy', 'user_agreement')),
  CONSTRAINT uniq_user_consent_version UNIQUE (user_id, consent_type, version)
);

CREATE INDEX IF NOT EXISTS idx_user_consents_user_active
  ON user_consents(user_id, consent_type, consented_at DESC)
  WHERE withdrawn_at IS NULL;

CREATE TABLE IF NOT EXISTS user_account_deletion_requests (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint NOT NULL REFERENCES users(id),
  status varchar(32) NOT NULL DEFAULT 'completed',
  reason text,
  requested_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT chk_user_account_deletion_status CHECK (status IN ('pending', 'completed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_user_account_deletion_user_requested
  ON user_account_deletion_requests(user_id, requested_at DESC);
