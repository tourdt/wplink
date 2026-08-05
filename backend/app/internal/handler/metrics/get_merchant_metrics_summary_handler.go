package metrics

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func GetMerchantMetricsSummaryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireMetricsDependencies(r, svcCtx, false); err != nil {
			response.JSON(w, nil, err)
			return
		}
		merchantID := pathvar.Vars(r)["merchantId"]
		if err := handlerx.RequireMerchant(r, handlerx.MerchantPermissionDeps{
			UserTokenService: svcCtx.UserTokenService, AdminTokenService: svcCtx.AdminTokenService, Store: svcCtx.APIStore,
		}, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := metricslogic.NewGetMerchantMetricsLogic(svcCtx.APIStore).GetMerchantMetrics(r.Context(), merchantID)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.MerchantMetricsSummaryResp{
			MerchantId: resp.MerchantID, PublishedResourceCount: resp.PublishedResourceCount,
			ExpiringResourceCount: resp.ExpiringResourceCount, DealtResourceCount: resp.DealtResourceCount,
			Last7Days: types.MerchantLast7DaysMetrics{
				ExposureCount: resp.Last7Days.ExposureCount, DetailViewCount: resp.Last7Days.DetailViewCount,
				ContactClickCount: resp.Last7Days.ContactClickCount,
			},
		}, nil)
	}
}
