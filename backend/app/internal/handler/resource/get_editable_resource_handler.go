package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func GetEditableResourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return ownResourceHTTPHandler(svcCtx, true)
}
