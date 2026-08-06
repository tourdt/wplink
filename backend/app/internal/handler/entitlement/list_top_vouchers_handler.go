package entitlement

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListTopVouchersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listTopVouchersHTTPHandler(entitlementStoreFromServiceContext(svcCtx), entitlementPermissionDepsFromServiceContext(svcCtx))
}
