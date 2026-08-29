CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS city_stations (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
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
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  city_station_id bigint NOT NULL REFERENCES city_stations(id),
  name varchar(128) NOT NULL,
  merchant_type varchar(64) NOT NULL,
  main_categories jsonb NOT NULL DEFAULT '[]'::jsonb,
  description text,
  contact_name varchar(64) NOT NULL,
  contact_phone varchar(32) NOT NULL,
  contact_wechat varchar(64),
  address_text varchar(255),
  market_name varchar(128),
  building_name varchar(64),
  floor_no varchar(32),
  booth_no varchar(64),
  latitude double precision,
  longitude double precision,
  location_name varchar(128),
  location_status varchar(32) NOT NULL DEFAULT 'hidden',
  location_updated_at timestamptz,
  logo_url text,
  images jsonb NOT NULL DEFAULT '[]'::jsonb,
  profile_status varchar(32) NOT NULL DEFAULT 'completed',
  status varchar(32) NOT NULL DEFAULT 'active',
  last_active_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT chk_merchants_location_pair CHECK ((latitude IS NULL) = (longitude IS NULL)),
  CONSTRAINT chk_merchants_latitude CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
  CONSTRAINT chk_merchants_longitude CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180),
  CONSTRAINT chk_merchants_location_status CHECK (location_status IN ('visible', 'hidden', 'risk_hidden')),
  CONSTRAINT chk_merchants_visible_location CHECK (location_status <> 'visible' OR (latitude IS NOT NULL AND longitude IS NOT NULL)),
  CONSTRAINT chk_merchants_risk_hidden_location CHECK (location_status <> 'risk_hidden' OR (latitude IS NOT NULL AND longitude IS NOT NULL)),
  CONSTRAINT chk_merchants_profile_status CHECK (profile_status IN ('incomplete', 'completed'))
);

CREATE INDEX IF NOT EXISTS idx_merchants_city_type ON merchants(city_station_id, merchant_type);
CREATE INDEX IF NOT EXISTS idx_merchants_last_active_at ON merchants(last_active_at);
CREATE INDEX IF NOT EXISTS idx_merchants_directory_location
  ON merchants(city_station_id, status, profile_status, location_status)
  WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_merchants_location_coordinates
  ON merchants(latitude, longitude)
  WHERE location_status = 'visible' AND latitude IS NOT NULL AND longitude IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_merchants_main_categories_gin
  ON merchants USING gin(main_categories);
