package vip

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func CreateVIPOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return createVIPOrderHTTPHandler(vipStoreFromServiceContext(svcCtx), vipPermissionDepsFromServiceContext(svcCtx))
}
