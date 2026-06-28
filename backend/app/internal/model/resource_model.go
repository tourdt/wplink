package model

import (
	"context"
	"database/sql"
	"time"
)

const (
	ResourceStatusDraft     = "draft"
	ResourceStatusPending   = "pending"
	ResourceStatusPublished = "published"
	ResourceStatusRejected  = "rejected"
	ResourceStatusTakenDown = "taken_down"
	ResourceStatusExpired   = "expired"
)

type ResourcePublishConfig struct {
	ID               string
	TypeCode         string
	RequiredFields   []string
	DefaultValidDays int64
}

type CreateResourceInput struct {
	MerchantID           string
	CityCode             string
	ResourceTypeConfigID string
	TypeCode             string
	Status               string
	Title                string
	Category             string
	District             string
	PriceText            string
	QuantityText         string
	Description          string
	Attributes           JSONMap
	Tags                 []string
	Images               []string
	ContactName          string
	ContactPhone         string
	ContactWechat        string
	CreatedByUser        string
}

type CreateResourceResult struct {
	ID     string
	Status string
}

type SubmitResourceResult struct {
	ID     string
	Status string
}

type ResourceMerchantBrief struct {
	ID                 string
	Name               string
	VerificationStatus string
}

type ResourceListItem struct {
	ID           string
	TypeCode     string
	Title        string
	Category     string
	District     string
	PriceText    string
	QuantityText string
	Merchant     ResourceMerchantBrief
	CreditTags   []string
	RefreshedAt  string
}

type ListResourcesFilter struct {
	CityCode     string
	MerchantID   string
	TypeCode     string
	Keyword      string
	Category     string
	VerifiedOnly bool
	Status       string
	Page         int64
	PageSize     int64
}

type ListResourcesResult struct {
	Items    []ResourceListItem
	Page     int64
	PageSize int64
	Total    int64
}

type ResourceDetail struct {
	ID                         string
	Status                     string
	TypeCode                   string
	Title                      string
	Category                   string
	Description                string
	PriceText                  string
	QuantityText               string
	Attributes                 JSONMap
	MerchantID                 string
	MerchantName               string
	MerchantVerificationStatus string
	ContactName                string
	PhoneMasked                string
	WechatMasked               string
	PublishedAt                string
	ExpiresAt                  string
}

type ReviewResourceInput struct {
	Action     string
	Reason     string
	ReviewerID string
}

type ReviewResourceResult struct {
	ID     string
	Status string
}

type ListPendingResourcesFilter struct {
	CityCode string
	TypeCode string
	Status   string
	Page     int64
	PageSize int64
}

type PendingResourceItem struct {
	ID           string
	Title        string
	TypeCode     string
	MerchantName string
	CreatedAt    string
}

type ListPendingResourcesResult struct {
	Items    []PendingResourceItem
	Page     int64
	PageSize int64
	Total    int64
}

type MyResourceMetrics struct {
	ExposureCount   int64
	DetailViewCount int64
	PhoneClickCount int64
	WechatCopyCount int64
}

type MyResourceItem struct {
	ID          string
	TypeCode    string
	Title       string
	Category    string
	Status      string
	PublishedAt string
	ExpiresAt   string
	DealtAt     string
	Metrics     MyResourceMetrics
}

type ListMyResourcesFilter struct {
	MerchantID string
	Status     string
	Page       int64
	PageSize   int64
}

type ListMyResourcesResult struct {
	Items    []MyResourceItem
	Page     int64
	PageSize int64
	Total    int64
}

type ResourceOwnershipStatus struct {
	ID         string
	MerchantID string
	Status     string
	IsExpired  bool
	IsDealt    bool
}

type RefreshResourceResult struct {
	ID                    string
	RefreshedAt           string
	RemainingRefreshQuota int64
}

type MarkDealtInput struct {
	MerchantID              string
	ResourceID              string
	IsDealt                 bool
	IsReal                  bool
	ResponseTimely          bool
	WillingToCooperateAgain bool
	Note                    string
}

type DealFeedbackResult struct {
	ID     string
	Status string
}

type TakeDownOwnResourceInput struct {
	MerchantID string
	ResourceID string
	Reason     string
}

type TakeDownOwnResourceResult struct {
	ID     string
	Status string
}

type RepostSimilarResult struct {
	ID     string
	Status string
}

type ResourceModel struct {
	db *sql.DB
}

func NewResourceModel(db *sql.DB) *ResourceModel {
	return &ResourceModel{db: db}
}

