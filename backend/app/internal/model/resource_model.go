package model

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

const (
	ResourceStatusDraft        = "draft"
	ResourceStatusPending      = "pending"
	ResourceStatusManualReview = "manual_review"
	ResourceStatusAuditRetry   = "audit_retry"
	ResourceStatusPublished    = "published"
	ResourceStatusRejected     = "rejected"
	ResourceStatusTakenDown    = "taken_down"
	ResourceStatusExpired      = "expired"

	ResourceDirectionSupply = "supply"
	ResourceDirectionDemand = "demand"

	EntitlementTypePublishQuota     = "publish_quota"
	EntitlementTypeRefreshQuota     = "refresh_quota"
	EntitlementTypeTopVoucher       = "top_voucher"
	EntitlementSourceProfileMonthly = "profile_monthly"

	ActionTypePublishResource = "publish_resource"
	ActionTypeRefreshResource = "refresh_resource"
	ActionTypeTopResource     = "top_resource"
)

var (
	ErrPublishQuotaInsufficient = errors.New("publish quota insufficient")
	ErrPublishDisabled          = errors.New("publish disabled")
)

const consumePublishQuotaSQL = `
UPDATE merchant_entitlements
SET used_amount = used_amount + 1,
    remaining_amount = remaining_amount - 1,
    updated_at = now()
WHERE id = (
  SELECT id
  FROM merchant_entitlements
  WHERE merchant_id = $1
    AND entitlement_type = 'publish_quota'
    AND status = 'active'
    AND remaining_amount > 0
    AND starts_at <= now()
    AND (expires_at IS NULL OR expires_at > now())
  ORDER BY expires_at NULLS LAST, created_at ASC
  LIMIT 1
)
  AND remaining_amount > 0
RETURNING id::text, remaining_amount + 1, remaining_amount
`

const consumeRefreshQuotaSQL = `
UPDATE merchant_entitlements
SET used_amount = used_amount + 1,
    remaining_amount = remaining_amount - 1,
    updated_at = now()
WHERE id = (
  SELECT id
  FROM merchant_entitlements
  WHERE merchant_id = $1
    AND entitlement_type = 'refresh_quota'
    AND status = 'active'
    AND remaining_amount > 0
    AND starts_at <= now()
    AND (expires_at IS NULL OR expires_at > now())
  ORDER BY expires_at NULLS LAST, created_at ASC
  LIMIT 1
)
  AND remaining_amount > 0
RETURNING id::text, remaining_amount + 1, remaining_amount
`

type ResourcePublishConfig struct {
	ID               string
	Version          int64
	TypeCode         string
	TypeName         string
	Direction        string
	FieldSchema      JSONMap
	RequiredFields   []string
	FilterFields     []string
	DisplayTemplate  JSONMap
	ReviewRules      JSONMap
	SortWeights      JSONMap
	MessageRules     JSONMap
	CommercialRules  JSONMap
	DefaultValidDays int64
}

func ResourceTypeSnapshotFromPublishConfig(config ResourcePublishConfig) JSONMap {
	version := config.Version
	if version <= 0 {
		// 测试桩和本地旧数据的兜底；真实配置由数据库约束保证版本从 1 开始。
		version = 1
	}
	return JSONMap{
		"version":          version,
		"typeCode":         config.TypeCode,
		"typeName":         config.TypeName,
		"direction":        config.Direction,
		"fieldSchema":      config.FieldSchema,
		"requiredFields":   append([]string(nil), config.RequiredFields...),
		"filterFields":     append([]string(nil), config.FilterFields...),
		"displayTemplate":  config.DisplayTemplate,
		"reviewRules":      config.ReviewRules,
		"sortWeights":      config.SortWeights,
		"messageRules":     config.MessageRules,
		"commercialRules":  config.CommercialRules,
		"defaultValidDays": config.DefaultValidDays,
	}
}

type CreateResourceInput struct {
	MerchantID                string
	CityCode                  string
	ResourceTypeConfigID      string
	ResourceTypeConfigVersion int64
	ResourceTypeSnapshot      JSONMap
	TypeCode                  string
	Direction                 string
	Status                    string
	Title                     string
	Category                  string
	District                  string
	PriceText                 string
	QuantityText              string
	CoverURL                  string
	Description               string
	Attributes                JSONMap
	Tags                      []string
	Images                    []string
	ContactName               string
	ContactPhone              string
	ContactWechat             string
	CreatedByUser             string
	CreatedByOperator         string
	ConsumePublishQuota       bool
}

type CreateResourceResult struct {
	ID     string
	Status string
}

type SubmitResourceResult struct {
	ID     string
	Status string
}

type ResourceAuditSnapshot struct {
	ID            string
	MerchantID    string
	TypeCode      string
	OpenID        string
	Title         string
	Category      string
	District      string
	PriceText     string
	QuantityText  string
	Description   string
	Attributes    JSONMap
	Tags          []string
	Images        []string
	ContactName   string
	ContactWechat string
}

type ResourceContentAuditTaskInput struct {
	TraceID   string
	AuditType string
	MediaURL  string
}

type ResourceContentAuditTaskResultInput struct {
	TraceID    string
	Status     string
	Suggest    string
	Label      int64
	Reason     string
	RawPayload JSONMap
}

type ResourceContentAuditTaskCompletion struct {
	ResourceID    string
	PendingCount  int64
	RejectedCount int64
	FailedCount   int64
}

type ResourceAuditRetryClaim struct {
	ResourceID   string
	RetryCount   int64
	ManualReview bool
}

type ResourceAuditDecisionInput struct {
	ResourceID string
	Action     string
	Decision   string
	Reason     string
	Labels     []string
	TraceIDs   []string
}

type ResourceMerchantBrief struct {
	ID        string
	Name      string
	VIPStatus string
}

type ResourceListItem struct {
	ID           string
	Direction    string
	TypeCode     string
	TypeName     string
	Title        string
	Category     string
	CoverURL     string
	District     string
	PriceText    string
	QuantityText string
	Tags         []string
	Merchant     ResourceMerchantBrief
	CreditTags   []string
	RefreshedAt  string
	DealtAt      string
}

// RelatedResourceSource 仅保存访问控制与同类推荐排序所需的源资源信息，避免公开接口提前读取私有详情字段。
type RelatedResourceSource struct {
	Status     string
	MerchantID string
	TypeCode   string
	Direction  string
	GroupCode  string
}

type ListResourcesFilter struct {
	CityCode   string
	MerchantID string
	GroupCode  string
	TypeCode   string
	Direction  string
	Keyword    string
	Category   string
	Tags       []string
	Status     string
	Page       int64
	PageSize   int64
}

type ListResourcesResult struct {
	Items    []ResourceListItem
	Page     int64
	PageSize int64
	Total    int64
}

