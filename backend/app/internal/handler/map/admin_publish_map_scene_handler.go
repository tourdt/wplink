package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminPublishMapSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminPublishMapSceneHTTPHandler(svcCtx)
}