func (m *ResourceModel) GetResourcePublishConfig(ctx context.Context, cityCode string, typeCode string) (ResourcePublishConfig, error) {
	var config ResourcePublishConfig
	var requiredFields JSONStringSlice
	err := m.db.QueryRowContext(ctx, `
SELECT rtc.id, rtc.type_code, rtc.required_fields, rtc.default_valid_days
FROM resource_type_configs rtc
JOIN city_stations cs ON cs.id = rtc.city_station_id
WHERE cs.code = $1
  AND cs.status = 'active'
  AND rtc.type_code = $2
  AND rtc.status = 'active'
`, cityCode, typeCode).Scan(&config.ID, &config.TypeCode, &requiredFields, &config.DefaultValidDays)
	config.RequiredFields = []string(requiredFields)
	return config, err
}

func (m *ResourceModel) GetMerchantPublishStatus(ctx context.Context, merchantID string) (string, error) {
	var status string
	err := m.db.QueryRowContext(ctx, `
SELECT status
FROM merchants
WHERE id = $1 AND deleted_at IS NULL
`, merchantID).Scan(&status)
	return status, err
}

func (m *ResourceModel) CreateResource(ctx context.Context, input CreateResourceInput) (CreateResourceResult, error) {
	var result CreateResourceResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO resources (
  merchant_id,
  city_station_id,
  resource_type_config_id,
  type_code,
  status,
  title,
  category,
  district,
  price_text,
  quantity_text,
  description,
  attributes,
  tags,
  images,
  contact_name,
  contact_phone,
  contact_wechat,
  created_by
)
SELECT
  $1,
  cs.id,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11,
  $12,
  $13,
  $14,
  $15,
  $16,
  $17,
  NULLIF($18, '')::uuid
FROM city_stations cs
WHERE cs.code = $2 AND cs.status = 'active'
RETURNING id, status
`,
		input.MerchantID,
		input.CityCode,
		input.ResourceTypeConfigID,
		input.TypeCode,
		input.Status,
		input.Title,
		input.Category,
		input.District,
		input.PriceText,
		input.QuantityText,
		input.Description,
		input.Attributes,
		JSONStringSlice(input.Tags),
		JSONStringSlice(input.Images),
		input.ContactName,
		input.ContactPhone,
		input.ContactWechat,
		input.CreatedByUser,
	).Scan(&result.ID, &result.Status)
	return result, err
}

func (m *ResourceModel) SubmitResourceForReview(ctx context.Context, resourceID string) (SubmitResourceResult, error) {
	var result SubmitResourceResult
	err := m.db.QueryRowContext(ctx, `
UPDATE resources
SET status = 'pending', updated_at = now()
WHERE id = $1
  AND status IN ('draft', 'rejected')
  AND deleted_at IS NULL
RETURNING id, status
`, resourceID).Scan(&result.ID, &result.Status)
	return result, err
}

func (m *ResourceModel) ListResources(ctx context.Context, filter ListResourcesFilter) (ListResourcesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	rows, err := m.db.QueryContext(ctx, `
SELECT
  r.id,
  r.type_code,
  r.title,
  r.category,
  COALESCE(r.district, ''),
  COALESCE(r.price_text, ''),
  COALESCE(r.quantity_text, ''),
  m.id,
  m.name,
  m.verification_status,
  COALESCE(r.refreshed_at, r.published_at, r.created_at),
  COUNT(*) OVER() AS total
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
JOIN city_stations cs ON cs.id = r.city_station_id
WHERE r.deleted_at IS NULL
  AND r.status = $1
  AND ($2 = '' OR cs.code = $2)
  AND ($3 = '' OR r.merchant_id = $3)
  AND ($4 = '' OR r.type_code = $4)
  AND ($5 = '' OR r.category = $5)
  AND (
    $6 = ''
    OR r.title ILIKE '%' || $6 || '%'
    OR r.description ILIKE '%' || $6 || '%'
    OR r.category ILIKE '%' || $6 || '%'
    OR m.name ILIKE '%' || $6 || '%'
    OR r.attributes::text ILIKE '%' || $6 || '%'
  )
  AND ($7 = false OR r.is_verified = true OR m.verification_status = 'verified')
  AND (r.expires_at IS NULL OR r.expires_at > now())
ORDER BY COALESCE(r.refreshed_at, r.published_at, r.created_at) DESC
LIMIT $8 OFFSET $9
`, filter.Status, filter.CityCode, filter.MerchantID, filter.TypeCode, filter.Category, filter.Keyword, filter.VerifiedOnly, pageSize, offset)
	if err != nil {
		return ListResourcesResult{}, err
	}
	defer rows.Close()

	var items []ResourceListItem
	var total int64
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
			&total,
		); err != nil {
			return ListResourcesResult{}, err
		}
		item.RefreshedAt = refreshedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListResourcesResult{}, err
	}
	return ListResourcesResult{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (m *ResourceModel) GetPublishedResourceDetail(ctx context.Context, resourceID string) (ResourceDetail, error) {
	var detail ResourceDetail
	var publishedAt sql.NullTime
	var expiresAt sql.NullTime
	err := m.db.QueryRowContext(ctx, `
SELECT
  r.id,
  r.status,
  r.type_code,
  r.title,
  r.category,
  r.description,
  COALESCE(r.price_text, ''),
  COALESCE(r.quantity_text, ''),
  r.attributes,
  m.id,
  m.name,
  m.verification_status,
  r.contact_name,
  r.contact_phone,
  COALESCE(r.contact_wechat, ''),
  r.published_at,
  r.expires_at
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
WHERE r.id = $1
  AND r.status = 'published'
  AND r.deleted_at IS NULL
  AND (r.expires_at IS NULL OR r.expires_at > now())
`, resourceID).Scan(
		&detail.ID,
		&detail.Status,
		&detail.TypeCode,
		&detail.Title,
		&detail.Category,
		&detail.Description,
		&detail.PriceText,
		&detail.QuantityText,
		&detail.Attributes,
		&detail.MerchantID,
		&detail.MerchantName,
		&detail.MerchantVerificationStatus,
		&detail.ContactName,
		&detail.PhoneMasked,
		&detail.WechatMasked,
		&publishedAt,
		&expiresAt,
	)
	if err != nil {
		return ResourceDetail{}, err
	}
	detail.PhoneMasked = maskContact(detail.PhoneMasked)
	detail.WechatMasked = maskWechat(detail.WechatMasked)
	if publishedAt.Valid {
		detail.PublishedAt = publishedAt.Time.Format(time.RFC3339)
	}
	if expiresAt.Valid {
		detail.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
	}
	return detail, nil
}

func (m *ResourceModel) ReviewResource(ctx context.Context, resourceID string, input ReviewResourceInput) (ReviewResourceResult, error) {
	now := time.Now().UTC()
	status := ResourceStatusPublished
	if input.Action == "reject" {
		status = ResourceStatusRejected
	}
	if input.Action == "take_down" {
		status = ResourceStatusTakenDown
	}

	var result ReviewResourceResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var merchantID string
		var title string
		row := tx.QueryRowContext(ctx, `
UPDATE resources
SET
  status = $2,
  published_at = CASE WHEN $3 = 'approve' THEN $4 ELSE published_at END,
  refreshed_at = CASE WHEN $3 = 'approve' THEN $4 ELSE refreshed_at END,
  expires_at = CASE WHEN $3 = 'approve' THEN $4 + interval '7 days' ELSE expires_at END,
  reject_reason = CASE WHEN $3 = 'reject' THEN $5 ELSE reject_reason END,
  take_down_reason = CASE WHEN $3 = 'take_down' THEN $5 ELSE take_down_reason END,
  taken_down_at = CASE WHEN $3 = 'take_down' THEN $4 ELSE taken_down_at END,
  updated_at = $4
WHERE id = $1
RETURNING id, merchant_id, title, status
`, resourceID, status, input.Action, now, input.Reason)
		if err := row.Scan(&result.ID, &merchantID, &title, &result.Status); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO resource_review_records (resource_id, reviewer_id, action, reason, snapshot)
VALUES ($1, NULLIF($2, '')::uuid, $3, $4, '{}'::jsonb)
`, resourceID, input.ReviewerID, input.Action, input.Reason)
		if err != nil {
			return err
		}
		messageTitle := "资源审核通过"
		messageContent := title + " 已审核通过并公开展示"
		if input.Action == "reject" {
			messageTitle = "资源审核驳回"
			messageContent = title + " 审核未通过，请修改后重新提交"
		}
		if input.Action == "take_down" {
			messageTitle = "资源已下架"
			messageContent = title + " 已由运营下架"
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'resource_review', $2, $3, $4, $5, $6, 'unread')
`, "merchant:"+merchantID, "resource_"+input.Action, resourceID, messageTitle, messageContent, MerchantMyResourcesTargetURL(merchantID))
		if err != nil {
			return err
		}
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   input.ReviewerID,
			OperatorRole: "platform_operator",
			Action:       "resource_" + input.Action,
			ObjectType:   "resource",
			ObjectID:     resourceID,
			AfterSnapshot: JSONMap{
				"status": result.Status,
				"reason": input.Reason,
			},
		})
	})
	return result, err
}

func (m *ResourceModel) ListPendingResources(ctx context.Context, filter ListPendingResourcesFilter) (ListPendingResourcesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	rows, err := m.db.QueryContext(ctx, `
SELECT
  r.id,
  r.title,
  r.type_code,
  m.name,
  r.created_at,
  COUNT(*) OVER() AS total
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
JOIN city_stations cs ON cs.id = r.city_station_id
WHERE r.deleted_at IS NULL
  AND r.status = $1
  AND ($2 = '' OR cs.code = $2)
  AND ($3 = '' OR r.type_code = $3)
