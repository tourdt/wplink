package admindashboard

import (
	"context"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func adminDashboardOverviewHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, err := handlerx.AdminFromContext(adminDashboardRequestContext(r))
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.AdminDashboardModel == nil {
			logx.WithContext(adminDashboardRequestContext(r)).Errorw("后台看板 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "后台看板服务暂不可用，请稍后重试"))
			return
		}
		var req types.AdminDashboardOverviewReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "看板筛选参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewDashboardLogic(svcCtx.APIStore).GetOverview(r.Context(), adminlogic.DashboardOverviewReq{CityCode: req.CityCode})
		if err != nil {
			adminlogic.LogAdminFailure(r.Context(), "加载后台看板失败", "get_admin_dashboard", err,
				logx.Field("operatorId", admin.OperatorID), logx.Field("cityFiltered", req.CityCode != ""))
		}
		response.JSON(w, resp, err)
	}
}

func adminDashboardRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
