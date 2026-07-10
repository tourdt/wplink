CREATE TABLE IF NOT EXISTS growth_campaigns (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  code varchar(64) NOT NULL UNIQUE,
  name varchar(128) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'draft',
  starts_at timestamptz NOT NULL DEFAULT now(),
  ends_at timestamptz,
  city_scope jsonb NOT NULL DEFAULT '[]'::jsonb,
  user_scope jsonb NOT NULL DEFAULT '[]'::jsonb,
  total_reward_limit integer,
  daily_reward_limit integer,
  config_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_by bigint REFERENCES admin_operator_profiles(id),
  updated_by bigint REFERENCES admin_operator_profiles(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (status IN ('draft', 'active', 'paused', 'ended', 'disabled')),
  CHECK (ends_at IS NULL OR ends_at > starts_at),
  CHECK (total_reward_limit IS NULL OR total_reward_limit > 0),
  CHECK (daily_reward_limit IS NULL OR daily_reward_limit > 0)
);

CREATE TABLE IF NOT EXISTS growth_campaign_rules (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  campaign_code varchar(64) NOT NULL REFERENCES growth_campaigns(code),
  rule_code varchar(64) NOT NULL,
  rule_name varchar(128) NOT NULL DEFAULT '',
  trigger_event varchar(64) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  priority integer NOT NULL DEFAULT 100,
  conditions jsonb NOT NULL DEFAULT '{}'::jsonb,
  reward_type varchar(64) NOT NULL,
  reward_amount integer NOT NULL,
  valid_days integer NOT NULL,
  per_user_limit integer,
  per_user_daily_limit integer,
  per_resource_daily_limit integer,
  description varchar(255) NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (campaign_code, rule_code),
  CHECK (status IN ('active', 'inactive')),
  CHECK (trigger_event IN ('user_first_login', 'resource_first_approved', 'resource_approved_count_reached', 'resource_share_effective_view', 'resource_share_effective_contact', 'invitee_first_resource_approved')),
  CHECK (reward_type IN ('publish_quota', 'refresh_quota')),
  CHECK (reward_amount > 0),
  CHECK (valid_days > 0),
  CHECK (per_user_limit IS NULL OR per_user_limit > 0),
  CHECK (per_user_daily_limit IS NULL OR per_user_daily_limit > 0),
  CHECK (per_resource_daily_limit IS NULL OR per_resource_daily_limit > 0)
);

CREATE TABLE IF NOT EXISTS growth_reward_grants (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  campaign_code varchar(64) NOT NULL,
  rule_code varchar(64) NOT NULL,
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  user_id bigint REFERENCES users(id),
  resource_id bigint REFERENCES resources(id),
  event_id bigint,
  idempotency_key varchar(255) NOT NULL,
  source_type varchar(64) NOT NULL DEFAULT 'growth_campaign',
  reward_type varchar(64) NOT NULL,
  reward_amount integer NOT NULL,
  valid_days integer NOT NULL,
  entitlement_id bigint REFERENCES merchant_entitlements(id),
  status varchar(32) NOT NULL DEFAULT 'granted',
  reason varchar(128) NOT NULL DEFAULT '',
  snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (idempotency_key),
  CHECK (status IN ('granted', 'skipped', 'failed', 'revoked')),
  CHECK (reward_amount > 0),
  CHECK (valid_days > 0)
);

CREATE INDEX IF NOT EXISTS idx_growth_campaigns_status_time
  ON growth_campaigns(status, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS idx_growth_campaign_rules_campaign_status
  ON growth_campaign_rules(campaign_code, status, trigger_event, priority);
CREATE INDEX IF NOT EXISTS idx_growth_reward_grants_merchant
  ON growth_reward_grants(merchant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_growth_reward_grants_resource
  ON growth_reward_grants(resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_growth_reward_grants_campaign_rule
  ON growth_reward_grants(campaign_code, rule_code, created_at DESC);

ALTER TABLE IF EXISTS growth_campaign_rules
  ADD COLUMN IF NOT EXISTS rule_name varchar(128) NOT NULL DEFAULT '';

UPDATE growth_campaign_rules
SET rule_name = COALESCE(NULLIF(description, ''), rule_code)
WHERE rule_name = '';

INSERT INTO growth_campaigns (
  code, name, status, starts_at, ends_at, config_snapshot
) VALUES (
  'starter_growth_2026_q3',
  '新手成长权益活动',
  'active',
  now(),
  now() + interval '90 days',
  '{"sourceType":"growth_campaign","frontendTitle":"新手发布权益","frontendHint":"发布优质资源、有效分享可获得更多曝光权益"}'::jsonb
) ON CONFLICT (code) DO NOTHING;

INSERT INTO growth_campaign_rules (
  campaign_code, rule_code, rule_name, trigger_event, status, priority, conditions, reward_type, reward_amount, valid_days, per_user_limit, per_user_daily_limit, per_resource_daily_limit, description
) VALUES
  ('starter_growth_2026_q3', 'first_login_publish_quota', '首次登录赠送发布次数', 'user_first_login', 'active', 10, '{}'::jsonb, 'publish_quota', 3, 30, 1, NULL, NULL, '首次登录赠送发布次数'),
  ('starter_growth_2026_q3', 'first_resource_approved_publish_quota', '首条资源审核通过奖励', 'resource_first_approved', 'active', 20, '{}'::jsonb, 'publish_quota', 5, 30, 1, NULL, NULL, '首条资源审核通过赠送发布次数'),
  ('starter_growth_2026_q3', 'three_approved_resources_publish_quota', '连续优质发布奖励发布次数', 'resource_approved_count_reached', 'active', 30, '{"approvedCount":3,"windowDays":30}'::jsonb, 'publish_quota', 5, 30, 1, NULL, NULL, '连续优质发布赠送发布次数'),
  ('starter_growth_2026_q3', 'three_approved_resources_refresh_quota', '连续优质发布奖励刷新次数', 'resource_approved_count_reached', 'active', 31, '{"approvedCount":3,"windowDays":30}'::jsonb, 'refresh_quota', 3, 30, 1, NULL, NULL, '连续优质发布赠送刷新次数'),
  ('starter_growth_2026_q3', 'share_contact_refresh_quota', '分享带来有效联系奖励', 'resource_share_effective_contact', 'active', 40, '{}'::jsonb, 'refresh_quota', 1, 15, NULL, 3, NULL, '分享带来有效联系赠送刷新次数'),
  ('starter_growth_2026_q3', 'share_view_refresh_quota', '分享有效浏览奖励', 'resource_share_effective_view', 'inactive', 50, '{"viewThreshold":5,"windowHours":24}'::jsonb, 'refresh_quota', 1, 15, NULL, 3, 1, '分享有效浏览奖励，首期默认关闭'),
  ('starter_growth_2026_q3', 'invitee_first_resource_approved', '邀请成功奖励', 'invitee_first_resource_approved', 'inactive', 60, '{}'::jsonb, 'publish_quota', 5, 30, NULL, NULL, NULL, '邀请奖励预留，等待邀请关系模型')
ON CONFLICT (campaign_code, rule_code) DO NOTHING;
