package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminSaveMapObjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveMapObjectHTTPHandler(svcCtx, false)
}
