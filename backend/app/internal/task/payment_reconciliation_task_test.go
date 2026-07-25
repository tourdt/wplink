package task

import (
	"context"
	"database/sql"
	"testing"
	"time"

	paymentlogic "wplink/backend/app/internal/logic/payment"
	"wplink/backend/app/internal/model"
)

type fakePaymentReconciliationStore struct {
	orders              []model.PendingPaymentOrder
	verificationMark    model.MarkVerificationPaymentPaidInput
	contactUnlockMark   model.MarkContactUnlockOrderPaidInput
	vipMark             model.MarkVIPOrderPaidInput
	closedBusinessType  string
	closedBusinessOrder string
	closedOutTradeNo    string
	markClosedErr       error
}

func (s *fakePaymentReconciliationStore) ListPendingPaymentOrders(ctx context.Context, createdBefore time.Time, limit int64) ([]model.PendingPaymentOrder, error) {
	return s.orders, nil
}

func (s *fakePaymentReconciliationStore) MarkPaymentOrderClosed(ctx context.Context, businessType string, businessOrderID string, outTradeNo string) error {
	s.closedBusinessType = businessType
	s.closedBusinessOrder = businessOrderID
	s.closedOutTradeNo = outTradeNo
	return s.markClosedErr
}

func (s *fakePaymentReconciliationStore) MarkVerificationPaymentPaid(ctx context.Context, input model.MarkVerificationPaymentPaidInput) (model.VerificationPaymentResult, error) {
	s.verificationMark = input
	return model.VerificationPaymentResult{OrderID: input.BusinessOrderID, Status: model.PaymentOrderStatusPaid}, nil
}

func (s *fakePaymentReconciliationStore) MarkContactUnlockOrderPaid(ctx context.Context, input model.MarkContactUnlockOrderPaidInput) (model.ContactUnlockPaymentResult, error) {
	s.contactUnlockMark = input
	return model.ContactUnlockPaymentResult{OrderID: input.BusinessOrderID, Status: model.PaymentOrderStatusPaid}, nil
}

func (s *fakePaymentReconciliationStore) MarkVIPOrderPaid(ctx context.Context, input model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error) {
	s.vipMark = input
	return model.VIPPaymentResult{OrderID: input.BusinessOrderID, Status: model.PaymentOrderStatusPaid}, nil
}

type fakePaymentOrderGateway struct {
	orders   map[string]paymentlogic.WechatPayOrder
	queryErr error
	closed   []string
	closeErr error
}

func (g *fakePaymentOrderGateway) QueryOrder(ctx context.Context, outTradeNo string) (paymentlogic.WechatPayOrder, error) {
	if g.queryErr != nil {
		return paymentlogic.WechatPayOrder{}, g.queryErr
	}
	return g.orders[outTradeNo], nil
}

func (g *fakePaymentOrderGateway) CloseOrder(ctx context.Context, outTradeNo string) error {
	g.closed = append(g.closed, outTradeNo)
	return g.closeErr
}

func TestPaymentReconciliationTaskRecoversSuccessfulVIPPayment(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	store := &fakePaymentReconciliationStore{
		orders: []model.PendingPaymentOrder{{
			BusinessType:    model.PaymentBusinessVIP,
			BusinessOrderID: "order-1",
			OutTradeNo:      "VIP202607250001",
			AmountTotal:     1990,
			CreatedAt:       now.Add(-5 * time.Minute),
		}},
	}
	gateway := &fakePaymentOrderGateway{orders: map[string]paymentlogic.WechatPayOrder{
		"VIP202607250001": {
			OutTradeNo:    "VIP202607250001",
			TransactionID: "wx-transaction-1",
			Attach:        "vip:order-1",
			TradeState:    "SUCCESS",
			SuccessTime:   now.Add(-time.Minute).Format(time.RFC3339),
			AmountTotal:   1990,
			RawPayload:    map[string]interface{}{"trade_state": "SUCCESS"},
		},
	}}
	reconciler := NewPaymentReconciliationTask(store, gateway, 2*time.Minute, 30*time.Minute, 100)
	reconciler.now = func() time.Time { return now }

	result, err := reconciler.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ScannedCount != 1 || result.PaidCount != 1 || result.FailedCount != 0 {
		t.Fatalf("result = %#v, want one recovered payment", result)
	}
	if store.vipMark.BusinessOrderID != "order-1" || store.vipMark.TransactionID != "wx-transaction-1" {
		t.Fatalf("vipMark = %#v, want reconciled payment", store.vipMark)
	}
}

func TestPaymentReconciliationTaskClosesExpiredUnpaidOrder(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	store := &fakePaymentReconciliationStore{
		orders: []model.PendingPaymentOrder{{
			BusinessType:    model.PaymentBusinessContactUnlock,
			BusinessOrderID: "order-2",
			OutTradeNo:      "CU202607250001",
			AmountTotal:     500,
			CreatedAt:       now.Add(-31 * time.Minute),
			ExpiresAt:       sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
		}},
		markClosedErr: sql.ErrNoRows,
	}
	gateway := &fakePaymentOrderGateway{orders: map[string]paymentlogic.WechatPayOrder{
		"CU202607250001": {
			OutTradeNo: "CU202607250001",
			TradeState: "NOTPAY",
		},
	}}
	reconciler := NewPaymentReconciliationTask(store, gateway, 2*time.Minute, 30*time.Minute, 100)
	reconciler.now = func() time.Time { return now }

	result, err := reconciler.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ClosedCount != 1 || result.FailedCount != 0 {
		t.Fatalf("result = %#v, want one idempotently closed order", result)
	}
	if len(gateway.closed) != 1 || gateway.closed[0] != "CU202607250001" {
		t.Fatalf("closed = %#v, want remote order closed first", gateway.closed)
	}
	if store.closedBusinessOrder != "order-2" {
		t.Fatalf("closed business order = %q, want order-2", store.closedBusinessOrder)
	}
}
