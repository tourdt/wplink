package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminBatchGenerateMapObjectsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminBatchGenerateMapObjectsHTTPHandler(svcCtx)
}
