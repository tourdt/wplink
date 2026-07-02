-- MVP 端到端演示数据。
-- 前置条件：已按顺序执行 migrations 目录下的所有 up migration。

BEGIN;

INSERT INTO users (id, phone, wechat_openid, nickname, default_city_station_id, status, last_login_at)
SELECT u.id, u.phone, u.openid, u.nickname, cs.id, 'active', now()
FROM city_stations cs
CROSS JOIN (
  VALUES
    (8010000000000000001, '19900000001', 'demo_operator_openid', '演示运营'),
    (8010000000000000002, '19900000002', 'demo_factory_admin_openid', '认证工厂管理员'),
    (8010000000000000003, '19900000003', 'demo_stockist_admin_openid', '认证库存商管理员'),
    (8010000000000000004, '19900000004', 'demo_service_admin_openid', '服务商管理员'),
    (8010000000000000005, '19900000005', 'demo_buyer_openid', '采购商买家')
) AS u(id, phone, openid, nickname)
WHERE cs.code = 'zhili'
ON CONFLICT (id) DO UPDATE SET
  phone = EXCLUDED.phone,
  wechat_openid = EXCLUDED.wechat_openid,
  nickname = EXCLUDED.nickname,
  default_city_station_id = EXCLUDED.default_city_station_id,
  status = 'active',
  updated_at = now();

