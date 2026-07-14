package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	ContactUnlockSourcePaid      = "paid"
	ContactUnlockSourceVIP       = "vip"
	ContactUnlockSourceLoginFree = "login_free"
	ContactUnlockSourceOwner     = "owner"
	ContactUnlockSourceManual    = "manual"
)

type ResourceContactUnlockModel struct {
	db *sql.DB
}

func NewResourceContactUnlockModel(db *sql.DB) *ResourceContactUnlockModel {
	return &ResourceContactUnlockModel{db: db}
}

type CreateContactUnlockOrderInput struct {
	ResourceID       string
	UserID           string
	ViewerMerchantID string
}

type ContactUnlockOrderResult struct {
	ID              string
	ResourceID      string
	OutTradeNo      string
	Status          string
	AlreadyUnlocked bool
	PriceCent       int64
	Currency        string
	ResourceTitle   string
}

type ContactUnlockState struct {
	Unlocked  bool
	ID        string
	Source    string
	OrderID   string
	ExpiresAt time.Time
}

type VIPManagedMerchant struct {
	MerchantID string
}

type ContactUnlockInput struct {
	ResourceID       string
	UserID           string
	ViewerMerchantID string
	SourceType       string
	OrderID          string
	StartsAt         time.Time
	ExpiresAt        time.Time
}

type ContactUnlockResult struct {
	ID         string
	ResourceID string
	UserID     string
	SourceType string
	ExpiresAt  time.Time
}

type GetContactUnlockPaymentContextInput struct {
	ResourceID string
	OrderID    string
	UserID     string
}

type ContactUnlockPaymentContext struct {
	OrderID       string
	ResourceID    string
	UserID        string
	OpenID        string
	Status        string
	OutTradeNo    string
	AmountTotal   int64
	Currency      string
	ResourceTitle string
}

type CreateContactUnlockPaymentOrderInput struct {
	ResourceID string
	OrderID    string
	UserID     string
}

type ContactUnlockPaymentOrder struct {
	ID            string
	ResourceID    string
	OutTradeNo    string
	AmountTotal   int64
	Currency      string
	Status        string
	ResourceTitle string
}

type MarkContactUnlockOrderPaidInput struct {
	OutTradeNo    string
	TransactionID string
	AmountTotal   int64
	SuccessTime   string
	NotifyPayload JSONMap
}

type ContactUnlockPaymentResult struct {
	OrderID    string
	ResourceID string
	Status     string
}

const createContactUnlockOrderResourceSQL = `
SELECT
  r.id::text,
  r.type_code,
  r.title,
  rtc.commercial_rules
FROM resources r
JOIN merchants m ON m.id = r.merchant_id
JOIN resource_type_configs rtc ON rtc.id = r.resource_type_config_id
WHERE r.id = $1
  AND r.status = 'published'
  AND r.deleted_at IS NULL
  AND m.deleted_at IS NULL
  AND m.status = 'active'
  AND (r.expires_at IS NULL OR r.expires_at > now())
LIMIT 1
`

const insertContactUnlockOrderSQL = `
INSERT INTO resource_contact_unlock_orders (
  resource_id,
  buyer_user_id,
  buyer_merchant_id,
  type_code,
  out_trade_no,
  price_cent,
  currency,
  commercial_rules_snapshot,
  expires_at
)
VALUES ($1, $2, NULLIF($3, '')::bigint, $4, $5, $6, $7, $8, now() + interval '30 minutes')
RETURNING id::text, out_trade_no, price_cent, currency, status
`

const getContactUnlockPaymentContextSQL = `
SELECT
  o.id::text,
  o.resource_id::text,
  o.buyer_user_id::text,
  COALESCE(u.wechat_openid, ''),
  o.status,
  o.out_trade_no,
  o.price_cent,
  o.currency,
  r.title
FROM resource_contact_unlock_orders o
JOIN resources r ON r.id = o.resource_id
JOIN users u ON u.id = $3
WHERE o.resource_id = $1
  AND o.id = $2
  AND o.buyer_user_id = $3
LIMIT 1
`

const createContactUnlockPaymentOrderSQL = `
UPDATE resource_contact_unlock_orders
SET buyer_user_id = $3,
    updated_at = now()
WHERE resource_id = $1
  AND id = $2
  AND buyer_user_id = $3
  AND status = 'pending'
  AND expires_at > now()
RETURNING id::text,
  resource_id::text,
  out_trade_no,
  price_cent,
  currency,
  status,
  (SELECT title FROM resources WHERE resources.id = resource_contact_unlock_orders.resource_id) AS resource_title
`

const selectContactUnlockOrderForPaymentSQL = `
SELECT
  id::text,
  resource_id::text,
  buyer_user_id::text,
  buyer_merchant_id::text,
  status,
  price_cent,
  commercial_rules_snapshot
FROM resource_contact_unlock_orders
WHERE out_trade_no = $1
  AND status IN ('pending', 'paid')
FOR UPDATE
`

