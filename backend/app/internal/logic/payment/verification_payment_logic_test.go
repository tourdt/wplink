package payment

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestCreateVerificationPaymentCreatesWechatPrepay(t *testing.T) {
	store := &fakeVerificationPaymentStore{
		context: model.VerificationPaymentContext{
			VerificationID: "verification-1",
			MerchantID:     "merchant-1",
			UserID:         "user-1",
			OpenID:         "openid-1",
			Status:         model.VerificationStatusPaymentPending,
			Billing: model.VerificationBillingConfig{
				ChargeEnabled: true,
				FeeAmount:     29900,
				Currency:      "CNY",
			},
		},
		order: model.VerificationPaymentOrder{
			ID:          "payment-1",
			OutTradeNo:  "VP202606300001",
			AmountTotal: 29900,
			Currency:    "CNY",
			Status:      model.PaymentOrderStatusPending,
		},
	}
	gateway := &fakeWechatPayGateway{
		prepay: WechatPayParams{
			TimeStamp: "1893456000",
			NonceStr:  "nonce",
			Package:   "prepay_id=wx-prepay",
			SignType:  "RSA",
			PaySign:   "pay-sign",
		},
	}
	logic := NewCreateVerificationPaymentLogic(store, gateway)

	resp, err := logic.CreateVerificationPayment(context.Background(), CreateVerificationPaymentReq{
		MerchantID: " merchant-1 ", VerificationID: " verification-1 ", UserID: " user-1 ",
	})
	if err != nil {
		t.Fatalf("CreateVerificationPayment() error = %v", err)
	}

	if store.createInput.AmountTotal != 29900 || store.createInput.OpenID != "openid-1" {
		t.Fatalf("createInput = %#v, want amount and openid", store.createInput)
	}
	if gateway.prepayInput.OutTradeNo != "VP202606300001" || gateway.prepayInput.OpenID != "openid-1" {
		t.Fatalf("prepayInput = %#v, want order and openid", gateway.prepayInput)
	}
	if resp.OrderID != "payment-1" || resp.Payment.Package != "prepay_id=wx-prepay" {
		t.Fatalf("resp = %#v, want payment params", resp)
	}
}

func TestHandleWechatPayNotifyActivatesVerification(t *testing.T) {
	store := &fakeVerificationPaymentStore{
		markResult: model.VerificationPaymentResult{
			OrderID:        "payment-1",
			VerificationID: "verification-1",
			MerchantID:     "merchant-1",
			Status:         model.PaymentOrderStatusPaid,
		},
	}
	gateway := &fakeWechatPayGateway{
		notify: WechatPayNotification{
			OutTradeNo:    "VP202606300001",
			TransactionID: "wx-transaction-1",
			AmountTotal:   29900,
			SuccessTime:   "2026-06-30T12:00:00+08:00",
			RawPayload:    map[string]interface{}{"trade_state": "SUCCESS"},
		},
	}
	logic := NewWechatPayNotifyLogic(store, gateway)

	resp, err := logic.HandleNotify(context.Background(), WechatPayNotifyReq{Headers: map[string]string{"Wechatpay-Signature": "sig"}, Body: []byte(`{}`)})
	if err != nil {
		t.Fatalf("HandleNotify() error = %v", err)
	}

	if store.markInput.OutTradeNo != "VP202606300001" || store.markInput.TransactionID != "wx-transaction-1" {
		t.Fatalf("markInput = %#v, want notification identifiers", store.markInput)
	}
	if resp.Code != "SUCCESS" || resp.Message != "成功" {
		t.Fatalf("resp = %#v, want wechat success response", resp)
	}
}

type fakeVerificationPaymentStore struct {
	context     model.VerificationPaymentContext
	order       model.VerificationPaymentOrder
	createInput model.CreateVerificationPaymentOrderInput
	markInput   model.MarkVerificationPaymentPaidInput
	markResult  model.VerificationPaymentResult
}

func (s *fakeVerificationPaymentStore) GetVerificationPaymentContext(ctx context.Context, input model.GetVerificationPaymentContextInput) (model.VerificationPaymentContext, error) {
	return s.context, nil
}

func (s *fakeVerificationPaymentStore) CreateVerificationPaymentOrder(ctx context.Context, input model.CreateVerificationPaymentOrderInput) (model.VerificationPaymentOrder, error) {
	s.createInput = input
	return s.order, nil
}

func (s *fakeVerificationPaymentStore) MarkVerificationPaymentPaid(ctx context.Context, input model.MarkVerificationPaymentPaidInput) (model.VerificationPaymentResult, error) {
	s.markInput = input
	return s.markResult, nil
}

type fakeWechatPayGateway struct {
	prepayInput WechatPrepayInput
	prepay      WechatPayParams
	notify      WechatPayNotification
}

func (g *fakeWechatPayGateway) CreatePrepay(ctx context.Context, input WechatPrepayInput) (WechatPayParams, error) {
	g.prepayInput = input
	return g.prepay, nil
}

func (g *fakeWechatPayGateway) DecodeNotify(ctx context.Context, req WechatPayNotifyReq) (WechatPayNotification, error) {
	return g.notify, nil
}
