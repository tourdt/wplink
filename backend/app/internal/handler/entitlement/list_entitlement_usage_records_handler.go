package entitlement

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListEntitlementUsageRecordsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listEntitlementUsageRecordsHTTPHandler(entitlementStoreFromServiceContext(svcCtx), entitlementPermissionDepsFromServiceContext(svcCtx))
}
