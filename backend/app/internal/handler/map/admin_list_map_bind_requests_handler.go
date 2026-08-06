package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListMapBindRequestsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListMapBindRequestsHTTPHandler(svcCtx)
}
