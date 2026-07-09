package payment

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestCreateVIPPaymentCreatesWechatPrepay(t *testing.T) {
	store := &fakeVIPPaymentStore{
		context: model.VIPPaymentContext{
			OrderID:     "order-1",
			MerchantID:  "merchant-1",
			UserID:      "user-1",
			OpenID:      "openid-1",
			Status:      model.PaymentOrderStatusPending,
			OutTradeNo:  "VIP202607090001",
			AmountTotal: 1990,
			Currency:    "CNY",
			PlanName:    "VIP 月卡",
		},
		order: model.VIPPaymentOrder{
			ID:          "order-1",
			OutTradeNo:  "VIP202607090001",
			AmountTotal: 1990,
			Currency:    "CNY",
			Status:      model.PaymentOrderStatusPending,
			PlanName:    "VIP 月卡",
		},
	}
	gateway := &fakeWechatPayGateway{
		prepay: WechatPayParams{
			TimeStamp: "1893456000",
			NonceStr:  "nonce",
			Package:   "prepay_id=vip-prepay",
			SignType:  "RSA",
			PaySign:   "pay-sign",
		},
	}
	logic := NewCreateVIPPaymentLogic(store, gateway)

	resp, err := logic.CreateVIPPayment(context.Background(), CreateVIPPaymentReq{
		MerchantID: " merchant-1 ", OrderID: " order-1 ", UserID: " user-1 ",
	})
	if err != nil {
		t.Fatalf("CreateVIPPayment() error = %v", err)
	}
	if store.createInput.OrderID != "order-1" || store.createInput.UserID != "user-1" {
		t.Fatalf("createInput = %#v, want trimmed order and user", store.createInput)
	}
	if gateway.prepayInput.OutTradeNo != "VIP202607090001" || gateway.prepayInput.AmountTotal != 1990 || gateway.prepayInput.Attach != "vip:order-1" {
		t.Fatalf("prepayInput = %#v, want vip payment order", gateway.prepayInput)
	}
	if resp.OrderID != "order-1" || resp.Payment.Package != "prepay_id=vip-prepay" {
		t.Fatalf("resp = %#v, want vip prepay params", resp)
	}
}

func TestCreateVIPPaymentCreatesQuotaPackPrepay(t *testing.T) {
	store := &fakeVIPPaymentStore{
		context: model.VIPPaymentContext{
			OrderID:     "order-2",
			MerchantID:  "merchant-1",
			UserID:      "user-1",
			OpenID:      "openid-1",
			Status:      model.PaymentOrderStatusPending,
			OutTradeNo:  "VIP202607090002",
			AmountTotal: 2500,
			Currency:    "CNY",
			ProductType: model.VIPProductTypeQuotaPack,
			ProductName: "发布次数包",
		},
		order: model.VIPPaymentOrder{
			ID:          "order-2",
			OutTradeNo:  "VIP202607090002",
			AmountTotal: 2500,
			Currency:    "CNY",
			Status:      model.PaymentOrderStatusPending,
			ProductType: model.VIPProductTypeQuotaPack,
			ProductName: "发布次数包",
		},
	}
	gateway := &fakeWechatPayGateway{
		prepay: WechatPayParams{
			TimeStamp: "1893456000",
			NonceStr:  "nonce",
			Package:   "prepay_id=quota-prepay",
			SignType:  "RSA",
			PaySign:   "pay-sign",
		},
	}
	logic := NewCreateVIPPaymentLogic(store, gateway)

	resp, err := logic.CreateVIPPayment(context.Background(), CreateVIPPaymentReq{
		MerchantID: "merchant-1", OrderID: "order-2", UserID: "user-1",
	})
	if err != nil {
		t.Fatalf("CreateVIPPayment() error = %v", err)
	}
	if gateway.prepayInput.Description != "权益次数包 - 发布次数包" || gateway.prepayInput.AmountTotal != 2500 {
		t.Fatalf("prepayInput = %#v, want quota pack payment description", gateway.prepayInput)
	}
	if resp.OrderID != "order-2" || resp.Payment.Package != "prepay_id=quota-prepay" {
		t.Fatalf("resp = %#v, want quota prepay params", resp)
	}
}

