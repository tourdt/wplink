package adminentitlement

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminGrantMerchantEntitlementHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminGrantMerchantEntitlementHTTPHandler(svcCtx)
}
