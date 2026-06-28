package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
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

type resourceTypeConfigRow struct {
	ID               string          `db:"id"`
	TypeCode         string          `db:"type_code"`
	TypeName         string          `db:"type_name"`
	DefaultValidDays int64           `db:"default_valid_days"`
	RequiredFields   JSONStringSlice `db:"required_fields"`
	FilterFields     JSONStringSlice `db:"filter_fields"`
	DisplayTemplate  JSONMap         `db:"display_template"`
}

type adminResourceTypeConfigRow struct {
	ID               string          `db:"id"`
	CityCode         string          `db:"city_code"`
	TypeCode         string          `db:"type_code"`
	TypeName         string          `db:"type_name"`
	FieldSchema      JSONMap         `db:"field_schema"`
	RequiredFields   JSONStringSlice `db:"required_fields"`
	FilterFields     JSONStringSlice `db:"filter_fields"`
	DisplayTemplate  JSONMap         `db:"display_template"`
	ReviewRules      JSONMap         `db:"review_rules"`
	SortWeights      JSONMap         `db:"sort_weights"`
	MessageRules     JSONMap         `db:"message_rules"`
	DefaultValidDays int64           `db:"default_valid_days"`
	Status           string          `db:"status"`
}

type ResourceTypeConfigModel struct {
	conn  sqlx.SqlConn
	table ResourceTypeConfigsModel
}

func NewResourceTypeConfigModel(db *sql.DB) *ResourceTypeConfigModel {
	conn := sqlx.NewSqlConnFromDB(db)
	return &ResourceTypeConfigModel{
		conn:  conn,
		table: NewResourceTypeConfigsModel(conn),
	}
}

func (m *ResourceTypeConfigModel) ListActiveResourceTypesByCityCode(ctx context.Context, cityCode string) ([]ResourceTypeConfig, error) {
	var rows []resourceTypeConfigRow
	err := m.conn.QueryRowsCtx(ctx, &rows, `
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
	configs := make([]ResourceTypeConfig, 0, len(rows))
	for _, row := range rows {
		configs = append(configs, ResourceTypeConfig{
			ID:               row.ID,
			TypeCode:         row.TypeCode,
			TypeName:         row.TypeName,
			DefaultValidDays: row.DefaultValidDays,
			RequiredFields:   []string(row.RequiredFields),
			FilterFields:     []string(row.FilterFields),
			DisplayTemplate:  row.DisplayTemplate,
		})
	}
	return configs, nil
}

func (m *ResourceTypeConfigModel) ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]AdminResourceTypeConfig, error) {
	var rows []adminResourceTypeConfigRow
	err := m.conn.QueryRowsCtx(ctx, &rows, `
SELECT
  rtc.id,
  COALESCE(cs.code, '') AS city_code,
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
	configs := make([]AdminResourceTypeConfig, 0, len(rows))
	for _, row := range rows {
		configs = append(configs, AdminResourceTypeConfig{
			ID:               row.ID,
			CityCode:         row.CityCode,
			TypeCode:         row.TypeCode,
			TypeName:         row.TypeName,
			FieldSchema:      row.FieldSchema,
			RequiredFields:   []string(row.RequiredFields),
			FilterFields:     []string(row.FilterFields),
			DisplayTemplate:  row.DisplayTemplate,
			ReviewRules:      row.ReviewRules,
			SortWeights:      row.SortWeights,
			MessageRules:     row.MessageRules,
			DefaultValidDays: row.DefaultValidDays,
			Status:           row.Status,
		})
	}
	return configs, nil
}

func (m *ResourceTypeConfigModel) UpdateResourceTypeConfig(ctx context.Context, configID string, patch ResourceTypeConfigPatch) (string, error) {
	updatedAt := time.Now().UTC()
	_, err := m.conn.ExecCtx(ctx, `
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
