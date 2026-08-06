package callback

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	"wplink/backend/app/internal/handler/handlerx"
	paymentlogic "wplink/backend/app/internal/logic/payment"
	"wplink/backend/app/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
)

const wechatPayNotifyBodyLimit = int64(1 << 20)

var (
	errSafeWechatPayGateway = errors.New("微信支付通知校验失败")
	errSafeWechatPayStore   = errors.New("微信支付通知入账失败")
)

func wechatPayNotifyHTTPHandler(store paymentlogic.UnifiedWechatPayNotifyStore, gateway paymentlogic.WechatPayGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if callbackDependencyMissing(store) || callbackDependencyMissing(gateway) {
			logx.WithContext(callbackRequestContext(r)).Error("微信支付通知 Handler 依赖未配置")
			writeWechatPayNotifyFailure(w)
			return
		}
		body, err := handlerx.ReadLimitedBody(r, wechatPayNotifyBodyLimit)
		if err != nil {
			// 不记录 Body、长度探测内容或读取错误原文，供应商只接收固定失败确认并安全重试。
			logx.WithContext(callbackRequestContext(r)).Error("读取微信支付通知失败")
			writeWechatPayNotifyFailure(w)
			return
		}
		headers := make(map[string]string, len(r.Header))
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		// 安全包装器只替换依赖错误文本，不修改原始 Body、首 Header 映射或成功业务结果。
		resp, err := paymentlogic.NewUnifiedWechatPayNotifyLogic(
			safeUnifiedWechatPayNotifyStore{delegate: store},
			safeWechatPayNotifyGateway{delegate: gateway},
		).HandleNotify(r.Context(), paymentlogic.WechatPayNotifyReq{Headers: headers, Body: body})
		if err != nil {
			writeWechatPayNotifyFailure(w)
			return
		}
		if resp.Code != "SUCCESS" || resp.Message != "成功" {
			logx.WithContext(r.Context()).Error("微信支付通知业务返回非成功确认")
			writeWechatPayNotifyFailure(w)
			return
		}
		writeWechatPayNotifySuccess(w)
	}
}

type safeWechatPayNotifyGateway struct {
	delegate paymentlogic.WechatPayGateway
}

func (g safeWechatPayNotifyGateway) CreatePrepay(ctx context.Context, input paymentlogic.WechatPrepayInput) (paymentlogic.WechatPayParams, error) {
	params, err := g.delegate.CreatePrepay(ctx, input)
	if err != nil {
		return paymentlogic.WechatPayParams{}, errSafeWechatPayGateway
	}
	return params, nil
}

func (g safeWechatPayNotifyGateway) DecodeNotify(ctx context.Context, req paymentlogic.WechatPayNotifyReq) (paymentlogic.WechatPayNotification, error) {
	notification, err := g.delegate.DecodeNotify(ctx, req)
	if err != nil {
		return paymentlogic.WechatPayNotification{}, errSafeWechatPayGateway
	}
	return notification, nil
}

type safeUnifiedWechatPayNotifyStore struct {
	delegate paymentlogic.UnifiedWechatPayNotifyStore
}

func (s safeUnifiedWechatPayNotifyStore) MarkContactUnlockOrderPaid(ctx context.Context, input model.MarkContactUnlockOrderPaidInput) (model.ContactUnlockPaymentResult, error) {
	result, err := s.delegate.MarkContactUnlockOrderPaid(ctx, input)
	if err != nil {
		return model.ContactUnlockPaymentResult{}, errSafeWechatPayStore
	}
	return result, nil
}

func (s safeUnifiedWechatPayNotifyStore) MarkVIPOrderPaid(ctx context.Context, input model.MarkVIPOrderPaidInput) (model.VIPPaymentResult, error) {
	result, err := s.delegate.MarkVIPOrderPaid(ctx, input)
	if err != nil {
		return model.VIPPaymentResult{}, errSafeWechatPayStore
	}
	return result, nil
}

func callbackDependencyMissing(dependency any) bool {
	if dependency == nil {
		return true
	}
	value := reflect.ValueOf(dependency)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func callbackRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

func writeWechatPayNotifySuccess(w http.ResponseWriter) {
	writeWechatPayNotifyJSON(w, http.StatusOK, `{"code":"SUCCESS","message":"成功"}`)
}

func writeWechatPayNotifyFailure(w http.ResponseWriter) {
	writeWechatPayNotifyJSON(w, http.StatusInternalServerError, `{"code":"FAIL","message":"处理失败"}`)
}

func writeWechatPayNotifyJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
