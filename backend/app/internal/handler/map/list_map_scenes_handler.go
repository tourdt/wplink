package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListMapScenesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listMapScenesHTTPHandler(svcCtx)
}
