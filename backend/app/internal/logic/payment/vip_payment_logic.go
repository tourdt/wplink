package payment

import (
	"context"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type VIPPaymentStore interface {
	GetVIPPaymentContext(ctx context.Context, input model.GetVIPPaymentContextInput) (model.VIPPaymentContext, error)
	CreateVIPPaymentOrder(ctx context.Context, input model.CreateVIPPaymentOrderInput) (model.VIPPaymentOrder, error)
	MarkVIPOrderPaid(ctx context.Context, input model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error)
}

type CreateVIPPaymentReq struct {
	MerchantID string
	OrderID    string
	UserID     string
}

type CreateVIPPaymentResp struct {
	OrderID string          `json:"orderId"`
	Status  string          `json:"status"`
	Payment WechatPayParams `json:"payment"`
}

type CreateVIPPaymentLogic struct {
	store          VIPPaymentStore
	gateway        WechatPayGateway
	devMockEnabled bool
}

func NewCreateVIPPaymentLogic(store VIPPaymentStore, gateway WechatPayGateway, devMockEnabled ...bool) *CreateVIPPaymentLogic {
	mockEnabled := false
	if len(devMockEnabled) > 0 {
		mockEnabled = devMockEnabled[0]
	}
	return &CreateVIPPaymentLogic{store: store, gateway: gateway, devMockEnabled: mockEnabled}
}

func (l *CreateVIPPaymentLogic) CreateVIPPayment(ctx context.Context, req CreateVIPPaymentReq) (CreateVIPPaymentResp, error) {
	input := model.GetVIPPaymentContextInput{
		MerchantID: strings.TrimSpace(req.MerchantID),
		OrderID:    strings.TrimSpace(req.OrderID),
		UserID:     strings.TrimSpace(req.UserID),
	}
	if input.MerchantID == "" || input.OrderID == "" || input.UserID == "" {
		return CreateVIPPaymentResp{}, errx.New(errx.CodeValidationFailed, "VIP 订单不存在或未登录")
	}
	if l.gateway == nil && !l.devMockEnabled {
		return CreateVIPPaymentResp{}, errx.New(errx.CodeInternalError, "微信支付暂未配置，请联系平台运营")
	}
	contextInfo, err := l.store.GetVIPPaymentContext(ctx, input)
	if err != nil {
		logx.Errorf("查询 VIP 支付上下文失败: merchantId=%s orderId=%s userId=%s err=%+v", input.MerchantID, input.OrderID, input.UserID, err)
		return CreateVIPPaymentResp{}, err
	}
	if contextInfo.Status != model.PaymentOrderStatusPending {
		return CreateVIPPaymentResp{}, errx.New(errx.CodeStateConflict, "VIP 订单已失效，请重新选择套餐")
	}
	if strings.TrimSpace(contextInfo.OpenID) == "" {
		return CreateVIPPaymentResp{}, errx.New(errx.CodeValidationFailed, "请先使用微信登录后再支付")
	}
	order, err := l.store.CreateVIPPaymentOrder(ctx, model.CreateVIPPaymentOrderInput{
		OrderID:    contextInfo.OrderID,
		MerchantID: contextInfo.MerchantID,
		UserID:     contextInfo.UserID,
	})
	if err != nil {
		logx.Errorf("创建 VIP 支付单失败: merchantId=%s orderId=%s userId=%s err=%+v", input.MerchantID, input.OrderID, input.UserID, err)
		return CreateVIPPaymentResp{}, err
	}
	if l.gateway == nil {
		return l.completeDevMockVIPPayment(ctx, order)
	}
	params, err := l.gateway.CreatePrepay(ctx, WechatPrepayInput{
		OutTradeNo:  order.OutTradeNo,
		Description: "VIP 会员 - " + order.PlanName,
		OpenID:      contextInfo.OpenID,
		AmountTotal: order.AmountTotal,
		Currency:    order.Currency,
		Attach:      "vip:" + order.ID,
	})
	if err != nil {
		logx.Errorf("创建微信 VIP 预支付失败: merchantId=%s orderId=%s userId=%s err=%+v", input.MerchantID, input.OrderID, input.UserID, err)
		return CreateVIPPaymentResp{}, err
	}
	return CreateVIPPaymentResp{OrderID: order.ID, Status: order.Status, Payment: params}, nil
}

func (l *CreateVIPPaymentLogic) completeDevMockVIPPayment(ctx context.Context, order model.VIPPaymentOrder) (CreateVIPPaymentResp, error) {
	result, err := l.store.MarkVIPOrderPaid(ctx, model.MarkVIPOrderPaidInput{
		OutTradeNo:    order.OutTradeNo,
		TransactionID: "mock-" + order.OutTradeNo,
		AmountTotal:   order.AmountTotal,
		SuccessTime:   nowFunc().Format(time.RFC3339),
		NotifyPayload: model.JSONMap{
			"trade_state":  "SUCCESS",
			"mock":         true,
			"out_trade_no": order.OutTradeNo,
		},
	})
	if err != nil {
		return CreateVIPPaymentResp{}, err
	}
	logx.Infof("开发模拟 VIP 支付已完成: merchantId=%s orderId=%s outTradeNo=%s", result.MerchantID, result.OrderID, order.OutTradeNo)
	return CreateVIPPaymentResp{OrderID: result.OrderID, Status: result.Status}, nil
}

type VIPWechatPayNotifyLogic struct {
	store   VIPPaymentStore
	gateway WechatPayGateway
}

func NewVIPWechatPayNotifyLogic(store VIPPaymentStore, gateway WechatPayGateway) *VIPWechatPayNotifyLogic {
	return &VIPWechatPayNotifyLogic{store: store, gateway: gateway}
}

func (l *VIPWechatPayNotifyLogic) HandleNotify(ctx context.Context, req WechatPayNotifyReq) (WechatPayNotifyResp, error) {
	if l.gateway == nil {
		return WechatPayNotifyResp{}, errx.New(errx.CodeInternalError, "微信支付暂未配置")
	}
	notification, err := l.gateway.DecodeNotify(ctx, req)
	if err != nil {
		return WechatPayNotifyResp{}, err
	}
	if strings.TrimSpace(notification.OutTradeNo) == "" || strings.TrimSpace(notification.TransactionID) == "" {
		return WechatPayNotifyResp{}, errx.New(errx.CodeValidationFailed, "支付通知数据不完整")
	}
	result, err := l.store.MarkVIPOrderPaid(ctx, model.MarkVIPOrderPaidInput{
		OutTradeNo:    notification.OutTradeNo,
		TransactionID: notification.TransactionID,
		AmountTotal:   notification.AmountTotal,
		SuccessTime:   notification.SuccessTime,
		NotifyPayload: model.JSONMap(notification.RawPayload),
	})
	if err != nil {
		logx.Errorf("处理 VIP 支付回调失败: orderId=%s outTradeNo=%s err=%+v", result.OrderID, notification.OutTradeNo, err)
		return WechatPayNotifyResp{}, err
	}
	return WechatPayNotifyResp{Code: "SUCCESS", Message: "成功"}, nil
}
