package adminresource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListResourceReportsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListResourceReportsHTTPHandler(svcCtx)
}
