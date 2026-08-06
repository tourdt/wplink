package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func GetOwnResourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return ownResourceHTTPHandler(svcCtx, false)
}
