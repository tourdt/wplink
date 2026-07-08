WITH zhili AS (
  SELECT id FROM city_stations WHERE code = 'zhili'
)
UPDATE resource_type_configs rtc
SET
  field_schema = cfg.field_schema::jsonb,
  updated_at = now()
FROM zhili
CROSS JOIN (
  VALUES
    (
      'inventory',
      '{"fields":[{"key":"season","label":"季节","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["春季","夏季","秋季","冬季"],"allowCustom":false},{"key":"sizeRange","label":"尺码段","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：90-140"},{"key":"allowSample","label":"支持拿样","type":"boolean","required":false,"filterable":false,"displayIn":["detail"]},{"key":"allowLiveSale","label":"支持直播","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]}]}'
    ),
    (
      'goods',
      '{"fields":[{"key":"style","label":"风格","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["韩版","学院风","运动风","国潮","基础款"],"allowCustom":true},{"key":"minOrderQuantity","label":"起批量","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：20 套"},{"key":"spotAvailable","label":"是否现货","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"dropshipping","label":"一件代发","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]}]}'
    ),
    (
      'factory',
      '{"fields":[{"key":"dailyCapacity","label":"日产能","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：1200 件/天"},{"key":"minOrderQuantity","label":"起订量","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：300 件"},{"key":"acceptSmallOrders","label":"接小单","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"availableSchedule","label":"空档期","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["本周可排单","下周可排单","本月可排单","需沟通排期"],"allowCustom":true}]}'
    ),
    (
      'order',
      '{"fields":[{"key":"orderQuantity","label":"订单数量","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：5000 件"},{"key":"deliveryDeadline","label":"交期","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：20 天"},{"key":"sampleRequired","label":"需要打样","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"longTermCooperation","label":"长期合作","type":"boolean","required":false,"filterable":false,"displayIn":["detail"]}]}'
    ),
    (
      'job',
      '{"fields":[{"key":"position","label":"岗位","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["平车工","拷边工","裁剪工","后道","包装工"],"allowCustom":true},{"key":"payText","label":"工价","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：计件 0.8-1.2 元"},{"key":"headcount","label":"人数","type":"number","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：8"},{"key":"includeMealsHousing","label":"包吃住","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]}]}'
    ),
    (
      'rental',
      '{"fields":[{"key":"areaText","label":"面积","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：120 平"},{"key":"rentText","label":"租金","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：6800 元/月"},{"key":"floor","label":"楼层","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：1 楼"},{"key":"transferFee","label":"转让费","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：无"}]}'
    ),
    (
      'service',
      '{"fields":[{"key":"serviceType","label":"服务类型","type":"select","required":false,"filterable":true,"displayIn":["detail"],"options":["物流","辅料","印花","绣花","摄影","直播","包装"],"allowCustom":true},{"key":"serviceArea","label":"服务范围","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：织里及周边"},{"key":"leadTime","label":"交付时效","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：当天出样"},{"key":"caseAvailable","label":"有案例","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]}]}'
    )
) AS cfg(type_code, field_schema)
WHERE rtc.city_station_id = zhili.id
  AND rtc.type_code = cfg.type_code;
