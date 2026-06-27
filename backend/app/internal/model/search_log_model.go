package model

import (
	"context"
	"database/sql"
)

type SearchLogInput struct {
	UserID      string
	CityCode    string
	Keyword     string
	Filters     JSONMap
	ResultCount int64
}

type SearchLogModel struct {
	db *sql.DB
}

func NewSearchLogModel(db *sql.DB) *SearchLogModel {
	return &SearchLogModel{db: db}
}

func (m *SearchLogModel) RecordSearchLog(ctx context.Context, input SearchLogInput) error {
	_, err := m.db.ExecContext(ctx, `
INSERT INTO search_logs (user_id, city_station_id, keyword, filters, result_count)
SELECT
  NULLIF($1, '')::uuid,
  cs.id,
  $3,
  $4,
  $5
FROM city_stations cs
WHERE ($2 = '' OR cs.code = $2)
ORDER BY CASE WHEN cs.code = $2 THEN 0 ELSE 1 END
LIMIT 1
`, input.UserID, input.CityCode, input.Keyword, input.Filters, input.ResultCount)
	return err
}
