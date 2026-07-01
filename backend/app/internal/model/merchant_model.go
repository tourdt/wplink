package model

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

const (
	MerchantStatusActive = "active"
)

type CreateMerchantInput struct {
	CreatorUserID  string
	CityCode       string
	Name           string
	MerchantType   string
	MainCategories []string
	ContactName    string
	ContactPhone   string
	ContactWechat  string
	AddressText    string
	Description    string
}

type CreateMerchantResult struct {
	ID                 string
	Name               string
	VerificationStatus string
	Status             string
}

type CreditTag struct {
	Code  string
	Label string
}

type MerchantDetail struct {
	ID                 string
	Name               string
	MerchantType       string
	CityCode           string
	MainCategories     []string
	VerificationStatus string
	CreditTags         []CreditTag
	ContactName        string
	PhoneMasked        string
	WechatMasked       string
	PublishedCount     int64
	DealtCount         int64
	FollowerCount      int64
	AddressText        string
	Location           JSONMap
	Description        string
	LogoURL            string
	Images             []string
	LastActiveAt       string
}

type UpdateMerchantPatch struct {
	MainCategories []string
	MerchantType   string
	Description    string
	LogoURL        string
	Images         []string
	ContactName    string
	ContactPhone   string
	ContactWechat  string
	AddressText    string
	Location       JSONMap
	LocationSet    bool
}

type ListMerchantsFilter struct {
	CityCode     string
	MerchantType string
	Status       string
	Keyword      string
	Page         int64
	PageSize     int64
}

type MerchantListItem struct {
	ID                 string
	Name               string
	MerchantType       string
	VerificationStatus string
	Status             string
	LastActiveAt       string
}

type ListMerchantsResult struct {
	Items    []MerchantListItem
	Page     int64
	PageSize int64
	Total    int64
}

type MerchantModel struct {
	db *sql.DB
}

func NewMerchantModel(db *sql.DB) *MerchantModel {
	return &MerchantModel{db: db}
}

