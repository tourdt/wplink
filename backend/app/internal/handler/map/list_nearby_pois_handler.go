package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListNearbyPoisHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listNearbyPoisHTTPHandler(svcCtx)
}
