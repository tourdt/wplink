package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListMyResourcesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listMyResourcesHTTPHandler(svcCtx)
}