CREATE INDEX IF NOT EXISTS idx_merchants_market_name_trgm
  ON merchants USING gin(market_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_merchants_booth_no_trgm
  ON merchants USING gin(booth_no gin_trgm_ops);

CREATE TABLE IF NOT EXISTS merchant_location_reports (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
  reporter_user_id bigint NOT NULL REFERENCES users(id),
  reason_code varchar(64) NOT NULL,
  description varchar(200) NOT NULL DEFAULT '',
  status varchar(32) NOT NULL DEFAULT 'pending',
  location_snapshot jsonb NOT NULL,
  reviewed_by bigint REFERENCES admin_operators(id),
  review_note text,
  reviewed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_merchant_location_report_reason CHECK (length(btrim(reason_code)) > 0),
  CONSTRAINT chk_merchant_location_report_status CHECK (status IN ('pending', 'valid', 'invalid', 'resolved')),
  CONSTRAINT chk_merchant_location_report_snapshot CHECK (jsonb_typeof(location_snapshot) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_pending_merchant_location_report_user
  ON merchant_location_reports(merchant_id, reporter_user_id)
  WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_merchant_location_reports_merchant_status_created
  ON merchant_location_reports(merchant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_merchant_location_reports_pending_created
  ON merchant_location_reports(created_at DESC)
  WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_merchant_location_reports_reporter
  ON merchant_location_reports(reporter_user_id);
CREATE INDEX IF NOT EXISTS idx_merchant_location_reports_reviewer
  ON merchant_location_reports(reviewed_by)
  WHERE reviewed_by IS NOT NULL;

CREATE TABLE IF NOT EXISTS merchant_directory_events (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint REFERENCES users(id) ON DELETE SET NULL,
  merchant_id bigint NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
  target_merchant_id bigint REFERENCES merchants(id) ON DELETE SET NULL,
  visitor_key varchar(96) NOT NULL,
  session_id varchar(96) NOT NULL,
  event_type varchar(40) NOT NULL,
  source varchar(30) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_merchant_directory_event_type CHECK (event_type IN (
    'directory_exposure', 'directory_merchant_click', 'location_view',
    'nearby_marker_click', 'nearby_list_item_click', 'nearby_merchant_click',
    'nearby_drawer_open', 'navigation_click', 'location_report_submit'
  )),
  CONSTRAINT chk_merchant_directory_event_source CHECK (source IN (
    'directory', 'merchant_detail', 'merchant_location'
  ))
);

CREATE INDEX IF NOT EXISTS idx_merchant_directory_events_merchant_event_created
  ON merchant_directory_events(merchant_id, event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_merchant_directory_events_target_created
  ON merchant_directory_events(target_merchant_id, created_at DESC)
  WHERE target_merchant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_merchant_directory_events_user_created
  ON merchant_directory_events(user_id, created_at DESC)
  WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_merchant_directory_events_created
  ON merchant_directory_events(created_at);

CREATE TABLE IF NOT EXISTS merchant_admin_bindings (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  user_id bigint NOT NULL REFERENCES users(id),
  role varchar(32) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_by bigint REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_active_merchant_binding_per_user
  ON merchant_admin_bindings(user_id)
  WHERE status = 'active';
CREATE UNIQUE INDEX IF NOT EXISTS uniq_active_owner_per_merchant
  ON merchant_admin_bindings(merchant_id)
  WHERE status = 'active' AND role = 'owner';
CREATE INDEX IF NOT EXISTS idx_merchant_admin_bindings_user ON merchant_admin_bindings(user_id, status);

CREATE TABLE IF NOT EXISTS resource_type_configs (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  city_station_id bigint REFERENCES city_stations(id),
  type_code varchar(64) NOT NULL,
  type_name varchar(64) NOT NULL,
  version bigint NOT NULL DEFAULT 1,
  direction varchar(32) NOT NULL DEFAULT 'supply',
  field_schema jsonb NOT NULL DEFAULT '{}'::jsonb,
  required_fields jsonb NOT NULL DEFAULT '[]'::jsonb,
  filter_fields jsonb NOT NULL DEFAULT '[]'::jsonb,
  display_template jsonb NOT NULL DEFAULT '{}'::jsonb,
  review_rules jsonb NOT NULL DEFAULT '{}'::jsonb,
  sort_weights jsonb NOT NULL DEFAULT '{}'::jsonb,
  message_rules jsonb NOT NULL DEFAULT '{}'::jsonb,
  commercial_rules jsonb NOT NULL DEFAULT '{"publish":{"mode":"consume_quota"},"contactUnlock":{"mode":"login_free","priceCent":0,"currency":"CNY","vipFree":false,"repeatUnlockDays":30}}'::jsonb,
  default_valid_days integer NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_resource_type_configs_direction CHECK (direction IN ('supply', 'demand')),
  CONSTRAINT chk_resource_type_configs_version CHECK (version > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_resource_type_city_scope
  ON resource_type_configs(city_station_id, type_code)
  WHERE city_station_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uniq_resource_type_global_scope
  ON resource_type_configs(type_code)
  WHERE city_station_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_resource_type_configs_type_code ON resource_type_configs(type_code);
CREATE INDEX IF NOT EXISTS idx_resource_type_configs_direction ON resource_type_configs(direction, status);
CREATE INDEX IF NOT EXISTS idx_resource_type_configs_status ON resource_type_configs(status);

CREATE TABLE IF NOT EXISTS resources (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  city_station_id bigint NOT NULL REFERENCES city_stations(id),
  resource_type_config_id bigint NOT NULL REFERENCES resource_type_configs(id),
  resource_type_config_version bigint NOT NULL,
  resource_type_snapshot jsonb NOT NULL,
  type_code varchar(64) NOT NULL,
  direction varchar(32) NOT NULL DEFAULT 'supply',
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
  published_at timestamptz,
  refreshed_at timestamptz,
  top_started_at timestamptz,
  top_expires_at timestamptz,
  expires_at timestamptz,
  dealt_at timestamptz,
  taken_down_at timestamptz,
  archived_at timestamptz,
  reject_reason text,
  take_down_reason text,
  created_by_user_id bigint REFERENCES users(id),
  created_by_operator_id bigint REFERENCES admin_operators(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT chk_resources_direction CHECK (direction IN ('supply', 'demand')),
  CONSTRAINT chk_resources_type_config_version CHECK (resource_type_config_version > 0),
  CONSTRAINT chk_resources_type_snapshot CHECK (jsonb_typeof(resource_type_snapshot) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_resources_city_type_status ON resources(city_station_id, type_code, status);
CREATE INDEX IF NOT EXISTS idx_resources_city_direction_type_status ON resources(city_station_id, direction, type_code, status);
CREATE INDEX IF NOT EXISTS idx_resources_merchant_status ON resources(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_resources_category_status ON resources(category, status);
CREATE INDEX IF NOT EXISTS idx_resources_refreshed_at ON resources(refreshed_at DESC);
CREATE INDEX IF NOT EXISTS idx_resources_top_expires_at ON resources(top_expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_resources_expires_at ON resources(expires_at);
CREATE INDEX IF NOT EXISTS idx_resources_attributes_gin ON resources USING gin(attributes);

CREATE TABLE IF NOT EXISTS resource_review_records (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  reviewer_id bigint NOT NULL REFERENCES admin_operators(id),
  action varchar(32) NOT NULL,
  reason text,
  snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_resource_review_records_resource ON resource_review_records(resource_id);
CREATE INDEX IF NOT EXISTS idx_resource_review_records_reviewer ON resource_review_records(reviewer_id);

CREATE TABLE IF NOT EXISTS credit_records (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  resource_id bigint REFERENCES resources(id),
  source_type varchar(64) NOT NULL,
  tag_code varchar(64) NOT NULL,
  tag_label varchar(64) NOT NULL,
  description text,
  visibility varchar(32) NOT NULL DEFAULT 'public',
  created_by bigint REFERENCES admin_operators(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_credit_records_merchant_visibility ON credit_records(merchant_id, visibility);
CREATE INDEX IF NOT EXISTS idx_credit_records_tag_code ON credit_records(tag_code);

CREATE TABLE IF NOT EXISTS search_logs (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint REFERENCES users(id),
  city_station_id bigint REFERENCES city_stations(id),
  keyword varchar(128) NOT NULL,
  filters jsonb NOT NULL DEFAULT '{}'::jsonb,
  result_count integer NOT NULL DEFAULT 0,
  clicked_resource_id bigint REFERENCES resources(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_search_logs_keyword ON search_logs(keyword);
CREATE INDEX IF NOT EXISTS idx_search_logs_city_created ON search_logs(city_station_id, created_at);
CREATE INDEX IF NOT EXISTS idx_search_logs_result_count ON search_logs(result_count);

CREATE TABLE IF NOT EXISTS merchant_entitlements (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  entitlement_type varchar(64) NOT NULL,
  source_type varchar(64) NOT NULL,
  total_amount integer NOT NULL,
  used_amount integer NOT NULL DEFAULT 0,
  remaining_amount integer NOT NULL,
  allowed_type_codes jsonb NOT NULL DEFAULT '[]'::jsonb,
  top_duration_hours integer NOT NULL DEFAULT 0,
  starts_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_merchant_entitlements_merchant_type_status ON merchant_entitlements(merchant_id, entitlement_type, status);
CREATE INDEX IF NOT EXISTS idx_merchant_entitlements_expires_at ON merchant_entitlements(expires_at);

CREATE TABLE IF NOT EXISTS merchant_entitlement_usage_records (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  entitlement_id bigint NOT NULL REFERENCES merchant_entitlements(id),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  entitlement_type varchar(64) NOT NULL,
  action_type varchar(64) NOT NULL,
  amount integer NOT NULL DEFAULT 1,
  resource_id bigint REFERENCES resources(id),
  before_remaining_amount integer NOT NULL,
  after_remaining_amount integer NOT NULL,
  snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  used_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_entitlement_usage_entitlement ON merchant_entitlement_usage_records(entitlement_id, used_at DESC);
CREATE INDEX IF NOT EXISTS idx_entitlement_usage_merchant ON merchant_entitlement_usage_records(merchant_id, used_at DESC);
CREATE INDEX IF NOT EXISTS idx_entitlement_usage_resource ON merchant_entitlement_usage_records(resource_id);

CREATE TABLE IF NOT EXISTS resource_contact_events (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  user_id bigint REFERENCES users(id),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  action varchar(32) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_resource_contact_events_resource_action ON resource_contact_events(resource_id, action);
CREATE INDEX IF NOT EXISTS idx_resource_contact_events_merchant_created ON resource_contact_events(merchant_id, created_at);

CREATE TABLE IF NOT EXISTS resource_contact_unlock_orders (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  buyer_user_id bigint NOT NULL REFERENCES users(id),
  buyer_merchant_id bigint REFERENCES merchants(id),
  type_code varchar(64) NOT NULL,
  out_trade_no varchar(64) UNIQUE NOT NULL,
  price_cent integer NOT NULL,
  currency varchar(16) NOT NULL DEFAULT 'CNY',
  commercial_rules_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  status varchar(32) NOT NULL DEFAULT 'pending',
  paid_at timestamptz,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_resource_contact_unlock_orders_price CHECK (price_cent >= 0),
  CONSTRAINT chk_resource_contact_unlock_orders_status CHECK (status IN ('pending', 'paid', 'closed', 'refunded')),
  CONSTRAINT chk_resource_contact_unlock_orders_currency CHECK (currency = 'CNY')
);

CREATE INDEX IF NOT EXISTS idx_contact_unlock_orders_resource_status
  ON resource_contact_unlock_orders(resource_id, status);
CREATE INDEX IF NOT EXISTS idx_contact_unlock_orders_buyer_status
  ON resource_contact_unlock_orders(buyer_user_id, status);
CREATE INDEX IF NOT EXISTS idx_contact_unlock_orders_out_trade_no
  ON resource_contact_unlock_orders(out_trade_no);
CREATE INDEX IF NOT EXISTS idx_contact_unlock_orders_pending_created
  ON resource_contact_unlock_orders(created_at)
  WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS resource_contact_unlocks (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  user_id bigint NOT NULL REFERENCES users(id),
  viewer_merchant_id bigint REFERENCES merchants(id),
  source_type varchar(32) NOT NULL,
  order_id bigint REFERENCES resource_contact_unlock_orders(id),
  starts_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz,
  CONSTRAINT chk_resource_contact_unlocks_source CHECK (source_type IN ('paid', 'vip', 'login_free', 'owner', 'manual')),
  CONSTRAINT chk_resource_contact_unlocks_period CHECK (expires_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_contact_unlocks_resource_user
  ON resource_contact_unlocks(resource_id, user_id, viewer_merchant_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_contact_unlocks_order
  ON resource_contact_unlocks(order_id);

CREATE TABLE IF NOT EXISTS resource_metrics_daily (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  resource_id bigint NOT NULL REFERENCES resources(id),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
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
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  city_station_id bigint REFERENCES city_stations(id),
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
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  recipient_user_id bigint REFERENCES users(id),
  recipient_role_code varchar(64),
  message_type varchar(64) NOT NULL,
  trigger_type varchar(64) NOT NULL,
  trigger_id bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS uniq_messages_lifecycle_trigger
  ON messages(recipient_role_code, trigger_type, trigger_id)
  WHERE recipient_role_code IS NOT NULL
    AND trigger_id IS NOT NULL
    AND trigger_type IN (
      'resource_expired',
      'resource_expiring'
    );
