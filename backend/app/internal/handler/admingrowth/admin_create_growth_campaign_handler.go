package admingrowth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateGrowthCampaignHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveGrowthCampaignHTTPHandler(svcCtx, false)
}
