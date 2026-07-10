package model

import (
	"context"
	"database/sql"
)

type entitlementUsageInput struct {
	EntitlementID         string
	MerchantID            string
	EntitlementType       string
	ActionType            string
	Amount                int64
	ResourceID            string
	BeforeRemainingAmount int64
	AfterRemainingAmount  int64
	Snapshot              JSONMap
}

func recordEntitlementUsageTx(ctx context.Context, tx *sql.Tx, input entitlementUsageInput) error {
	if input.Amount <= 0 {
		input.Amount = 1
	}
	if input.Snapshot == nil {
		input.Snapshot = JSONMap{}
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO merchant_entitlement_usage_records (
  entitlement_id,
  merchant_id,
  entitlement_type,
  action_type,
  amount,
  resource_id,
  before_remaining_amount,
  after_remaining_amount,
  snapshot
)
VALUES (
  NULLIF($1, '')::bigint,
  NULLIF($2, '')::bigint,
  $3,
  $4,
  $5,
  NULLIF($6, '')::bigint,
  $7,
  $8,
  $9
)
`, input.EntitlementID, input.MerchantID, input.EntitlementType, input.ActionType, input.Amount, input.ResourceID, input.BeforeRemainingAmount, input.AfterRemainingAmount, input.Snapshot)
	return err
}
