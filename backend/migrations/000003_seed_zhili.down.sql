DELETE FROM banner_topics
WHERE city_station_id IN (
  SELECT id FROM city_stations WHERE code = 'zhili'
)
AND title IN ('织里童装库存精选', '织里童装现货对接');

DELETE FROM resource_type_configs
WHERE city_station_id IN (
  SELECT id FROM city_stations WHERE code = 'zhili'
)
AND type_code IN (
  'factory_direct',
  'spot_wholesale',
  'stock_clearance',
  'buy_kids_goods',
  'fabric_supply',
  'accessory_supply',
  'processing_accept',
  'find_factory',
  'production_support',
  'job_hiring',
  'job_seeking',
  'sample_rental',
  'shop_office_rental',
  'shop_sale',
  'seek_shop_office',
  'apartment_rental',
  'housing_sale',
  'seek_housing',
  'factory_warehouse_rental',
  'workshop_rental',
  'factory_sale',
  'seek_factory_warehouse',
  'secondhand_sale',
  'secondhand_buy',
  'education_training',
  'appliance_repair',
  'moving_cleaning',
  'other_local_service'
);

DELETE FROM city_stations
WHERE code = 'zhili';
