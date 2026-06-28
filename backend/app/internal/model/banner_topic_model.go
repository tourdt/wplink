package model

import (
	"context"
	"database/sql"
	"time"
)

type BannerTopicConfig struct {
	ID         string
	CityCode   string
	Kind       string
	Title      string
	Subtitle   string
	CoverURL   string
	TypeScope  []string
	JumpType   string
	JumpTarget string
	Tags       []string
	StartAt    string
	EndAt      string
	SortOrder  int64
	Status     string
	CreatedAt  string
	UpdatedAt  string
}

type BannerTopicFilter struct {
	CityCode string
	Kind     string
	Status   string
}

type SaveBannerTopicInput struct {
	ID         string
	CityCode   string
	Kind       string
	Title      string
	Subtitle   string
	CoverURL   string
	TypeScope  []string
	JumpType   string
	JumpTarget string
	Tags       []string
	StartAt    string
	EndAt      string
	SortOrder  int64
	Status     string
}

type SaveBannerTopicResult struct {
	ID        string
	UpdatedAt string
}

type BannerTopicModel struct {
	db *sql.DB
}

func NewBannerTopicModel(db *sql.DB) *BannerTopicModel {
	return &BannerTopicModel{db: db}
}

func (m *BannerTopicModel) ListBannerTopics(ctx context.Context, filter BannerTopicFilter) ([]BannerTopicConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  bt.id::text,
  COALESCE(cs.code, ''),
  bt.kind,
  bt.title,
  COALESCE(bt.subtitle, ''),
  COALESCE(bt.cover_url, ''),
  bt.type_scope,
  bt.jump_type,
  bt.jump_target,
  bt.tags,
  COALESCE(bt.start_at, to_timestamp(0)),
  COALESCE(bt.end_at, to_timestamp(0)),
  bt.sort_order,
  bt.status,
  bt.created_at,
  bt.updated_at
FROM banner_topics bt
LEFT JOIN city_stations cs ON cs.id = bt.city_station_id
WHERE ($1 = '' OR cs.code = $1)
  AND ($2 = '' OR bt.kind = $2)
  AND ($3 = '' OR bt.status = $3)
ORDER BY bt.sort_order DESC, bt.updated_at DESC
`, filter.CityCode, filter.Kind, filter.Status)
	if err != nil {
		return nil, err
	}
	return scanBannerTopics(rows)
}

func (m *BannerTopicModel) ListActiveBannerTopics(ctx context.Context, filter BannerTopicFilter) ([]BannerTopicConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  bt.id::text,
  COALESCE(cs.code, ''),
  bt.kind,
  bt.title,
  COALESCE(bt.subtitle, ''),
  COALESCE(bt.cover_url, ''),
  bt.type_scope,
  bt.jump_type,
  bt.jump_target,
  bt.tags,
  COALESCE(bt.start_at, to_timestamp(0)),
  COALESCE(bt.end_at, to_timestamp(0)),
  bt.sort_order,
  bt.status,
  bt.created_at,
  bt.updated_at
FROM banner_topics bt
LEFT JOIN city_stations cs ON cs.id = bt.city_station_id
WHERE ($1 = '' OR cs.code = $1)
  AND bt.kind = $2
  AND bt.status = 'active'
  AND (bt.start_at IS NULL OR bt.start_at <= now())
  AND (bt.end_at IS NULL OR bt.end_at >= now())
ORDER BY bt.sort_order DESC, bt.updated_at DESC
`, filter.CityCode, filter.Kind)
	if err != nil {
		return nil, err
	}
	return scanBannerTopics(rows)
}

