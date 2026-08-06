package adminresource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListResourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListResourcesHTTPHandler(svcCtx, false)
}
