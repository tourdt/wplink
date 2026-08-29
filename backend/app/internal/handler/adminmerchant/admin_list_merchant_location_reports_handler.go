package adminmerchant

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListMerchantLocationReportsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListMerchantLocationReportsHTTPHandler(svcCtx)
}
