package vip

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListQuotaPacksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listQuotaPacksHTTPHandler(vipStoreFromServiceContext(svcCtx))
}
