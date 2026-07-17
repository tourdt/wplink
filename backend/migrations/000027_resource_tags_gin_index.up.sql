CREATE INDEX IF NOT EXISTS idx_resources_tags_gin
  ON resources USING gin (tags);
