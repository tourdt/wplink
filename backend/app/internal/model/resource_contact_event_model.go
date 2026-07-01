package model

import (
	"context"
	"database/sql"
)

type ResourceContactEventInput struct {
	ResourceID string
	UserID     string
	Action     string
}

type ResourceContactEventResult struct {
	ID         string
	MerchantID string
}

type ResourceContactUnlockInfo struct {
	ResourceID string
	MerchantID string
	Status     string
	Phone      string
	Wechat     string
	ExpiresAt  sql.NullTime
}

type ResourceContactEventModel struct {
	db *sql.DB
}

func NewResourceContactEventModel(db *sql.DB) *ResourceContactEventModel {
	return &ResourceContactEventModel{db: db}
}

func (m *ResourceContactEventModel) GetResourceContactUnlockInfo(ctx context.Context, resourceID string) (ResourceContactUnlockInfo, error) {
	var info ResourceContactUnlockInfo
	err := m.db.QueryRowContext(ctx, `
SELECT
  r.id::text,
  r.merchant_id::text,
  r.status,
  r.contact_phone,
  COALESCE(r.contact_wechat, ''),
  r.expires_at
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
WHERE r.id = $1
  AND r.deleted_at IS NULL
  AND m.deleted_at IS NULL
  AND m.status = 'active'
`, resourceID).Scan(
		&info.ResourceID,
		&info.MerchantID,
		&info.Status,
		&info.Phone,
		&info.Wechat,
		&info.ExpiresAt,
	)
	return info, err
}

func (m *ResourceContactEventModel) RecordResourceContactEvent(ctx context.Context, input ResourceContactEventInput) (ResourceContactEventResult, error) {
	var result ResourceContactEventResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO resource_contact_events (
  resource_id,
  user_id,
  merchant_id,
  action
)
SELECT
  r.id,
  NULLIF($2, '')::bigint,
  r.merchant_id,
  $3
FROM resources r
WHERE r.id = $1
  AND r.deleted_at IS NULL
RETURNING id::text, merchant_id::text
`, input.ResourceID, input.UserID, input.Action).Scan(&result.ID, &result.MerchantID)
	return result, err
}
