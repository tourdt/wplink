package adminmerchant

import (
	"context"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/session"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
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

func adminListMerchantLocationReportsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMerchantGovernanceContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListMerchantLocationReportsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "位置反馈筛选参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewMerchantAdminLogic(store).ListMerchantLocationReports(r.Context(), adminlogic.ListMerchantLocationReportsReq{
			Status: req.Status, Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			logx.WithContext(r.Context()).Errorw("后台加载商家位置反馈失败",
				logx.Field("adminId", admin.OperatorID), logx.Field("action", "list_location_reports"), logx.Field("err", err))
		}
		response.JSON(w, resp, err)
	}
}

func adminReviewMerchantLocationReportHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMerchantGovernanceContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminReviewMerchantLocationReportReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "位置反馈处理参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewMerchantAdminLogic(store).ReviewMerchantLocationReport(
			r.Context(), pathvar.Vars(r)["reportId"],
			adminlogic.ReviewMerchantLocationReportReq{Status: req.Status, ReviewNote: req.ReviewNote},
			adminlogic.AdminPermissionActor{OperatorID: admin.OperatorID, Roles: admin.Roles},
		)
		response.JSON(w, resp, err)
	}
}

func adminUpdateMerchantLocationStatusHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminMerchantGovernanceContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminUpdateMerchantLocationStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "商家位置操作参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewMerchantAdminLogic(store).UpdateMerchantLocationStatus(
			r.Context(), pathvar.Vars(r)["merchantId"],
			adminlogic.UpdateMerchantLocationStatusReq{Action: req.Action, Reason: req.Reason},
			adminlogic.AdminPermissionActor{OperatorID: admin.OperatorID, Roles: admin.Roles},
		)
		response.JSON(w, resp, err)
	}
}

func requireAdminMerchantGovernanceContext(r *http.Request, svcCtx *svc.ServiceContext) (session.AdminTokenSubject, *svc.APIStore, error) {
	admin, err := handlerx.AdminFromContext(adminMerchantRequestContext(r))
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.MerchantLocationReportModel == nil {
		logx.WithContext(adminMerchantRequestContext(r)).Errorw("后台位置治理 Handler 存储依赖未配置", logx.Field("adminId", admin.OperatorID))
		return session.AdminTokenSubject{}, nil, errx.New(errx.CodeInternalError, "商家位置治理服务暂不可用，请稍后重试")
	}
	return admin, svcCtx.APIStore, nil
}

func adminMerchantRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
