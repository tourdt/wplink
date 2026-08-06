package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func SearchMapObjectsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return searchMapObjectsHTTPHandler(svcCtx)
}
