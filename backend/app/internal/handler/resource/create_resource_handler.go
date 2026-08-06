package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func CreateResourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return createResourceHTTPHandler(svcCtx, false)
}
