UPDATE resource_type_configs
SET status = 'active',
    updated_at = now()
WHERE type_code IN ('job_hiring', 'job_seeking')
  AND status = 'inactive';

UPDATE city_stations
SET config = jsonb_set(
      jsonb_set(
        config,
        '{enabledTypeCodes}',
        COALESCE(config->'enabledTypeCodes', '[]'::jsonb)
          || CASE
               WHEN COALESCE(config->'enabledTypeCodes', '[]'::jsonb) @> '["job_hiring"]'::jsonb
                 THEN '[]'::jsonb
               ELSE '["job_hiring"]'::jsonb
             END
          || CASE
               WHEN COALESCE(config->'enabledTypeCodes', '[]'::jsonb) @> '["job_seeking"]'::jsonb
                 THEN '[]'::jsonb
               ELSE '["job_seeking"]'::jsonb
             END
      ),
      '{primaryCategories}',
      COALESCE(config->'primaryCategories', '[]'::jsonb)
        || CASE
             WHEN COALESCE(config->'primaryCategories', '[]'::jsonb) @> '["招聘求职"]'::jsonb
               THEN '[]'::jsonb
             ELSE '["招聘求职"]'::jsonb
           END
    ),
    updated_at = now()
WHERE code = 'zhili';