const updateContactUnlockOrderPaidSQL = `
UPDATE resource_contact_unlock_orders
SET status = 'paid',
    paid_at = $2,
    updated_at = now()
WHERE id = $1
`

const hasActiveContactUnlockSQL = `
SELECT id::text, source_type, COALESCE(order_id::text, ''), expires_at
FROM resource_contact_unlocks
WHERE resource_id = $1
  AND user_id = $2
  AND starts_at <= now()
  AND expires_at > now()
ORDER BY expires_at DESC
LIMIT 1
`

const upsertContactUnlockSQL = `
INSERT INTO resource_contact_unlocks (
  resource_id,
  user_id,
  viewer_merchant_id,
  source_type,
  order_id,
  starts_at,
  expires_at
)
VALUES ($1, $2, NULLIF($3, '')::bigint, $4, NULLIF($5, '')::bigint, $6, $7)
RETURNING id::text, resource_id::text, user_id::text, source_type, expires_at
`

const findActiveVIPManagedMerchantSQL = `
SELECT m.id::text
FROM merchant_admin_bindings mab
JOIN merchants m ON m.id = mab.merchant_id
JOIN merchant_vip_subscriptions mvs ON mvs.merchant_id = m.id
WHERE mab.user_id = $1
  AND mab.status = 'active'
  AND m.deleted_at IS NULL
  AND m.status = 'active'
  AND mvs.status = 'active'
  AND mvs.starts_at <= now()
  AND mvs.expires_at > now()
ORDER BY mvs.expires_at DESC
LIMIT 1
`

func (m *ResourceContactUnlockModel) CreateContactUnlockOrder(ctx context.Context, input CreateContactUnlockOrderInput) (ContactUnlockOrderResult, error) {
	var result ContactUnlockOrderResult
	var typeCode string
	var commercialRules JSONMap
	err := m.db.QueryRowContext(ctx, createContactUnlockOrderResourceSQL, strings.TrimSpace(input.ResourceID)).Scan(
		&result.ResourceID,
		&typeCode,
		&result.ResourceTitle,
		&commercialRules,
	)
	if err != nil {
		return ContactUnlockOrderResult{}, err
	}
	rules := ContactUnlockRulesFromCommercialRules(commercialRules)
	state, err := m.HasActiveContactUnlock(ctx, result.ResourceID, strings.TrimSpace(input.UserID))
	if err != nil {
		return ContactUnlockOrderResult{}, err
	}
	result.PriceCent = rules.PriceCent
	result.Currency = strings.TrimSpace(rules.Currency)
	if result.Currency == "" {
		result.Currency = DefaultContactUnlockCurrency
	}
	if state.Unlocked {
		result.AlreadyUnlocked = true
		result.Status = PaymentOrderStatusPaid
		return result, nil
	}
	err = m.db.QueryRowContext(ctx, insertContactUnlockOrderSQL,
		result.ResourceID,
		strings.TrimSpace(input.UserID),
		strings.TrimSpace(input.ViewerMerchantID),
		typeCode,
		buildContactUnlockOutTradeNo(result.ResourceID+input.UserID),
		result.PriceCent,
		result.Currency,
		commercialRules,
	).Scan(
		&result.ID,
		&result.OutTradeNo,
		&result.PriceCent,
		&result.Currency,
		&result.Status,
	)
	return result, err
}

func (m *ResourceContactUnlockModel) GetContactUnlockPaymentContext(ctx context.Context, input GetContactUnlockPaymentContextInput) (ContactUnlockPaymentContext, error) {
	var result ContactUnlockPaymentContext
	err := m.db.QueryRowContext(ctx, getContactUnlockPaymentContextSQL,
		strings.TrimSpace(input.ResourceID),
		strings.TrimSpace(input.OrderID),
		strings.TrimSpace(input.UserID),
	).Scan(
		&result.OrderID,
		&result.ResourceID,
		&result.UserID,
		&result.OpenID,
		&result.Status,
		&result.OutTradeNo,
		&result.AmountTotal,
		&result.Currency,
		&result.ResourceTitle,
	)
	return result, err
}

func (m *ResourceContactUnlockModel) CreateContactUnlockPaymentOrder(ctx context.Context, input CreateContactUnlockPaymentOrderInput) (ContactUnlockPaymentOrder, error) {
	var result ContactUnlockPaymentOrder
	err := m.db.QueryRowContext(ctx, createContactUnlockPaymentOrderSQL,
		strings.TrimSpace(input.ResourceID),
		strings.TrimSpace(input.OrderID),
		strings.TrimSpace(input.UserID),
	).Scan(
		&result.ID,
		&result.ResourceID,
		&result.OutTradeNo,
		&result.AmountTotal,
		&result.Currency,
		&result.Status,
		&result.ResourceTitle,
	)
	return result, err
}

