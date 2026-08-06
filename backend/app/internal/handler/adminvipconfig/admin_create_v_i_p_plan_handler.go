package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateVIPPlanHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveVIPPlanHTTPHandler(svcCtx, false)
}
