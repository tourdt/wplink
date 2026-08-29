-- MVP 端到端演示数据。
-- 前置条件：已按顺序执行 migrations 目录下的所有 up migration。

BEGIN;

INSERT INTO users (id, phone, wechat_openid, nickname, default_city_station_id, status, last_login_at)
SELECT u.id, u.phone, u.openid, u.nickname, cs.id, 'active', now()
FROM city_stations cs
CROSS JOIN (
  VALUES
    (8010000000000000002, '19900000002', 'demo_factory_admin_openid', '童装工厂管理员'),
    (8010000000000000003, '19900000003', 'demo_stockist_admin_openid', '库存商管理员'),
    (8010000000000000004, '19900000004', 'demo_service_admin_openid', '服务商管理员'),
    (8010000000000000005, '19900000005', 'demo_buyer_openid', '采购商买家'),
    (8010000000000000006, '19900000006', 'demo_material_admin_openid', '面辅料商管理员'),
    (8010000000000000007, '19900000007', 'demo_property_admin_openid', '产业物业管理员')
) AS u(id, phone, openid, nickname)
WHERE cs.code = 'zhili'
ON CONFLICT (id) DO UPDATE SET
  phone = EXCLUDED.phone,
  wechat_openid = EXCLUDED.wechat_openid,
  nickname = EXCLUDED.nickname,
  default_city_station_id = EXCLUDED.default_city_station_id,
  status = 'active',
  updated_at = now();

INSERT INTO admin_operators (id, login_name, real_name, status)
VALUES (8009000000000000001, '19900000001', '演示超级管理员', 'enabled')
ON CONFLICT (id) DO UPDATE SET
  login_name = EXCLUDED.login_name,
  real_name = EXCLUDED.real_name,
  status = EXCLUDED.status,
  updated_at = now();

INSERT INTO admin_operator_role_assignments (operator_id, role_id)
SELECT 8009000000000000001, ar.id
FROM admin_roles ar
WHERE ar.code = 'super_admin'
  AND NOT EXISTS (
    SELECT 1
    FROM admin_operator_role_assignments aora
    WHERE aora.operator_id = 8009000000000000001
      AND aora.role_id = ar.id
  );

INSERT INTO merchants (
  id,
  city_station_id,
  name,
  merchant_type,
  main_categories,
  description,
  contact_name,
  contact_phone,
  contact_wechat,
  address_text,
  market_name,
  building_name,
  floor_no,
  booth_no,
  latitude,
  longitude,
  location_name,
  location_status,
  location_updated_at,
  images,
  status,
  profile_status,
  onboarded_at,
  last_active_at
)
SELECT
  m.id,
  cs.id,
  m.name,
  m.merchant_type,
  m.main_categories::jsonb,
  m.description,
  m.contact_name,
  m.contact_phone,
  m.contact_wechat,
  m.address_text,
  m.market_name,
  m.building_name,
  m.floor_no,
  m.booth_no,
  m.latitude,
  m.longitude,
  m.location_name,
  m.location_status,
  CASE WHEN m.location_status = 'visible' THEN now() END,
  m.images::jsonb,
  'active',
  'completed',
  now(),
  now()
FROM city_stations cs
CROSS JOIN (
  VALUES
    (
      8020000000000000001,
      '湖州织里晨星童装厂',
      'factory',
      '["童装","卫衣","套装"]',
      '主做童装卫衣和套装，可承接小单快反。',
      '陈厂长',
      '18800000001',
      'factory-demo',
      '织里镇利济路88号',
      '织里童装城',
      '一期',
      '1F',
      'A108',
      30.8732,
      120.2255,
      '织里童装城',
      'visible',
      '[]'
    ),
    (
      8020000000000000002,
      '织里云仓尾货',
      'stockist',
      '["童装","库存尾货"]',
      '长期处理整包库存和直播货盘。',
      '周经理',
      '18800000002',
      'stock-demo',
      '织里童装城3区',
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      'hidden',
      '[]'
    ),
    (
      8020000000000000003,
      '织里快印包装服务商',
      'service_provider',
      '["吊牌","包装","拍摄"]',
      '服务商，提供童装包装、快印和电商拍摄配套服务。',
      '李经理',
      '18800000003',
      'service-demo',
      '织里镇阿祥路19号',
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      'hidden',
      '[]'
    ),
    (
      8020000000000000004,
      '杭州童装采购商',
      'buyer',
      '["童装","电商供货"]',
      '采购商，长期寻找织里现货、工厂和快反供需信息。',
      '王采购',
      '18800000004',
      'buyer-demo',
      '杭州市滨江区',
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      'hidden',
      '[]'
    ),
    (
      8020000000000000005,
      '织里面辅料现货中心',
      'material_supplier',
      '["面料","辅料","现货"]',
      '提供童装面料、吊牌、洗标和包装辅料现货，可当天打样。',
      '赵经理',
      '18800000005',
      'material-demo',
      '织里镇吴兴大道面辅料市场',
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      'hidden',
      '[]'
    ),
    (
      8020000000000000006,
      '织里产业物业服务中心',
      'property_service',
      '["商铺","住宅","厂房仓库"]',
      '提供织里产业商铺、员工公寓、厂房和仓库出租出售服务。',
      '孙经理',
      '18800000006',
      'property-demo',
      '织里镇利济路产业服务中心',
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      NULL,
      'hidden',
      '[]'
    )
) AS m(
  id,
  name,
  merchant_type,
  main_categories,
  description,
  contact_name,
  contact_phone,
  contact_wechat,
  address_text,
  market_name,
  building_name,
  floor_no,
  booth_no,
  latitude,
  longitude,
  location_name,
  location_status,
  images
)
WHERE cs.code = 'zhili'
ON CONFLICT (id) DO UPDATE SET
  city_station_id = EXCLUDED.city_station_id,
  name = EXCLUDED.name,
  merchant_type = EXCLUDED.merchant_type,
  main_categories = EXCLUDED.main_categories,
  description = EXCLUDED.description,
  contact_name = EXCLUDED.contact_name,
  contact_phone = EXCLUDED.contact_phone,
  contact_wechat = EXCLUDED.contact_wechat,
  address_text = EXCLUDED.address_text,
  market_name = EXCLUDED.market_name,
  building_name = EXCLUDED.building_name,
  floor_no = EXCLUDED.floor_no,
  booth_no = EXCLUDED.booth_no,
  latitude = EXCLUDED.latitude,
  longitude = EXCLUDED.longitude,
  location_name = EXCLUDED.location_name,
  location_status = EXCLUDED.location_status,
  location_updated_at = EXCLUDED.location_updated_at,
  images = EXCLUDED.images,
  status = EXCLUDED.status,
  profile_status = EXCLUDED.profile_status,
  onboarded_at = COALESCE(merchants.onboarded_at, EXCLUDED.onboarded_at),
  last_active_at = EXCLUDED.last_active_at,
  updated_at = now();

