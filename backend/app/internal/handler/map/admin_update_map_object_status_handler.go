package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateMapObjectStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminUpdateMapObjectStatusHTTPHandler(svcCtx)
}
