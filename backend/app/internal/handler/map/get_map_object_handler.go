package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func GetMapObjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return getMapObjectHTTPHandler(svcCtx)
}
