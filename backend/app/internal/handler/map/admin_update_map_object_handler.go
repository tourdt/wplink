package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateMapObjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveMapObjectHTTPHandler(svcCtx, true)
}
