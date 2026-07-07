CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 资源列表和搜索页使用前后模糊 ILIKE，trigram 索引避免数据增长后退化为全表扫描。
CREATE INDEX IF NOT EXISTS idx_resources_title_trgm
  ON resources USING gin(title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_resources_description_trgm
  ON resources USING gin(description gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_resources_category_trgm
  ON resources USING gin(category gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_resources_attributes_text_trgm
  ON resources USING gin((attributes::text) gin_trgm_ops);

-- 商家名称会参与资源、地图和后台商家列表搜索，独立建索引可复用。
CREATE INDEX IF NOT EXISTS idx_merchants_name_trgm
  ON merchants USING gin(name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_merchants_contact_name_trgm
  ON merchants USING gin(contact_name gin_trgm_ops);

-- 地图点位搜索会匹配编码、名称、地址和聚合 search_text。
CREATE INDEX IF NOT EXISTS idx_map_object_code_trgm
  ON map_object USING gin(code gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_map_object_name_trgm
  ON map_object USING gin(name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_map_object_address_trgm
  ON map_object USING gin((COALESCE(address, '')) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_map_object_search_text_trgm
  ON map_object USING gin(search_text gin_trgm_ops);

-- 档口绑定申请后台审核会搜索申请备注，避免运营列表数据变大后扫描整表。
CREATE INDEX IF NOT EXISTS idx_map_bind_request_note_trgm
  ON map_object_bind_request USING gin(note gin_trgm_ops);
