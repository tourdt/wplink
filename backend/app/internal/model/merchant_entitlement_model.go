package model

import (
	"context"
	"database/sql"
	"time"
)

type MerchantEntitlement struct {
	Type            string
	SourceType      string
	TotalAmount     int64
	UsedAmount      int64
	RemainingAmount int64
	ExpiresAt       string
}

type TopVoucher struct {
	ID               string
	Status           string
	TopDurationHours int64
	AllowedTypeCodes []string
	ExpiresAt        string
}

type RedeemTopVoucherResult struct {
	VoucherID  string
	ResourceID string
	Status     string
}

type GrantEntitlementInput struct {
	MerchantID      string
	EntitlementType string
	SourceType      string
	TotalAmount     int64
	ExpiresAt       string
	Reason          string
	OperatorID      string
}

type GrantEntitlementResult struct {
	ID string
}

type MerchantEntitlementModel struct {
	db *sql.DB
}

func NewMerchantEntitlementModel(db *sql.DB) *MerchantEntitlementModel {
	return &MerchantEntitlementModel{db: db}
}

func (m *MerchantEntitlementModel) ListMerchantEntitlements(ctx context.Context, merchantID string) ([]MerchantEntitlement, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT entitlement_type, source_type, total_amount, used_amount, remaining_amount, expires_at
FROM merchant_entitlements
WHERE merchant_id = $1
  AND status = 'active'
ORDER BY created_at DESC
`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MerchantEntitlement
	for rows.Next() {
		var item MerchantEntitlement
		var expiresAt sql.NullTime
		if err := rows.Scan(&item.Type, &item.SourceType, &item.TotalAmount, &item.UsedAmount, &item.RemainingAmount, &expiresAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			item.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *MerchantEntitlementModel) ListTopVouchers(ctx context.Context, merchantID string) ([]TopVoucher, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT id, status, top_duration_hours, allowed_type_codes, expires_at
FROM top_vouchers
WHERE merchant_id = $1
ORDER BY created_at DESC
`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TopVoucher
	for rows.Next() {
		var item TopVoucher
		var allowedTypeCodes JSONStringSlice
		var expiresAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Status, &item.TopDurationHours, &allowedTypeCodes, &expiresAt); err != nil {
			return nil, err
		}
		item.AllowedTypeCodes = []string(allowedTypeCodes)
		if expiresAt.Valid {
			item.ExpiresAt = expiresAt.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *MerchantEntitlementModel) GetTopVoucherMerchantID(ctx context.Context, voucherID string) (string, error) {
	var merchantID string
	err := m.db.QueryRowContext(ctx, `
SELECT merchant_id
FROM top_vouchers
WHERE id = $1
`, voucherID).Scan(&merchantID)
	return merchantID, err
}

func (m *MerchantEntitlementModel) RedeemTopVoucher(ctx context.Context, voucherID string, resourceID string) (RedeemTopVoucherResult, error) {
	var result RedeemTopVoucherResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
UPDATE top_vouchers tv
SET used_resource_id = r.id, used_at = now(), status = 'used'
FROM resources r
WHERE tv.id = $1
  AND r.id = $2
  AND tv.status = 'unused'
  AND (tv.expires_at IS NULL OR tv.expires_at > now())
  AND r.merchant_id = tv.merchant_id
  AND r.status = 'published'
  AND r.deleted_at IS NULL
  AND (jsonb_array_length(tv.allowed_type_codes) = 0 OR tv.allowed_type_codes ? r.type_code)
RETURNING tv.id, r.id, tv.status
`, voucherID, resourceID).Scan(&result.VoucherID, &result.ResourceID, &result.Status)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE resources SET refreshed_at = now(), updated_at = now() WHERE id = $1`, resourceID)
		return err
	})
	return result, err
}

func (m *MerchantEntitlementModel) GrantMerchantEntitlement(ctx context.Context, input GrantEntitlementInput) (GrantEntitlementResult, error) {
	var result GrantEntitlementResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `
INSERT INTO merchant_entitlements (
  merchant_id,
  entitlement_type,
  source_type,
  total_amount,
  remaining_amount,
  expires_at,
  status
)
VALUES ($1, $2, $3, $4, $4, NULLIF($5, '')::timestamptz, 'active')
RETURNING id
`, input.MerchantID, input.EntitlementType, input.SourceType, input.TotalAmount, input.ExpiresAt).Scan(&result.ID); err != nil {
			return err
		}
		return recordOperationLogTx(ctx, tx, OperationLogInput{
			OperatorID:   input.OperatorID,
			OperatorRole: "platform_operator",
			Action:       "entitlement_grant",
			ObjectType:   "merchant",
			ObjectID:     input.MerchantID,
			AfterSnapshot: JSONMap{
				"entitlementId":   result.ID,
				"entitlementType": input.EntitlementType,
				"sourceType":      input.SourceType,
				"totalAmount":     input.TotalAmount,
				"reason":          input.Reason,
			},
		})
	})
	return result, err
}
