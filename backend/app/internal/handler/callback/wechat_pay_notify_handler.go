package callback

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func WechatPayNotifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.ResourceContactUnlockModel == nil || svcCtx.APIStore.VIPModel == nil {
		return wechatPayNotifyHTTPHandler(nil, nil)
	}
	return wechatPayNotifyHTTPHandler(svcCtx.APIStore, svcCtx.WechatPayGateway)
}
