package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func GetMerchantMapBindingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return getMerchantMapBindingHTTPHandler(svcCtx)
}
