package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	VIPStatusNone    = "none"
	VIPStatusActive  = "active"
	VIPStatusExpired = "expired"

	VIPPublishPolicyQuota = "quota"

	VIPProductTypeVIPPlan   = "vip_plan"
	VIPProductTypeQuotaPack = "quota_pack"

	vipBenefitValidityDays = 30
	quotaPackValidityDays  = 180
)

var (
	ErrTopServiceResourceRequired = errors.New("top service resource required")
	ErrTopServiceResourceInvalid  = errors.New("top service resource invalid")
	ErrTopServiceProductInvalid   = errors.New("top service product invalid")
)

type VIPBenefitSnapshot struct {
	PublishPolicy      string
	PublishQuota       int64
	RefreshQuota       int64
	TopVoucherCount    int64
	TopDurationHours   int64
	HomepageImageLimit int64
}

type VIPPlan struct {
	Code              string
	Name              string
	DurationMonths    int64
	StandardPriceCent int64
	SalePriceCent     int64
	SaleLabel         string
	PromotionCode     string
	Benefits          VIPBenefitSnapshot
}

type QuotaPack struct {
	Code              string
	Name              string
	Description       string
	StandardPriceCent int64
	SalePriceCent     int64
	SaleLabel         string
	Benefits          VIPBenefitSnapshot
}

type AdminVIPPlanConfig struct {
	Code              string
	Name              string
	DurationMonths    int64
	StandardPriceCent int64
	Status            string
	DisplayOrder      int64
	Benefits          VIPBenefitSnapshot
	UpdatedAt         string
}

type SaveAdminVIPPlanInput struct {
	Code              string
	Name              string
	DurationMonths    int64
	StandardPriceCent int64
	Status            string
	DisplayOrder      int64
	Benefits          VIPBenefitSnapshot
	OperatorID        string
}

type AdminQuotaPackConfig struct {
	Code              string
	Name              string
	Description       string
	StandardPriceCent int64
	SalePriceCent     int64
	SaleLabel         string
	Status            string
	DisplayOrder      int64
	Benefits          VIPBenefitSnapshot
	UpdatedAt         string
}

type SaveAdminQuotaPackInput struct {
	Code              string
	Name              string
	Description       string
	StandardPriceCent int64
	SalePriceCent     int64
	SaleLabel         string
	Status            string
	DisplayOrder      int64
	Benefits          VIPBenefitSnapshot
	OperatorID        string
}

type AdminVIPPromotionConfig struct {
	Code          string
	PlanCode      string
	PlanName      string
	PromotionType string
	SalePriceCent int64
	StartsAt      string
	EndsAt        string
	QuotaLimit    int64
	UsedCount     int64
	Status        string
	UpdatedAt     string
}

type SaveAdminVIPPromotionInput struct {
	Code          string
	PlanCode      string
	PromotionType string
	SalePriceCent int64
	StartsAt      string
	EndsAt        string
	QuotaLimit    int64
	Status        string
	OperatorID    string
}

type AdminVIPConfigSaveResult struct {
	Code      string
	UpdatedAt string
}

type VIPPromotion struct {
	ID            string
	Code          string
	PromotionType string
	SalePriceCent int64
}

type MerchantVIPSummary struct {
	MerchantID            string
	Status                string
	PlanCode              string
	PlanName              string
	StartsAt              string
	ExpiresAt             string
	PublishQuotaRemaining int64
	RefreshQuotaRemaining int64
	TopVoucherCount       int64
}

type CreateVIPOrderInput struct {
	MerchantID string
	UserID     string
	PlanCode   string
}

type CreateQuotaPackOrderInput struct {
	MerchantID string
	UserID     string
	PackCode   string
	ResourceID string
}

type VIPOrder struct {
	ID                string
	MerchantID        string
	UserID            string
	ProductType       string
	ProductCode       string
	ProductName       string
	PlanCode          string
	PlanName          string
	OutTradeNo        string
	Status            string
	Currency          string
	StandardPriceCent int64
	ActualPriceCent   int64
	PromotionCode     string
	Benefits          VIPBenefitSnapshot
}

type GetVIPPaymentContextInput struct {
	MerchantID string
	OrderID    string
	UserID     string
}

type VIPPaymentContext struct {
	OrderID     string
	MerchantID  string
	UserID      string
	OpenID      string
	Status      string
	OutTradeNo  string
	AmountTotal int64
	Currency    string
	PlanName    string
	ProductType string
	ProductName string
}

type CreateVIPPaymentOrderInput struct {
	OrderID    string
	MerchantID string
	UserID     string
}

type VIPPaymentOrder struct {
	ID          string
	OutTradeNo  string
	AmountTotal int64
	Currency    string
	Status      string
	PlanName    string
	ProductType string
	ProductName string
}

type MarkVIPOrderPaidInput struct {
	OutTradeNo    string
	TransactionID string
	AmountTotal   int64
	SuccessTime   string
	NotifyPayload JSONMap
}

type VIPPaymentResult struct {
	OrderID        string
	MerchantID     string
	Status         string
	SubscriptionID string
}

type VIPModel struct {
	db *sql.DB
}

func NewVIPModel(db *sql.DB) *VIPModel {
	return &VIPModel{db: db}
}

func VIPBenefitSnapshotFromJSON(values JSONMap) VIPBenefitSnapshot {
	snapshot := VIPBenefitSnapshot{
		PublishPolicy: VIPPublishPolicyQuota,
	}
	if values == nil {
		return snapshot
	}
	if policy := strings.TrimSpace(stringFromJSON(values["publishPolicy"])); policy != "" {
		snapshot.PublishPolicy = policy
	}
	snapshot.PublishQuota = int64FromJSON(values["publishQuota"])
	snapshot.RefreshQuota = int64FromJSON(values["refreshQuota"])
	snapshot.TopVoucherCount = int64FromJSON(values["topVoucherCount"])
	snapshot.TopDurationHours = int64FromJSON(values["topDurationHours"])
	snapshot.HomepageImageLimit = int64FromJSON(values["homepageImageLimit"])
	return snapshot
}

func (s VIPBenefitSnapshot) ToJSONMap() JSONMap {
	policy := strings.TrimSpace(s.PublishPolicy)
	if policy == "" {
		policy = VIPPublishPolicyQuota
	}
	return JSONMap{
		"publishPolicy":      policy,
		"publishQuota":       s.PublishQuota,
		"refreshQuota":       s.RefreshQuota,
		"topVoucherCount":    s.TopVoucherCount,
		"topDurationHours":   s.TopDurationHours,
		"homepageImageLimit": s.HomepageImageLimit,
	}
}

