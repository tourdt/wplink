package model

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPaymentReconciliationModelListsOnlyActivePaymentBusinessOrders(t *testing.T) {
	db, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp),
	)
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	// 认证支付已下线；补偿查询只能扫描仍在线的联系方式解锁和 VIP 订单。
	mock.ExpectQuery(`(?s)SELECT business_type, business_order_id, out_trade_no, amount_total, created_at, expires_at\s+FROM \(\s+SELECT\s+'contact_unlock'::text AS business_type,\s+id::text AS business_order_id,\s+out_trade_no,\s+price_cent::bigint AS amount_total,\s+created_at,\s+expires_at`).
		WillReturnRows(sqlmock.NewRows([]string{
			"business_type", "business_order_id", "out_trade_no", "amount_total", "created_at", "expires_at",
		}))

	orders, err := NewPaymentReconciliationModel(db).ListPendingPaymentOrders(
		context.Background(),
		time.Date(2026, 8, 2, 18, 14, 2, 0, time.UTC),
		100,
	)
	if err != nil {
		t.Fatalf("ListPendingPaymentOrders() error = %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("ListPendingPaymentOrders() orders = %#v, want empty", orders)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectation not met: %v", err)
	}
}

func TestPaymentReconciliationModelRejectsRetiredVerificationOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	err = NewPaymentReconciliationModel(db).MarkPaymentOrderClosed(context.Background(), "verification", "order-1", "VERIFY202608020001")
	if err == nil || !strings.Contains(err.Error(), "不支持的支付业务类型") {
		t.Fatalf("MarkPaymentOrderClosed() error = %v, want unsupported-business error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("retired verification order should not execute SQL: %v", err)
	}
}
