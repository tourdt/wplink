package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func CreateResourceContactEventHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return contactEventHTTPHandler(svcCtx)
}
