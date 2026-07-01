CREATE TABLE IF NOT EXISTS hot_search_keywords (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  city_station_id bigint REFERENCES city_stations(id),
  keyword varchar(64) NOT NULL,
  sort_order integer NOT NULL DEFAULT 0,
  status varchar(32) NOT NULL DEFAULT 'draft',
  start_at timestamptz,
  end_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hot_search_keywords_city_status ON hot_search_keywords (city_station_id, status);
CREATE INDEX IF NOT EXISTS idx_hot_search_keywords_sort ON hot_search_keywords (sort_order DESC, updated_at DESC);
