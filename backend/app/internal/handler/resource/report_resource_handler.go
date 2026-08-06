package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ReportResourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return reportResourceHTTPHandler(svcCtx)
}
