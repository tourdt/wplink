package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateQuotaPackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveQuotaPackHTTPHandler(svcCtx, false)
}
