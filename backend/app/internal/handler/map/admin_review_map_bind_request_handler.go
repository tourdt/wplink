package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminReviewMapBindRequestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminReviewMapBindRequestHTTPHandler(svcCtx)
}
