WITH zhili AS (
  INSERT INTO city_stations (code, name, province, city, primary_category, status, config)
  VALUES (
    'zhili',
    '织里',
    '浙江省',
    '湖州市',
    '童装',
    'active',
    '{
      "enabledTypeCodes": ["inventory", "goods", "factory", "order", "job", "rental", "service"],
      "primaryCategories": ["童装", "加工厂", "库存尾货", "辅料服务"]
    }'::jsonb
  )
  ON CONFLICT (code) DO UPDATE
  SET
    name = EXCLUDED.name,
    province = EXCLUDED.province,
    city = EXCLUDED.city,
    primary_category = EXCLUDED.primary_category,
    status = EXCLUDED.status,
    config = EXCLUDED.config,
    updated_at = now()
  RETURNING id
)
INSERT INTO resource_type_configs (
  city_station_id,
  type_code,
  type_name,
  field_schema,
  required_fields,
  filter_fields,
  display_template,
  review_rules,
  sort_weights,
  message_rules,
  default_valid_days,
  status
)
SELECT
  zhili.id,
  cfg.type_code,
  cfg.type_name,
  cfg.field_schema::jsonb,
  cfg.required_fields::jsonb,
  cfg.filter_fields::jsonb,
  cfg.display_template::jsonb,
  cfg.review_rules::jsonb,
  cfg.sort_weights::jsonb,
  cfg.message_rules::jsonb,
  cfg.default_valid_days,
  'active'
FROM zhili
CROSS JOIN (
  VALUES
    (
      'inventory',
      '库存',
      '{"fields":[{"key":"season","label":"季节","type":"select"},{"key":"sizeRange","label":"尺码段","type":"text"},{"key":"allowSample","label":"支持拿样","type":"boolean"},{"key":"allowLiveSale","label":"支持直播","type":"boolean"}]}',
      '["title","category","quantityText","contactPhone"]',
      '["season","sizeRange","allowLiveSale"]',
      '{"list":["priceText","quantityText","district"],"detail":["season","sizeRange","allowSample","allowLiveSale"]}',
      '{"resubmitOnChange":["title","priceText","quantityText","contactPhone"]}',
      '{"verified":20,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":2}',
      7
    ),
    (
      'goods',
      '货源',
      '{"fields":[{"key":"style","label":"风格","type":"text"},{"key":"minOrderQuantity","label":"起批量","type":"text"},{"key":"spotAvailable","label":"是否现货","type":"boolean"},{"key":"dropshipping","label":"一件代发","type":"boolean"}]}',
      '["title","category","priceText","contactPhone"]',
      '["style","spotAvailable","dropshipping"]',
      '{"list":["priceText","district"],"detail":["style","minOrderQuantity","spotAvailable","dropshipping"]}',
      '{"resubmitOnChange":["title","priceText","contactPhone"]}',
      '{"verified":20,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":3}',
      15
    ),
    (
      'factory',
      '工厂产能',
      '{"fields":[{"key":"dailyCapacity","label":"日产能","type":"text"},{"key":"minOrderQuantity","label":"起订量","type":"text"},{"key":"acceptSmallOrders","label":"接小单","type":"boolean"},{"key":"availableSchedule","label":"空档期","type":"text"}]}',
      '["title","category","quantityText","contactPhone"]',
      '["dailyCapacity","acceptSmallOrders","availableSchedule"]',
      '{"list":["quantityText","district"],"detail":["dailyCapacity","minOrderQuantity","acceptSmallOrders","availableSchedule"]}',
      '{"resubmitOnChange":["title","quantityText","contactPhone"]}',
      '{"verified":25,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":3}',
      15
    ),
    (
      'order',
      '订单需求',
      '{"fields":[{"key":"orderQuantity","label":"订单数量","type":"text"},{"key":"deliveryDeadline","label":"交期","type":"text"},{"key":"sampleRequired","label":"需要打样","type":"boolean"},{"key":"longTermCooperation","label":"长期合作","type":"boolean"}]}',
      '["title","category","quantityText","contactPhone"]',
      '["orderQuantity","deliveryDeadline","sampleRequired"]',
      '{"list":["quantityText","district"],"detail":["orderQuantity","deliveryDeadline","sampleRequired","longTermCooperation"]}',
      '{"resubmitOnChange":["title","quantityText","contactPhone"]}',
      '{"verified":15,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":3}',
      10
    ),
    (
      'job',
      '招聘',
      '{"fields":[{"key":"position","label":"岗位","type":"text"},{"key":"payText","label":"工价","type":"text"},{"key":"headcount","label":"人数","type":"number"},{"key":"includeMealsHousing","label":"包吃住","type":"boolean"}]}',
      '["title","category","priceText","contactPhone"]',
      '["position","payText","includeMealsHousing"]',
      '{"list":["priceText","district"],"detail":["position","payText","headcount","includeMealsHousing"]}',
      '{"resubmitOnChange":["title","priceText","contactPhone"]}',
      '{"verified":10,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":2}',
      15
    ),
    (
      'rental',
      '出租/转让',
      '{"fields":[{"key":"areaText","label":"面积","type":"text"},{"key":"rentText","label":"租金","type":"text"},{"key":"floor","label":"楼层","type":"text"},{"key":"transferFee","label":"转让费","type":"text"}]}',
      '["title","category","priceText","contactPhone"]',
      '["areaText","rentText","floor"]',
      '{"list":["priceText","district"],"detail":["areaText","rentText","floor","transferFee"]}',
      '{"resubmitOnChange":["title","priceText","contactPhone"]}',
      '{"verified":10,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":5}',
      30
    ),
    (
      'service',
      '服务',
      '{"fields":[{"key":"serviceType","label":"服务类型","type":"text"},{"key":"serviceArea","label":"服务范围","type":"text"},{"key":"leadTime","label":"交付时效","type":"text"},{"key":"caseAvailable","label":"有案例","type":"boolean"}]}',
      '["title","category","contactPhone"]',
      '["serviceType","serviceArea","caseAvailable"]',
      '{"list":["district"],"detail":["serviceType","serviceArea","leadTime","caseAvailable"]}',
      '{"resubmitOnChange":["title","contactPhone"]}',
      '{"verified":20,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":5}',
      30
    )
) AS cfg(
  type_code,
  type_name,
  field_schema,
  required_fields,
  filter_fields,
  display_template,
  review_rules,
  sort_weights,
  message_rules,
  default_valid_days
)
ON CONFLICT (city_station_id, type_code) WHERE city_station_id IS NOT NULL
DO UPDATE SET
  type_name = EXCLUDED.type_name,
  field_schema = EXCLUDED.field_schema,
  required_fields = EXCLUDED.required_fields,
  filter_fields = EXCLUDED.filter_fields,
  display_template = EXCLUDED.display_template,
  review_rules = EXCLUDED.review_rules,
  sort_weights = EXCLUDED.sort_weights,
  message_rules = EXCLUDED.message_rules,
  default_valid_days = EXCLUDED.default_valid_days,
  status = EXCLUDED.status,
  updated_at = now();

