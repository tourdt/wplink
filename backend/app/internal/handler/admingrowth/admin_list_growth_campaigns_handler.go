package admingrowth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListGrowthCampaignsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListGrowthCampaignsHTTPHandler(svcCtx)
}