INSERT INTO merchant_admin_bindings (id, merchant_id, user_id, role, status, created_by)
VALUES
  (8021000000000000001, 8020000000000000001, 8010000000000000002, 'owner', 'active', 8010000000000000002),
  (8021000000000000002, 8020000000000000002, 8010000000000000003, 'owner', 'active', 8010000000000000003),
  (8021000000000000003, 8020000000000000003, 8010000000000000004, 'owner', 'active', 8010000000000000004),
  (8021000000000000004, 8020000000000000004, 8010000000000000005, 'owner', 'active', 8010000000000000005),
  (8021000000000000005, 8020000000000000005, 8010000000000000006, 'owner', 'active', 8010000000000000006),
  (8021000000000000006, 8020000000000000006, 8010000000000000007, 'owner', 'active', 8010000000000000007)
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, revoked_at = NULL;

INSERT INTO resources (
  id,
  merchant_id,
  city_station_id,
  resource_type_config_id,
  resource_type_config_version,
  resource_type_snapshot,
  type_code,
  direction,
  status,
  title,
  category,
  district,
  price_text,
  quantity_text,
  cover_url,
  description,
  attributes,
  tags,
  images,
  contact_name,
  contact_phone,
  contact_wechat,
  published_at,
  refreshed_at,
  top_started_at,
  top_expires_at,
  expires_at,
  dealt_at,
  taken_down_at,
  reject_reason,
  take_down_reason,
  created_by_user_id
)
SELECT
  r.id,
  r.merchant_id,
  cs.id,
  rtc.id,
  rtc.version,
  jsonb_build_object(
    'version', rtc.version,
    'typeCode', rtc.type_code,
    'typeName', rtc.type_name,
    'direction', rtc.direction,
    'fieldSchema', rtc.field_schema,
    'requiredFields', rtc.required_fields,
    'filterFields', rtc.filter_fields,
    'displayTemplate', rtc.display_template,
    'reviewRules', rtc.review_rules,
    'sortWeights', rtc.sort_weights,
    'messageRules', rtc.message_rules,
    'commercialRules', rtc.commercial_rules,
    'defaultValidDays', rtc.default_valid_days
  ),
  r.type_code,
  rtc.direction,
  r.status,
  r.title,
  r.category,
  r.district,
  r.price_text,
  r.quantity_text,
  r.cover_url,
  r.description,
  r.attributes::jsonb,
  r.tags::jsonb,
  '[]'::jsonb,
  r.contact_name,
  r.contact_phone,
  r.contact_wechat,
  r.published_at,
  r.refreshed_at,
  r.top_started_at,
  r.top_expires_at,
  r.expires_at,
  r.dealt_at,
  r.taken_down_at,
  r.reject_reason,
  r.take_down_reason,
  r.created_by_user_id
