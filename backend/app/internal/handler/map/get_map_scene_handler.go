package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func GetMapSceneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return getMapSceneHTTPHandler(svcCtx)
}
