package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListRelatedResourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return relatedResourcesHTTPHandler(svcCtx)
}
