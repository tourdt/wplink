package vip

import (
	"net/http"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/svc"
)

func CreateVIPPaymentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	if svcCtx == nil {
		return createVIPPaymentHTTPHandler(nil, nil, vipPermissionDepsFromServiceContext(nil), config.Config{})
	}
	return createVIPPaymentHTTPHandler(vipStoreFromServiceContext(svcCtx), svcCtx.WechatPayGateway, vipPermissionDepsFromServiceContext(svcCtx), svcCtx.Config)
}