func (m *VIPModel) ListVIPPlans(ctx context.Context) ([]VIPPlan, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  p.code,
  p.name,
  p.duration_months,
  p.standard_price_cent,
  COALESCE(v.benefits, '{}'::jsonb) AS benefits,
  promo.code,
  promo.promotion_type,
  promo.sale_price_cent
FROM vip_plans p
JOIN LATERAL (
  SELECT benefits
  FROM vip_plan_versions
  WHERE plan_id = p.id
    AND status = 'active'
    AND starts_at <= now()
    AND (ends_at IS NULL OR ends_at > now())
  ORDER BY version DESC
  LIMIT 1
) v ON true
LEFT JOIN LATERAL (
  SELECT code, promotion_type, sale_price_cent
  FROM vip_promotions
  WHERE plan_id = p.id
    AND status = 'active'
    AND starts_at <= now()
    AND (ends_at IS NULL OR ends_at > now())
    AND (quota_limit IS NULL OR used_count < quota_limit)
  ORDER BY sale_price_cent ASC, created_at DESC
  LIMIT 1
) promo ON true
WHERE p.status = 'active'
ORDER BY p.display_order ASC, p.duration_months ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []VIPPlan
	for rows.Next() {
		var item VIPPlan
		var benefits JSONMap
		var promotionCode sql.NullString
		var promotionType sql.NullString
		var salePrice sql.NullInt64
		if err := rows.Scan(
			&item.Code,
			&item.Name,
			&item.DurationMonths,
			&item.StandardPriceCent,
			&benefits,
			&promotionCode,
			&promotionType,
			&salePrice,
		); err != nil {
			return nil, err
		}
		item.Benefits = VIPBenefitSnapshotFromJSON(benefits)
		if promotionCode.Valid {
			item.PromotionCode = promotionCode.String
			item.SaleLabel = vipPromotionSaleLabel(promotionType.String)
		}
		if salePrice.Valid {
			item.SalePriceCent = salePrice.Int64
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *VIPModel) ListQuotaPacks(ctx context.Context) ([]QuotaPack, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  code,
  name,
  description,
  standard_price_cent,
  sale_price_cent,
  sale_label,
  COALESCE(benefits, '{}'::jsonb) AS benefits
FROM vip_quota_packs
WHERE status = 'active'
ORDER BY display_order ASC, standard_price_cent ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QuotaPack
	for rows.Next() {
		var item QuotaPack
		var salePrice sql.NullInt64
		var saleLabel sql.NullString
		var benefits JSONMap
		if err := rows.Scan(
			&item.Code,
			&item.Name,
			&item.Description,
			&item.StandardPriceCent,
			&salePrice,
			&saleLabel,
			&benefits,
		); err != nil {
			return nil, err
		}
		if salePrice.Valid {
			item.SalePriceCent = salePrice.Int64
		}
		if saleLabel.Valid {
			item.SaleLabel = saleLabel.String
		}
		item.Benefits = VIPBenefitSnapshotFromJSON(benefits)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *VIPModel) ListAdminVIPPlans(ctx context.Context) ([]AdminVIPPlanConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  p.code,
  p.name,
  p.duration_months,
  p.standard_price_cent,
  p.status,
  p.display_order,
  COALESCE(v.benefits, '{}'::jsonb) AS benefits,
  p.updated_at
FROM vip_plans p
LEFT JOIN LATERAL (
  SELECT benefits
  FROM vip_plan_versions
  WHERE plan_id = p.id
  ORDER BY version DESC
  LIMIT 1
) v ON true
ORDER BY p.display_order ASC, p.duration_months ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminVIPPlanConfig, 0)
	for rows.Next() {
		var item AdminVIPPlanConfig
		var benefits JSONMap
		var updatedAt time.Time
		if err := rows.Scan(
			&item.Code,
			&item.Name,
			&item.DurationMonths,
			&item.StandardPriceCent,
			&item.Status,
			&item.DisplayOrder,
			&benefits,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		item.Benefits = VIPBenefitSnapshotFromJSON(benefits)
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *VIPModel) SaveAdminVIPPlan(ctx context.Context, input SaveAdminVIPPlanInput) (AdminVIPConfigSaveResult, error) {
	var result AdminVIPConfigSaveResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		beforeSnapshot, err := adminVIPPlanSnapshotTx(ctx, tx, input.Code)
		if err != nil {
			return err
		}

		var planID string
		var updatedAt time.Time
		err = tx.QueryRowContext(ctx, `
INSERT INTO vip_plans (
  code,
  name,
  duration_months,
  standard_price_cent,
  status,
  display_order
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  duration_months = EXCLUDED.duration_months,
  standard_price_cent = EXCLUDED.standard_price_cent,
  status = EXCLUDED.status,
  display_order = EXCLUDED.display_order,
  updated_at = now()
RETURNING id::text, code, updated_at
`, input.Code, input.Name, input.DurationMonths, input.StandardPriceCent, input.Status, input.DisplayOrder).Scan(&planID, &result.Code, &updatedAt)
		if err != nil {
			return err
		}

		// 套餐权益必须以版本形式追加，保证已创建订单继续按 benefits_snapshot 执行。
		if _, err := tx.ExecContext(ctx, `
INSERT INTO vip_plan_versions (plan_id, version, benefits, status)
SELECT $1::bigint, COALESCE(MAX(version), 0) + 1, $2, 'active'
FROM vip_plan_versions
WHERE plan_id = $1::bigint
`, planID, input.Benefits.ToJSONMap()); err != nil {
			return err
		}

		result.UpdatedAt = updatedAt.Format(time.RFC3339)
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:     input.OperatorID,
			OperatorRole:   "platform_operator",
			Action:         "vip_plan_save",
			ObjectType:     "vip_config",
			ObjectID:       "",
			BeforeSnapshot: beforeSnapshot,
			AfterSnapshot: JSONMap{
				"code":              input.Code,
				"name":              input.Name,
				"durationMonths":    input.DurationMonths,
				"standardPriceCent": input.StandardPriceCent,
				"status":            input.Status,
				"displayOrder":      input.DisplayOrder,
				"benefits":          input.Benefits.ToJSONMap(),
			},
		})
	})
	if err != nil {
		return AdminVIPConfigSaveResult{}, err
	}
	return result, nil
}