type ResourceDetail struct {
	ID                string
	Status            string
	TypeCode          string
	Direction         string
	TypeName          string
	Title             string
	Category          string
	CityCode          string
	District          string
	Description       string
	PriceText         string
	QuantityText      string
	Attributes        JSONMap
	FieldSchema       JSONMap
	DisplayTemplate   JSONMap
	CommercialRules   JSONMap
	Tags              []string
	Images            []string
	MerchantID        string
	MerchantName      string
	MerchantVIPStatus string
	ContactName       string
	PhoneMasked       string
	WechatMasked      string
	PublishedAt       string
	ExpiresAt         string
	DealtAt           string
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

const listResourcesSQL = `
SELECT
  r.id::text,
  r.direction,
  r.type_code,
  r.resource_type_snapshot ->> 'typeName',
  r.title,
  r.category,
  COALESCE(NULLIF(r.cover_url, ''), r.images ->> 0, ''),
  COALESCE(r.district, ''),
  COALESCE(r.price_text, ''),
  COALESCE(r.quantity_text, ''),
  r.tags,
  m.id::text,
  m.name,
  CASE WHEN EXISTS (
    SELECT 1
    FROM merchant_vip_subscriptions mvs
    WHERE mvs.merchant_id = m.id
      AND mvs.status = 'active'
      AND mvs.starts_at <= now()
      AND mvs.expires_at > now()
  ) THEN 'active' ELSE 'none' END AS vip_status,
  COALESCE(r.refreshed_at, r.published_at, r.created_at),
  r.dealt_at,
  COUNT(*) OVER() AS total
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
JOIN city_stations cs ON cs.id = r.city_station_id
WHERE r.deleted_at IS NULL
  AND m.status = 'active'
  AND r.status = $1
  AND ($2 = '' OR cs.code = $2)
  AND (NULLIF($3, '')::bigint IS NULL OR r.merchant_id = NULLIF($3, '')::bigint)
  AND ($4 = '' OR r.resource_type_snapshot #>> '{displayTemplate,group,code}' = $4)
  AND ($5 = '' OR r.type_code = $5)
  AND ($6 = '' OR r.direction = $6)
  AND ($7 = '' OR r.category = $7)
  AND (
    $8 = ''
    OR r.title ILIKE '%' || $8 || '%'
    OR r.description ILIKE '%' || $8 || '%'
    OR r.category ILIKE '%' || $8 || '%'
    OR m.name ILIKE '%' || $8 || '%'
    OR r.attributes::text ILIKE '%' || $8 || '%'
  )
  AND (r.expires_at IS NULL OR r.expires_at > now())
  AND (r.dealt_at IS NULL OR r.dealt_at > now() - interval '7 days')
  AND (cardinality($11::text[]) = 0 OR r.tags ?& $11::text[])
ORDER BY
  CASE WHEN r.top_expires_at IS NOT NULL AND r.top_expires_at > now() THEN 1 ELSE 0 END DESC,
  COALESCE(r.refreshed_at, r.published_at, r.created_at) DESC
LIMIT $9 OFFSET $10
`

const getRelatedResourceSourceSQL = `
SELECT
  r.status,
  r.merchant_id::text,
  r.type_code,
  r.direction,
  COALESCE(r.resource_type_snapshot #>> '{displayTemplate,group,code}', '')
FROM resources r
WHERE r.id = $1
  AND r.deleted_at IS NULL
`

const listRelatedResourcesSQL = `
WITH source AS (
  SELECT
    r.id,
    r.type_code,
    r.direction,
    COALESCE(r.resource_type_snapshot #>> '{displayTemplate,group,code}', '') AS group_code
  FROM resources r
  WHERE r.id = $1
    AND r.deleted_at IS NULL
)
SELECT
  candidate.id::text,
  candidate.direction,
  candidate.type_code,
  candidate.resource_type_snapshot ->> 'typeName',
  candidate.title,
  candidate.category,
  COALESCE(NULLIF(candidate.cover_url, ''), candidate.images ->> 0, ''),
  COALESCE(candidate.district, ''),
  COALESCE(candidate.price_text, ''),
  COALESCE(candidate.quantity_text, ''),
  candidate.tags,
  merchant.id::text,
  merchant.name,
  CASE WHEN EXISTS (
    SELECT 1
    FROM merchant_vip_subscriptions mvs
    WHERE mvs.merchant_id = merchant.id
      AND mvs.status = 'active'
      AND mvs.starts_at <= now()
      AND mvs.expires_at > now()
  ) THEN 'active' ELSE 'none' END AS vip_status,
  COALESCE(candidate.refreshed_at, candidate.published_at, candidate.created_at),
  candidate.dealt_at
FROM source
JOIN resources candidate ON candidate.id <> source.id
JOIN merchants merchant ON merchant.id = candidate.merchant_id
WHERE candidate.deleted_at IS NULL
  AND candidate.status = 'published'
  AND merchant.status = 'active'
  AND (candidate.expires_at IS NULL OR candidate.expires_at > now())
  AND (candidate.dealt_at IS NULL OR candidate.dealt_at > now() - interval '7 days')
  AND (
    candidate.type_code = source.type_code
    OR (
      source.group_code <> ''
      AND candidate.resource_type_snapshot #>> '{displayTemplate,group,code}' = source.group_code
    )
    OR candidate.direction = source.direction
  )
ORDER BY
  CASE
    WHEN candidate.type_code = source.type_code THEN 1
    WHEN candidate.resource_type_snapshot #>> '{displayTemplate,group,code}' = source.group_code THEN 2
    WHEN candidate.direction = source.direction THEN 3
  END ASC,
  CASE WHEN candidate.top_expires_at IS NOT NULL AND candidate.top_expires_at > now() THEN 1 ELSE 0 END DESC,
  COALESCE(candidate.refreshed_at, candidate.published_at, candidate.created_at) DESC,
  candidate.id DESC
LIMIT $2
`

const reviewResourceSQL = `
UPDATE resources
SET
  status = $2,
  published_at = CASE WHEN $3 = 'approve' THEN $4::timestamptz ELSE published_at END,
  refreshed_at = CASE WHEN $3 = 'approve' THEN $4::timestamptz ELSE refreshed_at END,
  expires_at = CASE WHEN $3 = 'approve' THEN $4::timestamptz + make_interval(days => GREATEST((resources.resource_type_snapshot ->> 'defaultValidDays')::int, 1)) ELSE expires_at END,
  reject_reason = CASE WHEN $3 = 'reject' THEN $5 ELSE reject_reason END,
  take_down_reason = CASE WHEN $3 = 'take_down' THEN $5 ELSE take_down_reason END,
  taken_down_at = CASE WHEN $3 = 'take_down' THEN $4::timestamptz ELSE taken_down_at END,
  updated_at = $4::timestamptz
WHERE resources.id = $1
  AND (
    ($3 IN ('approve', 'reject') AND resources.status IN ('pending', 'manual_review'))
    OR ($3 = 'take_down' AND resources.status = 'published')
  )
  AND resources.deleted_at IS NULL
RETURNING resources.id::text, resources.merchant_id::text, resources.title, resources.status
`

const publishResourceAfterAuditSQL = `
UPDATE resources
SET
  status = 'published',
  published_at = $2::timestamptz,
  refreshed_at = $2::timestamptz,
  expires_at = $2::timestamptz + make_interval(days => GREATEST((resources.resource_type_snapshot ->> 'defaultValidDays')::int, 1)),
  reject_reason = NULL,
  updated_at = $2::timestamptz
WHERE resources.id = $1
  AND resources.status = 'pending'
RETURNING resources.id::text, resources.status
`

const publishedResourceDetailSQL = `
SELECT
  r.id::text,
  r.status,
  r.type_code,
  r.direction,
  r.resource_type_snapshot ->> 'typeName',
  r.title,
  r.category,
  cs.code,
  COALESCE(r.district, ''),
  r.description,
  COALESCE(r.price_text, ''),
  COALESCE(r.quantity_text, ''),
  r.attributes,
  r.resource_type_snapshot -> 'fieldSchema',
  r.resource_type_snapshot -> 'displayTemplate',
  r.resource_type_snapshot -> 'commercialRules',
  r.tags,
  r.images,
  m.id::text,
  m.name,
  CASE WHEN EXISTS (
    SELECT 1
    FROM merchant_vip_subscriptions mvs
    WHERE mvs.merchant_id = m.id
      AND mvs.status = 'active'
      AND mvs.starts_at <= now()
      AND mvs.expires_at > now()
  ) THEN 'active' ELSE 'none' END AS vip_status,
  r.contact_name,
  r.contact_phone,
  COALESCE(r.contact_wechat, ''),
  r.published_at,
  r.expires_at,
  r.dealt_at
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
JOIN city_stations cs ON cs.id = r.city_station_id
WHERE r.id = $1
  AND r.status = 'published'
  AND m.status = 'active'
  AND r.deleted_at IS NULL
  AND (r.expires_at IS NULL OR r.expires_at > now())
  AND (r.dealt_at IS NULL OR r.dealt_at > now() - interval '7 days')
`

const ownResourceDetailSQL = `
SELECT
  r.id::text,
  r.status,
  r.type_code,
  r.direction,
  r.resource_type_snapshot ->> 'typeName',
  r.title,
  r.category,
  cs.code,
  COALESCE(r.district, ''),
  r.description,
  COALESCE(r.price_text, ''),
  COALESCE(r.quantity_text, ''),
  r.attributes,
  r.resource_type_snapshot -> 'fieldSchema',
  r.resource_type_snapshot -> 'displayTemplate',
  r.resource_type_snapshot -> 'commercialRules',
  r.tags,
  r.images,
  m.id::text,
  m.name,
  r.contact_name,
  r.contact_phone,
  COALESCE(r.contact_wechat, ''),
  r.published_at,
  r.expires_at,
  r.dealt_at
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
JOIN city_stations cs ON cs.id = r.city_station_id
WHERE r.id = $1
  AND r.merchant_id = $2
  AND r.deleted_at IS NULL
`

type ListPendingResourcesFilter struct {
	CityCode string
	TypeCode string
	Status   string
	Page     int64
	PageSize int64
}

type PendingResourceItem struct {
	ID           string
	Status       string
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
	ID           string
	Direction    string
	TypeCode     string
	TypeName     string
	Title        string
	Category     string
	CoverURL     string
	Status       string
	RejectReason string
	PublishedAt  string
	ExpiresAt    string
	DealtAt      string
	Metrics      MyResourceMetrics
}

type EditableResourceDetail struct {
	ID            string
	MerchantID    string
	CityCode      string
	TypeCode      string
	Direction     string
	Status        string
	Title         string
	Category      string
	District      string
	PriceText     string
	QuantityText  string
	Description   string
	Attributes    JSONMap
	Tags          []string
	Images        []string
	ContactName   string
	ContactPhone  string
	ContactWechat string
	RejectReason  string
}

type ListMyResourcesFilter struct {
	MerchantID string
	Status     string
	Direction  string
	Page       int64
	PageSize   int64
}

type ListMyResourcesResult struct {
	Items    []MyResourceItem
	Page     int64
	PageSize int64
	Total    int64
}

const listMyResourcesSQL = `
SELECT
  r.id::text,
  r.direction,
  r.type_code,
  r.resource_type_snapshot ->> 'typeName',
  r.title,
  r.category,
  COALESCE(NULLIF(r.cover_url, ''), r.images ->> 0, ''),
  r.status,
  COALESCE(r.reject_reason, ''),
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
    OR ($2 = 'needs_action' AND r.status IN ('draft', 'pending', 'manual_review', 'audit_retry', 'rejected'))
    OR ($2 = 'showing' AND r.status = 'published' AND r.dealt_at IS NULL AND (r.expires_at IS NULL OR r.expires_at > now()))
    OR ($2 = 'ended' AND (r.status IN ('expired', 'taken_down') OR r.dealt_at IS NOT NULL OR (r.expires_at IS NOT NULL AND r.expires_at <= now())))
    OR ($2 = 'expiring_soon' AND r.status = 'published' AND r.expires_at IS NOT NULL AND r.expires_at <= now() + interval '3 days' AND r.expires_at > now())
    OR ($2 = 'expired' AND ((r.expires_at IS NOT NULL AND r.expires_at <= now()) OR r.status = 'expired'))
    OR ($2 = 'dealt' AND r.dealt_at IS NOT NULL)
    OR r.status = $2
  )
  AND ($3 = '' OR r.direction = $3)
GROUP BY r.id
ORDER BY r.updated_at DESC
LIMIT $4 OFFSET $5
`

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

type DeleteTakenDownResourceResult struct {
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
	var filterFields JSONStringSlice
	err := m.db.QueryRowContext(ctx, `
SELECT
  rtc.id::text,
  rtc.version,
  rtc.type_code,
  rtc.type_name,
  rtc.direction,
  rtc.field_schema,
  rtc.required_fields,
  rtc.filter_fields,
  rtc.display_template,
  rtc.review_rules,
  rtc.sort_weights,
  rtc.message_rules,
  rtc.commercial_rules,
  rtc.default_valid_days
FROM resource_type_configs rtc
JOIN city_stations cs ON cs.id = rtc.city_station_id
WHERE cs.code = $1
  AND cs.status = 'active'
  AND rtc.type_code = $2
  AND rtc.status = 'active'
`, cityCode, typeCode).Scan(
		&config.ID,
		&config.Version,
		&config.TypeCode,
		&config.TypeName,
		&config.Direction,
		&config.FieldSchema,
		&requiredFields,
		&filterFields,
		&config.DisplayTemplate,
		&config.ReviewRules,
		&config.SortWeights,
		&config.MessageRules,
		&config.CommercialRules,
		&config.DefaultValidDays,
	)
	config.RequiredFields = []string(requiredFields)
	config.FilterFields = []string(filterFields)
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
	if input.ConsumePublishQuota {
		var result CreateResourceResult
		err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
			var err error
			result, err = insertResource(ctx, tx, input)
			if err != nil {
				return err
			}
			// 资源 id 生成后再扣减发布额度，便于权益使用记录精确关联到本次发布；事务会保证额度不足时资源不会落库。
			return consumePublishQuotaTx(ctx, tx, input.MerchantID, result.ID)
		})
		return result, err
	}
	return insertResource(ctx, m.db, input)
}

type resourceQueryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func insertResource(ctx context.Context, queryer resourceQueryRower, input CreateResourceInput) (CreateResourceResult, error) {
	var result CreateResourceResult
	err := queryer.QueryRowContext(ctx, `
INSERT INTO resources (
  merchant_id,
  city_station_id,
  resource_type_config_id,
  resource_type_config_version,
  resource_type_snapshot,
  type_code,
  direction,
  status,
  title,
  category,
  district,
  price_text,
  quantity_text,
  cover_url,
  description,
  attributes,
  tags,
  images,
	  contact_name,
	  contact_phone,
	  contact_wechat,
	  created_by_user_id,
	  created_by_operator_id
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
  $18,
  $19,
  $20,
  $21,
  NULLIF($22, '')::bigint,
  NULLIF($23, '')::bigint
	FROM city_stations cs
	WHERE cs.code = $2 AND cs.status = 'active'
	RETURNING id::text, status
`,
		input.MerchantID,
		input.CityCode,
		input.ResourceTypeConfigID,
		input.ResourceTypeConfigVersion,
		input.ResourceTypeSnapshot,
		input.TypeCode,
		input.Direction,
		input.Status,
		input.Title,
		input.Category,
		input.District,
		input.PriceText,
		input.QuantityText,
		input.CoverURL,
		input.Description,
		input.Attributes,
		JSONStringSlice(input.Tags),
		JSONStringSlice(input.Images),
		input.ContactName,
		input.ContactPhone,
		input.ContactWechat,
		input.CreatedByUser,
		input.CreatedByOperator,
	).Scan(&result.ID, &result.Status)
	return result, err
}

