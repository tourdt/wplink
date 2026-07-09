-- 兼容早期已建表的环境：当 000002 的建表语句演进后，
-- 已存在的旧表不会自动补 direction 列；本迁移在新增需求类型前先补齐统一供需方向 schema。
ALTER TABLE IF EXISTS resource_type_configs
  ADD COLUMN IF NOT EXISTS direction varchar(32) NOT NULL DEFAULT 'supply';

ALTER TABLE IF EXISTS resources
  ADD COLUMN IF NOT EXISTS direction varchar(32) NOT NULL DEFAULT 'supply';

UPDATE resource_type_configs
SET direction = 'demand'
WHERE type_code IN ('buy_goods', 'find_inventory', 'find_factory', 'find_service', 'find_rental');

UPDATE resources r
SET direction = rtc.direction
FROM resource_type_configs rtc
WHERE r.resource_type_config_id = rtc.id
  AND r.direction IS DISTINCT FROM rtc.direction;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'chk_resource_type_configs_direction'
  ) THEN
    ALTER TABLE resource_type_configs
      ADD CONSTRAINT chk_resource_type_configs_direction CHECK (direction IN ('supply', 'demand'));
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'chk_resources_direction'
  ) THEN
    ALTER TABLE resources
      ADD CONSTRAINT chk_resources_direction CHECK (direction IN ('supply', 'demand'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_resource_type_configs_direction ON resource_type_configs(direction, status);
CREATE INDEX IF NOT EXISTS idx_resources_city_direction_type_status ON resources(city_station_id, direction, type_code, status);

WITH zhili AS (
  SELECT id FROM city_stations WHERE code = 'zhili'
)
UPDATE city_stations cs
SET
  config = jsonb_set(
    cs.config,
    '{enabledTypeCodes}',
    COALESCE(cs.config -> 'enabledTypeCodes', '[]'::jsonb) || '["find_rental"]'::jsonb,
    true
  ),
  updated_at = now()
FROM zhili
WHERE cs.id = zhili.id
  AND NOT (COALESCE(cs.config -> 'enabledTypeCodes', '[]'::jsonb) ? 'find_rental');

WITH zhili AS (
  SELECT id FROM city_stations WHERE code = 'zhili'
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
  'find_rental',
  '找场地',
  'demand',
  '{"fields":[{"key":"rentalNeedType","label":"标的类型","type":"select","required":true,"filterable":true,"displayIn":["detail"],"options":["厂房","档口","仓库","商铺","商品房","宿舍","设备","其他"],"allowCustom":true},{"key":"expectedAreaText","label":"期望面积","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：100-200 平"},{"key":"budgetRentText","label":"预算租金","type":"text","required":false,"filterable":true,"displayIn":["detail"],"placeholder":"例如：5000-8000 元/月"},{"key":"preferredDistrict","label":"期望区域","type":"text","required":true,"filterable":true,"displayIn":["detail"],"placeholder":"例如：织里童装城附近"},{"key":"acceptTransferFee","label":"接受转让费","type":"boolean","required":false,"filterable":true,"displayIn":["detail"]},{"key":"moveInTime","label":"入驻时间","type":"text","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"例如：两周内"},{"key":"supportRequirement","label":"配套要求","type":"textarea","required":false,"filterable":false,"displayIn":["detail"],"placeholder":"说明水电、停车、楼层、货梯、住宿等要求"}]}'::jsonb,
  '["title","rentalNeedType","preferredDistrict","contactPhone"]'::jsonb,
  '["rentalNeedType","expectedAreaText","budgetRentText","preferredDistrict","acceptTransferFee"]'::jsonb,
  '{"summary":{"category":"rentalNeedType","quantityText":"expectedAreaText","priceText":"budgetRentText"},"list":["priceText","quantityText","preferredDistrict","district"],"detail":["rentalNeedType","expectedAreaText","budgetRentText","preferredDistrict","acceptTransferFee","moveInTime","supportRequirement"]}'::jsonb,
  '{"resubmitOnChange":["title","rentalNeedType","preferredDistrict","contactPhone"]}'::jsonb,
  '{"freshDemand":20,"refreshedAt":10}'::jsonb,
  '{"expiringSoonDays":3}'::jsonb,
  15,
  'active'
FROM zhili
ON CONFLICT (city_station_id, type_code) WHERE city_station_id IS NOT NULL
DO UPDATE SET
  type_name = EXCLUDED.type_name,
  direction = EXCLUDED.direction,
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