FROM city_stations cs
JOIN resource_type_configs rtc ON rtc.city_station_id = cs.id
CROSS JOIN (
  VALUES
    (8030000000000000001, 8020000000000000001, 'factory_direct', 'published', '童装套装源头工厂一件代发', '套装', '织里', '36-48 元/套', '20 套起批', '/static/home/factory-hero.jpg', '源头工厂现货套装，支持小单快反和一件代发。', '{"productCategory":"套装","factoryPriceText":"36-48 元/套","minOrderText":"20 套起批","factoryAdvantage":"小单快反","supportsDropship":true,"deliveryArea":"全国"}', '["源头工厂","一件代发","小单快反"]', '陈厂长', '18800000001', 'factory-demo', true, now() - interval '4 days', now() - interval '3 hours', NULL, NULL, now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000002),
    (8030000000000000002, 8020000000000000002, 'spot_wholesale', 'published', '夏款童裙尾货整包批发', '裙装', '织里', '15-22 元/件', '2600 件', '', '夏季童裙整包尾货，可现场看货并支持拿样。', '{"stockCategory":"裙装","stockQuantityText":"2600 件","wholesalePriceText":"15-22 元/件","season":"夏季","sizeRange":"90-140","allowSample":true}', '["尾货","裙装","可拿样"]', '周经理', '18800000002', 'stock-demo', true, now() - interval '3 days', now() - interval '4 hours', NULL, NULL, now() + interval '10 days', NULL, NULL, NULL, NULL, 8010000000000000003),
    (8030000000000000003, 8020000000000000002, 'stock_clearance', 'published', '女童春款卫衣库存整包清', '女童', '织里', '18-26 元/件', '3800 件', '/static/home/factory-hero.jpg', '女童春款卫衣库存，支持整包、混批和直播拿样。', '{"clearanceCategory":"女童","stockQuantityText":"3800 件","packagePriceText":"18-26 元/件","stockCondition":"整包","warehouseLocation":"织里童装城 3 区","allowMixedLot":true}', '["库存","整包","直播货盘"]', '周经理', '18800000002', 'stock-demo', true, now() - interval '2 days', now() - interval '1 hours', NULL, NULL, now() + interval '1 day', NULL, NULL, NULL, NULL, 8010000000000000003),
    (8030000000000000004, 8020000000000000004, 'buy_kids_goods', 'published', '求购中大童夏款混批尾货', '混款', '杭州', '单件 10-25 元', '3000 件', '', '采购商求购中大童夏款混批尾货，接受断码并要求一周内交付。', '{"targetCategory":"混款","demandQuantityText":"3000 件","budgetRange":"单件 10-25 元","expectedDelivery":"7 天内","acceptTailStock":true,"purchaseArea":"杭州"}', '["求购","中大童","尾货"]', '王采购', '18800000004', 'buyer-demo', false, now() - interval '2 days', now() - interval '5 hours', NULL, NULL, now() + interval '7 days', NULL, NULL, NULL, NULL, 8010000000000000005),
    (8030000000000000005, 8020000000000000005, 'fabric_supply', 'published', '320 克纯棉卫衣布现货', '卫衣布', '织里', '26 元/公斤', '现货 8 吨', '/static/home/factory-hero.jpg', '纯棉卫衣布现货，多色可选，适合秋冬童装。', '{"fabricType":"卫衣布","fabricComposition":"纯棉","widthWeight":"185cm / 320g","fabricPriceText":"26 元/公斤","inStockText":"现货 8 吨","colorCount":16}', '["面料","卫衣布","现货"]', '赵经理', '18800000005', 'material-demo', true, now() - interval '5 days', now() - interval '6 hours', NULL, NULL, now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000006),
    (8030000000000000006, 8020000000000000005, 'accessory_supply', 'published', '童装吊牌洗标当天打样', '吊牌', '织里', '0.12 元/套起', '日产 10 万套', '', '童装吊牌、洗标和包装辅料支持当天打样。', '{"accessoryType":"吊牌","materialSpec":"350g 白卡覆膜","stockQuantityText":"日产 10 万套","accessoryPriceText":"0.12 元/套起","minOrderText":"1000 套"}', '["辅料","吊牌","当天打样"]', '赵经理', '18800000005', 'material-demo', true, now() - interval '4 days', now() - interval '7 hours', NULL, NULL, now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000006),
    (8030000000000000007, 8020000000000000001, 'processing_accept', 'published', '童装卫衣工厂本周空档接单', '卫衣', '织里', '按工艺核价', '日产 1200 件', '', '童装工厂本周有空档，可承接童装卫衣来料加工。', '{"processCategory":"卫衣","dailyCapacity":"日产 1200 件","processingMode":"来料加工","processingPriceText":"按工艺核价","availableSchedule":"本周可排","acceptSmallOrders":true}', '["源头工厂","快反","本周可排"]', '陈厂长', '18800000001', 'factory-demo', true, now() - interval '1 day', now() - interval '30 minutes', now() - interval '1 hours', now() + interval '24 hours', now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000002),
    (8030000000000000008, 8020000000000000004, 'find_factory', 'published', '寻找女童防晒衣加工厂', '童装', '杭州', '面议', '5000 件', '', '采购商寻找织里工厂承接女童防晒衣订单，交期二十天。', '{"targetCategory":"童装","orderQuantity":"5000 件","deliveryDeadline":"20 天","processingMode":"包工包料","budgetRange":"面议","sampleRequired":true}', '["订单","招加工厂","需要打样"]', '王采购', '18800000004', 'buyer-demo', false, now() - interval '1 day', now() - interval '2 hours', NULL, NULL, now() + interval '10 days', NULL, NULL, NULL, NULL, 8010000000000000005),
    (8030000000000000009, 8020000000000000003, 'production_support', 'published', '童装裁床整烫配套服务', '裁床', '织里', '按件计费', '织里及周边', '', '提供童装裁床、整烫和检品配套，当天响应。', '{"supportType":"裁床","serviceArea":"织里及周边","supportPriceText":"按件计费","responseTime":"当天响应","equipmentAvailable":true}', '["生产配套","裁床","当天响应"]', '李经理', '18800000003', 'service-demo', true, now() - interval '6 days', now() - interval '8 hours', NULL, NULL, now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000004),
    (8030000000000000010, 8020000000000000001, 'job_hiring', 'taken_down', '历史平车工招聘信息', '平车工', '织里', '计件 0.8-1.2 元', '8 人', '', '招聘入口下线前的历史平车工招聘演示。', '{"position":"平车工","payText":"计件 0.8-1.2 元","headcount":8,"workLocation":"织里镇利济路 88 号","includeMealsHousing":true,"settlementMode":"计件"}', '["历史招聘","平车工"]', '陈厂长', '18800000001', 'factory-demo', true, now() - interval '30 days', now() - interval '10 days', NULL, NULL, now() + interval '1 day', NULL, now() - interval '5 days', NULL, '冷启动阶段暂停招聘入口', 8010000000000000002),
    (8030000000000000011, 8020000000000000004, 'job_seeking', 'expired', '历史熟练车工求职信息', '平车工', '织里', '月薪 9000 元以上', '5 年经验', '', '求职入口下线前的历史熟练车工求职演示。', '{"desiredPosition":"平车工","expectedPayText":"月薪 9000 元以上","experienceYears":"5 年","availableTime":"随时到岗","expectedLocation":"织里"}', '["历史求职","平车工"]', '王采购', '18800000004', 'buyer-demo', false, now() - interval '30 days', now() - interval '20 days', NULL, NULL, now() - interval '1 day', NULL, NULL, NULL, NULL, 8010000000000000005),
    (8030000000000000012, 8020000000000000003, 'sample_rental', 'published', '直播童装样衣按周出租', '直播样衣间', '织里', '1200 元/周', '80 平', '', '直播童装样衣间按周出租，可陈列五百款样衣。', '{"sampleRoomType":"直播样衣间","areaText":"80 平","rentText":"1200 元/周","locationText":"织里童装城 2 区","displayCapacity":"可陈列 500 款"}', '["样衣出租","直播","按周租"]', '李经理', '18800000003', 'service-demo', true, now() - interval '7 days', now() - interval '9 hours', NULL, NULL, now() + interval '20 days', NULL, NULL, NULL, NULL, 8010000000000000004),
    (8030000000000000013, 8020000000000000006, 'shop_office_rental', 'published', '童装城一楼沿街商铺出租', '沿街商铺', '织里', '9800 元/月', '95 平', '', '童装城一楼沿街商铺，适合档口和直播。', '{"spaceType":"沿街商铺","areaText":"95 平","rentText":"9800 元/月","locationText":"织里童装城一楼","floor":"1 楼","availableTime":"随时可租"}', '["商铺出租","一楼","沿街"]', '孙经理', '18800000006', 'property-demo', true, now() - interval '8 days', now() - interval '10 hours', NULL, NULL, now() + interval '30 days', NULL, NULL, NULL, NULL, 8010000000000000007),
    (8030000000000000014, 8020000000000000006, 'shop_sale', 'published', '利济路成熟童装商铺出售', '沿街商铺', '织里', '总价 260 万元', '110 平', '', '利济路成熟童装商铺出售，产权清晰可过户。', '{"saleType":"沿街商铺","areaText":"110 平","salePriceText":"总价 260 万元","locationText":"织里利济路中段","certificateStatus":"可过户"}', '["商铺出售","可过户","利济路"]', '孙经理', '18800000006', 'property-demo', true, now() - interval '9 days', now() - interval '11 hours', NULL, NULL, now() + interval '30 days', NULL, NULL, NULL, NULL, 8010000000000000007),
    (8030000000000000015, 8020000000000000004, 'seek_shop_office', 'published', '求租童装城一楼直播档口', '直播档口', '织里', '1.2 万元/月以内', '80-120 平', '', '求租童装城一楼直播档口，用于样衣展示和直播。', '{"desiredSpaceType":"直播档口","expectedAreaText":"80-120 平","budgetRentText":"1.2 万元/月以内","preferredDistrict":"织里童装城一楼","usagePurpose":"直播与样衣展示"}', '["求租","直播档口","一楼"]', '王采购', '18800000004', 'buyer-demo', false, now() - interval '2 days', now() - interval '12 hours', NULL, NULL, now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000005),
    (8030000000000000016, 8020000000000000006, 'apartment_rental', 'published', '织里工厂附近员工公寓出租', '员工公寓', '织里', '3200 元/月', '三室一厅', '', '工厂集中区员工公寓出租，家具齐全，本周可入住。', '{"housingType":"员工公寓","roomLayout":"三室一厅","rentText":"3200 元/月","locationText":"织里镇工厂集中区","moveInTime":"本周可入住","furnished":true}', '["公寓出租","员工宿舍","拎包入住"]', '孙经理', '18800000006', 'property-demo', true, now() - interval '10 days', now() - interval '13 hours', NULL, NULL, now() + interval '30 days', NULL, NULL, NULL, NULL, 8010000000000000007),
    (8030000000000000017, 8020000000000000006, 'housing_sale', 'published', '织里两居室住宅诚意出售', '普通住宅', '织里', '总价 138 万元', '89 平', '', '织里镇中心两居室住宅，满两年，诚意出售。', '{"housingType":"普通住宅","areaText":"89 平","salePriceText":"总价 138 万元","locationText":"织里镇中心","certificateStatus":"满两年"}', '["住宅出售","两居室","满两年"]', '孙经理', '18800000006', 'property-demo', true, now() - interval '11 days', now() - interval '14 hours', NULL, NULL, now() + interval '30 days', NULL, NULL, NULL, NULL, 8010000000000000007),
    (8030000000000000018, 8020000000000000004, 'seek_housing', 'published', '求租可住六人的员工宿舍', '员工宿舍', '织里', '3500 元/月以内', '90 平以上', '', '为工厂员工求租可住六人的宿舍，一周内入住。', '{"desiredHousingType":"员工宿舍","expectedAreaText":"90 平以上","budgetRentText":"3500 元/月以内","preferredDistrict":"织里工厂集中区","moveInTime":"一周内"}', '["求租住房","员工宿舍","一周入住"]', '王采购', '18800000004', 'buyer-demo', false, now() - interval '3 days', now() - interval '15 hours', NULL, NULL, now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000005),
    (8030000000000000019, 8020000000000000006, 'factory_warehouse_rental', 'published', '童装城旁一楼仓库出租', '仓库', '织里', '6800 元/月', '120 平', '', '童装城旁一楼仓库，可进货车并配备工业用电。', '{"placeType":"仓库","areaText":"120 平","rentText":"6800 元/月","locationText":"织里童装城旁","powerCapacity":"50kW","truckAccess":true}', '["仓库出租","一楼仓","货车可进"]', '孙经理', '18800000006', 'property-demo', true, now() - interval '5 days', now() - interval '16 hours', NULL, NULL, now() + interval '30 days', NULL, NULL, NULL, NULL, 8010000000000000007),
    (8030000000000000020, 8020000000000000006, 'workshop_rental', 'published', '带设备缝制加工间出租', '缝制间', '织里', '1.6 万元/月', '260 平', '', '带平车设备的缝制加工间出租，可直接进场生产。', '{"workshopType":"缝制间","equipmentIncluded":true,"areaText":"260 平","rentText":"1.6 万元/月","locationText":"织里镇阿祥路"}', '["加工间出租","带设备","缝制间"]', '孙经理', '18800000006', 'property-demo', true, now() - interval '6 days', now() - interval '17 hours', NULL, NULL, now() + interval '30 days', NULL, NULL, NULL, NULL, 8010000000000000007),
    (8030000000000000021, 8020000000000000006, 'factory_sale', 'published', '织里园区独栋厂房出售', '独栋厂房', '织里', '总价 1680 万元', '3200 平', '', '织里产业园独栋厂房，有证并配备六百三十千伏安电力。', '{"placeType":"独栋厂房","areaText":"3200 平","salePriceText":"总价 1680 万元","locationText":"织里产业园","certificateStatus":"有证","powerCapacity":"630kVA"}', '["厂房出售","独栋","有证"]', '孙经理', '18800000006', 'property-demo', true, now() - interval '12 days', now() - interval '18 hours', NULL, NULL, now() + interval '30 days', NULL, NULL, NULL, NULL, 8010000000000000007),
    (8030000000000000022, 8020000000000000004, 'seek_factory_warehouse', 'published', '求租可进货车的一楼仓库', '一楼仓', '织里', '3 万元/月以内', '500-800 平', '', '求租织里东部一楼仓库，需要货车进出和工业用电。', '{"desiredPlaceType":"一楼仓","expectedAreaText":"500-800 平","budgetRentText":"3 万元/月以内","preferredDistrict":"织里东部","powerRequirement":"100kW 以上","truckAccessRequired":true}', '["求租仓库","一楼仓","货车可进"]', '王采购', '18800000004', 'buyer-demo', false, now() - interval '4 days', now() - interval '19 hours', NULL, NULL, now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000005),
    (8030000000000000023, 8020000000000000002, 'secondhand_sale', 'published', '九成新自动裁床设备转让', '设备', '织里', '转让价 12 万元', '1 台', '', '九成新自动裁床设备转让，可现场试机并自提。', '{"itemType":"设备","conditionLevel":"九成新","secondhandPriceText":"转让价 12 万元","secondhandQuantityText":"1 台","pickupLocation":"织里镇利济路"}', '["二手设备","自动裁床","可试机"]', '周经理', '18800000002', 'stock-demo', true, now() - interval '8 days', now() - interval '20 hours', NULL, NULL, now() + interval '20 days', now() - interval '2 days', NULL, NULL, NULL, 8010000000000000003),
    (8030000000000000024, 8020000000000000004, 'secondhand_buy', 'published', '求购二手童装货架二十组', '货架', '杭州', '每组 300 元以内', '20 组', '', '求购二手童装货架二十组，接受正常使用成色。', '{"wantedItemType":"货架","budgetRange":"每组 300 元以内","demandQuantityText":"20 组","pickupArea":"织里及周边","acceptUsedCondition":true}', '["求购二手","童装货架","织里周边"]', '王采购', '18800000004', 'buyer-demo', false, now() - interval '5 days', now() - interval '21 hours', NULL, NULL, now() + interval '15 days', NULL, NULL, NULL, NULL, 8010000000000000005),
    (8030000000000000025, 8020000000000000003, 'education_training', 'published', '童装制版与电商运营培训', '童装制版', '织里', '1980 元/期', '工厂版师和创业团队', '', '面向工厂版师和创业团队的童装制版与电商运营培训。', '{"courseType":"童装制版","targetAudience":"工厂版师和创业团队","coursePriceText":"1980 元/期","classTime":"周一至周五晚间","serviceArea":"织里"}', '["教育培训","童装制版","电商运营"]', '李经理', '18800000003', 'service-demo', true, now() - interval '6 days', now() - interval '22 hours', NULL, NULL, now() + interval '30 days', NULL, NULL, NULL, NULL, 8010000000000000004),
    (8030000000000000026, 8020000000000000003, 'appliance_repair', 'published', '缝纫设备上门检修服务', '缝纫设备', '织里', '上门检测 80 元起', '织里全域', '', '缝纫设备上门检修，织里全域两小时内响应。', '{"applianceType":"缝纫设备","serviceArea":"织里全域","repairPriceText":"上门检测 80 元起","responseTime":"2 小时内响应"}', '["设备维修","上门服务","两小时响应"]', '李经理', '18800000003', 'service-demo', true, now() - interval '7 days', now() - interval '23 hours', NULL, NULL, now() + interval '20 days', NULL, NULL, NULL, NULL, 8010000000000000004),
    (8030000000000000027, 8020000000000000003, 'moving_cleaning', 'published', '仓库搬运与开荒保洁', '仓库搬运', '织里', '按车次和人工报价', '织里及周边 20 公里', '', '提供仓库搬运、设备拆装和开荒保洁，当天可预约。', '{"serviceType":"仓库搬运","serviceArea":"织里及周边 20 公里","movingPriceText":"按车次和人工报价","appointmentTime":"当天可预约"}', '["搬家保洁","仓库搬运","当天预约"]', '李经理', '18800000003', 'service-demo', true, now() - interval '8 days', now() - interval '1 day', NULL, NULL, now() + interval '20 days', NULL, NULL, NULL, NULL, 8010000000000000004),
    (8030000000000000028, 8020000000000000003, 'other_local_service', 'published', '童装拍摄与广告制作服务', '摄影拍摄', '织里', '主图套拍 399 元起', '织里童装城', '', '提供童装主图拍摄、短视频和广告物料制作，次日交片。', '{"serviceType":"摄影拍摄","serviceArea":"织里童装城","localServicePriceText":"主图套拍 399 元起","responseTime":"次日交片"}', '["童装拍摄","广告制作","次日交片"]', '李经理', '18800000003', 'service-demo', true, now() - interval '9 days', now() - interval '26 hours', NULL, NULL, now() + interval '20 days', NULL, NULL, NULL, NULL, 8010000000000000004),
    (8030000000000000029, 8020000000000000002, 'stock_clearance', 'draft', '尚未提交的秋款库存草稿', '中大童', '织里', '待补充', '待盘点', '', '商家尚未完成的秋款库存草稿，用于调试继续编辑。', '{"clearanceCategory":"中大童","stockQuantityText":"待盘点","packagePriceText":"待补充","stockCondition":"尾季","warehouseLocation":"织里仓库","allowMixedLot":false}', '["草稿","秋款库存"]', '周经理', '18800000002', 'stock-demo', true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 8010000000000000003),
    (8030000000000000030, 8020000000000000002, 'stock_clearance', 'pending', '待审核夏款短袖库存', '男童', '织里', '12-18 元/件', '1800 件', '', '已提交并等待内容审核的夏款短袖库存。', '{"clearanceCategory":"男童","stockQuantityText":"1800 件","packagePriceText":"12-18 元/件","stockCondition":"散货","warehouseLocation":"织里童装城 3 区","allowMixedLot":true}', '["待审核","夏款库存"]', '周经理', '18800000002', 'stock-demo', true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 8010000000000000003),
    (8030000000000000031, 8020000000000000001, 'factory_direct', 'manual_review', '等待人工复核的工厂货源', '卫衣', '织里', '30-42 元/件', '50 件起批', '', '自动审核转入人工复核的工厂卫衣货源。', '{"productCategory":"卫衣","factoryPriceText":"30-42 元/件","minOrderText":"50 件起批","factoryAdvantage":"来样定制","supportsDropship":false,"deliveryArea":"华东"}', '["人工复核","工厂货源"]', '陈厂长', '18800000001', 'factory-demo', true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 8010000000000000002),
    (8030000000000000032, 8020000000000000005, 'fabric_supply', 'audit_retry', '内容审核重试中的面料现货', '针织', '织里', '19 元/公斤', '现货 3 吨', '', '外部内容审核暂时失败，等待后台任务重试。', '{"fabricType":"针织","fabricComposition":"棉氨","widthWeight":"175cm / 220g","fabricPriceText":"19 元/公斤","inStockText":"现货 3 吨","colorCount":8}', '["审核重试","针织面料"]', '赵经理', '18800000005', 'material-demo', true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 8010000000000000006),
    (8030000000000000033, 8020000000000000001, 'factory_direct', 'rejected', '资料不完整的货源演示', '童装', '织里', '面议', '起批待确认', '', '资料不完整并已被驳回的工厂货源演示。', '{"productCategory":"童装","factoryPriceText":"面议","minOrderText":"起批待确认","factoryAdvantage":"源头工厂","supportsDropship":false,"deliveryArea":"织里"}', '["已驳回","资料不完整"]', '陈厂长', '18800000001', 'factory-demo', true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '缺少清晰价格和联系方式确认材料', NULL, 8010000000000000002),
    (8030000000000000034, 8020000000000000002, 'secondhand_sale', 'published', '已成交超过七天的旧设备', '设备', '织里', '成交价 3 万元', '2 台', '', '成交超过七天的旧设备，用于验证公开列表自动隐藏。', '{"itemType":"设备","conditionLevel":"正常使用","secondhandPriceText":"成交价 3 万元","secondhandQuantityText":"2 台","pickupLocation":"织里镇仓储区"}', '["历史成交","二手设备"]', '周经理', '18800000002', 'stock-demo', true, now() - interval '20 days', now() - interval '12 days', NULL, NULL, now() + interval '10 days', now() - interval '10 days', NULL, NULL, NULL, 8010000000000000003)
) AS r(id, merchant_id, type_code, status, title, category, district, price_text, quantity_text, cover_url, description, attributes, tags, contact_name, contact_phone, contact_wechat, seed_status, published_at, refreshed_at, top_started_at, top_expires_at, expires_at, dealt_at, taken_down_at, reject_reason, take_down_reason, created_by_user_id)
WHERE cs.code = 'zhili'
  AND rtc.type_code = r.type_code