const submitResourceForReviewLockSQL = `
SELECT r.merchant_id::text, r.resource_type_snapshot -> 'commercialRules'
FROM resources r
WHERE r.id = $1
  AND r.status = 'draft'
  AND r.deleted_at IS NULL
FOR UPDATE
`

const submitResourceForReviewUpdateSQL = `
UPDATE resources
SET status = 'pending', updated_at = now()
WHERE id = $1
  AND status = 'draft'
  AND deleted_at IS NULL
RETURNING id::text, status
`

func (m *ResourceModel) SubmitResourceForReview(ctx context.Context, resourceID string) (SubmitResourceResult, error) {
	var result SubmitResourceResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var commercialRules JSONMap
		// 草稿提交只进入自动内容审核等待态；发布额度在自动审核通过并真正上架时扣减，避免图片异步审核失败后误扣额度。
		var merchantID string
		if err := tx.QueryRowContext(ctx, submitResourceForReviewLockSQL, resourceID).Scan(&merchantID, &commercialRules); err != nil {
			return err
		}
		switch PublishModeFromCommercialRules(commercialRules) {
		case ResourcePublishModeDisabled:
			return ErrPublishDisabled
		}
		return tx.QueryRowContext(ctx, submitResourceForReviewUpdateSQL, resourceID).Scan(&result.ID, &result.Status)
	})
	return result, err
}

func (m *ResourceModel) GetResourceAuditSnapshot(ctx context.Context, resourceID string) (ResourceAuditSnapshot, error) {
	var snapshot ResourceAuditSnapshot
	var tags JSONStringSlice
	var images JSONStringSlice
	err := m.db.QueryRowContext(ctx, `
SELECT
  r.id::text,
  r.merchant_id::text,
  r.type_code,
  COALESCE(u.wechat_openid, ''),
  r.title,
  r.category,
  COALESCE(r.district, ''),
  COALESCE(r.price_text, ''),
  COALESCE(r.quantity_text, ''),
  COALESCE(r.description, ''),
  r.attributes,
  r.tags,
  r.images,
  COALESCE(r.contact_name, ''),
  COALESCE(r.contact_wechat, '')
FROM resources r
	LEFT JOIN users u ON u.id = r.created_by_user_id AND u.deleted_at IS NULL
WHERE r.id = $1
  AND r.status IN ('draft', 'pending', 'audit_retry')
  AND r.deleted_at IS NULL
`, resourceID).Scan(
		&snapshot.ID,
		&snapshot.MerchantID,
		&snapshot.TypeCode,
		&snapshot.OpenID,
		&snapshot.Title,
		&snapshot.Category,
		&snapshot.District,
		&snapshot.PriceText,
		&snapshot.QuantityText,
		&snapshot.Description,
		&snapshot.Attributes,
		&tags,
		&images,
		&snapshot.ContactName,
		&snapshot.ContactWechat,
	)
	snapshot.Tags = []string(tags)
	snapshot.Images = []string(images)
	return snapshot, err
}

func (m *ResourceModel) MarkResourceAuditRetry(ctx context.Context, resourceID string, reason string) (int64, error) {
	var retryCount int64
	err := m.db.QueryRowContext(ctx, `
UPDATE resources
SET
  status = 'audit_retry',
  audit_retry_at = now() + make_interval(secs => LEAST((30 * power(2, audit_retry_count))::int, 1800)),
  audit_last_error = NULLIF($2, ''),
  updated_at = now()
WHERE id = $1
  AND status IN ('pending', 'audit_retry')
  AND deleted_at IS NULL
RETURNING audit_retry_count
`, resourceID, reason).Scan(&retryCount)
	return retryCount, err
}

func (m *ResourceModel) RecordResourceAuditDecision(ctx context.Context, input ResourceAuditDecisionInput) error {
	_, err := m.db.ExecContext(ctx, `
INSERT INTO resource_content_audit_runs (
  resource_id, attempt_no, audit_action, decision, reason, labels, trace_ids
)
SELECT
  id,
  audit_retry_count,
  $2,
  $3,
  NULLIF($4, ''),
  $5,
  $6
FROM resources
WHERE id = $1
`, input.ResourceID, input.Action, input.Decision, input.Reason, JSONStringSlice(input.Labels), JSONStringSlice(input.TraceIDs))
	return err
}

