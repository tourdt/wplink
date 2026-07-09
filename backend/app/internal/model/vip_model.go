package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	VIPStatusNone    = "none"
	VIPStatusActive  = "active"
	VIPStatusExpired = "expired"

	VIPPublishPolicyQuota = "quota"
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

type VIPOrder struct {
	ID                string
	MerchantID        string
	UserID            string
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
  COALESCE(tv.top_voucher_count, 0) AS top_voucher_count
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
    SUM(remaining_amount) FILTER (WHERE entitlement_type = 'refresh_quota') AS refresh_quota_remaining
  FROM merchant_entitlements
  WHERE merchant_id = m.id
    AND source_type = 'vip'
    AND status = 'active'
    AND starts_at <= now()
    AND (expires_at IS NULL OR expires_at > now())
) ent ON true
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS top_voucher_count
  FROM top_vouchers
  WHERE merchant_id = m.id
    AND source_type = 'vip'
    AND status = 'unused'
    AND (expires_at IS NULL OR expires_at > now())
) tv ON true
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
  plan_id,
  plan_version_id,
  promotion_id,
  out_trade_no,
  standard_price_cent,
  actual_price_cent,
  currency,
  benefits_snapshot,
  status
)
VALUES ($1, $2, $3, $4, NULLIF($5, '')::bigint, $6, $7, $8, $9, $10, 'pending')
RETURNING id::text, out_trade_no, status
`, merchantID, userID, planID, planVersionID, nullStringToString(promotionID), buildVIPOutTradeNo(merchantID+planCode), order.StandardPriceCent, order.ActualPriceCent, order.Currency, order.Benefits.ToJSONMap()).Scan(
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
  p.name
FROM vip_orders o
JOIN vip_plans p ON p.id = o.plan_id
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
RETURNING id::text, out_trade_no, actual_price_cent, currency, status, (
  SELECT name
  FROM vip_plans
  WHERE id = vip_orders.plan_id
)
`, strings.TrimSpace(input.OrderID), strings.TrimSpace(input.MerchantID), strings.TrimSpace(input.UserID)).Scan(
		&result.ID,
		&result.OutTradeNo,
		&result.AmountTotal,
		&result.Currency,
		&result.Status,
		&result.PlanName,
	)
	return result, err
}

func (m *VIPModel) MarkVIPOrderPaid(ctx context.Context, input MarkVIPOrderPaidInput) (VIPPaymentResult, error) {
	var result VIPPaymentResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		paidAt := parseVIPPaymentTime(input.SuccessTime)
		var planID string
		var promotionID sql.NullString
		var currentStatus string
		var actualPriceCent int64
		var benefitsJSON JSONMap
		err := tx.QueryRowContext(ctx, `
SELECT id::text, merchant_id::text, plan_id::text, promotion_id::text, status, actual_price_cent, benefits_snapshot
FROM vip_orders
WHERE out_trade_no = $1
FOR UPDATE
`, strings.TrimSpace(input.OutTradeNo)).Scan(
			&result.OrderID,
			&result.MerchantID,
			&planID,
			&promotionID,
			&currentStatus,
			&actualPriceCent,
			&benefitsJSON,
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

		var durationMonths int
		if err := tx.QueryRowContext(ctx, `SELECT duration_months FROM vip_plans WHERE id = $1`, planID).Scan(&durationMonths); err != nil {
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
`, result.MerchantID, planID, result.OrderID, periodStart, periodEnd).Scan(&result.SubscriptionID)
		if err != nil {
			return err
		}
		// VIP 权益按月发放，半年/年卡只在支付成功后发放当前周期额度，后续周期由运营任务续发。
		grantEnd := periodStart.AddDate(0, 1, 0)
		if periodEnd.Before(grantEnd) {
			grantEnd = periodEnd
		}
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
		{entitlementType: "publish_quota", totalAmount: snapshot.PublishQuota},
		{entitlementType: "refresh_quota", totalAmount: snapshot.RefreshQuota},
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
	if snapshot.TopDurationHours > 0 {
		for i := int64(0); i < snapshot.TopVoucherCount; i++ {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO top_vouchers (merchant_id, source_type, allowed_type_codes, top_duration_hours, expires_at, status)
VALUES ($1, 'vip', '[]'::jsonb, $2, $3, 'unused')
`, merchantID, snapshot.TopDurationHours, periodEnd); err != nil {
				return err
			}
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
