package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListMapScenesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListMapScenesHTTPHandler(svcCtx)
}
