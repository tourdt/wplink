CREATE TABLE IF NOT EXISTS city_stations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code varchar(64) UNIQUE NOT NULL,
  name varchar(64) NOT NULL,
  province varchar(64),
  city varchar(64),
  primary_category varchar(64),
  status varchar(32) NOT NULL DEFAULT 'planned',
  config jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_city_stations_status ON city_stations(status);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_users_default_city_station'
  ) THEN
    ALTER TABLE users
      ADD CONSTRAINT fk_users_default_city_station
      FOREIGN KEY (default_city_station_id) REFERENCES city_stations(id);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_users_default_city ON users(default_city_station_id);

CREATE TABLE IF NOT EXISTS merchants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  city_station_id uuid NOT NULL REFERENCES city_stations(id),
  name varchar(128) NOT NULL,
  merchant_type varchar(64) NOT NULL,
  main_categories jsonb NOT NULL DEFAULT '[]'::jsonb,
  description text,
  contact_name varchar(64) NOT NULL,
  contact_phone varchar(32) NOT NULL,
  contact_wechat varchar(64),
  address_text varchar(255),
  location jsonb NOT NULL DEFAULT '{}'::jsonb,
  images jsonb NOT NULL DEFAULT '[]'::jsonb,
  verification_status varchar(32) NOT NULL DEFAULT 'unverified',
  status varchar(32) NOT NULL DEFAULT 'active',
  last_active_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_merchants_city_type ON merchants(city_station_id, merchant_type);
CREATE INDEX IF NOT EXISTS idx_merchants_verification_status ON merchants(verification_status);
CREATE INDEX IF NOT EXISTS idx_merchants_last_active_at ON merchants(last_active_at);

CREATE TABLE IF NOT EXISTS merchant_admin_bindings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  user_id uuid NOT NULL REFERENCES users(id),
  role varchar(32) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_active_merchant_user
  ON merchant_admin_bindings(merchant_id, user_id)
  WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_merchant_admin_bindings_user ON merchant_admin_bindings(user_id, status);

CREATE TABLE IF NOT EXISTS resource_type_configs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  city_station_id uuid REFERENCES city_stations(id),
  type_code varchar(64) NOT NULL,
  type_name varchar(64) NOT NULL,
  field_schema jsonb NOT NULL DEFAULT '{}'::jsonb,
  required_fields jsonb NOT NULL DEFAULT '[]'::jsonb,
  filter_fields jsonb NOT NULL DEFAULT '[]'::jsonb,
  display_template jsonb NOT NULL DEFAULT '{}'::jsonb,
  review_rules jsonb NOT NULL DEFAULT '{}'::jsonb,
  sort_weights jsonb NOT NULL DEFAULT '{}'::jsonb,
  message_rules jsonb NOT NULL DEFAULT '{}'::jsonb,
  default_valid_days integer NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_resource_type_city_scope
  ON resource_type_configs(city_station_id, type_code)
  WHERE city_station_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uniq_resource_type_global_scope
  ON resource_type_configs(type_code)
  WHERE city_station_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_resource_type_configs_type_code ON resource_type_configs(type_code);
CREATE INDEX IF NOT EXISTS idx_resource_type_configs_status ON resource_type_configs(status);