func (m *VIPModel) ListAdminQuotaPacks(ctx context.Context) ([]AdminQuotaPackConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  code,
  name,
  description,
  standard_price_cent,
  sale_price_cent,
  sale_label,
  status,
  display_order,
  COALESCE(benefits, '{}'::jsonb) AS benefits,
  updated_at
FROM vip_quota_packs
ORDER BY display_order ASC, standard_price_cent ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminQuotaPackConfig, 0)
	for rows.Next() {
		var item AdminQuotaPackConfig
		var salePrice sql.NullInt64
		var saleLabel sql.NullString
		var benefits JSONMap
		var updatedAt time.Time
		if err := rows.Scan(
			&item.Code,
			&item.Name,
			&item.Description,
			&item.StandardPriceCent,
			&salePrice,
			&saleLabel,
			&item.Status,
			&item.DisplayOrder,
			&benefits,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if salePrice.Valid {
			item.SalePriceCent = salePrice.Int64
		}
		if saleLabel.Valid {
			item.SaleLabel = saleLabel.String
		}
		item.Benefits = VIPBenefitSnapshotFromJSON(benefits)
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *VIPModel) SaveAdminQuotaPack(ctx context.Context, input SaveAdminQuotaPackInput) (AdminVIPConfigSaveResult, error) {
	var result AdminVIPConfigSaveResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		beforeSnapshot, err := adminQuotaPackSnapshotTx(ctx, tx, input.Code)
		if err != nil {
			return err
		}
		var updatedAt time.Time
		err = tx.QueryRowContext(ctx, `
INSERT INTO vip_quota_packs (
  code,
  name,
  description,
  standard_price_cent,
  sale_price_cent,
  sale_label,
  benefits,
  status,
  display_order
)
VALUES ($1, $2, $3, $4, NULLIF($5, 0), NULLIF($6, ''), $7, $8, $9)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  standard_price_cent = EXCLUDED.standard_price_cent,
  sale_price_cent = EXCLUDED.sale_price_cent,
  sale_label = EXCLUDED.sale_label,
  benefits = EXCLUDED.benefits,
  status = EXCLUDED.status,
  display_order = EXCLUDED.display_order,
  updated_at = now()
RETURNING code, updated_at
`, input.Code, input.Name, input.Description, input.StandardPriceCent, input.SalePriceCent, input.SaleLabel, input.Benefits.ToJSONMap(), input.Status, input.DisplayOrder).Scan(&result.Code, &updatedAt)
		if err != nil {
			return err
		}
		result.UpdatedAt = updatedAt.Format(time.RFC3339)
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:     input.OperatorID,
			OperatorRole:   "platform_operator",
			Action:         "vip_quota_pack_save",
			ObjectType:     "vip_config",
			ObjectID:       "",
			BeforeSnapshot: beforeSnapshot,
			AfterSnapshot: JSONMap{
				"code":              input.Code,
				"name":              input.Name,
				"description":       input.Description,
				"standardPriceCent": input.StandardPriceCent,
				"salePriceCent":     input.SalePriceCent,
				"saleLabel":         input.SaleLabel,
				"status":            input.Status,
				"displayOrder":      input.DisplayOrder,
				"benefits":          input.Benefits.ToJSONMap(),
			},
		})
	})
	if err != nil {
		return AdminVIPConfigSaveResult{}, err
	}
	return result, nil
}

func (m *VIPModel) ListAdminVIPPromotions(ctx context.Context) ([]AdminVIPPromotionConfig, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  promo.code,
  p.code AS plan_code,
  p.name AS plan_name,
  promo.promotion_type,
  promo.sale_price_cent,
  promo.starts_at,
  promo.ends_at,
  promo.quota_limit,
  promo.used_count,
  promo.status,
  promo.updated_at
FROM vip_promotions promo
JOIN vip_plans p ON p.id = promo.plan_id
ORDER BY promo.updated_at DESC, promo.created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminVIPPromotionConfig, 0)
	for rows.Next() {
		var item AdminVIPPromotionConfig
		var startsAt time.Time
		var endsAt sql.NullTime
		var quotaLimit sql.NullInt64
		var updatedAt time.Time
		if err := rows.Scan(
			&item.Code,
			&item.PlanCode,
			&item.PlanName,
			&item.PromotionType,
			&item.SalePriceCent,
			&startsAt,
			&endsAt,
			&quotaLimit,
			&item.UsedCount,
			&item.Status,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		item.StartsAt = startsAt.Format(time.RFC3339)
		if endsAt.Valid {
			item.EndsAt = endsAt.Time.Format(time.RFC3339)
		}
		if quotaLimit.Valid {
			item.QuotaLimit = quotaLimit.Int64
		}
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *VIPModel) SaveAdminVIPPromotion(ctx context.Context, input SaveAdminVIPPromotionInput) (AdminVIPConfigSaveResult, error) {
	var result AdminVIPConfigSaveResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		beforeSnapshot, err := adminVIPPromotionSnapshotTx(ctx, tx, input.Code)
		if err != nil {
			return err
		}

		var updatedAt time.Time
		err = tx.QueryRowContext(ctx, `
WITH selected_plan AS (
  SELECT id
  FROM vip_plans
  WHERE code = $2
  LIMIT 1
)
INSERT INTO vip_promotions (
  code,
  plan_id,
  promotion_type,
  sale_price_cent,
  starts_at,
  ends_at,
  quota_limit,
  status
)
SELECT
  $1,
  selected_plan.id,
  $3,
  $4,
  COALESCE(NULLIF($5, '')::timestamptz, now()),
  NULLIF($6, '')::timestamptz,
  NULLIF($7, 0),
  $8
FROM selected_plan
ON CONFLICT (code) DO UPDATE SET
  plan_id = EXCLUDED.plan_id,
  promotion_type = EXCLUDED.promotion_type,
  sale_price_cent = EXCLUDED.sale_price_cent,
  starts_at = EXCLUDED.starts_at,
  ends_at = EXCLUDED.ends_at,
  quota_limit = EXCLUDED.quota_limit,
  status = EXCLUDED.status,
  updated_at = now()
RETURNING code, updated_at
`, input.Code, input.PlanCode, input.PromotionType, input.SalePriceCent, input.StartsAt, input.EndsAt, input.QuotaLimit, input.Status).Scan(&result.Code, &updatedAt)
		if err != nil {
			return err
		}
		result.UpdatedAt = updatedAt.Format(time.RFC3339)
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:     input.OperatorID,
			OperatorRole:   "platform_operator",
			Action:         "vip_promotion_save",
			ObjectType:     "vip_config",
			ObjectID:       "",
			BeforeSnapshot: beforeSnapshot,
			AfterSnapshot: JSONMap{
				"code":          input.Code,
				"planCode":      input.PlanCode,
				"promotionType": input.PromotionType,
				"salePriceCent": input.SalePriceCent,
				"startsAt":      input.StartsAt,
				"endsAt":        input.EndsAt,
				"quotaLimit":    input.QuotaLimit,
				"status":        input.Status,
			},
		})
	})
	if err != nil {
		return AdminVIPConfigSaveResult{}, err
	}
	return result, nil
}

