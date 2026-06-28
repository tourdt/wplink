CREATE SEQUENCE IF NOT EXISTS global_tsid_seq AS bigint MINVALUE 0 MAXVALUE 4194303 CYCLE;

CREATE OR REPLACE FUNCTION next_tsid()
RETURNS bigint
LANGUAGE plpgsql
AS $$
DECLARE
  ts_millis bigint;
  seq_value bigint;
BEGIN
  -- 42 位毫秒时间戳 + 22 位序列，生成正数 BIGINT，便于 PostgreSQL B-tree 按时间趋势写入。
  ts_millis := floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint - 1767225600000;
  seq_value := nextval('global_tsid_seq') % 4194304;
  RETURN (ts_millis << 22) | seq_value;
END;
$$;

CREATE TABLE IF NOT EXISTS users (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  phone varchar(32) UNIQUE,
  wechat_openid varchar(128) UNIQUE,
  nickname varchar(64),
  avatar_url text,
  default_city_station_id bigint,
  status varchar(32) NOT NULL DEFAULT 'active',
  last_login_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT chk_users_identity CHECK (phone IS NOT NULL OR wechat_openid IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS roles (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  code varchar(64) UNIQUE NOT NULL,
  name varchar(64) NOT NULL,
  description text,
  permissions jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_role_assignments (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint NOT NULL REFERENCES users(id),
  role_id bigint NOT NULL REFERENCES roles(id),
  city_station_id bigint,
  merchant_id bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, role_id, city_station_id, merchant_id)
);

CREATE TABLE IF NOT EXISTS admin_operator_profiles (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint NOT NULL UNIQUE REFERENCES users(id),
  real_name varchar(64) NOT NULL,
  managed_city_station_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
  status varchar(32) NOT NULL DEFAULT 'active',
  last_login_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS admin_login_credentials (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint NOT NULL UNIQUE REFERENCES users(id),
  login_name varchar(64) UNIQUE NOT NULL,
  password_hash text NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'enabled',
  failed_attempts integer NOT NULL DEFAULT 0,
  locked_until timestamptz,
  password_changed_at timestamptz,
  last_login_at timestamptz,
  created_by bigint REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS operation_logs (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  operator_id bigint NOT NULL REFERENCES users(id),
  operator_role varchar(64) NOT NULL,
  action varchar(128) NOT NULL,
  object_type varchar(64) NOT NULL,
  object_id bigint,
  before_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  after_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  ip varchar(64),
  user_agent text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user ON user_role_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_role ON user_role_assignments(role_id);
CREATE INDEX IF NOT EXISTS idx_admin_login_credentials_login_name ON admin_login_credentials(login_name);
CREATE INDEX IF NOT EXISTS idx_admin_login_credentials_status ON admin_login_credentials(status);
CREATE INDEX IF NOT EXISTS idx_operation_logs_operator ON operation_logs(operator_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_object ON operation_logs(object_type, object_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_created_at ON operation_logs(created_at);

INSERT INTO roles (code, name, description, permissions)
VALUES
  ('normal_user', '普通用户', '可浏览、搜索、联系和提交需求', '[]'::jsonb),
  ('merchant_admin', '商家管理员', '可管理商家主页和资源发布', '[]'::jsonb),
  ('platform_operator', '平台运营', '可审核、代发、撮合和发放基础权益', '[]'::jsonb),
  ('super_admin', '超级管理员', '可管理运营账号、平台配置和全部后台能力', '[]'::jsonb)
ON CONFLICT (code) DO NOTHING;
