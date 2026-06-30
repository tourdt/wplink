package model

import (
	"context"
	"database/sql"
	"time"
)

type ResourceFavoriteInput struct {
	UserID     string
	ResourceID string
	Favorited  bool
}

type ResourceFavoriteState struct {
	ResourceID string
	Favorited  bool
}

type MerchantFollowInput struct {
	UserID     string
	MerchantID string
	Followed   bool
}

type MerchantFollowState struct {
	MerchantID string
	Followed   bool
}

type ListInteractionFilter struct {
	Page     int64
	PageSize int64
}

type FollowedMerchantItem struct {
	ID                 string
	Name               string
	MerchantType       string
	VerificationStatus string
	MainCategories     []string
	FollowedAt         string
}

type ListFollowedMerchantsResult struct {
	Items    []FollowedMerchantItem
	Page     int64
	PageSize int64
	Total    int64
}

type SavedSearchInput struct {
	UserID       string
	Name         string
	CityCode     string
	TypeCode     string
	Keyword      string
	Category     string
	VerifiedOnly bool
}

type SavedSearchResult struct {
	ID string
}

type SavedSearchItem struct {
	ID           string
	Name         string
	CityCode     string
	TypeCode     string
	Keyword      string
	Category     string
	VerifiedOnly bool
	CreatedAt    string
}

type ListSavedSearchesResult struct {
	Items    []SavedSearchItem
	Page     int64
	PageSize int64
	Total    int64
}

type FavoriteModel struct {
	db *sql.DB
}

func NewFavoriteModel(db *sql.DB) *FavoriteModel {
	return &FavoriteModel{db: db}
}

func (m *FavoriteModel) SetResourceFavorite(ctx context.Context, input ResourceFavoriteInput) (ResourceFavoriteState, error) {
	status := interactionStatus(input.Favorited)
	var state ResourceFavoriteState
	err := m.db.QueryRowContext(ctx, `
INSERT INTO user_favorite_resources (user_id, resource_id, status, created_at, updated_at)
SELECT $1::bigint, r.id, $3, now(), now()
FROM resources r
WHERE r.id = $2::bigint
  AND r.deleted_at IS NULL
  AND r.status = 'published'
  AND (r.expires_at IS NULL OR r.expires_at > now())
ON CONFLICT (user_id, resource_id)
DO UPDATE SET status = EXCLUDED.status, updated_at = now()
RETURNING resource_id::text, status = 'active'
`, input.UserID, input.ResourceID, status).Scan(&state.ResourceID, &state.Favorited)
	return state, err
}

func (m *FavoriteModel) ResourceBelongsToUser(ctx context.Context, userID string, resourceID string) (bool, error) {
	var exists bool
	err := m.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM resources r
  JOIN merchant_admin_bindings mab ON mab.merchant_id = r.merchant_id
  JOIN merchants m ON m.id = r.merchant_id
  WHERE r.id = $1::bigint
    AND mab.user_id = $2::bigint
    AND r.deleted_at IS NULL
    AND r.status = 'published'
    AND (r.expires_at IS NULL OR r.expires_at > now())
    AND mab.status = 'active'
    AND m.deleted_at IS NULL
    AND m.status = 'active'
)
`, resourceID, userID).Scan(&exists)
	return exists, err
}

func (m *FavoriteModel) GetResourceFavoriteState(ctx context.Context, userID string, resourceID string) (ResourceFavoriteState, error) {
	state := ResourceFavoriteState{ResourceID: resourceID}
	err := m.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM user_favorite_resources
  WHERE user_id = $1::bigint AND resource_id = $2::bigint AND status = 'active'
)
`, userID, resourceID).Scan(&state.Favorited)
	return state, err
}

func (m *FavoriteModel) ListFavoriteResources(ctx context.Context, userID string, filter ListInteractionFilter) (ListResourcesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT
  r.id::text,
  r.type_code,
  r.title,
  r.category,
  COALESCE(r.district, ''),
  COALESCE(r.price_text, ''),
  COALESCE(r.quantity_text, ''),
  m.id::text,
  m.name,
  m.verification_status,
  COALESCE(r.refreshed_at, r.published_at, r.created_at),
  COUNT(*) OVER() AS total
FROM user_favorite_resources ufr
JOIN resources r ON r.id = ufr.resource_id
JOIN merchants m ON m.id = r.merchant_id
WHERE ufr.user_id = $1::bigint
  AND ufr.status = 'active'
  AND r.deleted_at IS NULL
  AND r.status = 'published'
  AND (r.expires_at IS NULL OR r.expires_at > now())
ORDER BY ufr.updated_at DESC
LIMIT $2 OFFSET $3
`, userID, pageSize, offset)
	if err != nil {
		return ListResourcesResult{}, err
	}
	defer rows.Close()
	return scanResourceListRows(rows, page, pageSize)
}

func (m *FavoriteModel) SetMerchantFollow(ctx context.Context, input MerchantFollowInput) (MerchantFollowState, error) {
	status := interactionStatus(input.Followed)
	var state MerchantFollowState
	err := m.db.QueryRowContext(ctx, `
INSERT INTO user_followed_merchants (user_id, merchant_id, status, created_at, updated_at)
SELECT $1::bigint, m.id, $3, now(), now()
FROM merchants m
WHERE m.id = $2::bigint
  AND m.deleted_at IS NULL
  AND m.status = 'active'
ON CONFLICT (user_id, merchant_id)
DO UPDATE SET status = EXCLUDED.status, updated_at = now()
RETURNING merchant_id::text, status = 'active'
`, input.UserID, input.MerchantID, status).Scan(&state.MerchantID, &state.Followed)
	return state, err
}

func (m *FavoriteModel) GetMerchantFollowState(ctx context.Context, userID string, merchantID string) (MerchantFollowState, error) {
	state := MerchantFollowState{MerchantID: merchantID}
	err := m.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM user_followed_merchants
  WHERE user_id = $1::bigint AND merchant_id = $2::bigint AND status = 'active'
)
`, userID, merchantID).Scan(&state.Followed)
	return state, err
}

