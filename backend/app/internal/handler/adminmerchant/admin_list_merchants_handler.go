package adminmerchant

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListMerchantsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListMerchantsHTTPHandler(svcCtx)
}
