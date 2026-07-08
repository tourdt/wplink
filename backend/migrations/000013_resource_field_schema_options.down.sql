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
      '{"fields":[{"key":"season","label":"季节","type":"select"},{"key":"sizeRange","label":"尺码段","type":"text"},{"key":"allowSample","label":"支持拿样","type":"boolean"},{"key":"allowLiveSale","label":"支持直播","type":"boolean"}]}'
    ),
    (
      'goods',
      '{"fields":[{"key":"style","label":"风格","type":"text"},{"key":"minOrderQuantity","label":"起批量","type":"text"},{"key":"spotAvailable","label":"是否现货","type":"boolean"},{"key":"dropshipping","label":"一件代发","type":"boolean"}]}'
    ),
    (
      'factory',
      '{"fields":[{"key":"dailyCapacity","label":"日产能","type":"text"},{"key":"minOrderQuantity","label":"起订量","type":"text"},{"key":"acceptSmallOrders","label":"接小单","type":"boolean"},{"key":"availableSchedule","label":"空档期","type":"text"}]}'
    ),
    (
      'order',
      '{"fields":[{"key":"orderQuantity","label":"订单数量","type":"text"},{"key":"deliveryDeadline","label":"交期","type":"text"},{"key":"sampleRequired","label":"需要打样","type":"boolean"},{"key":"longTermCooperation","label":"长期合作","type":"boolean"}]}'
    ),
    (
      'job',
      '{"fields":[{"key":"position","label":"岗位","type":"text"},{"key":"payText","label":"工价","type":"text"},{"key":"headcount","label":"人数","type":"number"},{"key":"includeMealsHousing","label":"包吃住","type":"boolean"}]}'
    ),
    (
      'rental',
      '{"fields":[{"key":"areaText","label":"面积","type":"text"},{"key":"rentText","label":"租金","type":"text"},{"key":"floor","label":"楼层","type":"text"},{"key":"transferFee","label":"转让费","type":"text"}]}'
    ),
    (
      'service',
      '{"fields":[{"key":"serviceType","label":"服务类型","type":"text"},{"key":"serviceArea","label":"服务范围","type":"text"},{"key":"leadTime","label":"交付时效","type":"text"},{"key":"caseAvailable","label":"有案例","type":"boolean"}]}'
    )
) AS cfg(type_code, field_schema)
WHERE rtc.city_station_id = zhili.id
  AND rtc.type_code = cfg.type_code;
