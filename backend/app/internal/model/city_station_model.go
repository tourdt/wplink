package model

import (
	"context"
	"database/sql"

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
	conn  sqlx.SqlConn
	table CityStationsModel
}

func NewCityStationModel(db *sql.DB) *CityStationModel {
	conn := sqlx.NewSqlConnFromDB(db)
	return &CityStationModel{
		conn:  conn,
		table: NewCityStationsModel(conn),
	}
}

func (m *CityStationModel) ListActiveCityStations(ctx context.Context) ([]CityStation, error) {
	var stations []CityStation
	err := m.conn.QueryRowsCtx(ctx, &stations, `
SELECT id, code, name, COALESCE(primary_category, ''), status
FROM city_stations
WHERE status = 'active'
ORDER BY created_at ASC
`)
	if err != nil {
		return nil, err
	}
	return stations, nil
}
