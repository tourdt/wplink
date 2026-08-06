package adminconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateResourceTypeConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminCreateResourceTypeConfigHTTPHandler(svcCtx)
}
