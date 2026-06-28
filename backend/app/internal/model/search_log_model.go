package model

import (
	"context"
	"database/sql"
	"time"
)

type SearchLogInput struct {
	UserID      string
	CityCode    string
	Keyword     string
	Filters     JSONMap
	ResultCount int64
}

type SearchLogFilter struct {
	CityCode string
	Keyword  string
	Page     int64
	PageSize int64
}

type SearchLogItem struct {
	ID          string
	UserID      string
	CityCode    string
	CityName    string
	Keyword     string
	Filters     JSONMap
	ResultCount int64
	CreatedAt   string
}

type ListSearchLogsResult struct {
	Items    []SearchLogItem
	Page     int64
	PageSize int64
	Total    int64
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

func (m *SearchLogModel) ListSearchLogs(ctx context.Context, filter SearchLogFilter) (ListSearchLogsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT
  sl.id,
  COALESCE(sl.user_id::text, ''),
  COALESCE(cs.code, ''),
  COALESCE(cs.name, ''),
  sl.keyword,
  sl.filters,
  sl.result_count,
  sl.created_at,
  COUNT(*) OVER() AS total
FROM search_logs sl
LEFT JOIN city_stations cs ON cs.id = sl.city_station_id
WHERE ($1 = '' OR cs.code = $1)
  AND ($2 = '' OR sl.keyword ILIKE '%' || $2 || '%')
ORDER BY sl.created_at DESC
LIMIT $3 OFFSET $4
`, filter.CityCode, filter.Keyword, pageSize, offset)
	if err != nil {
		return ListSearchLogsResult{}, err
	}
	defer rows.Close()

	result := ListSearchLogsResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item SearchLogItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.UserID, &item.CityCode, &item.CityName, &item.Keyword, &item.Filters, &item.ResultCount, &createdAt, &result.Total); err != nil {
			return ListSearchLogsResult{}, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListSearchLogsResult{}, err
	}
	return result, nil
}
