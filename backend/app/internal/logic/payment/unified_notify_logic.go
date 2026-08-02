package payment

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	PaymentBusinessContactUnlock = "contact_unlock"
	PaymentBusinessVIP           = "vip"
)

type UnifiedWechatPayNotifyStore interface {
	MarkContactUnlockOrderPaid(ctx context.Context, input model.MarkContactUnlockOrderPaidInput) (model.ContactUnlockPaymentResult, error)
	MarkVIPOrderPaid(ctx context.Context, input model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error)
}

type UnifiedWechatPayNotifyLogic struct {
	store   UnifiedWechatPayNotifyStore
	gateway WechatPayGateway
}

func NewUnifiedWechatPayNotifyLogic(store UnifiedWechatPayNotifyStore, gateway WechatPayGateway) *UnifiedWechatPayNotifyLogic {
	return &UnifiedWechatPayNotifyLogic{store: store, gateway: gateway}
}

func (l *UnifiedWechatPayNotifyLogic) HandleNotify(ctx context.Context, req WechatPayNotifyReq) (WechatPayNotifyResp, error) {
	if l.gateway == nil {
		return WechatPayNotifyResp{}, errx.New(errx.CodeInternalError, "微信支付暂未配置")
	}
	notification, err := l.gateway.DecodeNotify(ctx, req)
	if err != nil {
		logx.Errorf("微信支付统一回调验签或解密失败: err=%+v", err)
		return WechatPayNotifyResp{}, err
	}
	return l.HandleNotification(ctx, notification)
}

func (l *UnifiedWechatPayNotifyLogic) HandleNotification(ctx context.Context, notification WechatPayNotification) (WechatPayNotifyResp, error) {
	if strings.TrimSpace(notification.OutTradeNo) == "" ||
		strings.TrimSpace(notification.TransactionID) == "" ||
		notification.AmountTotal <= 0 {
		return WechatPayNotifyResp{}, errx.New(errx.CodeValidationFailed, "支付通知数据不完整")
	}

	businessType, businessOrderID, err := parsePaymentAttach(notification.Attach)
	if err != nil {
		logx.Errorf("微信支付统一回调缺少合法业务标识: outTradeNo=%s attach=%q err=%+v", notification.OutTradeNo, notification.Attach, err)
		return WechatPayNotifyResp{}, err
	}
	if err := validatePaymentBusinessPrefix(businessType, notification.OutTradeNo); err != nil {
		logx.Errorf("微信支付统一回调业务类型与商户订单号不匹配: outTradeNo=%s businessType=%s businessOrderId=%s err=%+v", notification.OutTradeNo, businessType, businessOrderID, err)
		return WechatPayNotifyResp{}, err
	}

	switch businessType {
	case PaymentBusinessContactUnlock:
		result, markErr := l.store.MarkContactUnlockOrderPaid(ctx, model.MarkContactUnlockOrderPaidInput{
			BusinessOrderID: businessOrderID,
			OutTradeNo:      notification.OutTradeNo,
			TransactionID:   notification.TransactionID,
			AmountTotal:     notification.AmountTotal,
			SuccessTime:     notification.SuccessTime,
			NotifyPayload:   model.JSONMap(notification.RawPayload),
		})
		if markErr != nil {
			logx.Errorf("处理联系方式查看支付回调失败: businessOrderId=%s outTradeNo=%s err=%+v", businessOrderID, notification.OutTradeNo, markErr)
			return WechatPayNotifyResp{}, markErr
		}
		if result.OrderID != businessOrderID {
			return WechatPayNotifyResp{}, paymentOrderMismatchError(notification.OutTradeNo, businessOrderID, result.OrderID)
		}
	case PaymentBusinessVIP:
		result, markErr := l.store.MarkVIPOrderPaid(ctx, model.MarkVIPOrderPaidInput{
			BusinessOrderID: businessOrderID,
			OutTradeNo:      notification.OutTradeNo,
			TransactionID:   notification.TransactionID,
			AmountTotal:     notification.AmountTotal,
			SuccessTime:     notification.SuccessTime,
			NotifyPayload:   model.JSONMap(notification.RawPayload),
		})
		if markErr != nil {
			logx.Errorf("处理 VIP 支付回调失败: businessOrderId=%s outTradeNo=%s err=%+v", businessOrderID, notification.OutTradeNo, markErr)
			return WechatPayNotifyResp{}, markErr
		}
		if result.OrderID != businessOrderID {
			return WechatPayNotifyResp{}, paymentOrderMismatchError(notification.OutTradeNo, businessOrderID, result.OrderID)
		}
	default:
		return WechatPayNotifyResp{}, errx.New(errx.CodeValidationFailed, "支付通知业务类型不支持")
	}

	logx.Infof("微信支付统一回调处理成功: businessType=%s businessOrderId=%s outTradeNo=%s transactionId=%s", businessType, businessOrderID, notification.OutTradeNo, notification.TransactionID)
	return WechatPayNotifyResp{Code: "SUCCESS", Message: "成功"}, nil
}

func buildPaymentAttach(businessType string, businessOrderID string) string {
	return strings.TrimSpace(businessType) + ":" + strings.TrimSpace(businessOrderID)
}

func parsePaymentAttach(attach string) (string, string, error) {
	businessType, businessOrderID, ok := strings.Cut(strings.TrimSpace(attach), ":")
	businessType = strings.TrimSpace(businessType)
	businessOrderID = strings.TrimSpace(businessOrderID)
	if !ok || businessType == "" || businessOrderID == "" {
		return "", "", errx.New(errx.CodeValidationFailed, "支付通知业务标识不正确")
	}
	switch businessType {
	case PaymentBusinessContactUnlock, PaymentBusinessVIP:
		return businessType, businessOrderID, nil
	default:
		return "", "", errx.New(errx.CodeValidationFailed, "支付通知业务类型不支持")
	}
}

func validatePaymentBusinessPrefix(businessType string, outTradeNo string) error {
	outTradeNo = strings.ToUpper(strings.TrimSpace(outTradeNo))
	matched := false
	switch businessType {
	case PaymentBusinessContactUnlock:
		matched = strings.HasPrefix(outTradeNo, "CU")
	case PaymentBusinessVIP:
		matched = strings.HasPrefix(outTradeNo, "VIP")
	}
	if !matched {
		return errx.New(errx.CodeValidationFailed, "支付通知业务类型与订单不匹配")
	}
	return nil
}

func paymentOrderMismatchError(outTradeNo string, wantOrderID string, gotOrderID string) error {
	logx.Errorf("微信支付业务订单标识校验失败: outTradeNo=%s attachOrderId=%s actualOrderId=%s", outTradeNo, wantOrderID, gotOrderID)
	return errx.New(errx.CodeStateConflict, "支付订单状态异常，请联系平台客服")
}