func (m *MerchantModel) CreateMerchant(ctx context.Context, input CreateMerchantInput) (CreateMerchantResult, error) {
	var result CreateMerchantResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `
INSERT INTO merchants (
  city_station_id,
  name,
  merchant_type,
  main_categories,
  description,
  contact_name,
  contact_phone,
  contact_wechat,
  address_text
)
SELECT
  cs.id,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9
FROM city_stations cs
WHERE cs.code = $1 AND cs.status = 'active'
RETURNING id::text, name, verification_status, status
`,
			input.CityCode,
			input.Name,
			input.MerchantType,
			JSONStringSlice(input.MainCategories),
			input.Description,
			input.ContactName,
			input.ContactPhone,
			input.ContactWechat,
			input.AddressText,
		).Scan(&result.ID, &result.Name, &result.VerificationStatus, &result.Status); err != nil {
			return err
		}
		if input.CreatorUserID == "" {
			return nil
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO merchant_admin_bindings (merchant_id, user_id, role, status, created_by)
VALUES ($1, $2, 'owner', 'active', $2)
ON CONFLICT (merchant_id, user_id) WHERE status = 'active' DO NOTHING
`, result.ID, input.CreatorUserID)
		return err
	})
	return result, err
}

func (m *MerchantModel) GetMerchantDetail(ctx context.Context, merchantID string) (MerchantDetail, error) {
	var detail MerchantDetail
	var categories JSONStringSlice
	var images JSONStringSlice
	var lastActive sql.NullTime

	err := m.db.QueryRowContext(ctx, `
SELECT
  m.id::text,
  m.name,
  m.merchant_type,
  cs.code,
  m.main_categories,
  m.verification_status,
  m.contact_name,
  m.contact_phone,
  COALESCE(m.contact_wechat, ''),
  COALESCE(m.address_text, ''),
  m.location,
  COALESCE(m.description, ''),
  COALESCE(m.logo_url, ''),
  m.images,
  m.last_active_at,
  COUNT(r.id) FILTER (WHERE r.status = 'published') AS published_count,
  COUNT(r.id) FILTER (WHERE r.status = 'dealt') AS dealt_count,
  (
    SELECT COUNT(*)
    FROM user_followed_merchants ufm
    WHERE ufm.merchant_id = m.id AND ufm.status = 'active'
  ) AS follower_count
FROM merchants m
JOIN city_stations cs ON cs.id = m.city_station_id
LEFT JOIN resources r ON r.merchant_id = m.id AND r.deleted_at IS NULL
WHERE m.id = $1 AND m.deleted_at IS NULL
GROUP BY m.id, cs.code
`, merchantID).Scan(
		&detail.ID,
		&detail.Name,
		&detail.MerchantType,
		&detail.CityCode,
		&categories,
		&detail.VerificationStatus,
		&detail.ContactName,
		&detail.PhoneMasked,
		&detail.WechatMasked,
		&detail.AddressText,
		&detail.Location,
		&detail.Description,
		&detail.LogoURL,
		&images,
		&lastActive,
		&detail.PublishedCount,
		&detail.DealtCount,
		&detail.FollowerCount,
	)
	if err != nil {
		return MerchantDetail{}, err
	}

	detail.MainCategories = []string(categories)
	detail.Images = []string(images)
	detail.PhoneMasked = maskContact(detail.PhoneMasked)
	detail.WechatMasked = maskWechat(detail.WechatMasked)
	if lastActive.Valid {
		detail.LastActiveAt = lastActive.Time.Format(time.RFC3339)
	}

	tags, err := m.listMerchantCreditTags(ctx, merchantID)
	if err != nil {
		return MerchantDetail{}, err
	}
	detail.CreditTags = tags
	return detail, nil
}

func (m *MerchantModel) GetMerchantContactPhone(ctx context.Context, merchantID string) (string, error) {
	var phone string
	err := m.db.QueryRowContext(ctx, `
SELECT contact_phone
FROM merchants
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1
`, merchantID).Scan(&phone)
	return strings.TrimSpace(phone), err
}

func (m *MerchantModel) UpdateMerchant(ctx context.Context, merchantID string, patch UpdateMerchantPatch) (string, error) {
	updatedAt := time.Now().UTC()
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var oldMerchantType string
		var oldVerificationStatus string
		var hasPendingVerification bool
		if err := tx.QueryRowContext(ctx, `
SELECT
  merchant_type,
  verification_status,
  EXISTS (
    SELECT 1
    FROM verifications
    WHERE merchant_id = merchants.id AND status = 'pending'
  ) AS has_pending_verification
FROM merchants
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE
`, merchantID).Scan(&oldMerchantType, &oldVerificationStatus, &hasPendingVerification); err != nil {
			return err
		}

		nextMerchantType := oldMerchantType
		if patch.MerchantType != "" {
			nextMerchantType = patch.MerchantType
		}
		nextVerificationStatus := oldVerificationStatus
		merchantTypeChanged := patch.MerchantType != "" && patch.MerchantType != oldMerchantType
		if merchantTypeChanged && (oldVerificationStatus == "verified" || hasPendingVerification) {
			nextVerificationStatus = "unverified"
		}

		result, err := tx.ExecContext(ctx, `
UPDATE merchants
SET
  main_categories = $2,
  merchant_type = $3,
  verification_status = $4,
  description = $5,
  logo_url = $6,
  images = $7,
  updated_at = $8,
  contact_name = COALESCE(NULLIF($9, ''), contact_name),
  contact_phone = COALESCE(NULLIF($10, ''), contact_phone),
  contact_wechat = COALESCE(NULLIF($11, ''), contact_wechat),
  address_text = COALESCE(NULLIF($12, ''), address_text),
  location = CASE WHEN $13 THEN $14 ELSE location END
WHERE id = $1 AND deleted_at IS NULL
`, merchantID, JSONStringSlice(patch.MainCategories), nextMerchantType, nextVerificationStatus, patch.Description, patch.LogoURL, JSONStringSlice(patch.Images), updatedAt, patch.ContactName, patch.ContactPhone, patch.ContactWechat, patch.AddressText, patch.LocationSet, patch.Location)
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

		if !merchantTypeChanged {
			return nil
		}

		// 主要身份影响认证含义；一旦从已认证或待审状态切换身份，需要撤销旧认证痕迹并要求重新认证。
		if _, err := tx.ExecContext(ctx, `
UPDATE verifications
SET status = 'revoked', review_note = COALESCE(NULLIF(review_note, ''), '主要身份已变更，请按新身份重新提交认证'), reviewed_at = $2, updated_at = $2
WHERE merchant_id = $1 AND status = 'pending'
`, merchantID, updatedAt); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE credit_records
SET revoked_at = $2
WHERE merchant_id = $1 AND source_type = 'verification' AND revoked_at IS NULL
`, merchantID, updatedAt); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO merchant_type_change_logs (
  merchant_id,
  old_merchant_type,
  new_merchant_type,
  old_verification_status,
  new_verification_status,
  changed_at
)
VALUES ($1, $2, $3, $4, $5, $6)
`, merchantID, oldMerchantType, patch.MerchantType, oldVerificationStatus, nextVerificationStatus, updatedAt)
		return err
	})
	if err != nil {
		return "", err
	}
	return updatedAt.Format(time.RFC3339), nil
}

func (m *MerchantModel) ListMerchants(ctx context.Context, filter ListMerchantsFilter) (ListMerchantsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	rows, err := m.db.QueryContext(ctx, `
SELECT
  m.id::text,
  m.name,
  m.merchant_type,
  m.verification_status,
  m.status,
  m.last_active_at,
  COUNT(*) OVER() AS total
