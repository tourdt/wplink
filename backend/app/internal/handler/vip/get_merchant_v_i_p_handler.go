package vip

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func GetMerchantVIPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return getMerchantVIPHTTPHandler(vipStoreFromServiceContext(svcCtx), vipPermissionDepsFromServiceContext(svcCtx))
}
