package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	PaymentBusinessContactUnlock = "contact_unlock"
	PaymentBusinessVIP           = "vip"
)

type PendingPaymentOrder struct {
	BusinessType    string
	BusinessOrderID string
	OutTradeNo      string
	AmountTotal     int64
	CreatedAt       time.Time
	ExpiresAt       sql.NullTime
}

type PaymentReconciliationModel struct {
	db *sql.DB
}

func NewPaymentReconciliationModel(db *sql.DB) *PaymentReconciliationModel {
	return &PaymentReconciliationModel{db: db}
}

func (m *PaymentReconciliationModel) ListPendingPaymentOrders(ctx context.Context, createdBefore time.Time, limit int64) ([]PendingPaymentOrder, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := m.db.QueryContext(ctx, `
SELECT business_type, business_order_id, out_trade_no, amount_total, created_at, expires_at
FROM (
  SELECT
    'contact_unlock'::text,
    id::text,
    out_trade_no,
    price_cent::bigint,
    created_at,
    expires_at
  FROM resource_contact_unlock_orders
  WHERE status = 'pending' AND created_at <= $1

  UNION ALL

  SELECT
    'vip'::text,
    id::text,
    out_trade_no,
    actual_price_cent::bigint,
    created_at,
    NULL::timestamptz
  FROM vip_orders
  WHERE status = 'pending' AND created_at <= $1
) pending_orders
ORDER BY created_at
LIMIT $2
`, createdBefore, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]PendingPaymentOrder, 0)
	for rows.Next() {
		var order PendingPaymentOrder
		if err := rows.Scan(
			&order.BusinessType,
			&order.BusinessOrderID,
			&order.OutTradeNo,
			&order.AmountTotal,
			&order.CreatedAt,
			&order.ExpiresAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (m *PaymentReconciliationModel) MarkPaymentOrderClosed(ctx context.Context, businessType string, businessOrderID string, outTradeNo string) error {
	var query string
	switch strings.TrimSpace(businessType) {
	case PaymentBusinessContactUnlock:
		query = `
UPDATE resource_contact_unlock_orders
SET status = 'closed', updated_at = now()
WHERE id = $1 AND out_trade_no = $2 AND status = 'pending'`
	case PaymentBusinessVIP:
		query = `
UPDATE vip_orders
SET status = 'closed', closed_at = COALESCE(closed_at, now()), updated_at = now()
WHERE id = $1 AND out_trade_no = $2 AND status = 'pending'`
	default:
		return fmt.Errorf("不支持的支付业务类型: %s", businessType)
	}

	result, err := m.db.ExecContext(ctx, query, strings.TrimSpace(businessOrderID), strings.TrimSpace(outTradeNo))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		// 其他实例可能已先完成支付或关单；读取当前状态比盲目覆盖更安全。
		return sql.ErrNoRows
	}
	return nil
}