CREATE TABLE IF NOT EXISTS resources (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  city_station_id uuid NOT NULL REFERENCES city_stations(id),
  resource_type_config_id uuid NOT NULL REFERENCES resource_type_configs(id),
  type_code varchar(64) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'pending',
  title varchar(128) NOT NULL,
  category varchar(64) NOT NULL,
  district varchar(128),
  price_text varchar(128),
  quantity_text varchar(128),
  cover_url text,
  description text NOT NULL,
  attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
  tags jsonb NOT NULL DEFAULT '[]'::jsonb,
  images jsonb NOT NULL DEFAULT '[]'::jsonb,
  contact_name varchar(64) NOT NULL,
  contact_phone varchar(32) NOT NULL,
  contact_wechat varchar(64),
  is_verified boolean NOT NULL DEFAULT false,
  published_at timestamptz,
  refreshed_at timestamptz,
  expires_at timestamptz,
  dealt_at timestamptz,
  taken_down_at timestamptz,
  archived_at timestamptz,
  reject_reason text,
  take_down_reason text,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_resources_city_type_status ON resources(city_station_id, type_code, status);
CREATE INDEX IF NOT EXISTS idx_resources_merchant_status ON resources(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_resources_category_status ON resources(category, status);
CREATE INDEX IF NOT EXISTS idx_resources_refreshed_at ON resources(refreshed_at DESC);
CREATE INDEX IF NOT EXISTS idx_resources_expires_at ON resources(expires_at);
CREATE INDEX IF NOT EXISTS idx_resources_attributes_gin ON resources USING gin(attributes);

CREATE TABLE IF NOT EXISTS resource_review_records (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  resource_id uuid NOT NULL REFERENCES resources(id),
  reviewer_id uuid NOT NULL REFERENCES users(id),
  action varchar(32) NOT NULL,
  reason text,
  snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_resource_review_records_resource ON resource_review_records(resource_id);
CREATE INDEX IF NOT EXISTS idx_resource_review_records_reviewer ON resource_review_records(reviewer_id);

CREATE TABLE IF NOT EXISTS verifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  resource_id uuid REFERENCES resources(id),
  verification_type varchar(64) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'pending',
  applicant_user_id uuid NOT NULL REFERENCES users(id),
  business_name varchar(128),
  license_url text,
  storefront_url text,
  materials jsonb NOT NULL DEFAULT '{}'::jsonb,
  review_note text,
  reviewed_by uuid REFERENCES users(id),
  submitted_at timestamptz NOT NULL DEFAULT now(),
  reviewed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_verifications_merchant_status ON verifications(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_verifications_resource ON verifications(resource_id);

CREATE TABLE IF NOT EXISTS credit_records (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  resource_id uuid REFERENCES resources(id),
  source_type varchar(64) NOT NULL,
  tag_code varchar(64) NOT NULL,
  tag_label varchar(64) NOT NULL,
  description text,
  visibility varchar(32) NOT NULL DEFAULT 'public',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_credit_records_merchant_visibility ON credit_records(merchant_id, visibility);
CREATE INDEX IF NOT EXISTS idx_credit_records_tag_code ON credit_records(tag_code);

CREATE TABLE IF NOT EXISTS purchase_demands (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id),
  city_station_id uuid REFERENCES city_stations(id),
  demand_type varchar(64) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'pending',
  title varchar(128) NOT NULL,
  category varchar(64) NOT NULL,
  price_range jsonb NOT NULL DEFAULT '{}'::jsonb,
  quantity_requirement jsonb NOT NULL DEFAULT '{}'::jsonb,
  attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
  contact_name varchar(64) NOT NULL,
  contact_phone varchar(32) NOT NULL,
  contact_wechat varchar(64),
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_purchase_demands_city_type_status ON purchase_demands(city_station_id, demand_type, status);
CREATE INDEX IF NOT EXISTS idx_purchase_demands_user_status ON purchase_demands(user_id, status);
CREATE INDEX IF NOT EXISTS idx_purchase_demands_attributes_gin ON purchase_demands USING gin(attributes);

CREATE TABLE IF NOT EXISTS search_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid REFERENCES users(id),
  city_station_id uuid REFERENCES city_stations(id),
  keyword varchar(128) NOT NULL,
  filters jsonb NOT NULL DEFAULT '{}'::jsonb,
  result_count integer NOT NULL DEFAULT 0,
  clicked_resource_id uuid REFERENCES resources(id),
  generated_demand_id uuid REFERENCES purchase_demands(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_search_logs_keyword ON search_logs(keyword);
CREATE INDEX IF NOT EXISTS idx_search_logs_city_created ON search_logs(city_station_id, created_at);
CREATE INDEX IF NOT EXISTS idx_search_logs_result_count ON search_logs(result_count);

CREATE TABLE IF NOT EXISTS match_cases (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  purchase_demand_id uuid REFERENCES purchase_demands(id),
  city_station_id uuid REFERENCES city_stations(id),
  status varchar(32) NOT NULL DEFAULT 'open',
  source varchar(32) NOT NULL DEFAULT 'manual',
  operator_id uuid REFERENCES users(id),
  result_note text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  closed_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_match_cases_demand ON match_cases(purchase_demand_id);
CREATE INDEX IF NOT EXISTS idx_match_cases_operator_status ON match_cases(operator_id, status);

CREATE TABLE IF NOT EXISTS match_case_resources (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  match_case_id uuid NOT NULL REFERENCES match_cases(id),
  resource_id uuid NOT NULL REFERENCES resources(id),
  role varchar(32) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT uniq_match_case_resource UNIQUE (match_case_id, resource_id)
);

CREATE TABLE IF NOT EXISTS match_case_participants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  match_case_id uuid NOT NULL REFERENCES match_cases(id),
  user_id uuid REFERENCES users(id),
  merchant_id uuid REFERENCES merchants(id),
  participant_role varchar(32) NOT NULL,
  contact_status varchar(32) NOT NULL DEFAULT 'pending',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_match_case_participant_identity CHECK (user_id IS NOT NULL OR merchant_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_match_case_participants_case ON match_case_participants(match_case_id);

CREATE TABLE IF NOT EXISTS merchant_entitlements (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  entitlement_type varchar(64) NOT NULL,
  source_type varchar(64) NOT NULL,
  total_amount integer NOT NULL,
  used_amount integer NOT NULL DEFAULT 0,
  remaining_amount integer NOT NULL,
  starts_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_merchant_entitlements_merchant_type_status ON merchant_entitlements(merchant_id, entitlement_type, status);
CREATE INDEX IF NOT EXISTS idx_merchant_entitlements_expires_at ON merchant_entitlements(expires_at);

CREATE TABLE IF NOT EXISTS top_vouchers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  entitlement_id uuid REFERENCES merchant_entitlements(id),
  source_type varchar(64) NOT NULL,
  allowed_type_codes jsonb NOT NULL DEFAULT '[]'::jsonb,
  top_duration_hours integer NOT NULL,
  used_resource_id uuid REFERENCES resources(id),
  used_at timestamptz,
  expires_at timestamptz,
  status varchar(32) NOT NULL DEFAULT 'unused',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_top_vouchers_merchant_status ON top_vouchers(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_top_vouchers_used_resource ON top_vouchers(used_resource_id);

CREATE TABLE IF NOT EXISTS resource_contact_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  resource_id uuid NOT NULL REFERENCES resources(id),
  user_id uuid REFERENCES users(id),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  action varchar(32) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_resource_contact_events_resource_action ON resource_contact_events(resource_id, action);
CREATE INDEX IF NOT EXISTS idx_resource_contact_events_merchant_created ON resource_contact_events(merchant_id, created_at);

CREATE TABLE IF NOT EXISTS resource_metrics_daily (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  resource_id uuid NOT NULL REFERENCES resources(id),
  merchant_id uuid NOT NULL REFERENCES merchants(id),
  stat_date date NOT NULL,
  exposure_count integer NOT NULL DEFAULT 0,
  search_exposure_count integer NOT NULL DEFAULT 0,
  list_exposure_count integer NOT NULL DEFAULT 0,
  detail_view_count integer NOT NULL DEFAULT 0,
  contact_click_count integer NOT NULL DEFAULT 0,
  phone_click_count integer NOT NULL DEFAULT 0,
  wechat_copy_count integer NOT NULL DEFAULT 0,
  favorite_count integer NOT NULL DEFAULT 0,
  share_count integer NOT NULL DEFAULT 0,
  deal_feedback_count integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT uniq_resource_metric_daily UNIQUE (resource_id, stat_date)
);

CREATE INDEX IF NOT EXISTS idx_resource_metrics_daily_merchant_date ON resource_metrics_daily(merchant_id, stat_date);

CREATE TABLE IF NOT EXISTS banner_topics (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  city_station_id uuid REFERENCES city_stations(id),
  kind varchar(32) NOT NULL,
  title varchar(128) NOT NULL,
  subtitle varchar(255),
  cover_url text,
  type_scope jsonb NOT NULL DEFAULT '[]'::jsonb,
  jump_type varchar(32) NOT NULL,
  jump_target text NOT NULL,
  tags jsonb NOT NULL DEFAULT '[]'::jsonb,
  start_at timestamptz,
  end_at timestamptz,
  sort_order integer NOT NULL DEFAULT 0,
  status varchar(32) NOT NULL DEFAULT 'draft',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_banner_topics_city_kind_status ON banner_topics(city_station_id, kind, status);
CREATE INDEX IF NOT EXISTS idx_banner_topics_active_time ON banner_topics(kind, status, start_at, end_at);
CREATE INDEX IF NOT EXISTS idx_banner_topics_sort ON banner_topics(sort_order DESC, updated_at DESC);

CREATE TABLE IF NOT EXISTS messages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  recipient_user_id uuid REFERENCES users(id),
  recipient_role_code varchar(64),
  message_type varchar(64) NOT NULL,
  trigger_type varchar(64) NOT NULL,
  trigger_id uuid,
  title varchar(128) NOT NULL,
  content text NOT NULL,
  target_url text,
  channel varchar(32) NOT NULL DEFAULT 'in_app',
  status varchar(32) NOT NULL DEFAULT 'pending',
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  sent_at timestamptz,
  CONSTRAINT chk_messages_recipient CHECK (recipient_user_id IS NOT NULL OR recipient_role_code IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_messages_recipient_status ON messages(recipient_user_id, status);
CREATE INDEX IF NOT EXISTS idx_messages_trigger ON messages(trigger_type, trigger_id);
