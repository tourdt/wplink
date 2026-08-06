package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateQuotaPackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveQuotaPackHTTPHandler(svcCtx, true)
}
