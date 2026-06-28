package model

import (
	"context"
	"database/sql"
	"time"
)

type CreateDemandInput struct {
	UserID              string
	CityCode            string
	DemandType          string
	Title               string
	Category            string
	PriceRange          JSONMap
	QuantityRequirement JSONMap
	Attributes          JSONMap
	ContactName         string
	ContactPhone        string
	ContactWechat       string
}

type CreateDemandResult struct {
	ID     string
	Status string
}

type ListDemandsFilter struct {
	CityCode   string
	DemandType string
	Status     string
	Page       int64
	PageSize   int64
}

type DemandListItem struct {
	ID          string
	Title       string
	DemandType  string
	Category    string
	ContactName string
	Status      string
	CreatedAt   string
}

type ListDemandsResult struct {
	Items    []DemandListItem
	Page     int64
	PageSize int64
	Total    int64
}

type DemandDetail struct {
	ID                  string
	Title               string
	DemandType          string
	Category            string
	PriceRange          JSONMap
	QuantityRequirement JSONMap
	Attributes          JSONMap
	ContactName         string
	ContactPhone        string
	ContactWechat       string
	Status              string
	CreatedAt           string
}

type UpdateDemandStatusResult struct {
	ID     string
	Status string
}

type DemandModel struct {
	db *sql.DB
}

func NewDemandModel(db *sql.DB) *DemandModel {
	return &DemandModel{db: db}
}

func (m *DemandModel) CreateDemand(ctx context.Context, input CreateDemandInput) (CreateDemandResult, error) {
	var result CreateDemandResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO purchase_demands (
  user_id,
  city_station_id,
  demand_type,
  status,
  title,
  category,
  price_range,
  quantity_requirement,
  attributes,
  contact_name,
  contact_phone,
  contact_wechat
)
SELECT
  $1,
  cs.id,
  $3,
  'pending',
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11
FROM city_stations cs
WHERE ($2 = '' OR cs.code = $2)
ORDER BY CASE WHEN cs.code = $2 THEN 0 ELSE 1 END
LIMIT 1
RETURNING id, status
`,
		input.UserID,
		input.CityCode,
		input.DemandType,
		input.Title,
		input.Category,
		input.PriceRange,
		input.QuantityRequirement,
		input.Attributes,
		input.ContactName,
		input.ContactPhone,
		input.ContactWechat,
	).Scan(&result.ID, &result.Status)
	return result, err
}

func (m *DemandModel) ListMyDemands(ctx context.Context, userID string, filter ListDemandsFilter) (ListDemandsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	rows, err := m.db.QueryContext(ctx, `
SELECT id, title, demand_type, category, contact_name, status, created_at, COUNT(*) OVER() AS total
FROM purchase_demands
WHERE user_id = $1
  AND ($2 = '' OR status = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
`, userID, filter.Status, pageSize, offset)
	if err != nil {
		return ListDemandsResult{}, err
	}
	return scanDemandRows(rows, page, pageSize)
}

func (m *DemandModel) ListDemands(ctx context.Context, filter ListDemandsFilter) (ListDemandsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	rows, err := m.db.QueryContext(ctx, `
SELECT pd.id, pd.title, pd.demand_type, pd.category, pd.contact_name, pd.status, pd.created_at, COUNT(*) OVER() AS total
FROM purchase_demands pd
LEFT JOIN city_stations cs ON cs.id = pd.city_station_id
WHERE ($1 = '' OR cs.code = $1)
  AND ($2 = '' OR pd.demand_type = $2)
  AND ($3 = '' OR pd.status = $3)
ORDER BY pd.created_at DESC
LIMIT $4 OFFSET $5
`, filter.CityCode, filter.DemandType, filter.Status, pageSize, offset)
	if err != nil {
		return ListDemandsResult{}, err
	}
	return scanDemandRows(rows, page, pageSize)
}

func (m *DemandModel) GetDemand(ctx context.Context, demandID string) (DemandDetail, error) {
	var detail DemandDetail
	var createdAt time.Time
	err := m.db.QueryRowContext(ctx, `
SELECT
  id,
  title,
  demand_type,
  category,
  price_range,
  quantity_requirement,
  attributes,
  contact_name,
  contact_phone,
  contact_wechat,
  status,
  created_at
FROM purchase_demands
WHERE id = $1
`, demandID).Scan(
		&detail.ID,
		&detail.Title,
		&detail.DemandType,
		&detail.Category,
		&detail.PriceRange,
		&detail.QuantityRequirement,
		&detail.Attributes,
		&detail.ContactName,
		&detail.ContactPhone,
		&detail.ContactWechat,
		&detail.Status,
		&createdAt,
	)
	if err != nil {
		return DemandDetail{}, err
	}
	detail.CreatedAt = createdAt.Format(time.RFC3339)
	return detail, nil
}

func (m *DemandModel) UpdateDemandStatus(ctx context.Context, demandID string, status string) (UpdateDemandStatusResult, error) {
	var result UpdateDemandStatusResult
	err := m.db.QueryRowContext(ctx, `
UPDATE purchase_demands
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING id, status
`, demandID, status).Scan(&result.ID, &result.Status)
	return result, err
}

func scanDemandRows(rows *sql.Rows, page int64, pageSize int64) (ListDemandsResult, error) {
	defer rows.Close()
	var items []DemandListItem
	var total int64
	for rows.Next() {
		var item DemandListItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.DemandType, &item.Category, &item.ContactName, &item.Status, &createdAt, &total); err != nil {
			return ListDemandsResult{}, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListDemandsResult{}, err
	}
	return ListDemandsResult{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}
