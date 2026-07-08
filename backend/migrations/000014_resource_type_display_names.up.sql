WITH zhili AS (
  SELECT id FROM city_stations WHERE code = 'zhili'
),
type_names AS (
  SELECT *
  FROM (
    VALUES
      ('inventory', '库存清仓'),
      ('goods', '现货货源'),
      ('factory', '工厂接单'),
      ('order', '订单找厂'),
      ('job', '招工招聘'),
      ('rental', '出租转让'),
      ('service', '配套服务')
  ) AS cfg(type_code, type_name)
)
UPDATE resource_type_configs rtc
SET
  type_name = type_names.type_name,
  updated_at = now()
FROM zhili, type_names
WHERE rtc.city_station_id = zhili.id
  AND rtc.type_code = type_names.type_code;

WITH zhili AS (
  SELECT id FROM city_stations WHERE code = 'zhili'
)
UPDATE resource_type_configs rtc
SET
  field_schema = '{"fields":[{"key":"rentalType","label":"房源类型","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["厂房","档口","仓库","商铺","商品房","宿舍","设备","其他"],"allowCustom":true},{"key":"areaText","label":"面积","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：120 平"},{"key":"rentText","label":"租金","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：6800 元/月"},{"key":"floor","label":"楼层","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：1 楼"},{"key":"transferFee","label":"转让费","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：无"}]}'::jsonb,
  filter_fields = '["rentalType","areaText","rentText","floor"]'::jsonb,
  display_template = '{"list":["priceText","district"],"detail":["rentalType","areaText","rentText","floor","transferFee"]}'::jsonb,
  updated_at = now()
FROM zhili
WHERE rtc.city_station_id = zhili.id
  AND rtc.type_code = 'rental';
