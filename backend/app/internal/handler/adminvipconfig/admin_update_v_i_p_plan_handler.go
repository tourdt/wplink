package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateVIPPlanHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveVIPPlanHTTPHandler(svcCtx, true)
}
