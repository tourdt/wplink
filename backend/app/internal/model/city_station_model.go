package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type CityStation struct {
	ID              string `db:"id"`
	Code            string `db:"code"`
	Name            string `db:"name"`
	PrimaryCategory string `db:"primary_category"`
	Status          string `db:"status"`
}

type CityStationModel struct {
	db    *sql.DB
	conn  sqlx.SqlConn
	table CityStationsModel
}

func NewCityStationModel(db *sql.DB) *CityStationModel {
	conn := sqlx.NewSqlConnFromDB(db)
	return &CityStationModel{
		db:    db,
		conn:  conn,
		table: NewCityStationsModel(conn),
	}
}

func (m *CityStationModel) ListActiveCityStations(ctx context.Context) ([]CityStation, error) {
	var stations []CityStation
	err := m.conn.QueryRowsCtx(ctx, &stations, `
SELECT id::text, code, name, COALESCE(primary_category, ''), status
FROM city_stations
WHERE status = 'active'
ORDER BY created_at ASC
`)
	if err != nil {
		return nil, err
	}
	return stations, nil
}

func (m *CityStationModel) GetVerificationBillingConfig(ctx context.Context, cityCode string) (VerificationBillingConfig, error) {
	var code string
	var billingConfig JSONMap
	var updatedAt sql.NullTime
	row := m.db.QueryRowContext(ctx, `
SELECT code, COALESCE(config->'verificationBilling', '{}'::jsonb), updated_at
FROM city_stations
WHERE code = $1
LIMIT 1
`, cityCode)
	if err := row.Scan(&code, &billingConfig, &updatedAt); err != nil {
		return VerificationBillingConfig{}, err
	}
	config := VerificationBillingConfigFromJSON(code, billingConfig)
	if updatedAt.Valid && config.UpdatedAt == "" {
		config.UpdatedAt = updatedAt.Time.Format(time.RFC3339)
	}
	return config, nil
}

func (m *CityStationModel) UpdateVerificationBillingConfig(ctx context.Context, input VerificationBillingConfig) (VerificationBillingConfig, error) {
	now := time.Now().Format(time.RFC3339)
	input.UpdatedAt = now
	configJSON := input.ToJSONMap()
	var savedCode string
	var savedConfig JSONMap
	var updatedAt time.Time
	row := m.db.QueryRowContext(ctx, `
UPDATE city_stations
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{verificationBilling}', $2::jsonb, true),
    updated_at = now()
WHERE code = $1
RETURNING code, COALESCE(config->'verificationBilling', '{}'::jsonb), updated_at
`, input.CityCode, configJSON)
	if err := row.Scan(&savedCode, &savedConfig, &updatedAt); err != nil {
		return VerificationBillingConfig{}, err
	}
	result := VerificationBillingConfigFromJSON(savedCode, savedConfig)
	result.UpdatedAt = updatedAt.Format(time.RFC3339)
	return result, nil
}