func TestCreateVIPPaymentUsesDevMockWhenGatewayMissing(t *testing.T) {
	store := &fakeVIPPaymentStore{
		context: model.VIPPaymentContext{
			OrderID:     "order-1",
			MerchantID:  "merchant-1",
			UserID:      "user-1",
			OpenID:      "dev:openid",
			Status:      model.PaymentOrderStatusPending,
			OutTradeNo:  "VIP202607090002",
			AmountTotal: 1990,
			Currency:    "CNY",
			PlanName:    "VIP 月卡",
		},
		order: model.VIPPaymentOrder{
			ID:          "order-1",
			OutTradeNo:  "VIP202607090002",
			AmountTotal: 1990,
			Currency:    "CNY",
			Status:      model.PaymentOrderStatusPending,
			PlanName:    "VIP 月卡",
		},
		markResult: model.VIPPaymentResult{
			OrderID:    "order-1",
			MerchantID: "merchant-1",
			Status:     model.PaymentOrderStatusPaid,
		},
	}
	logic := NewCreateVIPPaymentLogic(store, nil, true)

	resp, err := logic.CreateVIPPayment(context.Background(), CreateVIPPaymentReq{
		MerchantID: "merchant-1", OrderID: "order-1", UserID: "user-1",
	})
	if err != nil {
		t.Fatalf("CreateVIPPayment() error = %v", err)
	}
	if store.markInput.OutTradeNo != "VIP202607090002" || store.markInput.TransactionID != "mock-VIP202607090002" {
		t.Fatalf("markInput = %#v, want mock vip payment", store.markInput)
	}
	if resp.Status != model.PaymentOrderStatusPaid || resp.Payment.Package != "" {
		t.Fatalf("resp = %#v, want paid mock response", resp)
	}
}

func TestCreateVIPPaymentReturnsFriendlyStateErrors(t *testing.T) {
	logic := NewCreateVIPPaymentLogic(&fakeVIPPaymentStore{}, nil)
	_, err := logic.CreateVIPPayment(context.Background(), CreateVIPPaymentReq{
		MerchantID: "merchant-1", OrderID: "order-1", UserID: "user-1",
	})
	if err == nil || errx.PublicMessage(err) != "微信支付暂未配置，请联系平台运营" {
		t.Fatalf("error = %v, want gateway config message", err)
	}

	store := &fakeVIPPaymentStore{
		context: model.VIPPaymentContext{
			OrderID:     "order-1",
			MerchantID:  "merchant-1",
			UserID:      "user-1",
			OpenID:      "openid-1",
			Status:      model.PaymentOrderStatusClosed,
			AmountTotal: 1990,
		},
	}
	_, err = NewCreateVIPPaymentLogic(store, &fakeWechatPayGateway{}).CreateVIPPayment(context.Background(), CreateVIPPaymentReq{
		MerchantID: "merchant-1", OrderID: "order-1", UserID: "user-1",
	})
	if err == nil || errx.PublicMessage(err) != "订单已失效，请重新选择商品" {
		t.Fatalf("error = %v, want invalid order message", err)
	}
}

type fakeVIPPaymentStore struct {
	context     model.VIPPaymentContext
	order       model.VIPPaymentOrder
	createInput model.CreateVIPPaymentOrderInput
	markInput   model.MarkVIPOrderPaidInput
	markResult  model.VIPPaymentResult
}

func (s *fakeVIPPaymentStore) GetVIPPaymentContext(ctx context.Context, input model.GetVIPPaymentContextInput) (model.VIPPaymentContext, error) {
	return s.context, nil
}

func (s *fakeVIPPaymentStore) CreateVIPPaymentOrder(ctx context.Context, input model.CreateVIPPaymentOrderInput) (model.VIPPaymentOrder, error) {
	s.createInput = input
	return s.order, nil
}

func (s *fakeVIPPaymentStore) MarkVIPOrderPaid(ctx context.Context, input model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error) {
	s.markInput = input
	return s.markResult, nil
}
