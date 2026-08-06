package growth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListActiveGrowthCampaignsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listActiveGrowthCampaignsHTTPHandler(publicGrowthStoreFromServiceContext(svcCtx))
}
