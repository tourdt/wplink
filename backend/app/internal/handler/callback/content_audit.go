package callback

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"wplink/backend/app/internal/handler/handlerx"
	contentauditlogic "wplink/backend/app/internal/logic/contentaudit"
	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"
)

const contentAuditCallbackBodyLimit = int64(256 << 10)

func verifyContentAuditCallbackHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if callbackVerifierMissing(svcCtx) {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "微信回调校验服务暂不可用"))
			return
		}
		query := r.URL.Query()
		if err := svcCtx.ContentAuditCallbackVerifier.Verify(query.Get("signature"), query.Get("timestamp"), query.Get("nonce"), false); err != nil {
			response.JSON(w, nil, safeCallbackVerificationError(err))
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(query.Get("echostr")))
	}
}

func handleContentAuditCallbackHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if callbackVerifierMissing(svcCtx) {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "微信回调校验服务暂不可用"))
			return
		}
		query := r.URL.Query()
		signature, timestamp, nonce := query.Get("signature"), query.Get("timestamp"), query.Get("nonce")

		// 验签阶段只检查历史指纹，不提前写入重放记录；业务失败时微信仍可安全重试。
		if err := svcCtx.ContentAuditCallbackVerifier.Verify(signature, timestamp, nonce, false); err != nil {
			if errors.Is(err, contentauditlogic.ErrWechatCallbackReplay) {
				writeContentAuditCallbackSuccess(w)
				return
			}
			response.JSON(w, nil, safeCallbackVerificationError(err))
			return
		}

		body, err := handlerx.ReadLimitedBody(r, contentAuditCallbackBodyLimit)
		if err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "微信回调内容过大或读取失败"))
			return
		}
		var payload model.JSONMap
		if err := json.Unmarshal(body, &payload); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "微信回调参数格式不正确"))
			return
		}
		if payload == nil {
			payload = model.JSONMap{}
		}

		configuredAppID := ""
		if svcCtx != nil {
			configuredAppID = strings.TrimSpace(svcCtx.Config.Wechat.AppID)
		}
		if configuredAppID == "" {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "微信回调服务暂不可用"))
			return
		}
		payloadAppID, _ := payload["appid"].(string)
		if strings.TrimSpace(payloadAppID) == "" || strings.TrimSpace(payloadAppID) != configuredAppID {
			// 响应不回显配置值或供应商原始值，避免 AppID 进入响应与诊断链路。
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "微信回调身份校验失败"))
			return
		}
		if svcCtx.APIStore == nil || svcCtx.APIStore.ResourceModel == nil {
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "图片审核回调服务暂不可用"))
			return
		}

		_, logicErr := contentauditlogic.NewMediaCheckCallbackLogic(svcCtx.APIStore).Handle(r.Context(), payload)
		if logicErr != nil && errx.CodeOf(logicErr) != errx.CodeStateConflict {
			response.JSON(w, nil, logicErr)
			return
		}

		// 只有业务完成或已被幂等状态机确认后才记录指纹；记录失败时不返回 success，保留供应商重试机会。
		if err := svcCtx.ContentAuditCallbackVerifier.Verify(signature, timestamp, nonce, true); err != nil {
			// 并发请求可能都通过只读检查；第二个请求完成幂等业务处理后再记录指纹时，
			// 命中重放表示同一回调已经被成功记忆，此时仍须向微信确认 success。
			if errors.Is(err, contentauditlogic.ErrWechatCallbackReplay) {
				writeContentAuditCallbackSuccess(w)
				return
			}
			response.JSON(w, nil, safeCallbackVerificationError(err))
			return
		}
		writeContentAuditCallbackSuccess(w)
	}
}

func callbackVerifierMissing(svcCtx *svc.ServiceContext) bool {
	if svcCtx == nil || svcCtx.ContentAuditCallbackVerifier == nil {
		return true
	}
	value := reflect.ValueOf(svcCtx.ContentAuditCallbackVerifier)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func safeCallbackVerificationError(err error) error {
	code := errx.CodeOf(err)
	if code == errx.CodeInternalError {
		return errx.New(errx.CodeInternalError, "微信回调校验服务暂不可用")
	}
	return errx.New(errx.CodeUnauthorized, "微信回调校验失败")
}

func writeContentAuditCallbackSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("success"))
}
