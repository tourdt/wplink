package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListMapObjectsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListMapObjectsHTTPHandler(svcCtx)
}