ON CONFLICT (id) DO UPDATE SET
  merchant_id = EXCLUDED.merchant_id,
  city_station_id = EXCLUDED.city_station_id,
  resource_type_config_id = EXCLUDED.resource_type_config_id,
  resource_type_config_version = EXCLUDED.resource_type_config_version,
  resource_type_snapshot = EXCLUDED.resource_type_snapshot,
  type_code = EXCLUDED.type_code,
  direction = EXCLUDED.direction,
  status = EXCLUDED.status,
  title = EXCLUDED.title,
  category = EXCLUDED.category,
  district = EXCLUDED.district,
  price_text = EXCLUDED.price_text,
  quantity_text = EXCLUDED.quantity_text,
  cover_url = EXCLUDED.cover_url,
  description = EXCLUDED.description,
  attributes = EXCLUDED.attributes,
  tags = EXCLUDED.tags,
  contact_name = EXCLUDED.contact_name,
  contact_phone = EXCLUDED.contact_phone,
  contact_wechat = EXCLUDED.contact_wechat,
  published_at = EXCLUDED.published_at,
  refreshed_at = EXCLUDED.refreshed_at,
  top_started_at = EXCLUDED.top_started_at,
  top_expires_at = EXCLUDED.top_expires_at,
  expires_at = EXCLUDED.expires_at,
  dealt_at = EXCLUDED.dealt_at,
  taken_down_at = EXCLUDED.taken_down_at,
  reject_reason = EXCLUDED.reject_reason,
  take_down_reason = EXCLUDED.take_down_reason,
  created_by_user_id = EXCLUDED.created_by_user_id,
  updated_at = now();

