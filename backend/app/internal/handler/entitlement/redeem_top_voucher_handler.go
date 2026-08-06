package entitlement

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func RedeemTopVoucherHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	store := entitlementStoreFromServiceContext(svcCtx)
	return redeemTopVoucherHTTPHandler(store, store, entitlementPermissionDepsFromServiceContext(svcCtx))
}
