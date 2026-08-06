package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func CreateResourceDraftHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return createResourceHTTPHandler(svcCtx, true)
}
