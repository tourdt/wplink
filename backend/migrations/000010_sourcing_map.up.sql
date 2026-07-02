CREATE TABLE IF NOT EXISTS map_scene (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  city_station_id bigint REFERENCES city_stations(id),
  code varchar(64) UNIQUE NOT NULL,
  name varchar(100) NOT NULL,
  type varchar(30) NOT NULL,
  parent_code varchar(64),
  background_url text NOT NULL,
  width int NOT NULL,
  height int NOT NULL,
  min_scale numeric(5,2) NOT NULL DEFAULT 0.5,
  max_scale numeric(5,2) NOT NULL DEFAULT 5,
  default_scale numeric(5,2) NOT NULL DEFAULT 1,
  default_center_x numeric(10,2),
  default_center_y numeric(10,2),
  floor_no varchar(20),
  sort int NOT NULL DEFAULT 0,
  revision int NOT NULL DEFAULT 1,
  status varchar(20) NOT NULL DEFAULT 'draft',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS map_object (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  scene_code varchar(64) NOT NULL REFERENCES map_scene(code),
  merchant_id bigint REFERENCES merchants(id),
  code varchar(64) NOT NULL,
  name varchar(100) NOT NULL,
  type varchar(30) NOT NULL,
  layer varchar(30) NOT NULL,
  geometry_type varchar(30) NOT NULL,
  geometry jsonb NOT NULL,
  center_x numeric(10,2),
  center_y numeric(10,2),
  min_x numeric(10,2),
  min_y numeric(10,2),
  max_x numeric(10,2),
  max_y numeric(10,2),
  min_zoom int NOT NULL DEFAULT 1,
  max_zoom int NOT NULL DEFAULT 5,
  category_codes jsonb NOT NULL DEFAULT '[]'::jsonb,
  service_tags jsonb NOT NULL DEFAULT '[]'::jsonb,
  platform_tags jsonb NOT NULL DEFAULT '[]'::jsonb,
  poi_service_tags jsonb NOT NULL DEFAULT '[]'::jsonb,
  address text,
  phone varchar(30),
  wechat varchar(50),
  lat numeric(10,7),
  lng numeric(10,7),
  search_text text NOT NULL DEFAULT '',
  extra jsonb NOT NULL DEFAULT '{}'::jsonb,
  sort int NOT NULL DEFAULT 0,
  status varchar(20) NOT NULL DEFAULT 'normal',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(scene_code, code)
);

CREATE TABLE IF NOT EXISTS map_category (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  code varchar(64) UNIQUE NOT NULL,
  name varchar(100) NOT NULL,
  type varchar(30) NOT NULL,
  icon_url text,
  sort int NOT NULL DEFAULT 0,
  is_visible boolean NOT NULL DEFAULT true,
  status varchar(20) NOT NULL DEFAULT 'normal',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_map_scene_city_status ON map_scene(city_station_id, status, sort);
CREATE INDEX IF NOT EXISTS idx_map_scene_parent ON map_scene(parent_code);
CREATE INDEX IF NOT EXISTS idx_map_object_scene ON map_object(scene_code);
CREATE INDEX IF NOT EXISTS idx_map_object_scene_type ON map_object(scene_code, type);
CREATE INDEX IF NOT EXISTS idx_map_object_scene_bounds ON map_object(scene_code, min_x, max_x, min_y, max_y);
CREATE INDEX IF NOT EXISTS idx_map_object_scene_status_sort ON map_object(scene_code, status, sort);

INSERT INTO map_category(code, name, type, sort)
VALUES
  ('girl', '女童', 'booth_category', 10),
  ('boy', '男童', 'booth_category', 20),
  ('middle_child', '中大童', 'booth_category', 30),
  ('baby', '婴童', 'booth_category', 40),
  ('spot', '现货', 'booth_service', 10),
  ('factory', '源头工厂', 'booth_service', 20),
  ('sample', '支持打样', 'booth_service', 30),
  ('dropship', '一件代发', 'booth_service', 40),
  ('verified', '实地认证', 'platform_tag', 10),
  ('hot', '热门推荐', 'platform_tag', 20),
  ('packing_station', '打包站', 'poi_type', 10),
  ('logistics_point', '物流点', 'poi_type', 20),
  ('express_point', '快递点', 'poi_type', 30),
  ('parking', '停车场', 'poi_type', 40),
  ('packing', '支持打包', 'poi_service', 10),
  ('labeling', '支持贴单', 'poi_service', 20),
  ('carton', '支持纸箱', 'poi_service', 30),
  ('national', '全国物流', 'poi_service', 40),
  ('bulk_shipping', '批量发货', 'poi_service', 50)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    type = EXCLUDED.type,
    sort = EXCLUDED.sort,
    status = 'normal',
    updated_at = now();
