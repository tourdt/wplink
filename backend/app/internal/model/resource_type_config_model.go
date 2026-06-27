package model

import (
	"context"
	"database/sql"
	"time"
)

type ResourceTypeConfig struct {
	ID               string
	TypeCode         string
	TypeName         string
	DefaultValidDays int64
	RequiredFields   []string
	FilterFields     []string
	DisplayTemplate  JSONMap
}

type AdminResourceTypeConfig struct {
	ID               string
	CityCode         string
	TypeCode         string
	TypeName         string
	FieldSchema      JSONMap
	RequiredFields   []string
	FilterFields     []string
	DisplayTemplate  JSONMap
	ReviewRules      JSONMap
	SortWeights      JSONMap
	MessageRules     JSONMap
	DefaultValidDays int64
	Status           string
}

type ResourceTypeConfigPatch struct {
	FieldSchema      JSONMap
	RequiredFields   []string
	FilterFields     []string
	DisplayTemplate  JSONMap
	ReviewRules      JSONMap
	SortWeights      JSONMap
	MessageRules     JSONMap
	DefaultValidDays int64
	Status           string
}

type ResourceTypeConfigModel struct {
	db *sql.DB
}

func NewResourceTypeConfigModel(db *sql.DB) *ResourceTypeConfigModel {
	return &ResourceTypeConfigModel{db: db}
}

func (m *ResourceTypeConfigModel) ListActiveResourceTypesByCityCode(ctx context.Context, cityCode string) ([]ResourceTypeConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  rtc.id,
  rtc.type_code,
  rtc.type_name,
  rtc.default_valid_days,
  rtc.required_fields,
  rtc.filter_fields,
  rtc.display_template
FROM resource_type_configs rtc
JOIN city_stations cs ON cs.id = rtc.city_station_id
WHERE cs.code = $1
  AND cs.status = 'active'
  AND rtc.status = 'active'
ORDER BY rtc.created_at ASC
`, cityCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []ResourceTypeConfig
	for rows.Next() {
		var config ResourceTypeConfig
		var requiredFields JSONStringSlice
		var filterFields JSONStringSlice
		if err := rows.Scan(
			&config.ID,
			&config.TypeCode,
			&config.TypeName,
			&config.DefaultValidDays,
			&requiredFields,
			&filterFields,
			&config.DisplayTemplate,
		); err != nil {
			return nil, err
		}
		config.RequiredFields = []string(requiredFields)
		config.FilterFields = []string(filterFields)
		configs = append(configs, config)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}

func (m *ResourceTypeConfigModel) ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]AdminResourceTypeConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  rtc.id,
  COALESCE(cs.code, ''),
  rtc.type_code,
  rtc.type_name,
  rtc.field_schema,
  rtc.required_fields,
  rtc.filter_fields,
  rtc.display_template,
  rtc.review_rules,
  rtc.sort_weights,
  rtc.message_rules,
  rtc.default_valid_days,
  rtc.status
FROM resource_type_configs rtc
LEFT JOIN city_stations cs ON cs.id = rtc.city_station_id
WHERE ($1 = '' OR cs.code = $1)
  AND ($2 = '' OR rtc.status = $2)
ORDER BY rtc.created_at ASC
`, cityCode, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []AdminResourceTypeConfig
	for rows.Next() {
		var config AdminResourceTypeConfig
		var requiredFields JSONStringSlice
		var filterFields JSONStringSlice
		if err := rows.Scan(
			&config.ID,
			&config.CityCode,
			&config.TypeCode,
			&config.TypeName,
			&config.FieldSchema,
			&requiredFields,
			&filterFields,
			&config.DisplayTemplate,
			&config.ReviewRules,
			&config.SortWeights,
			&config.MessageRules,
			&config.DefaultValidDays,
			&config.Status,
		); err != nil {
			return nil, err
		}
		config.RequiredFields = []string(requiredFields)
		config.FilterFields = []string(filterFields)
		configs = append(configs, config)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}

func (m *ResourceTypeConfigModel) UpdateResourceTypeConfig(ctx context.Context, configID string, patch ResourceTypeConfigPatch) (string, error) {
	updatedAt := time.Now().UTC()
	_, err := m.db.ExecContext(ctx, `
UPDATE resource_type_configs
SET
  field_schema = $2,
  required_fields = $3,
  filter_fields = $4,
  display_template = $5,
  review_rules = $6,
  sort_weights = $7,
  message_rules = $8,
  default_valid_days = $9,
  status = $10,
  updated_at = $11
WHERE id = $1
`, configID,
		patch.FieldSchema,
		JSONStringSlice(patch.RequiredFields),
		JSONStringSlice(patch.FilterFields),
		patch.DisplayTemplate,
		patch.ReviewRules,
		patch.SortWeights,
		patch.MessageRules,
		patch.DefaultValidDays,
		patch.Status,
		updatedAt,
	)
	if err != nil {
		return "", err
	}
	return updatedAt.Format(time.RFC3339), nil
}
