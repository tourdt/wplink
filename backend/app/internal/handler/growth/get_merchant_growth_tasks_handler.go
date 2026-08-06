package growth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func GetMerchantGrowthTasksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return getMerchantGrowthTasksHTTPHandler(growthTaskStoreFromServiceContext(svcCtx), growthPermissionDepsFromServiceContext(svcCtx))
}