ORDER BY r.created_at DESC
LIMIT $4 OFFSET $5
`, filter.Status, filter.CityCode, filter.TypeCode, pageSize, offset)
	if err != nil {
		return ListPendingResourcesResult{}, err
	}
	defer rows.Close()

	var items []PendingResourceItem
	var total int64
	for rows.Next() {
		var item PendingResourceItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.TypeCode, &item.MerchantName, &createdAt, &total); err != nil {
			return ListPendingResourcesResult{}, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListPendingResourcesResult{}, err
	}
	return ListPendingResourcesResult{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (m *ResourceModel) ListMyResources(ctx context.Context, filter ListMyResourcesFilter) (ListMyResourcesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT
  r.id,
  r.type_code,
  r.title,
  r.category,
  r.status,
  r.published_at,
  r.expires_at,
  r.dealt_at,
  COALESCE(SUM(rmd.exposure_count), 0),
  COALESCE(SUM(rmd.detail_view_count), 0),
  COALESCE(SUM(rmd.phone_click_count), 0),
  COALESCE(SUM(rmd.wechat_copy_count), 0),
  COUNT(*) OVER() AS total
FROM resources r
LEFT JOIN resource_metrics_daily rmd ON rmd.resource_id = r.id
WHERE r.merchant_id = $1
  AND r.deleted_at IS NULL
  AND (
    $2 = ''
    OR ($2 = 'expiring_soon' AND r.status = 'published' AND r.expires_at IS NOT NULL AND r.expires_at <= now() + interval '3 days' AND r.expires_at > now())
    OR ($2 = 'expired' AND ((r.expires_at IS NOT NULL AND r.expires_at <= now()) OR r.status = 'expired'))
    OR ($2 = 'dealt' AND r.dealt_at IS NOT NULL)
    OR r.status = $2
  )
GROUP BY r.id
ORDER BY r.updated_at DESC
LIMIT $3 OFFSET $4
`, filter.MerchantID, filter.Status, pageSize, offset)
	if err != nil {
		return ListMyResourcesResult{}, err
	}
	defer rows.Close()
	result := ListMyResourcesResult{Page: page, PageSize: pageSize}
	for rows.Next() {
		var item MyResourceItem
		var publishedAt sql.NullTime
		var expiresAt sql.NullTime
		var dealtAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.TypeCode, &item.Title, &item.Category, &item.Status,
			&publishedAt, &expiresAt, &dealtAt,
			&item.Metrics.ExposureCount, &item.Metrics.DetailViewCount,
			&item.Metrics.PhoneClickCount, &item.Metrics.WechatCopyCount,
			&result.Total,
		); err != nil {
			return ListMyResourcesResult{}, err
		}
		if publishedAt.Valid {
			item.PublishedAt = publishedAt.Time.Format(time.RFC3339)
		}
		if expiresAt.Valid {
			item.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
		}
		if dealtAt.Valid {
			item.DealtAt = dealtAt.Time.Format(time.RFC3339)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListMyResourcesResult{}, err
	}
	return result, nil
}

