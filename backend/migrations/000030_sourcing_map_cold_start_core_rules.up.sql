-- 冷启动阶段先聚焦拿货地图与供需发布，招聘类型保留历史数据但停止新发布。
UPDATE resource_type_configs
SET status = 'inactive',
    updated_at = now()
WHERE type_code IN ('job_hiring', 'job_seeking')
  AND status <> 'inactive';

UPDATE city_stations
SET config = jsonb_set(
      jsonb_set(
        config,
        '{enabledTypeCodes}',
        (
          SELECT COALESCE(jsonb_agg(item ORDER BY ord), '[]'::jsonb)
          FROM jsonb_array_elements_text(COALESCE(config->'enabledTypeCodes', '[]'::jsonb))
            WITH ORDINALITY AS enabled(item, ord)
          WHERE item NOT IN ('job_hiring', 'job_seeking')
        )
      ),
      '{primaryCategories}',
      (
        SELECT COALESCE(jsonb_agg(item ORDER BY ord), '[]'::jsonb)
        FROM jsonb_array_elements_text(COALESCE(config->'primaryCategories', '[]'::jsonb))
          WITH ORDINALITY AS category(item, ord)
        WHERE item <> '招聘求职'
      )
    ),
    updated_at = now()
WHERE code = 'zhili';

-- 唯一索引创建前显式拦截历史重复数据，避免迁移过程静默选择或解绑任一档口。
DO $$
DECLARE
  duplicate_map_merchant_binding bigint;
BEGIN
  SELECT merchant_id
  INTO duplicate_map_merchant_binding
  FROM map_object
  WHERE merchant_id IS NOT NULL
  GROUP BY merchant_id
  HAVING COUNT(*) > 1
  LIMIT 1;

  IF duplicate_map_merchant_binding IS NOT NULL THEN
    RAISE EXCEPTION 'duplicate_map_merchant_binding: merchant_id=%', duplicate_map_merchant_binding;
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_map_object_merchant
  ON map_object(merchant_id)
  WHERE merchant_id IS NOT NULL;