INSERT INTO merchant_entitlements (id, merchant_id, entitlement_type, source_type, total_amount, remaining_amount, allowed_type_codes, top_duration_hours, expires_at, status)
VALUES
  (8042000000000000001, 8020000000000000001, 'publish_quota', 'demo_seed', 20, 18, '[]'::jsonb, 0, now() + interval '30 days', 'active'),
  (8042000000000000002, 8020000000000000002, 'refresh_quota', 'demo_seed', 30, 27, '[]'::jsonb, 0, now() + interval '30 days', 'active'),
  (8042000000000000003, 8020000000000000003, 'publish_quota', 'demo_seed', 20, 16, '[]'::jsonb, 0, now() + interval '30 days', 'active'),
  (8042000000000000004, 8020000000000000004, 'publish_quota', 'demo_debug', 20, 14, '[]'::jsonb, 0, now() + interval '30 days', 'active'),
  (8042000000000000005, 8020000000000000005, 'publish_quota', 'demo_seed', 20, 17, '[]'::jsonb, 0, now() + interval '30 days', 'active'),
  (8042000000000000006, 8020000000000000006, 'publish_quota', 'demo_seed', 20, 11, '[]'::jsonb, 0, now() + interval '30 days', 'active'),
  (8043000000000000001, 8020000000000000002, 'top_voucher', 'demo_seed', 1, 1, '["stock_clearance","factory_direct"]'::jsonb, 24, now() + interval '20 days', 'active')
