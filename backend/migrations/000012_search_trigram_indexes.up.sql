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

-- 商家名称会参与资源和后台商家列表搜索，独立建索引可复用。
CREATE INDEX IF NOT EXISTS idx_merchants_name_trgm
  ON merchants USING gin(name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_merchants_contact_name_trgm
  ON merchants USING gin(contact_name gin_trgm_ops);
