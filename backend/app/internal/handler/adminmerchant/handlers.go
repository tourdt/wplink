package adminmerchant

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

func adminListMerchantsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, err := handlerx.AdminFromContext(adminMerchantRequestContext(r))
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MerchantModel == nil {
			logx.WithContext(adminMerchantRequestContext(r)).Errorw("后台商家列表 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "商家管理服务暂不可用，请稍后重试"))
			return
		}
		var req types.AdminListMerchantsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "商家筛选参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewMerchantAdminLogic(svcCtx.APIStore).ListMerchants(r.Context(), adminlogic.ListMerchantsReq{
			CityCode: req.CityCode, MerchantType: req.MerchantType, Status: req.Status,
			Keyword: req.Keyword, Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			adminlogic.LogAdminFailure(r.Context(), "加载后台商家列表失败", "list_admin_merchants", err,
				logx.Field("operatorId", admin.OperatorID), logx.Field("cityFiltered", req.CityCode != ""),
				logx.Field("merchantTypeFiltered", req.MerchantType != ""), logx.Field("statusFiltered", req.Status != ""),
				logx.Field("keywordFiltered", req.Keyword != ""), logx.Field("page", req.Page), logx.Field("pageSize", req.PageSize))
		}
		response.JSON(w, resp, err)
	}
}

func adminMerchantRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