ON CONFLICT (id) DO UPDATE SET
  entitlement_type = EXCLUDED.entitlement_type,
  source_type = EXCLUDED.source_type,
  total_amount = EXCLUDED.total_amount,
  remaining_amount = EXCLUDED.remaining_amount,
  allowed_type_codes = EXCLUDED.allowed_type_codes,
  top_duration_hours = EXCLUDED.top_duration_hours,
  expires_at = EXCLUDED.expires_at,
  status = EXCLUDED.status,
  updated_at = now();

INSERT INTO resource_review_records (id, resource_id, reviewer_id, action, reason, snapshot)
SELECT
  rr.id,
  r.id,
  8009000000000000001,
  rr.action,
  rr.reason,
  jsonb_build_object('title', r.title, 'status', r.status)
FROM resources r
JOIN (
  VALUES
    (8030000000000000001, 8044000000000000001, 'approve', '演示审核通过'),
    (8030000000000000002, 8044000000000000002, 'approve', '演示审核通过'),
    (8030000000000000003, 8044000000000000003, 'approve', '演示审核通过'),
    (8030000000000000004, 8044000000000000004, 'approve', '演示审核通过'),
    (8030000000000000005, 8044000000000000005, 'approve', '演示审核通过'),
    (8030000000000000006, 8044000000000000006, 'approve', '演示审核通过'),
    (8030000000000000007, 8044000000000000007, 'approve', '演示审核通过'),
    (8030000000000000008, 8044000000000000008, 'approve', '演示审核通过'),
    (8030000000000000009, 8044000000000000009, 'approve', '演示审核通过'),
    (8030000000000000010, 8044000000000000010, 'take_down', '冷启动阶段暂停招聘入口'),
    (8030000000000000011, 8044000000000000011, 'approve', '历史资源审核通过'),
    (8030000000000000012, 8044000000000000012, 'approve', '演示审核通过'),
    (8030000000000000013, 8044000000000000013, 'approve', '演示审核通过'),
    (8030000000000000014, 8044000000000000014, 'approve', '演示审核通过'),
    (8030000000000000015, 8044000000000000015, 'approve', '演示审核通过'),
    (8030000000000000016, 8044000000000000016, 'approve', '演示审核通过'),
    (8030000000000000017, 8044000000000000017, 'approve', '演示审核通过'),
    (8030000000000000018, 8044000000000000018, 'approve', '演示审核通过'),
    (8030000000000000019, 8044000000000000019, 'approve', '演示审核通过'),
    (8030000000000000020, 8044000000000000020, 'approve', '演示审核通过'),
    (8030000000000000021, 8044000000000000021, 'approve', '演示审核通过'),
    (8030000000000000022, 8044000000000000022, 'approve', '演示审核通过'),
    (8030000000000000023, 8044000000000000023, 'approve', '演示审核通过'),
    (8030000000000000024, 8044000000000000024, 'approve', '演示审核通过'),
    (8030000000000000025, 8044000000000000025, 'approve', '演示审核通过'),
    (8030000000000000026, 8044000000000000026, 'approve', '演示审核通过'),
    (8030000000000000027, 8044000000000000027, 'approve', '演示审核通过'),
    (8030000000000000028, 8044000000000000028, 'approve', '演示审核通过'),
    (8030000000000000033, 8044000000000000033, 'reject', '缺少清晰价格和联系方式确认材料'),
    (8030000000000000034, 8044000000000000034, 'approve', '历史成交资源审核通过')
) AS rr(resource_id, id, action, reason) ON rr.resource_id = r.id
ON CONFLICT (id) DO UPDATE SET
  resource_id = EXCLUDED.resource_id,
  reviewer_id = EXCLUDED.reviewer_id,
  action = EXCLUDED.action,
  reason = EXCLUDED.reason,
  snapshot = EXCLUDED.snapshot;