func (m *ResourceModel) MarkResourceManualReview(ctx context.Context, resourceID string, reason string) error {
	_, err := m.db.ExecContext(ctx, `
UPDATE resources
SET
  status = 'manual_review',
  audit_retry_at = NULL,
  audit_last_error = NULLIF($2, ''),
  updated_at = now()
WHERE id = $1
  AND status IN ('pending', 'audit_retry')
  AND deleted_at IS NULL
`, resourceID, reason)
	return err
}

func (m *ResourceModel) MarkStaleContentAuditTasksForRetry(ctx context.Context, staleBefore time.Time) (int64, error) {
	result, err := m.db.ExecContext(ctx, `
WITH stale_resources AS (
  SELECT DISTINCT rcat.resource_id
  FROM resource_content_audit_tasks rcat
  JOIN resources r ON r.id = rcat.resource_id
  WHERE rcat.status = 'pending'
    AND rcat.created_at < $1
    AND r.status = 'pending'
    AND r.deleted_at IS NULL
),
failed_tasks AS (
  UPDATE resource_content_audit_tasks
  SET status = 'failed', reason = '图片审核回调超时，系统将自动重试', completed_at = now(), updated_at = now()
  WHERE resource_id IN (SELECT resource_id FROM stale_resources)
    AND status = 'pending'
)
UPDATE resources
SET
  status = 'audit_retry',
  audit_retry_at = now(),
  audit_last_error = '图片审核回调超时',
  updated_at = now()
WHERE id IN (SELECT resource_id FROM stale_resources)
`, staleBefore)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *ResourceModel) ClaimDueResourceAuditRetries(ctx context.Context, batchSize int64, maxRetries int64) ([]ResourceAuditRetryClaim, error) {
	if batchSize <= 0 {
		batchSize = 20
	}
	if maxRetries <= 0 {
		maxRetries = 5
	}
	rows, err := m.db.QueryContext(ctx, `
WITH candidates AS (
  SELECT id
  FROM resources
  WHERE status = 'audit_retry'
    AND audit_retry_at <= now()
    AND deleted_at IS NULL
  ORDER BY audit_retry_at ASC, updated_at ASC
  LIMIT $1
  FOR UPDATE SKIP LOCKED
),
claimed AS (
  UPDATE resources r
  SET
    status = CASE WHEN r.audit_retry_count >= $2 THEN 'manual_review' ELSE 'pending' END,
    audit_retry_count = CASE WHEN r.audit_retry_count >= $2 THEN r.audit_retry_count ELSE r.audit_retry_count + 1 END,
    audit_retry_at = NULL,
    updated_at = now()
  WHERE r.id IN (SELECT id FROM candidates)
  RETURNING r.id::text, r.audit_retry_count, r.status
)
SELECT id, audit_retry_count, status = 'manual_review'
FROM claimed
ORDER BY id
`, batchSize, maxRetries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var claims []ResourceAuditRetryClaim
	for rows.Next() {
		var claim ResourceAuditRetryClaim
		if err := rows.Scan(&claim.ResourceID, &claim.RetryCount, &claim.ManualReview); err != nil {
			return nil, err
		}
		claims = append(claims, claim)
	}
	return claims, rows.Err()
}

func (m *ResourceModel) CreateResourceContentAuditTasks(ctx context.Context, resourceID string, tasks []ResourceContentAuditTaskInput) error {
	return WithTx(ctx, m.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM resource_content_audit_tasks WHERE resource_id = $1`, resourceID); err != nil {
			return err
		}
		for _, task := range tasks {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO resource_content_audit_tasks (resource_id, trace_id, audit_type, media_url, status)
VALUES ($1, $2, $3, $4, 'pending')
`, resourceID, task.TraceID, task.AuditType, task.MediaURL); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *ResourceModel) CompleteResourceContentAuditTask(ctx context.Context, input ResourceContentAuditTaskResultInput) (ResourceContentAuditTaskCompletion, error) {
	var completion ResourceContentAuditTaskCompletion
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `
UPDATE resource_content_audit_tasks
SET
  status = $2,
  suggest = $3,
  label = NULLIF($4, 0),
  reason = $5,
  raw_payload = $6,
  completed_at = now(),
  updated_at = now()
WHERE trace_id = $1
RETURNING resource_id::text
`, input.TraceID, input.Status, input.Suggest, input.Label, input.Reason, input.RawPayload).Scan(&completion.ResourceID); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, `
SELECT
  COUNT(*) FILTER (WHERE status = 'pending'),
  COUNT(*) FILTER (WHERE status = 'rejected'),
  COUNT(*) FILTER (WHERE status = 'failed')
FROM resource_content_audit_tasks
WHERE resource_id = $1
`, completion.ResourceID).Scan(&completion.PendingCount, &completion.RejectedCount, &completion.FailedCount)
	})
	return completion, err
}

func (m *ResourceModel) PublishResourceAfterAudit(ctx context.Context, resourceID string) (ReviewResourceResult, error) {
	now := time.Now().UTC()
	var result ReviewResourceResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var merchantID string
		var title string
		var commercialRules JSONMap
		if err := tx.QueryRowContext(ctx, `
SELECT r.merchant_id::text, r.title, r.resource_type_snapshot -> 'commercialRules'
FROM resources r
WHERE r.id = $1
  AND r.status = 'pending'
  AND r.deleted_at IS NULL
FOR UPDATE
`, resourceID).Scan(&merchantID, &title, &commercialRules); err != nil {
			return err
		}
		switch PublishModeFromCommercialRules(commercialRules) {
		case ResourcePublishModeDisabled:
			return ErrPublishDisabled
		case ResourcePublishModeFree:
			// 免费发布分类通过自动审核后直接上架，不消耗发布额度。
		default:
			if err := consumePublishQuotaTx(ctx, tx, merchantID, resourceID); err != nil {
				return err
			}
		}
		if err := tx.QueryRowContext(ctx, publishResourceAfterAuditSQL, resourceID, now).Scan(&result.ID, &result.Status); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'resource_review', 'resource_auto_approve', $2, '资源已自动发布', $3, $4, 'unread')
`, "merchant:"+merchantID, resourceID, title+" 已通过内容审核并公开展示", MerchantMyResourcesTargetURL(merchantID))
		return err
	})
	return result, err
}

func (m *ResourceModel) RejectResourceAfterAudit(ctx context.Context, resourceID string, reason string) (ReviewResourceResult, error) {
	var result ReviewResourceResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var merchantID string
		var title string
		if err := tx.QueryRowContext(ctx, `
UPDATE resources
SET status = 'rejected', reject_reason = $2, updated_at = now()
WHERE id = $1
  AND status = 'pending'
  AND deleted_at IS NULL
RETURNING id::text, merchant_id::text, title, status
`, resourceID, reason).Scan(&result.ID, &merchantID, &title, &result.Status); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'resource_review', 'resource_auto_reject', $2, '资源审核未通过', $3, $4, 'unread')
`, "merchant:"+merchantID, resourceID, title+" "+reason, MerchantMyResourcesTargetURL(merchantID))
		return err
	})
	return result, err
}

