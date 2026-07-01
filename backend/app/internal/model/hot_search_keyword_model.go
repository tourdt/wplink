package model

import (
	"context"
	"database/sql"
	"time"
)

type HotSearchKeywordConfig struct {
	ID        string
	CityCode  string
	Keyword   string
	SortOrder int64
	Status    string
	StartAt   string
	EndAt     string
	CreatedAt string
	UpdatedAt string
}

type HotSearchKeywordFilter struct {
	CityCode string
	Status   string
}

type SaveHotSearchKeywordInput struct {
	ID        string
	CityCode  string
	Keyword   string
	SortOrder int64
	Status    string
	StartAt   string
	EndAt     string
}

type SaveHotSearchKeywordResult struct {
	ID        string
	UpdatedAt string
}

type HotSearchKeywordModel struct {
	db *sql.DB
}

func NewHotSearchKeywordModel(db *sql.DB) *HotSearchKeywordModel {
	return &HotSearchKeywordModel{db: db}
}

func (m *HotSearchKeywordModel) ListActiveHotSearchKeywords(ctx context.Context, cityCode string) ([]HotSearchKeywordConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  hsk.id::text,
  COALESCE(cs.code, ''),
  hsk.keyword,
  hsk.sort_order,
  hsk.status,
  COALESCE(hsk.start_at, to_timestamp(0)),
  COALESCE(hsk.end_at, to_timestamp(0)),
  hsk.created_at,
  hsk.updated_at
FROM hot_search_keywords hsk
LEFT JOIN city_stations cs ON cs.id = hsk.city_station_id
WHERE ($1 = '' OR cs.code = $1)
  AND hsk.status = 'active'
  AND (hsk.start_at IS NULL OR hsk.start_at <= now())
  AND (hsk.end_at IS NULL OR hsk.end_at >= now())
ORDER BY hsk.sort_order DESC, hsk.updated_at DESC
`, cityCode)
	if err != nil {
		return nil, err
	}
	return scanHotSearchKeywords(rows)
}

func (m *HotSearchKeywordModel) ListHotSearchKeywords(ctx context.Context, filter HotSearchKeywordFilter) ([]HotSearchKeywordConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  hsk.id::text,
  COALESCE(cs.code, ''),
  hsk.keyword,
  hsk.sort_order,
  hsk.status,
  COALESCE(hsk.start_at, to_timestamp(0)),
  COALESCE(hsk.end_at, to_timestamp(0)),
  hsk.created_at,
  hsk.updated_at
FROM hot_search_keywords hsk
LEFT JOIN city_stations cs ON cs.id = hsk.city_station_id
WHERE ($1 = '' OR cs.code = $1)
  AND ($2 = '' OR hsk.status = $2)
ORDER BY hsk.sort_order DESC, hsk.updated_at DESC
`, filter.CityCode, filter.Status)
	if err != nil {
		return nil, err
	}
	return scanHotSearchKeywords(rows)
}

func (m *HotSearchKeywordModel) CreateHotSearchKeyword(ctx context.Context, input SaveHotSearchKeywordInput) (SaveHotSearchKeywordResult, error) {
	var result SaveHotSearchKeywordResult
	var updatedAt time.Time
	err := m.db.QueryRowContext(ctx, `
INSERT INTO hot_search_keywords (
  city_station_id,
  keyword,
  sort_order,
  status,
  start_at,
  end_at
)
VALUES (
  (SELECT id FROM city_stations WHERE code = NULLIF($1, '') LIMIT 1),
  $2,
  $3,
  $4,
  NULLIF($5, '')::timestamptz,
  NULLIF($6, '')::timestamptz
)
RETURNING id::text, updated_at
`, input.CityCode, input.Keyword, input.SortOrder, input.Status, input.StartAt, input.EndAt).Scan(&result.ID, &updatedAt)
	if err != nil {
		return SaveHotSearchKeywordResult{}, err
	}
	result.UpdatedAt = updatedAt.Format(time.RFC3339)
	return result, nil
}

func (m *HotSearchKeywordModel) UpdateHotSearchKeyword(ctx context.Context, configID string, input SaveHotSearchKeywordInput) (SaveHotSearchKeywordResult, error) {
	var result SaveHotSearchKeywordResult
	var updatedAt time.Time
	err := m.db.QueryRowContext(ctx, `
UPDATE hot_search_keywords
SET
  city_station_id = (SELECT id FROM city_stations WHERE code = NULLIF($2, '') LIMIT 1),
  keyword = $3,
  sort_order = $4,
  status = $5,
  start_at = NULLIF($6, '')::timestamptz,
  end_at = NULLIF($7, '')::timestamptz,
  updated_at = now()
WHERE id = $1
RETURNING id::text, updated_at
`, configID, input.CityCode, input.Keyword, input.SortOrder, input.Status, input.StartAt, input.EndAt).Scan(&result.ID, &updatedAt)
	if err != nil {
		return SaveHotSearchKeywordResult{}, err
	}
	result.UpdatedAt = updatedAt.Format(time.RFC3339)
	return result, nil
}

func scanHotSearchKeywords(rows *sql.Rows) ([]HotSearchKeywordConfig, error) {
	defer rows.Close()
	var items []HotSearchKeywordConfig
	for rows.Next() {
		var item HotSearchKeywordConfig
		var startAt time.Time
		var endAt time.Time
		var createdAt time.Time
		var updatedAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.CityCode,
			&item.Keyword,
			&item.SortOrder,
			&item.Status,
			&startAt,
			&endAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if !startAt.IsZero() && startAt.Unix() != 0 {
			item.StartAt = startAt.Format(time.RFC3339)
		}
		if !endAt.IsZero() && endAt.Unix() != 0 {
			item.EndAt = endAt.Format(time.RFC3339)
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
