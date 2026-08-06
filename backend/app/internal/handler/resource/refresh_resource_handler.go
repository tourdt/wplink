package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func RefreshResourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return ownerActionHTTPHandler(svcCtx, "refresh")
}