func (m *FavoriteModel) ListFollowedMerchants(ctx context.Context, userID string, filter ListInteractionFilter) (ListFollowedMerchantsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT
  m.id::text,
  m.name,
  m.merchant_type,
  m.verification_status,
  m.main_categories,
  ufm.updated_at,
  COUNT(*) OVER() AS total
FROM user_followed_merchants ufm
JOIN merchants m ON m.id = ufm.merchant_id
WHERE ufm.user_id = $1::bigint
  AND ufm.status = 'active'
  AND m.deleted_at IS NULL
  AND m.status = 'active'
ORDER BY ufm.updated_at DESC
LIMIT $2 OFFSET $3
`, userID, pageSize, offset)
	if err != nil {
		return ListFollowedMerchantsResult{}, err
	}
	defer rows.Close()

	result := ListFollowedMerchantsResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item FollowedMerchantItem
		var categories JSONStringSlice
		var followedAt time.Time
		if err := rows.Scan(&item.ID, &item.Name, &item.MerchantType, &item.VerificationStatus, &categories, &followedAt, &result.Total); err != nil {
			return ListFollowedMerchantsResult{}, err
		}
		item.MainCategories = []string(categories)
		item.FollowedAt = followedAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListFollowedMerchantsResult{}, err
	}
	return result, nil
}

func (m *FavoriteModel) CreateSavedSearch(ctx context.Context, input SavedSearchInput) (SavedSearchResult, error) {
	var result SavedSearchResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO user_saved_searches (
  user_id,
  name,
  city_station_id,
  type_code,
  keyword,
  category,
  verified_only
)
SELECT
  $1::bigint,
  $2,
  cs.id,
  $4,
  $5,
  $6,
  $7
FROM city_stations cs
WHERE ($3 = '' OR cs.code = $3)
ORDER BY CASE WHEN cs.code = $3 THEN 0 ELSE 1 END
LIMIT 1
RETURNING id::text
`, input.UserID, input.Name, input.CityCode, input.TypeCode, input.Keyword, input.Category, input.VerifiedOnly).Scan(&result.ID)
	return result, err
}

func (m *FavoriteModel) ListSavedSearches(ctx context.Context, userID string, filter ListInteractionFilter) (ListSavedSearchesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT
  uss.id::text,
  uss.name,
  COALESCE(cs.code, ''),
  COALESCE(uss.type_code, ''),
  COALESCE(uss.keyword, ''),
  COALESCE(uss.category, ''),
  uss.verified_only,
  uss.created_at,
  COUNT(*) OVER() AS total
FROM user_saved_searches uss
LEFT JOIN city_stations cs ON cs.id = uss.city_station_id
WHERE uss.user_id = $1::bigint
  AND uss.deleted_at IS NULL
ORDER BY uss.created_at DESC
LIMIT $2 OFFSET $3
`, userID, pageSize, offset)
	if err != nil {
		return ListSavedSearchesResult{}, err
	}
	defer rows.Close()

	result := ListSavedSearchesResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item SavedSearchItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Name, &item.CityCode, &item.TypeCode, &item.Keyword, &item.Category, &item.VerifiedOnly, &createdAt, &result.Total); err != nil {
			return ListSavedSearchesResult{}, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListSavedSearchesResult{}, err
	}
	return result, nil
}

func (m *FavoriteModel) DeleteSavedSearch(ctx context.Context, userID string, savedSearchID string) error {
	result, err := m.db.ExecContext(ctx, `
UPDATE user_saved_searches
SET deleted_at = now()
WHERE id = $1::bigint AND user_id = $2::bigint AND deleted_at IS NULL
`, savedSearchID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func interactionStatus(active bool) string {
	if active {
		return "active"
	}
	return "canceled"
}

func scanResourceListRows(rows *sql.Rows, page int64, pageSize int64) (ListResourcesResult, error) {
	result := ListResourcesResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item ResourceListItem
		var refreshedAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.TypeCode,
			&item.Title,
			&item.Category,
			&item.District,
			&item.PriceText,
			&item.QuantityText,
			&item.Merchant.ID,
			&item.Merchant.Name,
			&item.Merchant.VerificationStatus,
			&refreshedAt,
			&result.Total,
		); err != nil {
			return ListResourcesResult{}, err
		}
		item.RefreshedAt = refreshedAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListResourcesResult{}, err
	}
	return result, nil
}
