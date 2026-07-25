package payment

import (
	"context"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContactUnlockPaymentStore interface {
	GetContactUnlockPaymentContext(ctx context.Context, input model.GetContactUnlockPaymentContextInput) (model.ContactUnlockPaymentContext, error)
	CreateContactUnlockPaymentOrder(ctx context.Context, input model.CreateContactUnlockPaymentOrderInput) (model.ContactUnlockPaymentOrder, error)
	MarkContactUnlockOrderPaid(ctx context.Context, input model.MarkContactUnlockOrderPaidInput) (model.ContactUnlockPaymentResult, error)
}

type CreateContactUnlockPaymentReq struct {
	ResourceID string
	OrderID    string
	UserID     string
}

type CreateContactUnlockPaymentResp struct {
	OrderID string          `json:"orderId"`
	Status  string          `json:"status"`
	Payment WechatPayParams `json:"payment"`
}

type CreateContactUnlockPaymentLogic struct {
	store          ContactUnlockPaymentStore
	gateway        WechatPayGateway
	devMockEnabled bool
}

func NewCreateContactUnlockPaymentLogic(store ContactUnlockPaymentStore, gateway WechatPayGateway, devMockEnabled ...bool) *CreateContactUnlockPaymentLogic {
	mockEnabled := false
	if len(devMockEnabled) > 0 {
		mockEnabled = devMockEnabled[0]
	}
	return &CreateContactUnlockPaymentLogic{store: store, gateway: gateway, devMockEnabled: mockEnabled}
}

func (l *CreateContactUnlockPaymentLogic) CreateContactUnlockPayment(ctx context.Context, req CreateContactUnlockPaymentReq) (CreateContactUnlockPaymentResp, error) {
	input := model.GetContactUnlockPaymentContextInput{
		ResourceID: strings.TrimSpace(req.ResourceID),
		OrderID:    strings.TrimSpace(req.OrderID),
		UserID:     strings.TrimSpace(req.UserID),
	}
	if input.ResourceID == "" || input.OrderID == "" || input.UserID == "" {
		return CreateContactUnlockPaymentResp{}, errx.New(errx.CodeValidationFailed, "联系方式查看订单不存在或未登录")
	}
	if l.gateway == nil && !l.devMockEnabled {
		return CreateContactUnlockPaymentResp{}, errx.New(errx.CodeInternalError, "微信支付暂未配置，请联系平台运营")
	}
	contextInfo, err := l.store.GetContactUnlockPaymentContext(ctx, input)
	if err != nil {
		logx.Errorf("查询联系方式查看支付上下文失败: resourceId=%s orderId=%s userId=%s err=%+v", input.ResourceID, input.OrderID, input.UserID, err)
		return CreateContactUnlockPaymentResp{}, err
	}
	if contextInfo.Status != model.PaymentOrderStatusPending {
		return CreateContactUnlockPaymentResp{}, errx.New(errx.CodeStateConflict, "订单已失效，请重新查看联系方式")
	}
	if strings.TrimSpace(contextInfo.OpenID) == "" {
		return CreateContactUnlockPaymentResp{}, errx.New(errx.CodeValidationFailed, "请先使用微信登录后再支付")
	}
	order, err := l.store.CreateContactUnlockPaymentOrder(ctx, model.CreateContactUnlockPaymentOrderInput{
		ResourceID: contextInfo.ResourceID,
		OrderID:    contextInfo.OrderID,
		UserID:     contextInfo.UserID,
	})
	if err != nil {
		logx.Errorf("创建联系方式查看支付单失败: resourceId=%s orderId=%s userId=%s err=%+v", input.ResourceID, input.OrderID, input.UserID, err)
		return CreateContactUnlockPaymentResp{}, err
	}
	if l.gateway == nil {
		return l.completeDevMockContactUnlockPayment(ctx, order)
	}
	params, err := l.gateway.CreatePrepay(ctx, WechatPrepayInput{
		OutTradeNo:  order.OutTradeNo,
		Description: contactUnlockPaymentDescription(order.ResourceTitle),
		OpenID:      contextInfo.OpenID,
		AmountTotal: order.AmountTotal,
		Currency:    order.Currency,
		Attach:      buildPaymentAttach(PaymentBusinessContactUnlock, order.ID),
	})
	if err != nil {
		logx.Errorf("创建微信联系方式查看预支付失败: resourceId=%s orderId=%s userId=%s err=%+v", input.ResourceID, input.OrderID, input.UserID, err)
		return CreateContactUnlockPaymentResp{}, err
	}
	return CreateContactUnlockPaymentResp{OrderID: order.ID, Status: order.Status, Payment: params}, nil
}

func contactUnlockPaymentDescription(resourceTitle string) string {
	title := strings.TrimSpace(resourceTitle)
	if title == "" {
		title = "需求信息"
	}
	return "联系方式查看 - " + title
}

func (l *CreateContactUnlockPaymentLogic) completeDevMockContactUnlockPayment(ctx context.Context, order model.ContactUnlockPaymentOrder) (CreateContactUnlockPaymentResp, error) {
	result, err := l.store.MarkContactUnlockOrderPaid(ctx, model.MarkContactUnlockOrderPaidInput{
		BusinessOrderID: order.ID,
		OutTradeNo:      order.OutTradeNo,
		TransactionID:   "mock-" + order.OutTradeNo,
		AmountTotal:     order.AmountTotal,
		SuccessTime:     nowFunc().Format(time.RFC3339),
		NotifyPayload: model.JSONMap{
			"trade_state":  "SUCCESS",
			"mock":         true,
			"out_trade_no": order.OutTradeNo,
		},
	})
	if err != nil {
		return CreateContactUnlockPaymentResp{}, err
	}
	logx.Infof("开发模拟联系方式查看支付已完成: resourceId=%s orderId=%s outTradeNo=%s", result.ResourceID, result.OrderID, order.OutTradeNo)
	return CreateContactUnlockPaymentResp{OrderID: result.OrderID, Status: result.Status}, nil
}
