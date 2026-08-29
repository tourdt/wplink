package adminmerchant

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminReviewMerchantLocationReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminReviewMerchantLocationReportHTTPHandler(svcCtx)
}
