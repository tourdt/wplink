package model

import (
	"context"
	"database/sql"
)

type CityStation struct {
	ID              string
	Code            string
	Name            string
	PrimaryCategory string
	Status          string
}

type CityStationModel struct {
	db *sql.DB
}

func NewCityStationModel(db *sql.DB) *CityStationModel {
	return &CityStationModel{db: db}
}

func (m *CityStationModel) ListActiveCityStations(ctx context.Context) ([]CityStation, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT id, code, name, COALESCE(primary_category, ''), status
FROM city_stations
WHERE status = 'active'
ORDER BY created_at ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stations []CityStation
	for rows.Next() {
		var station CityStation
		if err := rows.Scan(&station.ID, &station.Code, &station.Name, &station.PrimaryCategory, &station.Status); err != nil {
			return nil, err
		}
		stations = append(stations, station)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stations, nil
}
