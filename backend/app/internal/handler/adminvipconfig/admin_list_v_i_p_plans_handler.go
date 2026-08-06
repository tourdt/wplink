package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListVIPPlansHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListVIPConfigHTTPHandler(svcCtx, vipConfigPlans)
}
