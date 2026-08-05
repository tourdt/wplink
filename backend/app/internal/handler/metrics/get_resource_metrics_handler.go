package metrics

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func GetResourceMetricsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireMetricsDependencies(r, svcCtx, true); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resourceID := pathvar.Vars(r)["resourceId"]
		merchantID, err := svcCtx.APIStore.GetResourceMerchantID(r.Context(), resourceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.JSON(w, nil, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架"))
				return
			}
			logx.WithContext(r.Context()).Errorw("读取资源指标所属商家失败", logx.Field("resourceId", resourceID), logx.Field("errorType", fmt.Sprintf("%T", err)))
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "资源指标加载失败，请稍后重试"))
			return
		}
		if err := handlerx.RequireMerchant(r, handlerx.MerchantPermissionDeps{
			UserTokenService: svcCtx.UserTokenService, AdminTokenService: svcCtx.AdminTokenService, Store: svcCtx.APIStore,
		}, merchantID); err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.ResourceMetricsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "资源指标查询参数格式不正确"))
			return
		}
		resp, err := metricslogic.NewGetResourceMetricsLogic(svcCtx.APIStore).GetResourceMetrics(r.Context(), metricslogic.GetResourceMetricsReq{
			ResourceID: resourceID, From: req.From, To: req.To,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		daily := make([]types.ResourceMetricsDailyItem, 0, len(resp.Daily))
		for _, item := range resp.Daily {
			daily = append(daily, types.ResourceMetricsDailyItem{
				Date: item.Date, ExposureCount: item.ExposureCount, DetailViewCount: item.DetailViewCount,
				PhoneClickCount: item.PhoneClickCount, WechatCopyCount: item.WechatCopyCount,
			})
		}
		response.JSON(w, types.ResourceMetricsResp{
			ResourceId: resp.ResourceID,
			Summary: types.ResourceMetricsSummary{
				ExposureCount: resp.Summary.ExposureCount, DetailViewCount: resp.Summary.DetailViewCount,
				PhoneClickCount: resp.Summary.PhoneClickCount, WechatCopyCount: resp.Summary.WechatCopyCount,
				DealFeedbackCount: resp.Summary.DealFeedbackCount,
			},
			Daily: daily,
		}, nil)
	}
}

func requireMetricsDependencies(r *http.Request, svcCtx *svc.ServiceContext, requireResourceStore bool) error {
	missing := svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.ResourceMetricDailyModel == nil || svcCtx.APIStore.UserModel == nil || svcCtx.AdminTokenService == nil
	if !missing && requireResourceStore {
		missing = svcCtx.APIStore.ResourceModel == nil
	}
	if missing {
		logx.WithContext(r.Context()).Errorw("指标 Handler 依赖未配置", logx.Field("resourceStoreRequired", requireResourceStore))
		return errx.New(errx.CodeInternalError, "指标服务暂不可用，请稍后重试")
	}
	return nil
}
