package adminconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateResourceTypeConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminUpdateResourceTypeConfigHTTPHandler(svcCtx)
}
