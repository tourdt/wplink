package admingrowth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateGrowthCampaignHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveGrowthCampaignHTTPHandler(svcCtx, true)
}
