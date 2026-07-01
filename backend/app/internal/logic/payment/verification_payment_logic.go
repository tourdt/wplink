package payment

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type VerificationPaymentStore interface {
	GetVerificationPaymentContext(ctx context.Context, input model.GetVerificationPaymentContextInput) (model.VerificationPaymentContext, error)
	CreateVerificationPaymentOrder(ctx context.Context, input model.CreateVerificationPaymentOrderInput) (model.VerificationPaymentOrder, error)
	MarkVerificationPaymentPaid(ctx context.Context, input model.MarkVerificationPaymentPaidInput) (model.VerificationPaymentResult, error)
}

type WechatPayGateway interface {
	CreatePrepay(ctx context.Context, input WechatPrepayInput) (WechatPayParams, error)
	DecodeNotify(ctx context.Context, req WechatPayNotifyReq) (WechatPayNotification, error)
}

type CreateVerificationPaymentReq struct {
	MerchantID     string
	VerificationID string
	UserID         string
}

type CreateVerificationPaymentResp struct {
	OrderID string          `json:"orderId"`
	Status  string          `json:"status"`
	Payment WechatPayParams `json:"payment"`
}

type WechatPrepayInput struct {
	OutTradeNo  string
	Description string
	OpenID      string
	AmountTotal int64
	Currency    string
	Attach      string
}

type WechatPayParams struct {
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

type WechatPayNotifyReq struct {
	Headers map[string]string
	Body    []byte
}

type WechatPayNotification struct {
	OutTradeNo    string
	TransactionID string
	AmountTotal   int64
	SuccessTime   string
	RawPayload    map[string]interface{}
}

type WechatPayNotifyResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateVerificationPaymentLogic struct {
	store   VerificationPaymentStore
	gateway WechatPayGateway
}

func NewCreateVerificationPaymentLogic(store VerificationPaymentStore, gateway WechatPayGateway) *CreateVerificationPaymentLogic {
	return &CreateVerificationPaymentLogic{store: store, gateway: gateway}
}

func (l *CreateVerificationPaymentLogic) CreateVerificationPayment(ctx context.Context, req CreateVerificationPaymentReq) (CreateVerificationPaymentResp, error) {
	if l.gateway == nil {
		return CreateVerificationPaymentResp{}, errx.New(errx.CodeInternalError, "微信支付暂未配置，请联系平台运营")
	}
	input := model.GetVerificationPaymentContextInput{
		MerchantID:     strings.TrimSpace(req.MerchantID),
		VerificationID: strings.TrimSpace(req.VerificationID),
		UserID:         strings.TrimSpace(req.UserID),
	}
	if input.MerchantID == "" || input.VerificationID == "" || input.UserID == "" {
		return CreateVerificationPaymentResp{}, errx.New(errx.CodeValidationFailed, "认证记录不存在或未登录")
	}
	contextInfo, err := l.store.GetVerificationPaymentContext(ctx, input)
	if err != nil {
		return CreateVerificationPaymentResp{}, err
	}
	if contextInfo.Status != model.VerificationStatusPaymentPending {
		return CreateVerificationPaymentResp{}, errx.New(errx.CodeStateConflict, "当前认证无需支付，请刷新后重试")
	}
	if strings.TrimSpace(contextInfo.OpenID) == "" {
		return CreateVerificationPaymentResp{}, errx.New(errx.CodeValidationFailed, "请先使用微信登录后再支付")
	}
	if !contextInfo.Billing.RequiresOnlinePayment(nowFunc()) {
		return CreateVerificationPaymentResp{}, errx.New(errx.CodeStateConflict, "当前认证处于免费期，请刷新认证状态")
	}
	order, err := l.store.CreateVerificationPaymentOrder(ctx, model.CreateVerificationPaymentOrderInput{
		VerificationID: contextInfo.VerificationID,
		MerchantID:     contextInfo.MerchantID,
		UserID:         contextInfo.UserID,
		OpenID:         contextInfo.OpenID,
		AmountTotal:    contextInfo.Billing.FeeAmount,
		Currency:       contextInfo.Billing.Currency,
	})
	if err != nil {
		return CreateVerificationPaymentResp{}, err
	}
	params, err := l.gateway.CreatePrepay(ctx, WechatPrepayInput{
		OutTradeNo:  order.OutTradeNo,
		Description: "商家认证费",
		OpenID:      contextInfo.OpenID,
		AmountTotal: order.AmountTotal,
		Currency:    order.Currency,
		Attach:      contextInfo.VerificationID,
	})
	if err != nil {
		return CreateVerificationPaymentResp{}, err
	}
	return CreateVerificationPaymentResp{OrderID: order.ID, Status: order.Status, Payment: params}, nil
}

type WechatPayNotifyLogic struct {
	store   VerificationPaymentStore
	gateway WechatPayGateway
}

func NewWechatPayNotifyLogic(store VerificationPaymentStore, gateway WechatPayGateway) *WechatPayNotifyLogic {
	return &WechatPayNotifyLogic{store: store, gateway: gateway}
}

func (l *WechatPayNotifyLogic) HandleNotify(ctx context.Context, req WechatPayNotifyReq) (WechatPayNotifyResp, error) {
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
	_, err = l.store.MarkVerificationPaymentPaid(ctx, model.MarkVerificationPaymentPaidInput{
		OutTradeNo:    notification.OutTradeNo,
		TransactionID: notification.TransactionID,
		AmountTotal:   notification.AmountTotal,
		SuccessTime:   notification.SuccessTime,
		NotifyPayload: model.JSONMap(notification.RawPayload),
	})
	if err != nil {
		return WechatPayNotifyResp{}, err
	}
	return WechatPayNotifyResp{Code: "SUCCESS", Message: "成功"}, nil
}