INSERT INTO resource_metrics_daily (
  resource_id,
  merchant_id,
  stat_date,
  exposure_count,
  search_exposure_count,
  list_exposure_count,
  detail_view_count,
  contact_click_count,
  phone_click_count,
  wechat_copy_count,
  share_count,
  deal_feedback_count
)
SELECT
  r.id,
  r.merchant_id,
  current_date,
  120 + n.idx * 8,
  70 + n.idx * 5,
  50 + n.idx * 3,
  24 + n.idx,
  8 + n.idx,
  3 + n.idx,
  5 + n.idx,
  2,
  CASE WHEN n.idx = 1 THEN 1 ELSE 0 END
FROM resources r
JOIN (
  VALUES
    (8030000000000000001, 1),
    (8030000000000000003, 2),
    (8030000000000000007, 3),
    (8030000000000000004, 4),
    (8030000000000000008, 5),
    (8030000000000000015, 6)
) AS n(resource_id, idx) ON n.resource_id = r.id
ON CONFLICT (resource_id, stat_date) DO UPDATE SET
  exposure_count = EXCLUDED.exposure_count,
  search_exposure_count = EXCLUDED.search_exposure_count,
  list_exposure_count = EXCLUDED.list_exposure_count,
  detail_view_count = EXCLUDED.detail_view_count,
  contact_click_count = EXCLUDED.contact_click_count,
  phone_click_count = EXCLUDED.phone_click_count,
  wechat_copy_count = EXCLUDED.wechat_copy_count,
  share_count = EXCLUDED.share_count,
  deal_feedback_count = EXCLUDED.deal_feedback_count,
  updated_at = now();

