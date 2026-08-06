package adminpermission

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateOperatorStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminUpdateOperatorStatusHTTPHandler(svcCtx)
}