func (m *ResourceContactUnlockModel) MarkContactUnlockOrderPaid(ctx context.Context, input MarkContactUnlockOrderPaidInput) (ContactUnlockPaymentResult, error) {
	var result ContactUnlockPaymentResult
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var userID string
		var viewerMerchantID sql.NullString
		var currentStatus string
		var priceCent int64
		var rulesSnapshot JSONMap
		if err := tx.QueryRowContext(ctx, selectContactUnlockOrderForPaymentSQL, strings.TrimSpace(input.OutTradeNo)).Scan(
			&result.OrderID,
			&result.ResourceID,
			&userID,
			&viewerMerchantID,
			&currentStatus,
			&priceCent,
			&rulesSnapshot,
		); err != nil {
			return err
		}
		if input.AmountTotal > 0 && input.AmountTotal != priceCent {
			return sql.ErrNoRows
		}
		if currentStatus == PaymentOrderStatusPaid {
			result.Status = PaymentOrderStatusPaid
			return nil
		}
		if currentStatus != PaymentOrderStatusPending {
			return sql.ErrNoRows
		}
		paidAt := parseVIPPaymentTime(input.SuccessTime)
		if _, err := tx.ExecContext(ctx, updateContactUnlockOrderPaidSQL, result.OrderID, paidAt); err != nil {
			return err
		}
		rules := ContactUnlockRulesFromCommercialRules(rulesSnapshot)
		expiresAt := paidAt.AddDate(0, 0, int(rules.RepeatUnlockDays))
		var unlock ContactUnlockResult
		// 支付成功后按订单快照写入解锁记录，避免后台之后调整价格或有效期影响已经支付的用户权益。
		if err := tx.QueryRowContext(ctx, upsertContactUnlockSQL,
			result.ResourceID,
			userID,
			nullStringToString(viewerMerchantID),
			ContactUnlockSourcePaid,
			result.OrderID,
			paidAt,
			expiresAt,
		).Scan(&unlock.ID, &unlock.ResourceID, &unlock.UserID, &unlock.SourceType, &unlock.ExpiresAt); err != nil {
			return err
		}
		result.Status = PaymentOrderStatusPaid
		return nil
	})
	return result, err
}

func (m *ResourceContactUnlockModel) HasActiveContactUnlock(ctx context.Context, resourceID string, userID string) (ContactUnlockState, error) {
	var state ContactUnlockState
	if strings.TrimSpace(resourceID) == "" || strings.TrimSpace(userID) == "" {
		return state, nil
	}
	err := m.db.QueryRowContext(ctx, hasActiveContactUnlockSQL, strings.TrimSpace(resourceID), strings.TrimSpace(userID)).Scan(
		&state.ID,
		&state.Source,
		&state.OrderID,
		&state.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return ContactUnlockState{}, nil
		}
		return ContactUnlockState{}, err
	}
	state.Unlocked = true
	return state, nil
}

func (m *ResourceContactUnlockModel) UpsertContactUnlock(ctx context.Context, input ContactUnlockInput) (ContactUnlockResult, error) {
	startsAt := input.StartsAt
	if startsAt.IsZero() {
		startsAt = time.Now()
	}
	expiresAt := input.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = startsAt.AddDate(0, 0, int(DefaultContactRepeatUnlockDays))
	}
	var result ContactUnlockResult
	err := m.db.QueryRowContext(ctx, upsertContactUnlockSQL,
		strings.TrimSpace(input.ResourceID),
		strings.TrimSpace(input.UserID),
		strings.TrimSpace(input.ViewerMerchantID),
		strings.TrimSpace(input.SourceType),
		strings.TrimSpace(input.OrderID),
		startsAt,
		expiresAt,
	).Scan(&result.ID, &result.ResourceID, &result.UserID, &result.SourceType, &result.ExpiresAt)
	return result, err
}

func (m *ResourceContactUnlockModel) FindActiveVIPManagedMerchant(ctx context.Context, userID string) (VIPManagedMerchant, error) {
	var result VIPManagedMerchant
	err := m.db.QueryRowContext(ctx, findActiveVIPManagedMerchantSQL, strings.TrimSpace(userID)).Scan(&result.MerchantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return VIPManagedMerchant{}, nil
		}
		return VIPManagedMerchant{}, err
	}
	return result, nil
}

func buildContactUnlockOutTradeNo(seed string) string {
	cleanID := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.TrimSpace(seed))
	if len(cleanID) > 10 {
		cleanID = cleanID[len(cleanID)-10:]
	}
	outTradeNo := fmt.Sprintf("CU%d%s", time.Now().UnixNano(), cleanID)
	if len(outTradeNo) > 32 {
		outTradeNo = outTradeNo[:32]
	}
	return outTradeNo
}
