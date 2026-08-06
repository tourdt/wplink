package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListMerchantPlacesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listMerchantPlacesHTTPHandler(svcCtx)
}
