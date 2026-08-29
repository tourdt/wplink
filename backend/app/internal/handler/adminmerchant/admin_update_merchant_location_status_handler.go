package adminmerchant

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateMerchantLocationStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminUpdateMerchantLocationStatusHTTPHandler(svcCtx)
}