INSERT INTO resource_contact_events (id, resource_id, user_id, merchant_id, action, created_at)
VALUES
  (8045000000000000001, 8030000000000000001, 8010000000000000005, 8020000000000000001, 'phone', now() - interval '2 hours'),
  (8045000000000000002, 8030000000000000003, 8010000000000000005, 8020000000000000002, 'wechat', now() - interval '1 hours'),
  (8045000000000000003, 8030000000000000008, 8010000000000000002, 8020000000000000004, 'phone', now() - interval '30 minutes')
ON CONFLICT (id) DO UPDATE SET
  resource_id = EXCLUDED.resource_id,
  user_id = EXCLUDED.user_id,
  merchant_id = EXCLUDED.merchant_id,
  action = EXCLUDED.action,
  created_at = EXCLUDED.created_at;

INSERT INTO messages (id, recipient_user_id, recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status, sent_at)
VALUES
  (8052000000000000001, NULL, 'merchant:8020000000000000002', 'resource_review', 'resource_approve', 8030000000000000003, '供需信息审核通过', '女童春款卫衣库存整包清 已公开展示', '/pages/my-resources/index', 'unread', now()),
  (8052000000000000002, NULL, 'merchant:8020000000000000001', 'resource_review', 'resource_reject', 8030000000000000033, '供需信息审核未通过', '资料不完整的货源演示 未通过审核，请补充清晰价格和联系方式确认材料', '/pages/my-resources/index', 'unread', now()),
  (8052000000000000003, NULL, 'merchant:8020000000000000002', 'resource_expiring', 'resource_expiring', 8030000000000000003, '供需信息即将过期', '女童春款卫衣库存整包清 将在 1 天后过期', '/pages/my-resources/index', 'unread', now())
ON CONFLICT (id) DO UPDATE SET
  recipient_user_id = EXCLUDED.recipient_user_id,
  recipient_role_code = EXCLUDED.recipient_role_code,
  message_type = EXCLUDED.message_type,
  trigger_type = EXCLUDED.trigger_type,
  trigger_id = EXCLUDED.trigger_id,
  title = EXCLUDED.title,
  content = EXCLUDED.content,
  target_url = EXCLUDED.target_url,
  status = EXCLUDED.status,
  sent_at = EXCLUDED.sent_at;

INSERT INTO banner_topics (id, city_station_id, kind, title, subtitle, cover_url, type_scope, jump_type, jump_target, tags, sort_order, status, start_at, end_at)
SELECT
  8053000000000000001,
  cs.id,
  'banner',
  '演示活动：童装现货上新周',
  '点击进入活动 web-view 验证页',
  '',
  '["factory_direct","stock_clearance","processing_accept"]'::jsonb,
  'webview',
  'https://example.com/wplink-demo',
  '["演示","活动"]'::jsonb,
  110,
  'active',
  now() - interval '1 day',
  now() + interval '30 days'
FROM city_stations cs
WHERE cs.code = 'zhili'
ON CONFLICT (id) DO UPDATE SET subtitle = EXCLUDED.subtitle, jump_target = EXCLUDED.jump_target, status = EXCLUDED.status, updated_at = now();

INSERT INTO operation_logs (id, operator_id, operator_role, action, object_type, object_id, after_snapshot)
VALUES
  (8054000000000000001, 8009000000000000001, 'super_admin', 'merchant_verify', 'merchant', 8020000000000000001, '{"demo":true}'::jsonb),
  (8054000000000000002, 8009000000000000001, 'super_admin', 'resource_approve', 'resource', 8030000000000000003, '{"demo":true}'::jsonb),
  (8054000000000000003, 8009000000000000001, 'super_admin', 'resource_reject', 'resource', 8030000000000000033, '{"demo":true,"reason":"缺少清晰价格和联系方式确认材料"}'::jsonb)
ON CONFLICT (id) DO UPDATE SET
  operator_id = EXCLUDED.operator_id,
  operator_role = EXCLUDED.operator_role,
  action = EXCLUDED.action,
  object_type = EXCLUDED.object_type,
  object_id = EXCLUDED.object_id,
  after_snapshot = EXCLUDED.after_snapshot,
  created_at = now();

COMMIT;
