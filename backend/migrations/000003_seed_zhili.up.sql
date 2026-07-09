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
      "enabledTypeCodes": ["inventory", "goods", "factory", "job", "rental", "service", "buy_goods", "find_inventory", "find_factory", "find_service", "find_rental"],
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
  direction,
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
  cfg.direction,
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
      '库存清仓',
      'supply',
      '{"fields":[{"key":"stockCategory","label":"库存品类","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["童装","女装","卫衣","套装","面料","辅料"],"allowCustom":true},{"key":"season","label":"季节","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["春季","夏季","秋季","冬季"],"allowCustom":false},{"key":"sizeRange","label":"尺码段","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：90-140"},{"key":"stockQuantityText","label":"库存数量","type":"text","required":true,"filterable":false,"displayIn":["detail"],"placeholder":"例如：3200 件、80 包"},{"key":"stockPriceText","label":"打包价/单件价","type":"text","required":true,"filterable":false,"displayIn":["detail"],"placeholder":"例如：打包 18 元/件"},{"key":"allowSample","label":"支持拿样","type":"boolean","required":false,"filterable":false,"displayIn":["detail"]},{"key":"allowLiveSale","label":"支持直播","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"warehouseLocation","label":"仓库位置","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：织里童装城附近"}]}',
      '["title","stockCategory","stockQuantityText","stockPriceText","contactPhone"]',
      '["stockCategory","season","sizeRange","allowLiveSale","warehouseLocation"]',
      '{"summary":{"category":"stockCategory","quantityText":"stockQuantityText","priceText":"stockPriceText"},"list":["priceText","quantityText","district"],"detail":["stockCategory","season","sizeRange","stockQuantityText","stockPriceText","allowSample","allowLiveSale","warehouseLocation"]}',
      '{"resubmitOnChange":["title","stockCategory","stockQuantityText","stockPriceText","contactPhone"]}',
      '{"verified":20,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":2}',
      7
    ),
    (
      'goods',
      '现货货源',
      'supply',
      '{"fields":[{"key":"goodsCategory","label":"货源品类","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["童装","女装","面料","配饰"],"allowCustom":true},{"key":"style","label":"风格","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["韩版","学院风","运动风","国潮","基础款"],"allowCustom":true},{"key":"supplyPriceText","label":"供货价/价格带","type":"text","required":true,"filterable":false,"displayIn":["detail"],"placeholder":"例如：18-25 元/件"},{"key":"minOrderQuantity","label":"起批量","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：20 套"},{"key":"spotAvailable","label":"是否现货","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"dropshipping","label":"一件代发","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"supplyArea","label":"供货区域","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：织里及周边、全国发货"}]}',
      '["title","goodsCategory","supplyPriceText","contactPhone"]',
      '["goodsCategory","style","spotAvailable","dropshipping","supplyArea"]',
      '{"summary":{"category":"goodsCategory","quantityText":"minOrderQuantity","priceText":"supplyPriceText"},"list":["priceText","quantityText","district"],"detail":["goodsCategory","style","supplyPriceText","minOrderQuantity","spotAvailable","dropshipping","supplyArea"]}',
      '{"resubmitOnChange":["title","goodsCategory","supplyPriceText","contactPhone"]}',
      '{"verified":20,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":3}',
      15
    ),
    (
      'factory',
      '工厂接单',
      'supply',
      '{"fields":[{"key":"factoryCategory","label":"擅长品类","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["童装","女装","羽绒服","针织","梭织"],"allowCustom":true},{"key":"dailyCapacity","label":"日产能","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：1200 件/天"},{"key":"minOrderQuantity","label":"起订量","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：300 件"},{"key":"acceptSmallOrders","label":"接小单","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"availableSchedule","label":"当前空档期","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["本周可排单","下周可排单","本月可排单","需沟通排期"],"allowCustom":true},{"key":"laborPriceText","label":"工价范围","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：按款报价、计件区间"},{"key":"processingMode","label":"加工方式","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["包工包料","来料加工","清加工","可沟通"],"allowCustom":true}]}',
      '["title","factoryCategory","dailyCapacity","contactPhone"]',
      '["factoryCategory","dailyCapacity","acceptSmallOrders","availableSchedule","processingMode"]',
      '{"summary":{"category":"factoryCategory","quantityText":"dailyCapacity","priceText":"laborPriceText"},"list":["quantityText","priceText","district"],"detail":["factoryCategory","dailyCapacity","minOrderQuantity","acceptSmallOrders","availableSchedule","laborPriceText","processingMode"]}',
      '{"resubmitOnChange":["title","factoryCategory","dailyCapacity","laborPriceText","contactPhone"]}',
      '{"verified":25,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":3}',
      15
    ),
    (
      'job',
      '招工招聘',
      'supply',
      '{"fields":[{"key":"position","label":"岗位","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["平车工","拷边工","裁剪工","后道","包装工"],"allowCustom":true},{"key":"payText","label":"工价/薪资","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：计件 0.8-1.2 元、月薪区间"},{"key":"headcount","label":"人数","type":"number","required":true,"filterable":false,"displayIn":["detail"],"placeholder":"例如：8"},{"key":"includeMealsHousing","label":"包吃住","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"workLocation","label":"工作地点","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：织里某园区"},{"key":"settlementMode","label":"结算方式","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["计件","日结","月结","面议"],"allowCustom":true},{"key":"shiftTime","label":"班次/工作时长","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：白班、晚班、长白班"}]}',
      '["title","position","payText","headcount","workLocation","contactPhone"]',
      '["position","payText","includeMealsHousing","workLocation","settlementMode"]',
      '{"summary":{"category":"position","quantityText":"headcount","priceText":"payText"},"list":["priceText","quantityText","district"],"detail":["position","payText","headcount","includeMealsHousing","workLocation","settlementMode","shiftTime"]}',
      '{"resubmitOnChange":["title","position","payText","headcount","workLocation","contactPhone"]}',
      '{"verified":10,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":2}',
      15
    ),
    (
      'rental',
      '出租转让',
      'supply',
      '{"fields":[{"key":"rentalType","label":"房源类型","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["厂房","档口","仓库","商铺","商品房","宿舍","设备","其他"],"allowCustom":true},{"key":"areaText","label":"面积","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：120 平"},{"key":"rentText","label":"租金","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：6800 元/月"},{"key":"locationText","label":"位置","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：童装城附近、某园区"},{"key":"floor","label":"楼层","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：1 楼"},{"key":"transferFee","label":"转让费","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：无"},{"key":"leaseTerms","label":"租期条件","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：押金、租期、可入驻时间"}]}',
      '["title","rentalType","areaText","rentText","locationText","contactPhone"]',
      '["rentalType","areaText","rentText","locationText","floor"]',
      '{"summary":{"category":"rentalType","quantityText":"areaText","priceText":"rentText"},"list":["priceText","quantityText","district"],"detail":["rentalType","areaText","rentText","locationText","floor","transferFee","leaseTerms"]}',
      '{"resubmitOnChange":["title","rentalType","areaText","rentText","locationText","contactPhone"]}',
      '{"verified":10,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":5}',
      30
    ),
    (
      'service',
      '配套服务',
      'supply',
      '{"fields":[{"key":"serviceType","label":"服务类型","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["物流","辅料","印花","绣花","摄影","直播","包装"],"allowCustom":true},{"key":"serviceArea","label":"服务范围","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：织里及周边、全国接单"},{"key":"servicePriceText","label":"价格方式","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：按单报价、按件计费、面议"},{"key":"responseTime","label":"响应时效","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：当天出样、24 小时响应"},{"key":"caseAvailable","label":"服务案例","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"serviceRegion","label":"可服务区域","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：湖州、杭州、长三角"}]}',
      '["title","serviceType","serviceArea","contactPhone"]',
      '["serviceType","serviceArea","caseAvailable","serviceRegion"]',
      '{"summary":{"category":"serviceType","quantityText":"serviceArea","priceText":"servicePriceText"},"list":["quantityText","priceText","district"],"detail":["serviceType","serviceArea","servicePriceText","responseTime","caseAvailable","serviceRegion"]}',
      '{"resubmitOnChange":["title","serviceType","serviceArea","servicePriceText","contactPhone"]}',
      '{"verified":20,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":5}',
      30
    ),
    (
      'buy_goods',
      '找现货',
      'demand',
      '{"fields":[{"key":"targetCategory","label":"目标品类","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["童装","女装","卫衣","套装","面料"],"allowCustom":true},{"key":"targetStyle","label":"目标风格","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["韩版","学院风","运动风","国潮","基础款"],"allowCustom":true},{"key":"demandQuantityText","label":"需求数量","type":"text","required":true,"filterable":false,"displayIn":["detail"],"placeholder":"例如：3000 件、长期每周 500 件"},{"key":"budgetRange","label":"预算范围","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：20-30 元/件"},{"key":"deliveryDeadline","label":"交付时效","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：7 天内发货"},{"key":"acceptRemoteShipping","label":"接受外地发货","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]}]}',
      '["title","targetCategory","demandQuantityText","contactPhone"]',
      '["targetCategory","targetStyle","deliveryDeadline","acceptRemoteShipping"]',
      '{"summary":{"category":"targetCategory","quantityText":"demandQuantityText","priceText":"budgetRange"},"list":["quantityText","priceText","district"],"detail":["targetCategory","targetStyle","demandQuantityText","budgetRange","deliveryDeadline","acceptRemoteShipping"]}',
      '{"resubmitOnChange":["title","targetCategory","demandQuantityText","budgetRange","contactPhone"]}',
      '{"freshDemand":20,"refreshedAt":10}',
      '{"expiringSoonDays":2}',
      7
    ),
    (
      'find_inventory',
      '找库存',
      'demand',
      '{"fields":[{"key":"targetCategory","label":"目标品类","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["童装","女装","面料","辅料"],"allowCustom":true},{"key":"season","label":"季节","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["春季","夏季","秋季","冬季"],"allowCustom":false},{"key":"sizeRange","label":"尺码段","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：90-140"},{"key":"demandQuantityText","label":"需求数量","type":"text","required":true,"filterable":false,"displayIn":["detail"],"placeholder":"例如：10 包、5000 件"},{"key":"budgetRange","label":"预算范围","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：按包议价、低价清仓优先"},{"key":"acceptMixedLot","label":"接受混批","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]}]}',
      '["title","targetCategory","demandQuantityText","contactPhone"]',
      '["targetCategory","season","sizeRange","acceptMixedLot"]',
      '{"summary":{"category":"targetCategory","quantityText":"demandQuantityText","priceText":"budgetRange"},"list":["quantityText","priceText","district"],"detail":["targetCategory","season","sizeRange","demandQuantityText","budgetRange","acceptMixedLot"]}',
      '{"resubmitOnChange":["title","targetCategory","demandQuantityText","budgetRange","contactPhone"]}',
      '{"freshDemand":20,"refreshedAt":10}',
      '{"expiringSoonDays":2}',
      7
    ),
    (
      'find_factory',
      '找工厂',
      'demand',
      '{"fields":[{"key":"targetCategory","label":"加工品类","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["童装","女装","羽绒服","针织","梭织"],"allowCustom":true},{"key":"orderQuantity","label":"订单数量","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：5000 件"},{"key":"deliveryDeadline","label":"交期","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：20 天"},{"key":"sampleRequired","label":"需要打样","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"longTermCooperation","label":"长期合作","type":"boolean","required":false,"filterable":false,"displayIn":["detail"]},{"key":"processRequirement","label":"工艺要求","type":"textarea","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"说明面料、版型、工艺和质检要求"},{"key":"budgetRange","label":"预算范围","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：按款报价、目标成本区间"}]}',
      '["title","targetCategory","orderQuantity","deliveryDeadline","contactPhone"]',
      '["targetCategory","orderQuantity","deliveryDeadline","sampleRequired"]',
      '{"summary":{"category":"targetCategory","quantityText":"orderQuantity","priceText":"budgetRange"},"list":["quantityText","priceText","district"],"detail":["targetCategory","orderQuantity","deliveryDeadline","sampleRequired","longTermCooperation","processRequirement","budgetRange"]}',
      '{"resubmitOnChange":["title","targetCategory","orderQuantity","deliveryDeadline","budgetRange","contactPhone"]}',
      '{"verified":15,"refreshedAt":10,"contactCount":5}',
      '{"expiringSoonDays":3}',
      10
    ),
    (
      'find_service',
      '找服务',
      'demand',
      '{"fields":[{"key":"serviceType","label":"服务类型","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["物流","辅料","印花","绣花","摄影","直播","包装"],"allowCustom":true},{"key":"serviceArea","label":"服务区域","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：织里及周边、湖州、杭州"},{"key":"budgetRange","label":"预算范围","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：按单报价、面议、目标费用区间"},{"key":"deliveryDeadline","label":"交付时效","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：3 天内、当天响应"},{"key":"longTermCooperation","label":"长期合作","type":"boolean","required":false,"filterable":false,"displayIn":["detail"]},{"key":"serviceRequirement","label":"服务要求","type":"textarea","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"说明质量、响应、案例、发票等补充要求"}]}',
      '["title","serviceType","serviceArea","contactPhone"]',
      '["serviceType","serviceArea"]',
      '{"summary":{"category":"serviceType","quantityText":"serviceArea","priceText":"budgetRange"},"list":["quantityText","priceText","district"],"detail":["serviceType","serviceArea","budgetRange","deliveryDeadline","longTermCooperation","serviceRequirement"]}',
      '{"resubmitOnChange":["title","serviceType","serviceArea","budgetRange","contactPhone"]}',
      '{"freshDemand":20,"refreshedAt":10}',
      '{"expiringSoonDays":3}',
      15
    )
) AS cfg(
  type_code,
  type_name,
  direction,
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
  direction = EXCLUDED.direction,
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
    'internal',
    '/pages/search/index',
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
  '库存、货源、工厂资源一站式服务',
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