func (m *ResourceModel) GetResourceOwnershipStatus(ctx context.Context, merchantID string, resourceID string) (ResourceOwnershipStatus, error) {
	var result ResourceOwnershipStatus
	var expiresAt sql.NullTime
	var dealtAt sql.NullTime
	err := m.db.QueryRowContext(ctx, `
SELECT id, merchant_id, status, expires_at, dealt_at
FROM resources
WHERE id = $1
  AND merchant_id = $2
  AND deleted_at IS NULL
`, resourceID, merchantID).Scan(&result.ID, &result.MerchantID, &result.Status, &expiresAt, &dealtAt)
	if err != nil {
		return ResourceOwnershipStatus{}, err
	}
	result.IsExpired = expiresAt.Valid && !expiresAt.Time.After(time.Now().UTC())
	result.IsDealt = dealtAt.Valid
	return result, nil
}

func (m *ResourceModel) RefreshResource(ctx context.Context, merchantID string, resourceID string) (RefreshResourceResult, error) {
	var result RefreshResourceResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var entitlementID string
		var remaining int64
		if err := tx.QueryRowContext(ctx, `
UPDATE merchant_entitlements
SET used_amount = used_amount + 1, remaining_amount = remaining_amount - 1, updated_at = now()
WHERE id = (
  SELECT id
  FROM merchant_entitlements
  WHERE merchant_id = $1
    AND entitlement_type = 'refresh_quota'
    AND status = 'active'
    AND remaining_amount > 0
    AND (expires_at IS NULL OR expires_at > now())
  ORDER BY expires_at NULLS LAST, created_at ASC
  LIMIT 1
)
RETURNING id, remaining_amount
`, merchantID).Scan(&entitlementID, &remaining); err != nil {
			return err
		}
		var refreshedAt time.Time
		if err := tx.QueryRowContext(ctx, `
UPDATE resources
SET refreshed_at = now(), updated_at = now()
WHERE id = $1
  AND merchant_id = $2
  AND status = 'published'
  AND deleted_at IS NULL
RETURNING id, refreshed_at
`, resourceID, merchantID).Scan(&result.ID, &refreshedAt); err != nil {
			return err
		}
		result.RefreshedAt = refreshedAt.Format(time.RFC3339)
		result.RemainingRefreshQuota = remaining
		_ = entitlementID
		return nil
	})
	return result, err
}