func adminVIPPlanSnapshotTx(ctx context.Context, tx *sql.Tx, code string) (JSONMap, error) {
	var snapshot JSONMap
	var benefits JSONMap
	var updatedAt time.Time
	var name string
	var durationMonths int64
	var standardPriceCent int64
	var status string
	var displayOrder int64
	err := tx.QueryRowContext(ctx, `
SELECT
  p.name,
  p.duration_months,
  p.standard_price_cent,
  p.status,
  p.display_order,
  COALESCE(v.benefits, '{}'::jsonb) AS benefits,
  p.updated_at
FROM vip_plans p
LEFT JOIN LATERAL (
  SELECT benefits
  FROM vip_plan_versions
  WHERE plan_id = p.id
  ORDER BY version DESC
  LIMIT 1
) v ON true
WHERE p.code = $1
LIMIT 1
`, strings.TrimSpace(code)).Scan(&name, &durationMonths, &standardPriceCent, &status, &displayOrder, &benefits, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return JSONMap{}, nil
	}
	if err != nil {
		return nil, err
	}
	snapshot = JSONMap{
		"code":              strings.TrimSpace(code),
		"name":              name,
		"durationMonths":    durationMonths,
		"standardPriceCent": standardPriceCent,
		"status":            status,
		"displayOrder":      displayOrder,
		"benefits":          benefits,
		"updatedAt":         updatedAt.Format(time.RFC3339),
	}
	return snapshot, nil
}

func adminQuotaPackSnapshotTx(ctx context.Context, tx *sql.Tx, code string) (JSONMap, error) {
	var benefits JSONMap
	var salePrice sql.NullInt64
	var saleLabel sql.NullString
	var updatedAt time.Time
	var name string
	var description string
	var standardPriceCent int64
	var status string
	var displayOrder int64
	err := tx.QueryRowContext(ctx, `
SELECT
  name,
  description,
  standard_price_cent,
  sale_price_cent,
  sale_label,
  status,
  display_order,
  COALESCE(benefits, '{}'::jsonb) AS benefits,
  updated_at
FROM vip_quota_packs
WHERE code = $1
LIMIT 1
`, strings.TrimSpace(code)).Scan(&name, &description, &standardPriceCent, &salePrice, &saleLabel, &status, &displayOrder, &benefits, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return JSONMap{}, nil
	}
	if err != nil {
		return nil, err
	}
	snapshot := JSONMap{
		"code":              strings.TrimSpace(code),
		"name":              name,
		"description":       description,
		"standardPriceCent": standardPriceCent,
		"status":            status,
		"displayOrder":      displayOrder,
		"benefits":          benefits,
		"updatedAt":         updatedAt.Format(time.RFC3339),
	}
	if salePrice.Valid {
		snapshot["salePriceCent"] = salePrice.Int64
	}
	if saleLabel.Valid {
		snapshot["saleLabel"] = saleLabel.String
	}
	return snapshot, nil
}

func adminVIPPromotionSnapshotTx(ctx context.Context, tx *sql.Tx, code string) (JSONMap, error) {
	var startsAt time.Time
	var endsAt sql.NullTime
	var quotaLimit sql.NullInt64
	var updatedAt time.Time
	var planCode string
	var promotionType string
	var salePriceCent int64
	var usedCount int64
	var status string
	err := tx.QueryRowContext(ctx, `
SELECT
  p.code,
  promo.promotion_type,
  promo.sale_price_cent,
  promo.starts_at,
  promo.ends_at,
  promo.quota_limit,
  promo.used_count,
  promo.status,
  promo.updated_at
FROM vip_promotions promo
JOIN vip_plans p ON p.id = promo.plan_id
WHERE promo.code = $1
LIMIT 1
`, strings.TrimSpace(code)).Scan(&planCode, &promotionType, &salePriceCent, &startsAt, &endsAt, &quotaLimit, &usedCount, &status, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return JSONMap{}, nil
	}
	if err != nil {
		return nil, err
	}
	snapshot := JSONMap{
		"code":          strings.TrimSpace(code),
		"planCode":      planCode,
		"promotionType": promotionType,
		"salePriceCent": salePriceCent,
		"startsAt":      startsAt.Format(time.RFC3339),
		"usedCount":     usedCount,
		"status":        status,
		"updatedAt":     updatedAt.Format(time.RFC3339),
	}
	if endsAt.Valid {
		snapshot["endsAt"] = endsAt.Time.Format(time.RFC3339)
	}
	if quotaLimit.Valid {
		snapshot["quotaLimit"] = quotaLimit.Int64
	}
	return snapshot, nil
}

