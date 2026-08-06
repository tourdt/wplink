package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListMapObjectsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listMapObjectsHTTPHandler(svcCtx)
}