WITH zhili AS (
  SELECT id FROM city_stations WHERE code = 'zhili'
),
inserted_topic AS (
  INSERT INTO banner_topics (
    city_station_id,
    kind,
    title,
    subtitle,
    cover_url,
    type_scope,
    jump_type,
    jump_target,
    tags,
    sort_order,
    status
  )
  SELECT
    zhili.id,
    'topic',
    '织里童装库存精选',
    '快速发现可拿样、可直播的优质库存',
    '',
    '["inventory"]'::jsonb,
    'demand',
    '/pages/demand/index',
    '["童装","库存"]'::jsonb,
    90,
    'active'
  FROM zhili
  WHERE NOT EXISTS (
    SELECT 1
    FROM banner_topics bt
    WHERE bt.city_station_id = zhili.id
      AND bt.kind = 'topic'
      AND bt.title = '织里童装库存精选'
  )
  RETURNING id, city_station_id
),
topic AS (
  SELECT id, city_station_id FROM inserted_topic
  UNION ALL
  SELECT bt.id, bt.city_station_id
  FROM banner_topics bt
  JOIN zhili ON zhili.id = bt.city_station_id
  WHERE bt.kind = 'topic'
    AND bt.title = '织里童装库存精选'
  LIMIT 1
)
INSERT INTO banner_topics (
  city_station_id,
  kind,
  title,
  subtitle,
  cover_url,
  type_scope,
  jump_type,
  jump_target,
  tags,
  sort_order,
  status
)
SELECT
  topic.city_station_id,
  'banner',
  '织里童装现货对接',
  '库存、货源、工厂资源一站式撮合',
  '',
  '["inventory","goods","factory"]'::jsonb,
  'topic',
  topic.id::text,
  '["首页推荐"]'::jsonb,
  100,
  'active'
FROM topic
WHERE NOT EXISTS (
  SELECT 1
  FROM banner_topics bt
  WHERE bt.city_station_id = topic.city_station_id
    AND bt.kind = 'banner'
    AND bt.title = '织里童装现货对接'
);
