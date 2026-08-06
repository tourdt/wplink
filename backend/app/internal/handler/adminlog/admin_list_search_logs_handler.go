package adminlog

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListSearchLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListSearchLogsHTTPHandler(svcCtx)
}
