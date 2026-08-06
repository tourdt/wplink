package metrics

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	metricslogic "wplink/backend/app/internal/logic/metrics"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RecordResourceExposuresHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.ResourceMetricDailyModel == nil {
			logx.WithContext(r.Context()).Error("资源曝光 Handler 存储依赖未配置")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "曝光服务暂不可用，请稍后重试"))
			return
		}
		subject, authenticated, err := handlerx.OptionalUser(r, svcCtx.UserTokenService)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.RecordResourceExposuresReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "曝光参数格式不正确"))
			return
		}
		items := make([]metricslogic.ResourceExposureItem, 0, len(req.Items))
		for _, item := range req.Items {
			items = append(items, metricslogic.ResourceExposureItem{ResourceID: item.ResourceId, VisibleDurationMS: item.VisibleDurationMs})
		}
		userID := ""
		if authenticated {
			userID = subject.UserID
		}
		resp, err := metricslogic.NewRecordResourceExposuresLogic(svcCtx.APIStore).RecordResourceExposures(r.Context(), metricslogic.RecordResourceExposuresReq{
			UserID: userID, VisitorKey: req.VisitorKey, SessionID: req.SessionId, Source: req.Source, Items: items,
		})
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		response.JSON(w, types.RecordResourceExposuresResp{RecordedCount: resp.RecordedCount}, nil)
	}
}
