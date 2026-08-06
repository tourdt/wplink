package entitlement

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListMerchantEntitlementsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listMerchantEntitlementsHTTPHandler(entitlementStoreFromServiceContext(svcCtx), entitlementPermissionDepsFromServiceContext(svcCtx))
}
