package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminGetMapSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminGetMapSceneHTTPHandler(svcCtx)
}