func (m *VIPModel) GetMerchantVIPSummary(ctx context.Context, merchantID string) (MerchantVIPSummary, error) {
	var result MerchantVIPSummary
	result.MerchantID = strings.TrimSpace(merchantID)
	var startsAt sql.NullTime
	var expiresAt sql.NullTime
	err := m.db.QueryRowContext(ctx, `
SELECT
  m.id::text,
  CASE
    WHEN s.id IS NULL THEN 'none'
    WHEN s.status = 'active' AND s.starts_at <= now() AND s.expires_at > now() THEN 'active'
    ELSE 'expired'
  END AS vip_status,
  COALESCE(p.code, '') AS plan_code,
  COALESCE(p.name, '') AS plan_name,
  s.starts_at,
  s.expires_at,
  COALESCE(ent.publish_quota_remaining, 0) AS publish_quota_remaining,
  COALESCE(ent.refresh_quota_remaining, 0) AS refresh_quota_remaining,
  COALESCE(ent.top_voucher_count, 0) AS top_voucher_count
FROM merchants m
LEFT JOIN LATERAL (
  SELECT *
  FROM merchant_vip_subscriptions
  WHERE merchant_id = m.id
  ORDER BY expires_at DESC, created_at DESC
  LIMIT 1
) s ON true
LEFT JOIN vip_plans p ON p.id = s.plan_id
LEFT JOIN LATERAL (
  SELECT
    SUM(remaining_amount) FILTER (WHERE entitlement_type = 'publish_quota') AS publish_quota_remaining,
    SUM(remaining_amount) FILTER (WHERE entitlement_type = 'refresh_quota') AS refresh_quota_remaining,
    SUM(remaining_amount) FILTER (WHERE entitlement_type = 'top_voucher') AS top_voucher_count
  FROM merchant_entitlements
  WHERE merchant_id = m.id
    AND source_type = 'vip'
    AND status = 'active'
    AND starts_at <= now()
    AND (expires_at IS NULL OR expires_at > now())
) ent ON true
WHERE m.id = $1
  AND m.deleted_at IS NULL
LIMIT 1
`, result.MerchantID).Scan(
		&result.MerchantID,
		&result.Status,
		&result.PlanCode,
		&result.PlanName,
		&startsAt,
		&expiresAt,
		&result.PublishQuotaRemaining,
		&result.RefreshQuotaRemaining,
		&result.TopVoucherCount,
	)
	if err != nil {
		return MerchantVIPSummary{}, err
	}
	if startsAt.Valid {
		result.StartsAt = startsAt.Time.Format(time.RFC3339)
	}
	if expiresAt.Valid {
		result.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
	}
	return result, nil
}

func (m *VIPModel) CreateVIPOrder(ctx context.Context, input CreateVIPOrderInput) (VIPOrder, error) {
	merchantID := strings.TrimSpace(input.MerchantID)
	userID := strings.TrimSpace(input.UserID)
	planCode := strings.TrimSpace(input.PlanCode)

	var planID string
	var planVersionID string
	var benefitsJSON JSONMap
	var promotionID sql.NullString
	var promotionCode sql.NullString
	var promotionType sql.NullString
	var salePrice sql.NullInt64
	var order VIPOrder
	err := m.db.QueryRowContext(ctx, `
SELECT
  p.id::text,
  v.id::text,
  p.code,
  p.name,
  p.duration_months,
  p.standard_price_cent,
  COALESCE(v.benefits, '{}'::jsonb) AS benefits,
  promo.id::text,
  promo.code,
  promo.promotion_type,
  promo.sale_price_cent
FROM vip_plans p
JOIN LATERAL (
  SELECT id, benefits
  FROM vip_plan_versions
  WHERE plan_id = p.id
    AND status = 'active'
    AND starts_at <= now()
    AND (ends_at IS NULL OR ends_at > now())
  ORDER BY version DESC
  LIMIT 1
) v ON true
LEFT JOIN LATERAL (
  SELECT id, code, promotion_type, sale_price_cent
  FROM vip_promotions
  WHERE plan_id = p.id
    AND status = 'active'
    AND starts_at <= now()
    AND (ends_at IS NULL OR ends_at > now())
    AND (quota_limit IS NULL OR used_count < quota_limit)
    AND (
      promotion_type <> 'first_purchase'
      OR NOT EXISTS (
        SELECT 1
        FROM vip_orders
        WHERE merchant_id = $1
          AND product_type = 'vip_plan'
          AND status = 'paid'
      )
    )
  ORDER BY sale_price_cent ASC, created_at DESC
  LIMIT 1
) promo ON true
WHERE p.code = $2
  AND p.status = 'active'
LIMIT 1
`, merchantID, planCode).Scan(
		&planID,
		&planVersionID,
		&order.PlanCode,
		&order.PlanName,
		new(int64),
		&order.StandardPriceCent,
		&benefitsJSON,
		&promotionID,
		&promotionCode,
		&promotionType,
		&salePrice,
	)
	if err != nil {
		return VIPOrder{}, err
	}
	order.MerchantID = merchantID
	order.UserID = userID
	order.ProductType = VIPProductTypeVIPPlan
	order.ProductCode = order.PlanCode
	order.ProductName = order.PlanName
	order.Currency = "CNY"
	order.ActualPriceCent = order.StandardPriceCent
	order.Benefits = VIPBenefitSnapshotFromJSON(benefitsJSON)
	if promotionID.Valid {
		order.PromotionCode = promotionCode.String
		if salePrice.Valid && salePrice.Int64 <= order.StandardPriceCent {
			order.ActualPriceCent = salePrice.Int64
		}
	}

	err = m.db.QueryRowContext(ctx, `
INSERT INTO vip_orders (
  merchant_id,
  user_id,
  product_type,
  product_code,
  product_name,
  plan_id,
  plan_version_id,
  promotion_id,
  out_trade_no,
  standard_price_cent,
  actual_price_cent,
  currency,
  benefits_snapshot,
  product_snapshot,
  status
)
VALUES ($1, $2, 'vip_plan', $3, $4, $5, $6, NULLIF($7, '')::bigint, $8, $9, $10, $11, $12, $13, 'pending')
RETURNING id::text, out_trade_no, status
`, merchantID, userID, order.PlanCode, order.PlanName, planID, planVersionID, nullStringToString(promotionID), buildVIPOutTradeNo(merchantID+planCode), order.StandardPriceCent, order.ActualPriceCent, order.Currency, order.Benefits.ToJSONMap(), JSONMap{
		"planCode":      order.PlanCode,
		"planName":      order.PlanName,
		"promotionCode": order.PromotionCode,
	}).Scan(
		&order.ID,
		&order.OutTradeNo,
		&order.Status,
	)
	if err != nil {
		return VIPOrder{}, err
	}
	return order, nil
}