func consumePublishQuotaTx(ctx context.Context, tx *sql.Tx, merchantID string, resourceID string) error {
	if err := ensureProfileMonthlyEntitlementsTx(ctx, tx, merchantID); err != nil {
		return err
	}
	var usage entitlementUsageInput
	err := tx.QueryRowContext(ctx, consumePublishQuotaSQL, merchantID).Scan(&usage.EntitlementID, &usage.BeforeRemainingAmount, &usage.AfterRemainingAmount)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPublishQuotaInsufficient
	}
	if err != nil {
		return err
	}
	usage.MerchantID = merchantID
	usage.EntitlementType = EntitlementTypePublishQuota
	usage.ActionType = ActionTypePublishResource
	usage.ResourceID = resourceID
	usage.Amount = 1
	usage.Snapshot = JSONMap{"resourceId": resourceID}
	return recordEntitlementUsageTx(ctx, tx, usage)
}

func ensureProfileMonthlyEntitlementsTx(ctx context.Context, tx *sql.Tx, merchantID string) error {
	var profileStatus string
	var hasActiveVIP bool
	if err := tx.QueryRowContext(ctx, `
SELECT
  COALESCE(profile_status, 'incomplete') AS profile_status,
  EXISTS (
    SELECT 1
    FROM merchant_vip_subscriptions
    WHERE merchant_id = merchants.id
      AND status = 'active'
      AND starts_at <= now()
      AND expires_at > now()
  ) AS has_active_vip
FROM merchants
WHERE id = $1
  AND deleted_at IS NULL
  AND status = 'active'
FOR UPDATE
`, merchantID).Scan(&profileStatus, &hasActiveVIP); err != nil {
		return err
	}
	if hasActiveVIP {
		return nil
	}

	publishQuota, refreshQuota := profileMonthlyBenefitsForStatus(profileStatus)
	periodStart, periodEnd := currentMonthlyEntitlementPeriod(time.Now().UTC())
	for _, item := range []struct {
		entitlementType string
		totalAmount     int64
	}{
		{entitlementType: EntitlementTypePublishQuota, totalAmount: publishQuota},
		{entitlementType: EntitlementTypeRefreshQuota, totalAmount: refreshQuota},
	} {
		if err := upsertProfileMonthlyEntitlementTx(ctx, tx, merchantID, item.entitlementType, item.totalAmount, periodStart, periodEnd); err != nil {
			return err
		}
	}
	return nil
}

func profileMonthlyBenefitsForStatus(profileStatus string) (int64, int64) {
	// 冷启动基础额度与资料完整度解耦：所有商家每月都有 3 次免费发布，完善名片不再是领取门槛。
	return 3, 0
}

func currentMonthlyEntitlementPeriod(now time.Time) (time.Time, time.Time) {
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return periodStart, periodStart.AddDate(0, 1, 0)
}