func (m *BannerTopicModel) GetActiveTopic(ctx context.Context, topicID string, cityCode string) (BannerTopicConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  bt.id::text,
  COALESCE(cs.code, ''),
  bt.kind,
  bt.title,
  COALESCE(bt.subtitle, ''),
  COALESCE(bt.cover_url, ''),
  bt.type_scope,
  bt.jump_type,
  bt.jump_target,
  bt.tags,
  COALESCE(bt.start_at, to_timestamp(0)),
  COALESCE(bt.end_at, to_timestamp(0)),
  bt.sort_order,
  bt.status,
  bt.created_at,
  bt.updated_at
FROM banner_topics bt
LEFT JOIN city_stations cs ON cs.id = bt.city_station_id
WHERE bt.id = $1
  AND bt.kind = 'topic'
  AND bt.status = 'active'
  AND ($2 = '' OR cs.code = $2)
  AND (bt.start_at IS NULL OR bt.start_at <= now())
  AND (bt.end_at IS NULL OR bt.end_at >= now())
`, topicID, cityCode)
	if err != nil {
		return BannerTopicConfig{}, err
	}
	items, err := scanBannerTopics(rows)
	if err != nil {
		return BannerTopicConfig{}, err
	}
	if len(items) == 0 {
		return BannerTopicConfig{}, sql.ErrNoRows
	}
	return items[0], nil
}

func (m *BannerTopicModel) CreateBannerTopic(ctx context.Context, input SaveBannerTopicInput) (SaveBannerTopicResult, error) {
	var result SaveBannerTopicResult
	var updatedAt time.Time
	err := m.db.QueryRowContext(ctx, `
INSERT INTO banner_topics (
  city_station_id,
  kind,
  title,
  subtitle,
  cover_url,
  type_scope,
  jump_type,
  jump_target,
  tags,
  start_at,
  end_at,
  sort_order,
  status
)
VALUES (
  (SELECT id FROM city_stations WHERE code = NULLIF($1, '') LIMIT 1),
  $2,
  $3,
  NULLIF($4, ''),
  NULLIF($5, ''),
  $6,
  $7,
  $8,
  $9,
  NULLIF($10, '')::timestamptz,
  NULLIF($11, '')::timestamptz,
  $12,
  $13
)
RETURNING id::text, updated_at
`, input.CityCode, input.Kind, input.Title, input.Subtitle, input.CoverURL, JSONStringSlice(input.TypeScope), input.JumpType, input.JumpTarget, JSONStringSlice(input.Tags), input.StartAt, input.EndAt, input.SortOrder, input.Status).Scan(&result.ID, &updatedAt)
	if err != nil {
		return SaveBannerTopicResult{}, err
	}
	result.UpdatedAt = updatedAt.Format(time.RFC3339)
	return result, nil
}

func (m *BannerTopicModel) UpdateBannerTopic(ctx context.Context, configID string, input SaveBannerTopicInput) (SaveBannerTopicResult, error) {
	var result SaveBannerTopicResult
	var updatedAt time.Time
	err := m.db.QueryRowContext(ctx, `
UPDATE banner_topics
SET
  city_station_id = (SELECT id FROM city_stations WHERE code = NULLIF($2, '') LIMIT 1),
  kind = $3,
  title = $4,
  subtitle = NULLIF($5, ''),
  cover_url = NULLIF($6, ''),
  type_scope = $7,
  jump_type = $8,
  jump_target = $9,
  tags = $10,
  start_at = NULLIF($11, '')::timestamptz,
  end_at = NULLIF($12, '')::timestamptz,
  sort_order = $13,
  status = $14,
  updated_at = now()
WHERE id = $1
RETURNING id::text, updated_at
`, configID, input.CityCode, input.Kind, input.Title, input.Subtitle, input.CoverURL, JSONStringSlice(input.TypeScope), input.JumpType, input.JumpTarget, JSONStringSlice(input.Tags), input.StartAt, input.EndAt, input.SortOrder, input.Status).Scan(&result.ID, &updatedAt)
	if err != nil {
		return SaveBannerTopicResult{}, err
	}
	result.UpdatedAt = updatedAt.Format(time.RFC3339)
	return result, nil
}

func scanBannerTopics(rows *sql.Rows) ([]BannerTopicConfig, error) {
	defer rows.Close()
	var items []BannerTopicConfig
	for rows.Next() {
		var item BannerTopicConfig
		var typeScope JSONStringSlice
		var tags JSONStringSlice
		var startAt time.Time
		var endAt time.Time
		var createdAt time.Time
		var updatedAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.CityCode,
			&item.Kind,
			&item.Title,
			&item.Subtitle,
			&item.CoverURL,
			&typeScope,
			&item.JumpType,
			&item.JumpTarget,
			&tags,
			&startAt,
			&endAt,
			&item.SortOrder,
			&item.Status,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		item.TypeScope = []string(typeScope)
		item.Tags = []string(tags)
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