func (m *VIPModel) CreateQuotaPackOrder(ctx context.Context, input CreateQuotaPackOrderInput) (VIPOrder, error) {
	merchantID := strings.TrimSpace(input.MerchantID)
	userID := strings.TrimSpace(input.UserID)
	packCode := strings.TrimSpace(input.PackCode)
	resourceID := strings.TrimSpace(input.ResourceID)

	var packID string
	var benefitsJSON JSONMap
	var salePrice sql.NullInt64
	var saleLabel sql.NullString
	var quotaSaleLabel string
	var topResourceTitle string
	var topResourceTypeCode string
	var order VIPOrder
	err := m.db.QueryRowContext(ctx, `
SELECT
  id::text,
  code,
  name,
  standard_price_cent,
  sale_price_cent,
  sale_label,
  COALESCE(benefits, '{}'::jsonb) AS benefits
FROM vip_quota_packs
WHERE code = $1
  AND status = 'active'
LIMIT 1
`, packCode).Scan(
		&packID,
		&order.ProductCode,
		&order.ProductName,
		&order.StandardPriceCent,
		&salePrice,
		&saleLabel,
		&benefitsJSON,
	)
	if err != nil {
		return VIPOrder{}, err
	}
	order.MerchantID = merchantID
	order.UserID = userID
	order.ProductType = VIPProductTypeQuotaPack
	order.Currency = "CNY"
	order.ActualPriceCent = order.StandardPriceCent
	order.Benefits = VIPBenefitSnapshotFromJSON(benefitsJSON)
	if salePrice.Valid && salePrice.Int64 > 0 && salePrice.Int64 <= order.StandardPriceCent {
		order.ActualPriceCent = salePrice.Int64
	}
	if saleLabel.Valid {
		quotaSaleLabel = saleLabel.String
	}

	productSnapshot := JSONMap{
		"packId":    packID,
		"packCode":  order.ProductCode,
		"packName":  order.ProductName,
		"saleLabel": quotaSaleLabel,
	}
	if order.Benefits.TopVoucherCount > 0 {
		if resourceID == "" {
			return VIPOrder{}, ErrTopServiceResourceRequired
		}
		if order.Benefits.TopVoucherCount != 1 || order.Benefits.TopDurationHours <= 0 {
			return VIPOrder{}, ErrTopServiceProductInvalid
		}
		// 单独购买置顶服务必须绑定一条当前可置顶资源，避免支付后产生可囤积的置顶券余额。
		if err := m.db.QueryRowContext(ctx, `
SELECT title, type_code
FROM resources
WHERE id = $1
  AND merchant_id = $2
  AND status = 'published'
  AND deleted_at IS NULL
LIMIT 1
`, resourceID, merchantID).Scan(&topResourceTitle, &topResourceTypeCode); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return VIPOrder{}, ErrTopServiceResourceInvalid
			}
			return VIPOrder{}, err
		}
		productSnapshot["resourceId"] = resourceID
		productSnapshot["resourceTitle"] = topResourceTitle
		productSnapshot["resourceTypeCode"] = topResourceTypeCode
		productSnapshot["fulfillment"] = "redeem_after_payment"
	}

	err = m.db.QueryRowContext(ctx, `
INSERT INTO vip_orders (
  merchant_id,
  user_id,
  product_type,
  product_code,
  product_name,
  out_trade_no,
  standard_price_cent,
  actual_price_cent,
  currency,
  benefits_snapshot,
  product_snapshot,
  status
)
VALUES ($1, $2, 'quota_pack', $3, $4, $5, $6, $7, $8, $9, $10, 'pending')
RETURNING id::text, out_trade_no, status
`, merchantID, userID, order.ProductCode, order.ProductName, buildVIPOutTradeNo(merchantID+packCode+resourceID), order.StandardPriceCent, order.ActualPriceCent, order.Currency, order.Benefits.ToJSONMap(), productSnapshot).Scan(
		&order.ID,
		&order.OutTradeNo,
		&order.Status,
	)
	if err != nil {
		return VIPOrder{}, err
	}
	return order, nil
}

func (m *VIPModel) GetVIPPaymentContext(ctx context.Context, input GetVIPPaymentContextInput) (VIPPaymentContext, error) {
	var result VIPPaymentContext
	err := m.db.QueryRowContext(ctx, `
SELECT
  o.id::text,
  o.merchant_id::text,
  $3::text,
  COALESCE(u.wechat_openid, ''),
  o.status,
  o.out_trade_no,
  o.actual_price_cent,
  o.currency,
  COALESCE(p.name, '') AS plan_name,
  o.product_type,
  COALESCE(NULLIF(o.product_name, ''), p.name, '权益商品') AS product_name
FROM vip_orders o
LEFT JOIN vip_plans p ON p.id = o.plan_id
JOIN merchants m ON m.id = o.merchant_id
JOIN users u ON u.id = $3
JOIN merchant_admin_bindings mab ON mab.merchant_id = o.merchant_id AND mab.user_id = u.id AND mab.status = 'active'
WHERE o.merchant_id = $1
  AND o.id = $2
  AND m.deleted_at IS NULL
  AND m.status = 'active'
LIMIT 1
`, strings.TrimSpace(input.MerchantID), strings.TrimSpace(input.OrderID), strings.TrimSpace(input.UserID)).Scan(
		&result.OrderID,
		&result.MerchantID,
		&result.UserID,
		&result.OpenID,
		&result.Status,
		&result.OutTradeNo,
		&result.AmountTotal,
		&result.Currency,
		&result.PlanName,
		&result.ProductType,
		&result.ProductName,
	)
	return result, err
}

func (m *VIPModel) CreateVIPPaymentOrder(ctx context.Context, input CreateVIPPaymentOrderInput) (VIPPaymentOrder, error) {
	var result VIPPaymentOrder
	err := m.db.QueryRowContext(ctx, `
UPDATE vip_orders
SET user_id = $3,
    updated_at = now()
WHERE id = $1
  AND merchant_id = $2
  AND status = 'pending'
RETURNING id::text,
  out_trade_no,
  actual_price_cent,
  currency,
  status,
  COALESCE((SELECT name FROM vip_plans WHERE id = vip_orders.plan_id), '') AS plan_name,
  product_type,
  COALESCE(NULLIF(product_name, ''), '权益商品') AS product_name
`, strings.TrimSpace(input.OrderID), strings.TrimSpace(input.MerchantID), strings.TrimSpace(input.UserID)).Scan(
		&result.ID,
		&result.OutTradeNo,
		&result.AmountTotal,
		&result.Currency,
		&result.Status,
		&result.PlanName,
		&result.ProductType,
		&result.ProductName,
	)
	return result, err
}

