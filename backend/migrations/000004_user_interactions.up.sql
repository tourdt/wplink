CREATE TABLE IF NOT EXISTS user_favorite_resources (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint NOT NULL REFERENCES users(id),
  resource_id bigint NOT NULL REFERENCES resources(id),
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT uniq_user_favorite_resource UNIQUE (user_id, resource_id)
);

CREATE INDEX IF NOT EXISTS idx_user_favorite_resources_user_status
  ON user_favorite_resources(user_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_favorite_resources_resource_status
  ON user_favorite_resources(resource_id, status);

CREATE TABLE IF NOT EXISTS user_followed_merchants (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint NOT NULL REFERENCES users(id),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT uniq_user_followed_merchant UNIQUE (user_id, merchant_id)
);

CREATE INDEX IF NOT EXISTS idx_user_followed_merchants_user_status
  ON user_followed_merchants(user_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_followed_merchants_merchant_status
  ON user_followed_merchants(merchant_id, status);

CREATE TABLE IF NOT EXISTS user_saved_searches (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  user_id bigint NOT NULL REFERENCES users(id),
  name varchar(128) NOT NULL,
  city_station_id bigint REFERENCES city_stations(id),
  type_code varchar(64),
  keyword varchar(128),
  category varchar(64),
  verified_only boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_user_saved_searches_user_created
  ON user_saved_searches(user_id, created_at DESC)
  WHERE deleted_at IS NULL;