FROM merchants m
JOIN city_stations cs ON cs.id = m.city_station_id
WHERE m.deleted_at IS NULL
	  AND ($1 = '' OR cs.code = $1)
	  AND ($2 = '' OR m.merchant_type = $2)
	  AND ($3 = '' OR m.status = $3)
	  AND ($4 = '' OR m.name ILIKE '%' || $4 || '%' OR m.contact_name ILIKE '%' || $4 || '%')
	ORDER BY m.created_at DESC
	LIMIT $5 OFFSET $6
	`, filter.CityCode, filter.MerchantType, filter.Status, filter.Keyword, pageSize, offset)
	if err != nil {
		return ListMerchantsResult{}, err
	}
	defer rows.Close()

	var items []MerchantListItem
	var total int64
	for rows.Next() {
		var item MerchantListItem
		var lastActive sql.NullTime
		if err := rows.Scan(&item.ID, &item.Name, &item.MerchantType, &item.VerificationStatus, &item.Status, &lastActive, &total); err != nil {
			return ListMerchantsResult{}, err
		}
		if lastActive.Valid {
			item.LastActiveAt = lastActive.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListMerchantsResult{}, err
	}

	return ListMerchantsResult{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (m *MerchantModel) listMerchantCreditTags(ctx context.Context, merchantID string) ([]CreditTag, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT tag_code, tag_label
FROM credit_records
WHERE merchant_id = $1
  AND visibility = 'public'
  AND revoked_at IS NULL
ORDER BY created_at DESC
`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []CreditTag
	for rows.Next() {
		var tag CreditTag
		if err := rows.Scan(&tag.Code, &tag.Label); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func normalizePage(page int64, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func maskContact(value string) string {
	if len(value) < 7 {
		return value
	}
	return value[:3] + "****" + value[len(value)-4:]
}

func maskWechat(value string) string {
	if len(value) <= 4 {
		return value
	}
	return value[:4] + "_****"
}