func (m *VIPModel) MarkVIPOrderPaid(ctx context.Context, input MarkVIPOrderPaidInput) (VIPPaymentResult, error) {
	var result VIPPaymentResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		paidAt := parseVIPPaymentTime(input.SuccessTime)
		var planID sql.NullString
		var promotionID sql.NullString
		var productType string
		var currentStatus string
		var actualPriceCent int64
		var benefitsJSON JSONMap
		var productSnapshot JSONMap
		err := tx.QueryRowContext(ctx, `
SELECT id::text, merchant_id::text, plan_id::text, promotion_id::text, product_type, status, actual_price_cent, benefits_snapshot, product_snapshot
FROM vip_orders
WHERE out_trade_no = $1
FOR UPDATE
`, strings.TrimSpace(input.OutTradeNo)).Scan(
			&result.OrderID,
			&result.MerchantID,
			&planID,
			&promotionID,
			&productType,
			&currentStatus,
			&actualPriceCent,
			&benefitsJSON,
			&productSnapshot,
		)
		if err != nil {
			return err
		}
		if input.AmountTotal > 0 && input.AmountTotal != actualPriceCent {
			return sql.ErrNoRows
		}
		if currentStatus == PaymentOrderStatusPaid {
			result.Status = PaymentOrderStatusPaid
			return nil
		}
		if currentStatus != PaymentOrderStatusPending {
			return sql.ErrNoRows
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE vip_orders
SET transaction_id = $2,
    status = 'paid',
    notify_payload = $3,
    paid_at = $4,
    updated_at = now()
WHERE id = $1
`, result.OrderID, strings.TrimSpace(input.TransactionID), nonNilJSONMap(input.NotifyPayload), paidAt); err != nil {
			return err
		}
		result.Status = PaymentOrderStatusPaid
		if promotionID.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE vip_promotions SET used_count = used_count + 1, updated_at = now() WHERE id = $1`, promotionID.String); err != nil {
				return err
			}
		}

		// 同一张订单表承载 VIP 套餐和次数包：支付状态先幂等落库，再按商品类型发放不同权益。
		if strings.TrimSpace(productType) == VIPProductTypeQuotaPack {
			if err := GrantQuotaPackBenefits(ctx, tx, result.MerchantID, result.OrderID, VIPBenefitSnapshotFromJSON(benefitsJSON), paidAt, productSnapshot); err != nil {
				return err
			}
			messageTitle, messageContent, targetURL := quotaPackPaidMessage(result.MerchantID, productSnapshot)
			if _, err := tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'quota_pack', 'vip_order_paid', NULLIF($2, '')::bigint, $3, $4, $5, 'unread')
`, "merchant:"+result.MerchantID, result.OrderID, messageTitle, messageContent, targetURL); err != nil {
				return err
			}
			return nil
		}
		if strings.TrimSpace(productType) != "" && strings.TrimSpace(productType) != VIPProductTypeVIPPlan {
			return sql.ErrNoRows
		}
		if !planID.Valid {
			return sql.ErrNoRows
		}
		var durationMonths int
		if err := tx.QueryRowContext(ctx, `SELECT duration_months FROM vip_plans WHERE id = $1`, planID.String).Scan(&durationMonths); err != nil {
			return err
		}
		periodStart, periodEnd, err := vipSubscriptionPeriod(ctx, tx, result.MerchantID, paidAt, durationMonths)
		if err != nil {
			return err
		}
		if periodEnd.IsZero() {
			periodEnd = periodStart.AddDate(0, durationMonths, 0)
		}
		err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_vip_subscriptions (merchant_id, plan_id, order_id, status, source_type, starts_at, expires_at)
VALUES ($1, $2, $3, 'active', 'paid', $4, $5)
ON CONFLICT (merchant_id) WHERE status = 'active'
DO UPDATE SET
  plan_id = EXCLUDED.plan_id,
  order_id = EXCLUDED.order_id,
  starts_at = LEAST(merchant_vip_subscriptions.starts_at, EXCLUDED.starts_at),
  expires_at = EXCLUDED.expires_at,
  updated_at = now()
RETURNING id::text
`, result.MerchantID, planID.String, result.OrderID, periodStart, periodEnd).Scan(&result.SubscriptionID)
		if err != nil {
			return err
		}
		// VIP 赠送权益按批次发放，每批固定发放后 30 天内有效；若订阅提前到期，则不超过订阅到期时间。
		grantEnd := vipBenefitGrantEnd(periodStart, periodEnd)
		if err := GrantVIPMonthlyBenefits(ctx, tx, result.MerchantID, result.OrderID, VIPBenefitSnapshotFromJSON(benefitsJSON), periodStart, grantEnd); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'vip_membership', 'vip_order_paid', NULLIF($2, '')::bigint, 'VIP 已开通', 'VIP 发布权益已到账，可在发布资源时使用。', $3, 'unread')
