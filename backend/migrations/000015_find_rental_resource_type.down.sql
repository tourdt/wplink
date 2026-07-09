WITH zhili AS (
  SELECT id FROM city_stations WHERE code = 'zhili'
)
DELETE FROM resource_type_configs rtc
USING zhili
WHERE rtc.city_station_id = zhili.id
  AND rtc.type_code = 'find_rental';

WITH zhili AS (
  SELECT id FROM city_stations WHERE code = 'zhili'
),
filtered_codes AS (
  SELECT
    cs.id,
    COALESCE(jsonb_agg(code ORDER BY ord) FILTER (WHERE code IS NOT NULL), '[]'::jsonb) AS enabled_type_codes
  FROM city_stations cs
  JOIN zhili ON zhili.id = cs.id
  LEFT JOIN LATERAL jsonb_array_elements_text(COALESCE(cs.config -> 'enabledTypeCodes', '[]'::jsonb)) WITH ORDINALITY AS entry(code, ord)
    ON entry.code <> 'find_rental'
  GROUP BY cs.id
)
UPDATE city_stations cs
SET
  config = jsonb_set(cs.config, '{enabledTypeCodes}', filtered_codes.enabled_type_codes, true),
  updated_at = now()
FROM filtered_codes
WHERE cs.id = filtered_codes.id;
