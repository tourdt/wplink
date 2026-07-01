package model

import (
	"context"
	"database/sql"
	"net/url"
	"time"
)

type SubmitVerificationInput struct {
	MerchantID       string
	ApplicantUserID  string
	VerificationType string
	BusinessName     string
	LicenseURL       string
	StorefrontURL    string
	Materials        JSONMap
}

type VerificationResult struct {
	ID     string
	Status string
}

type VerificationBrief struct {
	ID               string
	VerificationType string
	Status           string
	ReviewedAt       string
}

type PendingVerificationsFilter struct {
	Page     int64
	PageSize int64
}

type PendingVerificationItem struct {
	ID               string
	MerchantID       string
	MerchantName     string
	VerificationType string
	Status           string
	SubmittedAt      string
}

type ListPendingVerificationsResult struct {
	Items    []PendingVerificationItem
	Page     int64
	PageSize int64
	Total    int64
}

type ReviewVerificationInput struct {
	VerificationID string
	ReviewerID     string
	Action         string
	ReviewNote     string
	RequirePayment bool
}

type ReviewVerificationResult struct {
	ID     string
	Status string
}

type VerificationModel struct {
	db *sql.DB
}

func NewVerificationModel(db *sql.DB) *VerificationModel {
	return &VerificationModel{db: db}
}

func (m *VerificationModel) SubmitVerification(ctx context.Context, input SubmitVerificationInput) (VerificationResult, error) {
	var result VerificationResult
	err := m.db.QueryRowContext(ctx, `
INSERT INTO verifications (
  merchant_id,
  verification_type,
  status,
  applicant_user_id,
  business_name,
  license_url,
  storefront_url,
  materials
)
VALUES ($1, $2, 'pending', $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7)
RETURNING id::text, status
`, input.MerchantID, input.VerificationType, input.ApplicantUserID, input.BusinessName, input.LicenseURL, input.StorefrontURL, input.Materials).Scan(&result.ID, &result.Status)
	return result, err
}

func (m *VerificationModel) GetLatestVerification(ctx context.Context, merchantID string) (VerificationBrief, error) {
	var result VerificationBrief
	var reviewedAt sql.NullTime
	err := m.db.QueryRowContext(ctx, `
SELECT id::text, verification_type, status, reviewed_at
FROM verifications
WHERE merchant_id = $1
ORDER BY submitted_at DESC, created_at DESC
LIMIT 1
`, merchantID).Scan(&result.ID, &result.VerificationType, &result.Status, &reviewedAt)
	if reviewedAt.Valid {
		result.ReviewedAt = reviewedAt.Time.Format(time.RFC3339)
	}
	return result, err
}

func (m *VerificationModel) GetVerificationBillingConfigForVerification(ctx context.Context, verificationID string) (VerificationBillingConfig, error) {
	var cityCode string
	var config JSONMap
	err := m.db.QueryRowContext(ctx, `
SELECT
  cs.code,
  COALESCE(cs.config->'verificationBilling', '{}'::jsonb)
FROM verifications v
JOIN merchants m ON m.id = v.merchant_id
JOIN city_stations cs ON cs.id = m.city_station_id
WHERE v.id = $1
LIMIT 1
`, verificationID).Scan(&cityCode, &config)
	if err != nil {
		return VerificationBillingConfig{}, err
	}
	return VerificationBillingConfigFromJSON(cityCode, config), nil
}

func (m *VerificationModel) ListPendingVerifications(ctx context.Context, filter PendingVerificationsFilter) (ListPendingVerificationsResult, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	rows, err := m.db.QueryContext(ctx, `
SELECT
  v.id::text,
  v.merchant_id::text,
  m.name,
  v.verification_type,
  v.status,
  v.submitted_at,
  COUNT(*) OVER() AS total
FROM verifications v
JOIN merchants m ON m.id = v.merchant_id
WHERE v.status = 'pending'
ORDER BY v.submitted_at DESC
LIMIT $1 OFFSET $2
`, pageSize, offset)
	if err != nil {
		return ListPendingVerificationsResult{}, err
	}
	defer rows.Close()
	var result ListPendingVerificationsResult
	result.Page = page
	result.PageSize = pageSize
	for rows.Next() {
		var item PendingVerificationItem
		var submittedAt time.Time
		if err := rows.Scan(&item.ID, &item.MerchantID, &item.MerchantName, &item.VerificationType, &item.Status, &submittedAt, &result.Total); err != nil {
			return ListPendingVerificationsResult{}, err
		}
		item.SubmittedAt = submittedAt.Format(time.RFC3339)
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ListPendingVerificationsResult{}, err
	}
	return result, nil
}