func (m *ResourceModel) MarkDealt(ctx context.Context, input MarkDealtInput) (DealFeedbackResult, error) {
	var result DealFeedbackResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var title string
		if err := tx.QueryRowContext(ctx, `
UPDATE resources
SET dealt_at = CASE WHEN $3 THEN now() ELSE NULL END, updated_at = now()
WHERE id = $1
  AND merchant_id = $2
  AND status = 'published'
  AND deleted_at IS NULL
RETURNING id, title, status
`, input.ResourceID, input.MerchantID, input.IsDealt).Scan(&result.ID, &title, &result.Status); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO resource_metrics_daily (resource_id, merchant_id, stat_date, deal_feedback_count)
SELECT id, merchant_id, CURRENT_DATE, 1
FROM resources
WHERE id = $1
ON CONFLICT (resource_id, stat_date)
DO UPDATE SET
  deal_feedback_count = resource_metrics_daily.deal_feedback_count + 1,
  updated_at = now()
`, input.ResourceID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'effect_feedback', 'deal_feedback', $2, '成交反馈已记录', $3, $4, 'unread')
`, "merchant:"+input.MerchantID, input.ResourceID, title+" 的成交反馈已记录", MerchantMyResourcesTargetURL(input.MerchantID))
		return err
	})
	return result, err
}

func (m *ResourceModel) TakeDownOwnResource(ctx context.Context, input TakeDownOwnResourceInput) (TakeDownOwnResourceResult, error) {
	var result TakeDownOwnResourceResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var title string
		if err := tx.QueryRowContext(ctx, `
UPDATE resources
SET status = 'taken_down', take_down_reason = $3, taken_down_at = now(), updated_at = now()
WHERE id = $1
  AND merchant_id = $2
  AND status = 'published'
  AND deleted_at IS NULL
RETURNING id, title, status
`, input.ResourceID, input.MerchantID, input.Reason).Scan(&result.ID, &title, &result.Status); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'resource_lifecycle', 'resource_taken_down', $2, '资源已下架', $3, $4, 'unread')
`, "merchant:"+input.MerchantID, input.ResourceID, title+" 已下架", MerchantMyResourcesTargetURL(input.MerchantID))
		return err
	})
	return result, err
}

func (m *ResourceModel) RepostSimilar(ctx context.Context, merchantID string, resourceID string) (RepostSimilarResult, error) {
	var result RepostSimilarResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO resources (
  merchant_id,
  city_station_id,
  resource_type_config_id,
  type_code,
  status,
  title,
  category,
  district,
  price_text,
  quantity_text,
  description,
  attributes,
  tags,
  images,
  contact_name,
  contact_phone,
  contact_wechat,
  created_by
)
SELECT
  merchant_id,
  city_station_id,
  resource_type_config_id,
  type_code,
  'draft',
  title,
  category,
  district,
  price_text,
  quantity_text,
  description,
  attributes,
  tags,
  images,
  contact_name,
  contact_phone,
  contact_wechat,
  created_by
FROM resources
WHERE id = $1
  AND merchant_id = $2
  AND deleted_at IS NULL
RETURNING id, status
`, resourceID, merchantID).Scan(&result.ID, &result.Status)
	return result, err
}