INSERT INTO user_role_assignments (user_id, role_id, city_station_id, merchant_id)
SELECT 8010000000000000001, r.id, cs.id, NULL::bigint
FROM roles r
JOIN city_stations cs ON cs.code = 'zhili'
WHERE r.code = 'platform_operator'
  AND NOT EXISTS (
    SELECT 1
    FROM user_role_assignments ura
    WHERE ura.user_id = 8010000000000000001
      AND ura.role_id = r.id
      AND ura.city_station_id = cs.id
      AND ura.merchant_id IS NULL
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
  images,
  verification_status,
  status,
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
  m.images::jsonb,
  m.verification_status,
  'active',
  now()
FROM city_stations cs
CROSS JOIN (
  VALUES
    (
      8020000000000000001,
      '湖州织里晨星童装厂',
      'factory',
      '["童装","卫衣","套装"]',
      '认证工厂，主做童装卫衣和套装，可承接小单快反。',
      '陈厂长',
      '18800000001',
      'factory-demo',
      '织里镇利济路88号',
      '[]',
      'verified'
    ),
    (
      8020000000000000002,
      '织里云仓尾货',
      'stockist',
      '["童装","库存尾货"]',
      '认证库存商，长期处理整包库存和直播货盘。',
      '周经理',
      '18800000002',
      'stock-demo',
      '织里童装城3区',
      '[]',
      'verified'
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
      '[]',
      'verified'
    ),
    (
      8020000000000000004,
      '杭州童装采购商',
      'buyer',
      '["童装","电商供货"]',
      '采购商，长期寻找织里现货、工厂和快反资源。',
      '王采购',
      '18800000004',
      'buyer-demo',
      '杭州市滨江区',
      '[]',
      'unverified'
    )
) AS m(id, name, merchant_type, main_categories, description, contact_name, contact_phone, contact_wechat, address_text, images, verification_status)
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
  images = EXCLUDED.images,
  verification_status = EXCLUDED.verification_status,
  status = EXCLUDED.status,
  last_active_at = EXCLUDED.last_active_at,
  updated_at = now();

INSERT INTO merchant_admin_bindings (id, merchant_id, user_id, role, status, created_by)
VALUES
  (8021000000000000001, 8020000000000000001, 8010000000000000002, 'owner', 'active', 8010000000000000001),
  (8021000000000000002, 8020000000000000002, 8010000000000000003, 'owner', 'active', 8010000000000000001),
  (8021000000000000003, 8020000000000000003, 8010000000000000004, 'owner', 'active', 8010000000000000001)
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, revoked_at = NULL;

INSERT INTO resources (
  id,
  merchant_id,
  city_station_id,
  resource_type_config_id,
  type_code,
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
  is_verified,
  published_at,
  refreshed_at,
  expires_at,
  reject_reason,
  created_by
)
SELECT
  r.id,
  r.merchant_id,
  cs.id,
  rtc.id,
  r.type_code,
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
  r.is_verified,
  r.published_at,
  r.refreshed_at,
  r.expires_at,
  r.reject_reason,
  r.created_by
FROM city_stations cs
JOIN resource_type_configs rtc ON rtc.city_station_id = cs.id
CROSS JOIN (
  VALUES
    (8030000000000000001, 8020000000000000002, 'inventory', 'published', '女童春款卫衣库存整包清', '童装卫衣', '织里', '18-26 元/件', '现货 3800 件', '', '春款卫衣库存，支持整包和直播拿样。', '{"season":"春季","sizeRange":"90-140","allowSample":true,"allowLiveSale":true}', '["库存","可拿样","直播货盘"]', '周经理', '18800000002', 'stock-demo', true, now() - interval '2 days', now() - interval '1 hours', now() + interval '5 days', NULL, 8010000000000000003),
    (8030000000000000002, 8020000000000000001, 'goods', 'published', '童装套装一件代发货源', '童装套装', '织里', '32-45 元/套', '起批 20 套', '', '工厂直供套装货源，可一件代发。', '{"style":"韩版休闲","minOrderQuantity":"20套","spotAvailable":true,"dropshipping":true}', '["货源","一件代发"]', '陈厂长', '18800000001', 'factory-demo', true, now() - interval '3 days', now() - interval '2 hours', now() + interval '12 days', NULL, 8010000000000000002),
    (8030000000000000003, 8020000000000000001, 'factory', 'published', '童装卫衣工厂空档期接单', '加工厂', '织里', '价格按工艺核算', '日产 1200 件', '', '认证工厂有空档期，可承接童装卫衣和套装快反。', '{"dailyCapacity":"1200件","minOrderQuantity":"300件","acceptSmallOrders":true,"availableSchedule":"本周可排单"}', '["认证工厂","快反"]', '陈厂长', '18800000001', 'factory-demo', true, now() - interval '1 days', now() - interval '30 minutes', now() + interval '14 days', NULL, 8010000000000000002),
    (8030000000000000004, 8020000000000000004, 'order', 'published', '采购 5000 件女童防晒衣订单', '订单需求', '杭州', '面议', '5000 件', '', '采购商寻找织里工厂承接防晒衣订单，交期 20 天。', '{"orderQuantity":"5000件","deliveryDeadline":"20天","sampleRequired":true,"longTermCooperation":true}', '["订单","采购商"]', '王采购', '18800000004', 'buyer-demo', false, now() - interval '1 days', now() - interval '1 days', now() + interval '8 days', NULL, 8010000000000000005),
    (8030000000000000005, 8020000000000000001, 'job', 'published', '童装平车熟练工招聘', '招聘', '织里', '计件 0.8-1.2 元', '招聘 8 人', '', '工厂招聘熟练平车工，订单稳定。', '{"position":"平车工","payText":"计件0.8-1.2元","headcount":8,"includeMealsHousing":true}', '["招聘","平车工"]', '陈厂长', '18800000001', 'factory-demo', true, now() - interval '4 days', now() - interval '2 days', now() + interval '10 days', NULL, 8010000000000000002),
    (8030000000000000006, 8020000000000000003, 'rental', 'published', '织里童装城旁 120 平仓库出租', '厂房仓库', '织里', '6800 元/月', '120 平', '', '童装城附近仓库出租，可短租。', '{"areaText":"120平","rentText":"6800元/月","floor":"1楼","transferFee":"无"}', '["出租","仓库"]', '李经理', '18800000003', 'service-demo', true, now() - interval '5 days', now() - interval '3 days', now() + interval '25 days', NULL, 8010000000000000004),
    (8030000000000000007, 8020000000000000003, 'service', 'published', '童装吊牌包装快印服务', '配套服务', '织里', '按量报价', '当天出样', '', '提供吊牌、洗标、包装袋和电商拍摄服务。', '{"serviceType":"包装快印","serviceArea":"织里及周边","leadTime":"当天出样","caseAvailable":true}', '["服务","包装"]', '李经理', '18800000003', 'service-demo', true, now() - interval '6 days', now() - interval '6 hours', now() + interval '28 days', NULL, 8010000000000000004),
    (8030000000000000008, 8020000000000000002, 'inventory', 'pending', '待审核夏款短袖库存', '童装短袖', '织里', '12-18 元/件', '1800 件', '', '演示待审核资源。', '{"season":"夏季","sizeRange":"90-130","allowSample":true,"allowLiveSale":false}', '["待审核"]', '周经理', '18800000002', 'stock-demo', true, NULL, NULL, now() + interval '7 days', NULL, 8010000000000000003),
    (8030000000000000009, 8020000000000000001, 'goods', 'rejected', '资料不完整的货源演示', '童装', '织里', '面议', '起批待确认', '', '演示已驳回资源。', '{"style":"基础款","spotAvailable":false}', '["已驳回"]', '陈厂长', '18800000001', 'factory-demo', true, NULL, NULL, now() + interval '7 days', '缺少清晰价格和联系方式确认材料', 8010000000000000002),
    (8030000000000000010, 8020000000000000002, 'inventory', 'published', '即将过期的直播童裙库存', '童裙', '织里', '22 元/件', '900 件', '', '演示即将过期资源。', '{"season":"夏季","sizeRange":"100-140","allowSample":true,"allowLiveSale":true}', '["即将过期"]', '周经理', '18800000002', 'stock-demo', true, now() - interval '6 days', now() - interval '5 days', now() + interval '1 day', NULL, 8010000000000000003),
    (8030000000000000011, 8020000000000000003, 'service', 'expired', '已过期的旧拍摄服务套餐', '电商拍摄', '织里', '套餐价 999 元', '限 10 套', '', '演示已过期资源。', '{"serviceType":"拍摄","serviceArea":"织里","leadTime":"3天","caseAvailable":true}', '["已过期"]', '李经理', '18800000003', 'service-demo', true, now() - interval '40 days', now() - interval '35 days', now() - interval '1 day', NULL, 8010000000000000004)
) AS r(id, merchant_id, type_code, status, title, category, district, price_text, quantity_text, cover_url, description, attributes, tags, contact_name, contact_phone, contact_wechat, is_verified, published_at, refreshed_at, expires_at, reject_reason, created_by)
WHERE cs.code = 'zhili'
  AND rtc.type_code = r.type_code
ON CONFLICT (id) DO UPDATE SET
  merchant_id = EXCLUDED.merchant_id,
  city_station_id = EXCLUDED.city_station_id,
  resource_type_config_id = EXCLUDED.resource_type_config_id,
  type_code = EXCLUDED.type_code,
  status = EXCLUDED.status,
  title = EXCLUDED.title,
  category = EXCLUDED.category,
  district = EXCLUDED.district,
  price_text = EXCLUDED.price_text,
  quantity_text = EXCLUDED.quantity_text,
  description = EXCLUDED.description,
  attributes = EXCLUDED.attributes,
  tags = EXCLUDED.tags,
  contact_name = EXCLUDED.contact_name,
  contact_phone = EXCLUDED.contact_phone,
  contact_wechat = EXCLUDED.contact_wechat,
  is_verified = EXCLUDED.is_verified,
  published_at = EXCLUDED.published_at,
  refreshed_at = EXCLUDED.refreshed_at,
  expires_at = EXCLUDED.expires_at,
  reject_reason = EXCLUDED.reject_reason,
  updated_at = now();

INSERT INTO verifications (id, merchant_id, verification_type, status, applicant_user_id, business_name, license_url, storefront_url, materials, review_note, reviewed_by, reviewed_at)
VALUES
  (8040000000000000001, 8020000000000000001, 'factory', 'verified', 8010000000000000002, '湖州织里晨星童装厂', 'https://example.com/demo/factory-license.jpg', 'https://example.com/demo/factory-store.jpg', '{"demo":true}'::jsonb, '演示认证通过', 8010000000000000001, now() - interval '5 days'),
  (8040000000000000002, 8020000000000000002, 'stockist', 'verified', 8010000000000000003, '织里云仓尾货', 'https://example.com/demo/stock-license.jpg', 'https://example.com/demo/stock-store.jpg', '{"demo":true}'::jsonb, '演示认证通过', 8010000000000000001, now() - interval '5 days'),
  (8040000000000000003, 8020000000000000003, 'service_provider', 'verified', 8010000000000000004, '织里快印包装服务商', 'https://example.com/demo/service-license.jpg', 'https://example.com/demo/service-store.jpg', '{"demo":true}'::jsonb, '演示认证通过', 8010000000000000001, now() - interval '5 days')
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, review_note = EXCLUDED.review_note, reviewed_at = EXCLUDED.reviewed_at;

INSERT INTO credit_records (id, merchant_id, source_type, tag_code, tag_label, description, visibility, created_by)
VALUES
  (8041000000000000001, 8020000000000000001, 'verification', 'factory_verified', '认证工厂', '演示认证信用标签', 'public', 8010000000000000001),
  (8041000000000000002, 8020000000000000002, 'verification', 'stockist_verified', '认证库存商', '演示认证信用标签', 'public', 8010000000000000001),
  (8041000000000000003, 8020000000000000003, 'verification', 'service_provider_verified', '认证服务商', '演示认证信用标签', 'public', 8010000000000000001)
ON CONFLICT (id) DO UPDATE SET tag_label = EXCLUDED.tag_label, description = EXCLUDED.description, revoked_at = NULL;

INSERT INTO merchant_entitlements (id, merchant_id, entitlement_type, source_type, total_amount, remaining_amount, expires_at, status)
VALUES
  (8042000000000000001, 8020000000000000001, 'publish_quota', 'verification_bonus', 20, 18, now() + interval '30 days', 'active'),
  (8042000000000000002, 8020000000000000002, 'refresh_quota', 'verification_bonus', 30, 27, now() + interval '30 days', 'active')
ON CONFLICT (id) DO UPDATE SET remaining_amount = EXCLUDED.remaining_amount, expires_at = EXCLUDED.expires_at, status = EXCLUDED.status;

INSERT INTO top_vouchers (id, merchant_id, entitlement_id, source_type, allowed_type_codes, top_duration_hours, expires_at, status)
VALUES
  (8043000000000000001, 8020000000000000002, 8042000000000000002, 'verification_bonus', '["inventory","goods"]'::jsonb, 24, now() + interval '20 days', 'unused')
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, expires_at = EXCLUDED.expires_at;

INSERT INTO resource_review_records (id, resource_id, reviewer_id, action, reason, snapshot)
SELECT
  rr.id,
  r.id,
  8010000000000000001,
  CASE WHEN r.status = 'rejected' THEN 'reject' ELSE 'approve' END,
  CASE WHEN r.status = 'rejected' THEN r.reject_reason ELSE '演示审核通过' END,
  jsonb_build_object('title', r.title, 'status', r.status)
FROM resources r
JOIN (
  VALUES
    (8030000000000000001, 8044000000000000001),
    (8030000000000000002, 8044000000000000002),
    (8030000000000000003, 8044000000000000003),
    (8030000000000000004, 8044000000000000004),
    (8030000000000000005, 8044000000000000005),
    (8030000000000000006, 8044000000000000006),
    (8030000000000000007, 8044000000000000007),
    (8030000000000000009, 8044000000000000009),
    (8030000000000000010, 8044000000000000010),
    (8030000000000000011, 8044000000000000011)
) AS rr(resource_id, id) ON rr.resource_id = r.id
WHERE r.id BETWEEN 8030000000000000001 AND 8030000000000000011
  AND r.status <> 'pending'
ON CONFLICT (id) DO NOTHING;

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
    (8030000000000000002, 2),
    (8030000000000000003, 3),
    (8030000000000000007, 4)
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
  (8045000000000000001, 8030000000000000001, 8010000000000000005, 8020000000000000002, 'phone', now() - interval '2 hours'),
  (8045000000000000002, 8030000000000000001, 8010000000000000005, 8020000000000000002, 'wechat', now() - interval '1 hours')
ON CONFLICT (id) DO UPDATE SET created_at = EXCLUDED.created_at;

INSERT INTO map_scene (
  city_station_id,
  code,
  name,
  type,
  background_url,
  width,
  height,
  min_scale,
  max_scale,
  default_scale,
  default_center_x,
  default_center_y,
  sort,
  status
)
SELECT
  cs.id,
  'zhili_lijilu_demo',
  '利济路童装拿货示范图',
  'street_segment',
  'https://dummyimage.com/1200x720/f4f7fb/334155.png&text=Zhili+Liji+Road+Map',
  1200,
  720,
  0.5,
  4,
  1,
  600,
  360,
  10,
  'published'
FROM city_stations cs
WHERE cs.code = 'zhili'
ON CONFLICT (code) DO UPDATE SET
  city_station_id = EXCLUDED.city_station_id,
  name = EXCLUDED.name,
  type = EXCLUDED.type,
  background_url = EXCLUDED.background_url,
  width = EXCLUDED.width,
  height = EXCLUDED.height,
  min_scale = EXCLUDED.min_scale,
  max_scale = EXCLUDED.max_scale,
  default_scale = EXCLUDED.default_scale,
  default_center_x = EXCLUDED.default_center_x,
  default_center_y = EXCLUDED.default_center_y,
  sort = EXCLUDED.sort,
  status = EXCLUDED.status,
  updated_at = now();

INSERT INTO map_object (
  scene_code,
  merchant_id,
  code,
  name,
  type,
  layer,
  geometry_type,
  geometry,
  center_x,
  center_y,
  min_x,
  min_y,
  max_x,
  max_y,
  min_zoom,
  max_zoom,
  category_codes,
  service_tags,
  platform_tags,
  poi_service_tags,
  address,
  phone,
  wechat,
  search_text,
  extra,
  sort,
  status
)
VALUES
  (
    'zhili_lijilu_demo',
    8020000000000000001,
    'A001',
    '晨星童装 A 区',
    'booth',
    'booth',
    'rect',
    '{"x":120,"y":120,"width":210,"height":110}'::jsonb,
    225,
    175,
    120,
    120,
    330,
    230,
    1,
    5,
    '["girl","baby"]'::jsonb,
    '["spot","factory","sample"]'::jsonb,
    '["verified","hot"]'::jsonb,
    '[]'::jsonb,
    '织里镇利济路88号',
    '18800000001',
    'factory-demo',
    'A001 晨星童装 A 区 女童 婴童 现货 源头工厂 支持打样 织里镇利济路88号 factory-demo',
    '{"demo":true,"floor":"1F"}'::jsonb,
    10,
    'normal'
  ),
  (
    'zhili_lijilu_demo',
    8020000000000000002,
    'B108',
    '云仓尾货 B 区',
    'booth',
    'booth',
    'rect',
    '{"x":420,"y":135,"width":220,"height":110}'::jsonb,
    530,
    190,
    420,
    135,
    640,
    245,
    1,
    5,
    '["girl","middle_child"]'::jsonb,
    '["spot","dropship"]'::jsonb,
    '["verified","hot"]'::jsonb,
    '[]'::jsonb,
    '织里童装城3区',
    '18800000002',
    'stock-demo',
    'B108 云仓尾货 B 区 女童 中大童 现货 一件代发 库存尾货 织里童装城3区 stock-demo',
    '{"demo":true,"floor":"1F"}'::jsonb,
    20,
    'normal'
  ),
  (
    'zhili_lijilu_demo',
    8020000000000000003,
    'P001',
    '利济路打包点',
    'packing_station',
    'poi',
    'point',
    '{"x":770,"y":220}'::jsonb,
    770,
    220,
    770,
    220,
    770,
    220,
    1,
    5,
    '[]'::jsonb,
    '[]'::jsonb,
    '[]'::jsonb,
    '["packing","labeling","carton"]'::jsonb,
    '利济路与阿祥路交叉口',
    '18800000003',
    'service-demo',
    'P001 利济路打包点 打包 贴单 纸箱 配套服务 利济路与阿祥路交叉口 service-demo',
    '{"demo":true}'::jsonb,
    30,
    'normal'
  ),
  (
    'zhili_lijilu_demo',
    NULL,
    'P002',
    '童装城停车场',
    'parking',
    'poi',
    'point',
    '{"x":955,"y":420}'::jsonb,
    955,
    420,
    955,
    420,
    955,
    420,
    1,
    5,
    '[]'::jsonb,
    '[]'::jsonb,
    '[]'::jsonb,
    '[]'::jsonb,
    '织里童装城东门',
    '',
    '',
    'P002 童装城停车场 停车 配套 织里童装城东门',
    '{"demo":true}'::jsonb,
    40,
    'normal'
  ),
  (
    'zhili_lijilu_demo',
    8020000000000000003,
    'P003',
    '全国物流发货点',
    'logistics_point',
    'poi',
    'point',
    '{"x":970,"y":185}'::jsonb,
    970,
    185,
    970,
    185,
    970,
    185,
    1,
    5,
    '[]'::jsonb,
    '[]'::jsonb,
    '[]'::jsonb,
    '["national","bulk_shipping"]'::jsonb,
    '利济路物流通道',
    '18800000003',
    'service-demo',
    'P003 全国物流发货点 物流 全国物流 批量发货 利济路物流通道 service-demo',
    '{"demo":true}'::jsonb,
    50,
    'normal'
  )
ON CONFLICT (scene_code, code) DO UPDATE SET
  merchant_id = EXCLUDED.merchant_id,
  name = EXCLUDED.name,
  type = EXCLUDED.type,
  layer = EXCLUDED.layer,
  geometry_type = EXCLUDED.geometry_type,
  geometry = EXCLUDED.geometry,
  center_x = EXCLUDED.center_x,
  center_y = EXCLUDED.center_y,
  min_x = EXCLUDED.min_x,
  min_y = EXCLUDED.min_y,
  max_x = EXCLUDED.max_x,
  max_y = EXCLUDED.max_y,
  min_zoom = EXCLUDED.min_zoom,
  max_zoom = EXCLUDED.max_zoom,
  category_codes = EXCLUDED.category_codes,
  service_tags = EXCLUDED.service_tags,
  platform_tags = EXCLUDED.platform_tags,
  poi_service_tags = EXCLUDED.poi_service_tags,
  address = EXCLUDED.address,
  phone = EXCLUDED.phone,
  wechat = EXCLUDED.wechat,
  search_text = EXCLUDED.search_text,
  extra = EXCLUDED.extra,
  sort = EXCLUDED.sort,
  status = EXCLUDED.status,
  updated_at = now();

INSERT INTO purchase_demands (id, user_id, city_station_id, demand_type, status, title, category, price_range, quantity_requirement, attributes, contact_name, contact_phone, contact_wechat, expires_at)
SELECT
  8050000000000000001,
  8010000000000000005,
  cs.id,
  'inventory',
  'matching',
  '找 3000 件女童春款卫衣现货',
  '童装卫衣',
  '{"min":15,"max":28,"unit":"元/件"}'::jsonb,
  '{"quantity":3000,"unit":"件"}'::jsonb,
  '{"sizeRange":"90-140","deadline":"3天内看样"}'::jsonb,
  '王采购',
  '18800000004',
  'buyer-demo',
  now() + interval '15 days'
FROM city_stations cs
WHERE cs.code = 'zhili'
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, updated_at = now();

INSERT INTO match_cases (id, purchase_demand_id, city_station_id, status, source, operator_id, result_note)
SELECT
  8051000000000000001,
  8050000000000000001,
  cs.id,
  'open',
  'manual',
  8010000000000000001,
  '演示撮合单：待联系库存商和工厂'
FROM city_stations cs
WHERE cs.code = 'zhili'
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, result_note = EXCLUDED.result_note, updated_at = now();

INSERT INTO match_case_resources (match_case_id, resource_id, role)
VALUES
  (8051000000000000001, 8030000000000000001, 'candidate'),
  (8051000000000000001, 8030000000000000003, 'candidate')
ON CONFLICT (match_case_id, resource_id) DO NOTHING;

INSERT INTO match_case_participants (match_case_id, merchant_id, participant_role, contact_status)
SELECT
  8051000000000000001,
  m.id,
  'merchant',
  'pending'
FROM merchants m
WHERE m.id IN (8020000000000000001, 8020000000000000002)
  AND NOT EXISTS (
    SELECT 1 FROM match_case_participants p
    WHERE p.match_case_id = 8051000000000000001
      AND p.merchant_id = m.id
  );

INSERT INTO messages (id, recipient_user_id, recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status, sent_at)
VALUES
  (8052000000000000001, NULL, 'merchant:8020000000000000002', 'resource_review', 'resource_approve', 8030000000000000001, '资源审核通过', '女童春款卫衣库存整包清 已公开展示', '/pages/my-resources/index', 'unread', now()),
  (8052000000000000002, NULL, 'merchant:8020000000000000002', 'resource_expiring', 'resource_expiring', 8030000000000000010, '资源即将过期', '即将过期的直播童裙库存 将在 1 天后过期', '/pages/my-resources/index', 'unread', now()),
  (8052000000000000003, NULL, 'merchant:8020000000000000001', 'match_progress', 'match_status_update', 8051000000000000001, '撮合进度更新', '运营已创建撮合单，等待联系确认。', '/pages/messages/index', 'unread', now()),
  (8052000000000000004, 8010000000000000005, NULL, 'match_progress', 'match_create', 8051000000000000001, '采购需求已进入撮合', '运营已受理您的采购需求。', '/pages/messages/index', 'unread', now())
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, content = EXCLUDED.content, sent_at = EXCLUDED.sent_at;

INSERT INTO banner_topics (id, city_station_id, kind, title, subtitle, cover_url, type_scope, jump_type, jump_target, tags, sort_order, status, start_at, end_at)
SELECT
  8053000000000000001,
  cs.id,
  'banner',
  '演示活动：童装现货撮合周',
  '点击进入活动 web-view 验证页',
  '',
  '["inventory","goods","factory"]'::jsonb,
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
  (8054000000000000001, 8010000000000000001, 'platform_operator', 'match_create', 'match_case', 8051000000000000001, '{"demo":true}'::jsonb),
  (8054000000000000002, 8010000000000000001, 'platform_operator', 'resource_approve', 'resource', 8030000000000000001, '{"demo":true}'::jsonb)
ON CONFLICT (id) DO UPDATE SET after_snapshot = EXCLUDED.after_snapshot, created_at = now();

COMMIT;
