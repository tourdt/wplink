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
