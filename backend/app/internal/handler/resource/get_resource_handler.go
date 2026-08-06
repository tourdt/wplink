package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func GetResourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return getResourceHTTPHandler(svcCtx)
}
