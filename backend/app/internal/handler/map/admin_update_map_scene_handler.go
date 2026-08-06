package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateMapSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveMapSceneHTTPHandler(svcCtx, true)
}
