package adminlog

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListOperationLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListOperationLogsHTTPHandler(svcCtx)
}
