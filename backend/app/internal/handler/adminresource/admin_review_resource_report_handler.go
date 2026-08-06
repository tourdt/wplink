package adminresource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminReviewResourceReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminReviewResourceReportHTTPHandler(svcCtx)
}