`, "merchant:"+result.MerchantID, result.OrderID, "/pages/vip/index?merchantId="+result.MerchantID); err != nil {
			return err
		}
		return nil
	})
	return result, err
}

func GrantVIPMonthlyBenefits(ctx context.Context, tx *sql.Tx, merchantID string, orderID string, snapshot VIPBenefitSnapshot, periodStart time.Time, periodEnd time.Time) error {
	for _, item := range []struct {
		entitlementType string
		totalAmount     int64
	}{
		{entitlementType: EntitlementTypePublishQuota, totalAmount: snapshot.PublishQuota},
		{entitlementType: EntitlementTypeRefreshQuota, totalAmount: snapshot.RefreshQuota},
	} {
		if item.totalAmount <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_entitlements (merchant_id, entitlement_type, source_type, total_amount, remaining_amount, starts_at, expires_at, status)
VALUES ($1, $2, 'vip', $3, $3, $4, $5, 'active')
`, merchantID, item.entitlementType, item.totalAmount, periodStart, periodEnd); err != nil {
			return err
		}
	}
	if snapshot.TopVoucherCount > 0 && snapshot.TopDurationHours > 0 {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_entitlements (merchant_id, entitlement_type, source_type, total_amount, remaining_amount, starts_at, expires_at, status, allowed_type_codes, top_duration_hours)
VALUES ($1, $2, 'vip', $3, $3, $4, $5, 'active', '[]'::jsonb, $6)
`, merchantID, EntitlementTypeTopVoucher, snapshot.TopVoucherCount, periodStart, periodEnd, snapshot.TopDurationHours); err != nil {
			return err
		}
	}
	return recordOperationLogTx(ctx, tx, OperationLogInput{
		OperatorID:   "",
		OperatorRole: "system",
		Action:       "vip_benefits_grant",
		ObjectType:   "merchant",
		ObjectID:     merchantID,
		AfterSnapshot: JSONMap{
			"orderId":          orderID,
			"publishQuota":     snapshot.PublishQuota,
			"refreshQuota":     snapshot.RefreshQuota,
			"topVoucherCount":  snapshot.TopVoucherCount,
			"topDurationHours": snapshot.TopDurationHours,
			"periodStart":      periodStart.Format(time.RFC3339),
			"periodEnd":        periodEnd.Format(time.RFC3339),
		},
	})
}

func vipBenefitGrantEnd(periodStart time.Time, periodEnd time.Time) time.Time {
	grantEnd := periodStart.AddDate(0, 0, vipBenefitValidityDays)
	if !periodEnd.IsZero() && periodEnd.Before(grantEnd) {
		return periodEnd
	}
	return grantEnd
}

func GrantQuotaPackBenefits(ctx context.Context, tx *sql.Tx, merchantID string, orderID string, snapshot VIPBenefitSnapshot, paidAt time.Time, productSnapshot JSONMap) error {
	quotaPackExpiresAt := paidAt.AddDate(0, 0, quotaPackValidityDays)
	topResourceID := strings.TrimSpace(stringFromJSON(productSnapshot["resourceId"]))
	var topRedeemResult RedeemTopVoucherResult
	for _, item := range []struct {
		entitlementType string
		totalAmount     int64
	}{
		{entitlementType: EntitlementTypePublishQuota, totalAmount: snapshot.PublishQuota},
		{entitlementType: EntitlementTypeRefreshQuota, totalAmount: snapshot.RefreshQuota},
	} {
		if item.totalAmount <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_entitlements (merchant_id, entitlement_type, source_type, total_amount, remaining_amount, starts_at, expires_at, status)
VALUES ($1, $2, 'quota_pack', $3, $3, $4, $5, 'active')
`, merchantID, item.entitlementType, item.totalAmount, paidAt, quotaPackExpiresAt); err != nil {
			return err
		}
	}
	if snapshot.TopVoucherCount > 0 && snapshot.TopDurationHours > 0 {
		if topResourceID != "" {
			if snapshot.TopVoucherCount != 1 {
				return ErrTopServiceProductInvalid
			}
			var entitlementID string
			if err := tx.QueryRowContext(ctx, `
INSERT INTO merchant_entitlements (merchant_id, entitlement_type, source_type, total_amount, remaining_amount, starts_at, expires_at, status, allowed_type_codes, top_duration_hours)
VALUES ($1, $2, 'top_service', 1, 1, $3, $4, 'active', '[]'::jsonb, $5)
RETURNING id::text
`, merchantID, EntitlementTypeTopVoucher, paidAt, quotaPackExpiresAt, snapshot.TopDurationHours).Scan(&entitlementID); err != nil {
				return err
			}
			// 单次置顶服务支付成功后立即核销，避免在商家账户里沉淀可提前购买的置顶券余额。
			result, err := redeemTopVoucherTx(ctx, tx, entitlementID, topResourceID)
			if err != nil {
				return err
			}
			topRedeemResult = result
		} else {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_entitlements (merchant_id, entitlement_type, source_type, total_amount, remaining_amount, starts_at, expires_at, status, allowed_type_codes, top_duration_hours)
VALUES ($1, $2, 'quota_pack', $3, $3, $4, $5, 'active', '[]'::jsonb, $6)
`, merchantID, EntitlementTypeTopVoucher, snapshot.TopVoucherCount, paidAt, quotaPackExpiresAt, snapshot.TopDurationHours); err != nil {
				return err
			}
		}
	}
	return recordOperationLogTx(ctx, tx, OperationLogInput{
		OperatorID:   "",
		OperatorRole: "system",
		Action:       "quota_pack_benefits_grant",
		ObjectType:   "merchant",
		ObjectID:     merchantID,
		AfterSnapshot: JSONMap{
			"orderId":          orderID,
			"publishQuota":     snapshot.PublishQuota,
			"refreshQuota":     snapshot.RefreshQuota,
			"topVoucherCount":  snapshot.TopVoucherCount,
			"topDurationHours": snapshot.TopDurationHours,
			"topResourceId":    topResourceID,
			"topExpiresAt":     topRedeemResult.TopExpiresAt,
			"paidAt":           paidAt.Format(time.RFC3339),
			"expiresAt":        quotaPackExpiresAt.Format(time.RFC3339),
		},
	})
}

func quotaPackPaidMessage(merchantID string, productSnapshot JSONMap) (string, string, string) {
	if strings.TrimSpace(stringFromJSON(productSnapshot["resourceId"])) != "" {
		return "置顶服务已生效", "购买的置顶服务已作用到对应供需信息，可在我的发布查看。", "/pages/my-resources/index?merchantId=" + merchantID
	}
	return "次数包已到账", "购买的发布或刷新次数已到账，可直接使用。", "/pages/vip/index?merchantId=" + merchantID
}

func buildVIPOutTradeNo(orderID string) string {
	cleanID := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.TrimSpace(orderID))
	if len(cleanID) > 10 {
		cleanID = cleanID[len(cleanID)-10:]
	}
	outTradeNo := fmt.Sprintf("VIP%d%s", time.Now().UnixNano(), cleanID)
	if len(outTradeNo) > 32 {
		outTradeNo = outTradeNo[:32]
	}
	return outTradeNo
}

func parseVIPPaymentTime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Now()
	}
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err == nil {
		return parsed
	}
	return time.Now()
}

func vipSubscriptionPeriod(ctx context.Context, tx *sql.Tx, merchantID string, paidAt time.Time, durationMonths int) (time.Time, time.Time, error) {
	if durationMonths <= 0 {
		durationMonths = 1
	}
	periodStart := paidAt
	var existingExpires sql.NullTime
	err := tx.QueryRowContext(ctx, `
SELECT expires_at
FROM merchant_vip_subscriptions
WHERE merchant_id = $1
  AND status = 'active'
  AND expires_at > $2
ORDER BY expires_at DESC
LIMIT 1
`, merchantID, paidAt).Scan(&existingExpires)
	if err == nil && existingExpires.Valid && existingExpires.Time.After(periodStart) {
		periodStart = existingExpires.Time
	}
	if err != nil && err != sql.ErrNoRows {
		return time.Time{}, time.Time{}, err
	}
	return periodStart, periodStart.AddDate(0, durationMonths, 0), nil
}

func vipPromotionSaleLabel(promotionType string) string {
	switch strings.TrimSpace(promotionType) {
	case "first_purchase":
		return "首月优惠"
	case "launch":
		return "限时优惠"
	default:
		return "优惠价"
	}
}

func nullStringToString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
