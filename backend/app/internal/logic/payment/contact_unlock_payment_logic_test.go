package payment

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestCreateContactUnlockPaymentUsesDevMock(t *testing.T) {
	store := &fakeContactUnlockPaymentStore{
		context: model.ContactUnlockPaymentContext{
			OrderID:       "order-1",
			ResourceID:    "resource-1",
			UserID:        "user-1",
			OpenID:        "openid-1",
			Status:        model.PaymentOrderStatusPending,
			OutTradeNo:    "contact_unlock_1",
			AmountTotal:   500,
			Currency:      "CNY",
			ResourceTitle: "求职需求",
		},
		order: model.ContactUnlockPaymentOrder{
			ID:            "order-1",
			ResourceID:    "resource-1",
			OutTradeNo:    "contact_unlock_1",
			AmountTotal:   500,
			Currency:      "CNY",
			Status:        model.PaymentOrderStatusPending,
			ResourceTitle: "求职需求",
		},
		markResult: model.ContactUnlockPaymentResult{
			OrderID:    "order-1",
			ResourceID: "resource-1",
			Status:     model.PaymentOrderStatusPaid,
		},
	}
	logic := NewCreateContactUnlockPaymentLogic(store, nil, true)

	resp, err := logic.CreateContactUnlockPayment(context.Background(), CreateContactUnlockPaymentReq{
		ResourceID: "resource-1",
		OrderID:    "order-1",
		UserID:     "user-1",
	})
	if err != nil {
		t.Fatalf("CreateContactUnlockPayment() error = %v", err)
	}
	if store.markInput.OutTradeNo != "contact_unlock_1" || store.markInput.TransactionID != "mock-contact_unlock_1" {
		t.Fatalf("markInput = %#v, want paid contact unlock", store.markInput)
	}
	if resp.Status != model.PaymentOrderStatusPaid || resp.Payment.Package != "" {
		t.Fatalf("resp = %#v, want paid mock contact unlock response", resp)
	}
}

func TestContactUnlockWechatPayNotifyMarksOrderPaid(t *testing.T) {
	store := &fakeContactUnlockPaymentStore{
		markResult: model.ContactUnlockPaymentResult{
			OrderID:    "order-1",
			ResourceID: "resource-1",
			Status:     model.PaymentOrderStatusPaid,
		},
	}
	gateway := &fakeWechatPayGateway{
		notify: WechatPayNotification{
			OutTradeNo:    "contact_unlock_1",
			TransactionID: "wx-transaction-1",
			AmountTotal:   500,
			SuccessTime:   "2026-07-14T12:00:00Z",
			RawPayload:    map[string]interface{}{"trade_state": "SUCCESS"},
		},
	}
	resp, err := NewContactUnlockWechatPayNotifyLogic(store, gateway).HandleNotify(context.Background(), WechatPayNotifyReq{})
	if err != nil {
		t.Fatalf("HandleNotify() error = %v", err)
	}
	if store.markInput.OutTradeNo != "contact_unlock_1" || store.markInput.TransactionID != "wx-transaction-1" {
		t.Fatalf("markInput = %#v, want contact unlock payment notification", store.markInput)
	}
	if resp.Code != "SUCCESS" || resp.Message != "成功" {
		t.Fatalf("resp = %#v, want wechat success", resp)
	}
}

type fakeContactUnlockPaymentStore struct {
	context     model.ContactUnlockPaymentContext
	order       model.ContactUnlockPaymentOrder
	createInput model.CreateContactUnlockPaymentOrderInput
	markInput   model.MarkContactUnlockOrderPaidInput
	markResult  model.ContactUnlockPaymentResult
}

func (s *fakeContactUnlockPaymentStore) GetContactUnlockPaymentContext(ctx context.Context, input model.GetContactUnlockPaymentContextInput) (model.ContactUnlockPaymentContext, error) {
	return s.context, nil
}

func (s *fakeContactUnlockPaymentStore) CreateContactUnlockPaymentOrder(ctx context.Context, input model.CreateContactUnlockPaymentOrderInput) (model.ContactUnlockPaymentOrder, error) {
	s.createInput = input
	return s.order, nil
}

func (s *fakeContactUnlockPaymentStore) MarkContactUnlockOrderPaid(ctx context.Context, input model.MarkContactUnlockOrderPaidInput) (model.ContactUnlockPaymentResult, error) {
	s.markInput = input
	return s.markResult, nil
}