func (m *VerificationModel) ReviewVerification(ctx context.Context, input ReviewVerificationInput) (ReviewVerificationResult, error) {
	var result ReviewVerificationResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var merchantID string
		var resourceID sql.NullString
		var verificationType string
		var nextStatus string
		switch input.Action {
		case "approve":
			if input.RequirePayment {
				nextStatus = VerificationStatusPaymentPending
			} else {
				nextStatus = VerificationStatusVerified
			}
		case "reject":
			nextStatus = VerificationStatusRejected
		case "revoke":
			nextStatus = VerificationStatusRevoked
		}

		err := tx.QueryRowContext(ctx, `
UPDATE verifications
SET status = $2, review_note = NULLIF($3, ''), reviewed_by = $4, reviewed_at = now(), updated_at = now()
WHERE id = $1
RETURNING id::text, merchant_id::text, resource_id::text, verification_type, status
`, input.VerificationID, nextStatus, input.ReviewNote, input.ReviewerID).Scan(&result.ID, &merchantID, &resourceID, &verificationType, &result.Status)
		if err != nil {
			return err
		}

		if input.Action == "approve" && !input.RequirePayment {
			if _, err := tx.ExecContext(ctx, `UPDATE merchants SET verification_status = 'verified', updated_at = now() WHERE id = $1`, merchantID); err != nil {
				return err
			}
			if resourceID.Valid {
				if _, err := tx.ExecContext(ctx, `UPDATE resources SET is_verified = true, updated_at = now() WHERE id = $1`, resourceID.String); err != nil {
					return err
				}
			}
			if err := grantVerificationBenefits(ctx, tx, merchantID, resourceID, verificationType, input.ReviewerID); err != nil {
				return err
			}
		}
		if input.Action == "revoke" {
			if _, err := tx.ExecContext(ctx, `UPDATE merchants SET verification_status = 'unverified', updated_at = now() WHERE id = $1`, merchantID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE credit_records SET revoked_at = now() WHERE merchant_id = $1 AND source_type = 'verification' AND revoked_at IS NULL`, merchantID); err != nil {
				return err
			}
		}
		messageTitle := "认证审核通过"
		messageContent := verificationLabel(verificationType) + " 已通过"
		if input.Action == "approve" && input.RequirePayment {
			messageTitle = "认证资料审核通过"
			messageContent = verificationLabel(verificationType) + " 资料已通过，请完成认证费支付后生效"
		}
		if input.Action == "reject" {
			messageTitle = "认证审核驳回"
			messageContent = verificationLabel(verificationType) + " 未通过，请修改资料后重新提交"
		}
		if input.Action == "revoke" {
			messageTitle = "认证已撤销"
			messageContent = verificationLabel(verificationType) + " 已撤销"
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO messages (recipient_role_code, message_type, trigger_type, trigger_id, title, content, target_url, status)
VALUES ($1, 'verification_result', $2, $3, $4, $5, $6, 'unread')
`, "merchant:"+merchantID, "verification_"+input.Action, input.VerificationID, messageTitle, messageContent, verificationMessageTargetURL(merchantID))
		if err != nil {
			return err
		}
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   input.ReviewerID,
			OperatorRole: "platform_operator",
			Action:       "verification_" + input.Action,
			ObjectType:   "verification",
			ObjectID:     input.VerificationID,
			AfterSnapshot: JSONMap{
				"status":     result.Status,
				"reviewNote": input.ReviewNote,
			},
		})
	})
	return result, err
}

func verificationMessageTargetURL(merchantID string) string {
	values := url.Values{}
	values.Set("merchantId", merchantID)
	return "/pages/verification/index?" + values.Encode()
}

func grantVerificationBenefits(ctx context.Context, tx *sql.Tx, merchantID string, resourceID sql.NullString, verificationType string, reviewerID string) error {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO credit_records (merchant_id, resource_id, source_type, tag_code, tag_label, description, visibility, created_by)
VALUES ($1, NULLIF($2, '')::bigint, 'verification', $3, $4, '认证审核通过后自动生成', 'public', NULLIF($5, '')::bigint)
`, merchantID, nullStringValue(resourceID), verificationType+"_verified", verificationLabel(verificationType), reviewerID); err != nil {
		return err
	}
	for _, item := range []struct {
		entitlementType string
		totalAmount     int64
	}{
		{entitlementType: "publish_quota", totalAmount: 20},
		{entitlementType: "refresh_quota", totalAmount: 30},
	} {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_entitlements (merchant_id, entitlement_type, source_type, total_amount, remaining_amount)
VALUES ($1, $2, 'verification', $3, $3)
`, merchantID, item.entitlementType, item.totalAmount); err != nil {
			return err
		}
	}
	for i := 0; i < 3; i++ {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO top_vouchers (merchant_id, source_type, allowed_type_codes, top_duration_hours, status)
VALUES ($1, 'verification', '[]'::jsonb, 24, 'unused')
`, merchantID); err != nil {
			return err
		}
	}
	return nil
}

func verificationLabel(verificationType string) string {
	switch verificationType {
	case "factory":
		return "工厂认证"
	case "stall":
		return "档口认证"
	case "stockist":
		return "库存商认证"
	case "service_provider":
		return "服务商认证"
	default:
		return "商家认证"
	}
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
