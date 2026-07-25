package payment

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type fakeUnifiedWechatPayNotifyStore struct {
	*fakeVerificationPaymentStore
	*fakeContactUnlockPaymentStore
	*fakeVIPPaymentStore
}

func newFakeUnifiedWechatPayNotifyStore() *fakeUnifiedWechatPayNotifyStore {
	return &fakeUnifiedWechatPayNotifyStore{
		fakeVerificationPaymentStore:  &fakeVerificationPaymentStore{},
		fakeContactUnlockPaymentStore: &fakeContactUnlockPaymentStore{},
		fakeVIPPaymentStore:           &fakeVIPPaymentStore{},
	}
}

func TestUnifiedWechatPayNotifyRoutesContactUnlockPayment(t *testing.T) {
	store := newFakeUnifiedWechatPayNotifyStore()
	store.fakeContactUnlockPaymentStore.markResult = model.ContactUnlockPaymentResult{
		OrderID:    "order-1",
		ResourceID: "resource-1",
		Status:     model.PaymentOrderStatusPaid,
	}
	gateway := &fakeWechatPayGateway{
		notify: WechatPayNotification{
			OutTradeNo:    "CU202607140001",
			TransactionID: "wx-transaction-1",
			Attach:        "contact_unlock:order-1",
			AmountTotal:   500,
			SuccessTime:   "2026-07-14T12:00:00Z",
			RawPayload:    map[string]interface{}{"trade_state": "SUCCESS"},
		},
	}

	resp, err := NewUnifiedWechatPayNotifyLogic(store, gateway).HandleNotify(context.Background(), WechatPayNotifyReq{})
	if err != nil {
		t.Fatalf("HandleNotify() error = %v", err)
	}
	if store.fakeContactUnlockPaymentStore.markInput.OutTradeNo != "CU202607140001" ||
		store.fakeContactUnlockPaymentStore.markInput.TransactionID != "wx-transaction-1" {
		t.Fatalf("markInput = %#v, want contact unlock payment notification", store.fakeContactUnlockPaymentStore.markInput)
	}
	if store.fakeVerificationPaymentStore.markInput.OutTradeNo != "" || store.fakeVIPPaymentStore.markInput.OutTradeNo != "" {
		t.Fatal("contact unlock notification must not be sent to another payment store")
	}
	if resp.Code != "SUCCESS" || resp.Message != "成功" {
		t.Fatalf("resp = %#v, want wechat success", resp)
	}
}

func TestUnifiedWechatPayNotifyRejectsMissingOrMismatchedAttach(t *testing.T) {
	tests := []struct {
		name   string
		notify WechatPayNotification
	}{
		{
			name: "missing attach",
			notify: WechatPayNotification{
				OutTradeNo: "CU202607140001", TransactionID: "wx-1", AmountTotal: 500,
			},
		},
		{
			name: "business prefix mismatch",
			notify: WechatPayNotification{
				OutTradeNo: "VIP202607140001", TransactionID: "wx-1", Attach: "contact_unlock:order-1", AmountTotal: 500,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			store := newFakeUnifiedWechatPayNotifyStore()
			_, err := NewUnifiedWechatPayNotifyLogic(store, &fakeWechatPayGateway{notify: testCase.notify}).
				HandleNotify(context.Background(), WechatPayNotifyReq{})
			if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
				t.Fatalf("error = %v, want validation failure", err)
			}
		})
	}
}
