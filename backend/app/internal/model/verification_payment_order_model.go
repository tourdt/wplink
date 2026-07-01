package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (m *VerificationModel) GetVerificationPaymentContext(ctx context.Context, input GetVerificationPaymentContextInput) (VerificationPaymentContext, error) {
	var result VerificationPaymentContext
	var billingConfig JSONMap
	err := m.db.QueryRowContext(ctx, `
SELECT
  v.id::text,
  v.merchant_id::text,
  $3::text,
  COALESCE(u.wechat_openid, ''),
  v.status,
  cs.code,
  COALESCE(cs.config->'verificationBilling', '{}'::jsonb)
FROM verifications v
JOIN merchants m ON m.id = v.merchant_id
JOIN city_stations cs ON cs.id = m.city_station_id
JOIN users u ON u.id = $3
JOIN merchant_admin_bindings mab ON mab.merchant_id = v.merchant_id AND mab.user_id = u.id AND mab.status = 'active'
WHERE v.id = $2
  AND v.merchant_id = $1
  AND m.deleted_at IS NULL
  AND m.status = 'active'
LIMIT 1
`, input.MerchantID, input.VerificationID, input.UserID).Scan(
		&result.VerificationID,
		&result.MerchantID,
		&result.UserID,
		&result.OpenID,
		&result.Status,
		&result.Billing.CityCode,
		&billingConfig,
	)
	if err != nil {
		return VerificationPaymentContext{}, err
	}
	result.Billing = VerificationBillingConfigFromJSON(result.Billing.CityCode, billingConfig)
	return result, nil
}

func (m *VerificationModel) CreateVerificationPaymentOrder(ctx context.Context, input CreateVerificationPaymentOrderInput) (VerificationPaymentOrder, error) {
	var result VerificationPaymentOrder
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "CNY"
	}
	outTradeNo := buildVerificationOutTradeNo(input.VerificationID)
	err := m.db.QueryRowContext(ctx, `
INSERT INTO verification_payment_orders (
  verification_id,
  merchant_id,
  user_id,
  out_trade_no,
  amount_total,
  currency,
  status
)
VALUES ($1, $2, $3, $4, $5, $6, 'pending')
ON CONFLICT (verification_id) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  amount_total = EXCLUDED.amount_total,
  currency = EXCLUDED.currency,
  updated_at = now()
WHERE verification_payment_orders.status = 'pending'
RETURNING id::text, out_trade_no, amount_total, currency, status
`, input.VerificationID, input.MerchantID, input.UserID, outTradeNo, input.AmountTotal, currency).Scan(
		&result.ID,
		&result.OutTradeNo,
		&result.AmountTotal,
		&result.Currency,
		&result.Status,
	)
	return result, err
}

func (m *VerificationModel) MarkVerificationPaymentPaid(ctx context.Context, input MarkVerificationPaymentPaidInput) (VerificationPaymentResult, error) {
	var result VerificationPaymentResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var verificationType string
		var paidAt sql.NullTime
		if strings.TrimSpace(input.SuccessTime) != "" {
			if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(input.SuccessTime)); err == nil {
				paidAt = sql.NullTime{Time: parsed, Valid: true}
			}
		}
		err := tx.QueryRowContext(ctx, `
UPDATE verification_payment_orders
SET transaction_id = $2,
    amount_total = CASE WHEN $3 > 0 THEN $3 ELSE amount_total END,
    status = 'paid',
    notify_payload = $4,
    paid_at = COALESCE($5, paid_at, now()),
    updated_at = now()
WHERE out_trade_no = $1
  AND status IN ('pending', 'paid')
  AND ($3 <= 0 OR amount_total = $3)
RETURNING id::text, verification_id::text, merchant_id::text, status
`, input.OutTradeNo, input.TransactionID, input.AmountTotal, input.NotifyPayload, paidAt).Scan(
			&result.OrderID,
			&result.VerificationID,
			&result.MerchantID,
			&result.Status,
		)
		if err != nil {
			return err
		}

		err = tx.QueryRowContext(ctx, `
UPDATE verifications
SET status = 'verified',
    reviewed_at = COALESCE(reviewed_at, now()),
    expires_at = COALESCE(expires_at, COALESCE($2, now()) + interval '1 year'),
    updated_at = now()
WHERE id = $1
  AND status IN ('payment_pending', 'verified')
RETURNING verification_type
`, result.VerificationID, paidAt).Scan(&verificationType)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE merchants SET verification_status = 'verified', updated_at = now() WHERE id = $1`, result.MerchantID); err != nil {
			return err
		}
		// 微信支付回调可能重复发送；权益只在该认证还没有认证来源权益时发放，避免重复赠送额度和置顶券。
		var alreadyGranted bool
		if err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM credit_records
  WHERE merchant_id = $1
    AND source_type = 'verification'
    AND tag_code = $2
    AND revoked_at IS NULL
)
`, result.MerchantID, verificationType+"_verified").Scan(&alreadyGranted); err != nil {
			return err
		}
		if !alreadyGranted {
			if err := grantVerificationBenefits(ctx, tx, result.MerchantID, sql.NullString{}, verificationType, ""); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'verification_result', 'verification_payment_paid', $2, '认证支付成功', $3, $4, 'unread')
`, "merchant:"+result.MerchantID, result.VerificationID, verificationLabel(verificationType)+" 已支付并生效", verificationMessageTargetURL(result.MerchantID))
		return err
	})
	return result, err
}

func buildVerificationOutTradeNo(verificationID string) string {
	cleanID := strings.NewReplacer("-", "", "_", "").Replace(strings.TrimSpace(verificationID))
	if len(cleanID) > 24 {
		cleanID = cleanID[len(cleanID)-24:]
	}
	return fmt.Sprintf("VP%s%s", time.Now().Format("20060102150405"), cleanID)
}
