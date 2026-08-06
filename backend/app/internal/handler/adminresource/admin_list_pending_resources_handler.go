package adminresource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListPendingResourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListResourcesHTTPHandler(svcCtx, true)
}