func upsertProfileMonthlyEntitlementTx(ctx context.Context, tx *sql.Tx, merchantID string, entitlementType string, totalAmount int64, periodStart time.Time, periodEnd time.Time) error {
	if totalAmount <= 0 {
		return nil
	}
	var entitlementID string
	err := tx.QueryRowContext(ctx, `
UPDATE merchant_entitlements
SET total_amount = CASE WHEN total_amount < $6 THEN $6 ELSE total_amount END,
    remaining_amount = CASE WHEN total_amount < $6 THEN remaining_amount + ($6 - total_amount) ELSE remaining_amount END,
    updated_at = CASE WHEN total_amount < $6 THEN now() ELSE updated_at END
WHERE merchant_id = $1
  AND entitlement_type = $2
  AND source_type = $3
  AND starts_at = $4
  AND expires_at = $5
  AND status = 'active'
RETURNING id::text
`, merchantID, entitlementType, EntitlementSourceProfileMonthly, periodStart, periodEnd, totalAmount).Scan(&entitlementID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return tx.QueryRowContext(ctx, `
INSERT INTO merchant_entitlements (merchant_id, entitlement_type, source_type, total_amount, remaining_amount, starts_at, expires_at, status)
VALUES ($1, $2, $3, $4, $4, $5, $6, 'active')
RETURNING id::text
`, merchantID, entitlementType, EntitlementSourceProfileMonthly, totalAmount, periodStart, periodEnd).Scan(&entitlementID)
}

func (m *ResourceModel) UpdateResourceDraft(ctx context.Context, resourceID string, input CreateResourceInput) (CreateResourceResult, error) {
	var result CreateResourceResult
	err := m.db.QueryRowContext(ctx, `
UPDATE resources
SET
  city_station_id = cs.id,
  resource_type_config_id = NULLIF($4, '')::bigint,
  resource_type_config_version = $5,
  resource_type_snapshot = $6,
  type_code = $7,
  direction = $8,
  status = 'draft',
  title = $9,
  category = $10,
  district = $11,
  price_text = $12,
  quantity_text = $13,
  cover_url = NULLIF($14, ''),
  description = $15,
  attributes = $16,
  tags = $17,
  images = $18,
  contact_name = $19,
  contact_phone = $20,
  contact_wechat = $21,
  reject_reason = NULL,
  updated_at = now()
FROM city_stations cs
WHERE resources.id = $1
  AND resources.merchant_id = $2
  AND cs.code = $3
  AND cs.status = 'active'
  AND resources.status IN ('draft', 'rejected')
  AND resources.deleted_at IS NULL
RETURNING resources.id::text, resources.status
`,
		resourceID,
		input.MerchantID,
		input.CityCode,
		input.ResourceTypeConfigID,
		input.ResourceTypeConfigVersion,
		input.ResourceTypeSnapshot,
		input.TypeCode,
		input.Direction,
		input.Title,
		input.Category,
		input.District,
		input.PriceText,
		input.QuantityText,
		input.CoverURL,
		input.Description,
		input.Attributes,
		JSONStringSlice(input.Tags),
		JSONStringSlice(input.Images),
		input.ContactName,
		input.ContactPhone,
		input.ContactWechat,
	).Scan(&result.ID, &result.Status)
	return result, err
}

func (m *ResourceModel) GetResourceMerchantID(ctx context.Context, resourceID string) (string, error) {
	var merchantID string
	err := m.db.QueryRowContext(ctx, `
SELECT merchant_id::text
FROM resources
WHERE id = $1
  AND deleted_at IS NULL
`, resourceID).Scan(&merchantID)
	return merchantID, err
}

func (m *ResourceModel) GetEditableResourceDetail(ctx context.Context, merchantID string, resourceID string) (EditableResourceDetail, error) {
	var detail EditableResourceDetail
	var attributes JSONMap
	var tags JSONStringSlice
	var images JSONStringSlice
	err := m.db.QueryRowContext(ctx, `
SELECT
  r.id::text,
  r.merchant_id::text,
  cs.code,
  r.type_code,
  r.direction,
  r.status,
  r.title,
  r.category,
  COALESCE(r.district, ''),
  COALESCE(r.price_text, ''),
  COALESCE(r.quantity_text, ''),
  r.description,
  r.attributes,
  r.tags,
  r.images,
  r.contact_name,
  r.contact_phone,
  COALESCE(r.contact_wechat, ''),
  COALESCE(r.reject_reason, '')
FROM resources r
JOIN city_stations cs ON cs.id = r.city_station_id
WHERE r.id = $1
  AND r.merchant_id = $2
  AND r.status IN ('draft', 'rejected')
  AND r.deleted_at IS NULL
`, resourceID, merchantID).Scan(
		&detail.ID,
		&detail.MerchantID,
		&detail.CityCode,
		&detail.TypeCode,
		&detail.Direction,
		&detail.Status,
		&detail.Title,
		&detail.Category,
		&detail.District,
		&detail.PriceText,
		&detail.QuantityText,
		&detail.Description,
		&attributes,
		&tags,
		&images,
		&detail.ContactName,
		&detail.ContactPhone,
		&detail.ContactWechat,
		&detail.RejectReason,
	)
	if err != nil {
		return EditableResourceDetail{}, err
	}
	detail.Attributes = attributes
	detail.Tags = []string(tags)
	detail.Images = []string(images)
	return detail, nil
}

func (m *ResourceModel) ListResources(ctx context.Context, filter ListResourcesFilter) (ListResourcesResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	rows, err := m.db.QueryContext(ctx, listResourcesSQL, filter.Status, filter.CityCode, filter.MerchantID, filter.GroupCode, filter.TypeCode, filter.Direction, filter.Category, filter.Keyword, pageSize, offset, pq.Array(filter.Tags))
	if err != nil {
		return ListResourcesResult{}, err
	}
	defer rows.Close()

	var items []ResourceListItem
	var total int64
	for rows.Next() {
		var item ResourceListItem
		var tags JSONStringSlice
		var refreshedAt time.Time
		var dealtAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.Direction,
			&item.TypeCode,
			&item.TypeName,
			&item.Title,
			&item.Category,
			&item.CoverURL,
			&item.District,
			&item.PriceText,
			&item.QuantityText,
			&tags,
			&item.Merchant.ID,
			&item.Merchant.Name,
			&item.Merchant.VIPStatus,
			&refreshedAt,
			&dealtAt,
			&total,
		); err != nil {
			return ListResourcesResult{}, err
		}
		item.Tags = []string(tags)
		item.RefreshedAt = refreshedAt.Format(time.RFC3339)
		if dealtAt.Valid {
			item.DealtAt = dealtAt.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListResourcesResult{}, err
	}
	return ListResourcesResult{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (m *ResourceModel) GetRelatedResourceSource(ctx context.Context, resourceID string) (RelatedResourceSource, error) {
	var source RelatedResourceSource
	err := m.db.QueryRowContext(ctx, getRelatedResourceSourceSQL, resourceID).Scan(
		&source.Status,
		&source.MerchantID,
		&source.TypeCode,
		&source.Direction,
		&source.GroupCode,
	)
	return source, err
}

func (m *ResourceModel) ListRelatedResources(ctx context.Context, resourceID string, limit int64) ([]ResourceListItem, error) {
	rows, err := m.db.QueryContext(ctx, listRelatedResourcesSQL, resourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ResourceListItem, 0)
	for rows.Next() {
		var item ResourceListItem
		var tags JSONStringSlice
		var refreshedAt time.Time
		var dealtAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.Direction,
			&item.TypeCode,
			&item.TypeName,
			&item.Title,
			&item.Category,
			&item.CoverURL,
			&item.District,
			&item.PriceText,
			&item.QuantityText,
			&tags,
			&item.Merchant.ID,
			&item.Merchant.Name,
			&item.Merchant.VIPStatus,
			&refreshedAt,
			&dealtAt,
		); err != nil {
			return nil, err
		}
		item.Tags = []string(tags)
		item.RefreshedAt = refreshedAt.Format(time.RFC3339)
		if dealtAt.Valid {
			item.DealtAt = dealtAt.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *ResourceModel) GetPublishedResourceDetail(ctx context.Context, resourceID string) (ResourceDetail, error) {
	var detail ResourceDetail
	var tags JSONStringSlice
	var images JSONStringSlice
	var publishedAt sql.NullTime
	var expiresAt sql.NullTime
	var dealtAt sql.NullTime
	err := m.db.QueryRowContext(ctx, publishedResourceDetailSQL, resourceID).Scan(
		&detail.ID,
		&detail.Status,
		&detail.TypeCode,
		&detail.Direction,
		&detail.TypeName,
		&detail.Title,
		&detail.Category,
		&detail.CityCode,
		&detail.District,
		&detail.Description,
		&detail.PriceText,
		&detail.QuantityText,
		&detail.Attributes,
		&detail.FieldSchema,
		&detail.DisplayTemplate,
		&detail.CommercialRules,
		&tags,
		&images,
		&detail.MerchantID,
		&detail.MerchantName,
		&detail.MerchantVIPStatus,
		&detail.ContactName,
		&detail.PhoneMasked,
		&detail.WechatMasked,
		&publishedAt,
		&expiresAt,
		&dealtAt,
	)
	if err != nil {
		return ResourceDetail{}, err
	}
	detail.Tags = []string(tags)
	detail.Images = []string(images)
	detail.PhoneMasked = maskContact(detail.PhoneMasked)
	detail.WechatMasked = maskWechat(detail.WechatMasked)
	if publishedAt.Valid {
		detail.PublishedAt = publishedAt.Time.Format(time.RFC3339)
	}
	if expiresAt.Valid {
		detail.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
	}
	if dealtAt.Valid {
		detail.DealtAt = dealtAt.Time.Format(time.RFC3339)
	}
	return detail, nil
}

func (m *ResourceModel) GetOwnResourceDetail(ctx context.Context, merchantID string, resourceID string) (ResourceDetail, error) {
	var detail ResourceDetail
	var tags JSONStringSlice
	var images JSONStringSlice
	var publishedAt sql.NullTime
	var expiresAt sql.NullTime
	var dealtAt sql.NullTime
	err := m.db.QueryRowContext(ctx, ownResourceDetailSQL, resourceID, merchantID).Scan(
		&detail.ID,
		&detail.Status,
		&detail.TypeCode,
		&detail.Direction,
		&detail.TypeName,
		&detail.Title,
		&detail.Category,
		&detail.CityCode,
		&detail.District,
		&detail.Description,
		&detail.PriceText,
		&detail.QuantityText,
		&detail.Attributes,
		&detail.FieldSchema,
		&detail.DisplayTemplate,
		&detail.CommercialRules,
		&tags,
		&images,
		&detail.MerchantID,
		&detail.MerchantName,
		&detail.ContactName,
		&detail.PhoneMasked,
		&detail.WechatMasked,
		&publishedAt,
		&expiresAt,
		&dealtAt,
	)
	if err != nil {
		return ResourceDetail{}, err
	}
	detail.Tags = []string(tags)
	detail.Images = []string(images)
	if publishedAt.Valid {
		detail.PublishedAt = publishedAt.Time.Format(time.RFC3339)
	}
	if expiresAt.Valid {
		detail.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
	}
	if dealtAt.Valid {
		detail.DealtAt = dealtAt.Time.Format(time.RFC3339)
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
		if input.Action == "approve" {
			var merchantID string
			var commercialRules JSONMap
			if err := tx.QueryRowContext(ctx, `
SELECT merchant_id::text, resource_type_snapshot -> 'commercialRules'
FROM resources
WHERE id = $1
  AND status IN ('pending', 'manual_review')
  AND deleted_at IS NULL
FOR UPDATE
`, resourceID).Scan(&merchantID, &commercialRules); err != nil {
				return err
			}
			switch PublishModeFromCommercialRules(commercialRules) {
			case ResourcePublishModeDisabled:
				return ErrPublishDisabled
			case ResourcePublishModeFree:
				// 免费发布分类经人工复核通过后不消耗额度。
			default:
				if err := consumePublishQuotaTx(ctx, tx, merchantID, resourceID); err != nil {
					return err
				}
			}
		}
		var merchantID string
		var title string
		row := tx.QueryRowContext(ctx, reviewResourceSQL, resourceID, status, input.Action, now, input.Reason)
		if err := row.Scan(&result.ID, &merchantID, &title, &result.Status); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO resource_review_records (resource_id, reviewer_id, action, reason, snapshot)
VALUES ($1, NULLIF($2, '')::bigint, $3, $4, '{}'::jsonb)
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
  r.id::text,
  r.status,
  r.title,
  r.type_code,
  m.name,
  r.created_at,
  COUNT(*) OVER() AS total
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
JOIN city_stations cs ON cs.id = r.city_station_id
WHERE r.deleted_at IS NULL
  AND ($1 = '' OR r.status = $1)
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
		if err := rows.Scan(&item.ID, &item.Status, &item.Title, &item.TypeCode, &item.MerchantName, &createdAt, &total); err != nil {
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
	rows, err := m.db.QueryContext(ctx, listMyResourcesSQL, filter.MerchantID, filter.Status, filter.Direction, pageSize, offset)
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
			&item.ID, &item.Direction, &item.TypeCode, &item.TypeName, &item.Title, &item.Category, &item.CoverURL, &item.Status,
			&item.RejectReason, &publishedAt, &expiresAt, &dealtAt,
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
SELECT id::text, merchant_id::text, status, expires_at, dealt_at
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
		if err := ensureProfileMonthlyEntitlementsTx(ctx, tx, merchantID); err != nil {
			return err
		}
		var entitlementID string
		var beforeRemaining int64
		var remaining int64
		if err := tx.QueryRowContext(ctx, consumeRefreshQuotaSQL, merchantID).Scan(&entitlementID, &beforeRemaining, &remaining); err != nil {
			return err
		}
		var refreshedAt time.Time
		if err := tx.QueryRowContext(ctx, `
UPDATE resources
SET refreshed_at = now(), updated_at = now()
WHERE id = $1
  AND merchant_id = $2
  AND status = 'published'
  AND dealt_at IS NULL
  AND deleted_at IS NULL
RETURNING id::text, refreshed_at
`, resourceID, merchantID).Scan(&result.ID, &refreshedAt); err != nil {
			return err
		}
		if err := recordEntitlementUsageTx(ctx, tx, entitlementUsageInput{
			EntitlementID:         entitlementID,
			MerchantID:            merchantID,
			EntitlementType:       EntitlementTypeRefreshQuota,
			ActionType:            ActionTypeRefreshResource,
			Amount:                1,
			ResourceID:            result.ID,
			BeforeRemainingAmount: beforeRemaining,
			AfterRemainingAmount:  remaining,
			Snapshot:              JSONMap{"refreshedAt": refreshedAt.Format(time.RFC3339)},
		}); err != nil {
			return err
		}
		result.RefreshedAt = refreshedAt.Format(time.RFC3339)
		result.RemainingRefreshQuota = remaining
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
RETURNING id::text, title, status
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
RETURNING id::text, title, status
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

func (m *ResourceModel) DeleteTakenDownResource(ctx context.Context, merchantID string, resourceID string) (DeleteTakenDownResourceResult, error) {
	var result DeleteTakenDownResourceResult
	err := m.db.QueryRowContext(ctx, `
UPDATE resources
SET deleted_at = now(), updated_at = now()
WHERE id = $1
  AND merchant_id = $2
  AND status = 'taken_down'
  AND deleted_at IS NULL
RETURNING id::text, status
`, resourceID, merchantID).Scan(&result.ID, &result.Status)
	return result, err
}

func (m *ResourceModel) RepostSimilar(ctx context.Context, merchantID string, resourceID string) (RepostSimilarResult, error) {
	var result RepostSimilarResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO resources (
  merchant_id,
  city_station_id,
  resource_type_config_id,
  resource_type_config_version,
  resource_type_snapshot,
  type_code,
  direction,
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
	  created_by_user_id,
	  created_by_operator_id
	)
SELECT
  source.merchant_id,
  source.city_station_id,
  rtc.id,
  rtc.version,
  jsonb_build_object(
    'version', rtc.version,
    'typeCode', rtc.type_code,
    'typeName', rtc.type_name,
    'direction', rtc.direction,
    'fieldSchema', rtc.field_schema,
    'requiredFields', rtc.required_fields,
    'filterFields', rtc.filter_fields,
    'displayTemplate', rtc.display_template,
    'reviewRules', rtc.review_rules,
    'sortWeights', rtc.sort_weights,
    'messageRules', rtc.message_rules,
    'commercialRules', rtc.commercial_rules,
    'defaultValidDays', rtc.default_valid_days
  ),
  rtc.type_code,
  rtc.direction,
  'draft',
  source.title,
  source.category,
  source.district,
  source.price_text,
  source.quantity_text,
  source.description,
  source.attributes,
  source.tags,
  source.images,
  source.contact_name,
  source.contact_phone,
  source.contact_wechat,
  source.created_by_user_id,
  source.created_by_operator_id
FROM resources source
JOIN resource_type_configs rtc
  ON rtc.id = source.resource_type_config_id
  AND rtc.status = 'active'
WHERE source.id = $1
  AND source.merchant_id = $2
  AND source.deleted_at IS NULL
RETURNING id::text, status
`, resourceID, merchantID).Scan(&result.ID, &result.Status)
	return result, err
}
