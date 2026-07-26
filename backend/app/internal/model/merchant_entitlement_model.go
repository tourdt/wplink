package model

import (
	"context"
	"database/sql"
	"time"
)

type MerchantEntitlement struct {
	ID               string
	Type             string
	SourceType       string
	Status           string
	TotalAmount      int64
	UsedAmount       int64
	RemainingAmount  int64
	TopDurationHours int64
	AllowedTypeCodes []string
	ExpiresAt        string
}

type TopVoucher struct {
	ID               string
	Status           string
	RemainingAmount  int64
	TopDurationHours int64
	AllowedTypeCodes []string
	ExpiresAt        string
}

type RedeemTopVoucherResult struct {
	VoucherID    string
	ResourceID   string
	Status       string
	TopExpiresAt string
}

type EntitlementUsageRecord struct {
	ID                    string
	EntitlementID         string
	EntitlementType       string
	ActionType            string
	Amount                int64
	ResourceID            string
	BeforeRemainingAmount int64
	AfterRemainingAmount  int64
	UsedAt                string
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
SELECT id::text, entitlement_type, source_type, status, total_amount, used_amount, remaining_amount, top_duration_hours, allowed_type_codes, expires_at
FROM merchant_entitlements
WHERE merchant_id = $1
  AND status = 'active'
  AND starts_at <= now()
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY expires_at NULLS LAST, created_at DESC
`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MerchantEntitlement
	for rows.Next() {
		var item MerchantEntitlement
		var expiresAt sql.NullTime
		var allowedTypeCodes JSONStringSlice
		if err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.SourceType,
			&item.Status,
			&item.TotalAmount,
			&item.UsedAmount,
			&item.RemainingAmount,
			&item.TopDurationHours,
			&allowedTypeCodes,
			&expiresAt,
		); err != nil {
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

func (m *MerchantEntitlementModel) ListTopVouchers(ctx context.Context, merchantID string) ([]TopVoucher, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  id::text,
  CASE WHEN remaining_amount > 0 THEN 'unused' ELSE status END AS voucher_status,
  remaining_amount,
  top_duration_hours,
  allowed_type_codes,
  expires_at
FROM merchant_entitlements
WHERE merchant_id = $1
  AND entitlement_type = $2
  AND status = 'active'
  AND remaining_amount > 0
  AND top_duration_hours > 0
  AND starts_at <= now()
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY expires_at NULLS LAST, created_at ASC
`, merchantID, EntitlementTypeTopVoucher)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TopVoucher
	for rows.Next() {
		var item TopVoucher
		var allowedTypeCodes JSONStringSlice
		var expiresAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Status, &item.RemainingAmount, &item.TopDurationHours, &allowedTypeCodes, &expiresAt); err != nil {
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
SELECT merchant_id::text
FROM merchant_entitlements
WHERE id = $1
  AND entitlement_type = $2
`, voucherID, EntitlementTypeTopVoucher).Scan(&merchantID)
	return merchantID, err
}

func (m *MerchantEntitlementModel) RedeemTopVoucher(ctx context.Context, voucherID string, resourceID string) (RedeemTopVoucherResult, error) {
	var result RedeemTopVoucherResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		redeemed, err := redeemTopVoucherTx(ctx, tx, voucherID, resourceID)
		if err != nil {
			return err
		}
		result = redeemed
		return nil
	})
	return result, err
}

func redeemTopVoucherTx(ctx context.Context, tx *sql.Tx, voucherID string, resourceID string) (RedeemTopVoucherResult, error) {
	var result RedeemTopVoucherResult
	var merchantID string
	var topDurationHours int64
	var beforeRemaining int64
	var afterRemaining int64
	var allowedTypeCodes JSONStringSlice
	var resourceTypeCode string
	var resourceTitle string
	err := tx.QueryRowContext(ctx, `
UPDATE merchant_entitlements me
SET used_amount = used_amount + 1,
    remaining_amount = remaining_amount - 1,
    updated_at = now()
FROM resources r
WHERE me.id = $1
  AND r.id = $2
  AND me.entitlement_type = $3
  AND me.status = 'active'
  AND me.remaining_amount > 0
  AND me.top_duration_hours > 0
  AND me.starts_at <= now()
  AND (me.expires_at IS NULL OR me.expires_at > now())
  AND r.merchant_id = me.merchant_id
  AND r.status = 'published'
  AND r.dealt_at IS NULL
  AND r.deleted_at IS NULL
  AND (jsonb_array_length(me.allowed_type_codes) = 0 OR me.allowed_type_codes ? r.type_code)
RETURNING
  me.id::text,
  me.merchant_id::text,
  r.id::text,
  me.top_duration_hours,
  me.remaining_amount + 1,
  me.remaining_amount,
  me.allowed_type_codes,
  r.type_code,
  r.title
`, voucherID, resourceID, EntitlementTypeTopVoucher).Scan(
		&result.VoucherID,
		&merchantID,
		&result.ResourceID,
		&topDurationHours,
		&beforeRemaining,
		&afterRemaining,
		&allowedTypeCodes,
		&resourceTypeCode,
		&resourceTitle,
	)
	if err != nil {
		return RedeemTopVoucherResult{}, err
	}
	var topExpiresAt time.Time
	if err := tx.QueryRowContext(ctx, `
UPDATE resources
SET top_started_at = now(),
    top_expires_at = now() + make_interval(hours => GREATEST($2, 1)::int),
    refreshed_at = now(),
    updated_at = now()
WHERE id = $1
  AND merchant_id = $3
  AND status = 'published'
  AND dealt_at IS NULL
  AND deleted_at IS NULL
RETURNING top_expires_at
`, result.ResourceID, topDurationHours, merchantID).Scan(&topExpiresAt); err != nil {
		return RedeemTopVoucherResult{}, err
	}
	result.Status = "used"
	result.TopExpiresAt = topExpiresAt.Format(time.RFC3339)
	return result, recordEntitlementUsageTx(ctx, tx, entitlementUsageInput{
		EntitlementID:         result.VoucherID,
		MerchantID:            merchantID,
		EntitlementType:       EntitlementTypeTopVoucher,
		ActionType:            ActionTypeTopResource,
		Amount:                1,
		ResourceID:            result.ResourceID,
		BeforeRemainingAmount: beforeRemaining,
		AfterRemainingAmount:  afterRemaining,
		Snapshot: JSONMap{
			"allowedTypeCodes": []string(allowedTypeCodes),
			"resourceTitle":    resourceTitle,
			"resourceTypeCode": resourceTypeCode,
			"topDurationHours": topDurationHours,
			"topExpiresAt":     result.TopExpiresAt,
		},
	})
}

func (m *MerchantEntitlementModel) ListMerchantEntitlementUsageRecords(ctx context.Context, merchantID string, entitlementID string) ([]EntitlementUsageRecord, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
  id::text,
  entitlement_id::text,
  entitlement_type,
  action_type,
  amount,
  COALESCE(resource_id::text, ''),
  before_remaining_amount,
  after_remaining_amount,
  used_at
FROM merchant_entitlement_usage_records
WHERE merchant_id = $1
  AND entitlement_id = $2
ORDER BY used_at DESC
LIMIT 100
`, merchantID, entitlementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []EntitlementUsageRecord
	for rows.Next() {
		var item EntitlementUsageRecord
		var usedAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.EntitlementID,
			&item.EntitlementType,
			&item.ActionType,
			&item.Amount,
			&item.ResourceID,
			&item.BeforeRemainingAmount,
			&item.AfterRemainingAmount,
			&usedAt,
		); err != nil {
			return nil, err
		}
		item.UsedAt = usedAt.Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
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
  status,
  top_duration_hours
)
VALUES ($1, $2, $3, $4, $4, NULLIF($5, '')::timestamptz, 'active', CASE WHEN $2 = $6 THEN 24 ELSE 0 END)
RETURNING id::text
`, input.MerchantID, input.EntitlementType, input.SourceType, input.TotalAmount, input.ExpiresAt, EntitlementTypeTopVoucher).Scan(&result.ID); err != nil {
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
