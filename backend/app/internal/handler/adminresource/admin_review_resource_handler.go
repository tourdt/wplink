package adminresource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminReviewResourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminReviewResourceHTTPHandler(svcCtx)
}
