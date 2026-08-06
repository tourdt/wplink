package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminSaveMapSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveMapSceneHTTPHandler(svcCtx, false)
}
