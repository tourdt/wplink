DELETE FROM banner_topics
WHERE city_station_id IN (
  SELECT id FROM city_stations WHERE code = 'zhili'
)
AND title IN ('织里童装库存精选', '织里童装现货对接');

DELETE FROM resource_type_configs
WHERE city_station_id IN (
  SELECT id FROM city_stations WHERE code = 'zhili'
)
AND type_code IN ('inventory', 'goods', 'factory', 'order', 'job', 'rental', 'service');

DELETE FROM city_stations
WHERE code = 'zhili';
