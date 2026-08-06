package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func SearchResourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return searchResourcesHTTPHandler(svcCtx)
}
