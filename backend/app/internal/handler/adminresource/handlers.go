package adminresource

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

func adminListResourcesHTTPHandler(svcCtx *svc.ServiceContext, pendingOnly bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminResourceContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		logic := adminlogic.NewListPendingResourcesLogic(store)
		if pendingOnly {
			var req types.AdminPendingResourcesReq
			if err := parseAdminResourceRequest(r, &req); err != nil {
				response.JSON(w, nil, err)
				return
			}
			resp, err := logic.ListPendingResources(r.Context(), adminlogic.ListPendingResourcesReq{
				CityCode: req.CityCode, TypeCode: req.TypeCode, Page: req.Page, PageSize: req.PageSize,
			})
			response.JSON(w, resp, err)
			return
		}
		var req types.AdminResourcesReq
		if err := parseAdminResourceRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := logic.ListAdminResources(r.Context(), adminlogic.ListPendingResourcesReq{
			CityCode: req.CityCode, TypeCode: req.TypeCode, Status: req.Status, Page: req.Page, PageSize: req.PageSize,
		})
		response.JSON(w, resp, err)
	}
}

func adminReviewResourceHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminResourceContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminReviewResourceReq
		if err := parseAdminResourceRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		// 审核人只能来自 AdminAuth context，客户端请求中的同名字段不会进入业务输入。
		resp, err := adminlogic.NewReviewResourceLogic(store).ReviewResource(r.Context(), pathvar.Vars(r)["resourceId"], adminlogic.ReviewResourceReq{
			Action: req.Action, Reason: req.Reason, ReviewerID: admin.OperatorID,
		})
		response.JSON(w, resp, err)
	}
}

func adminListResourceReportsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminResourceContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminResourceReportsReq
		if err := parseAdminResourceRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewResourceReportLogic(store).ListResourceReports(r.Context(), adminlogic.ListResourceReportsReq{
			Status: req.Status, Page: req.Page, PageSize: req.PageSize,
		})
		response.JSON(w, resp, err)
	}
}

func adminReviewResourceReportHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, store, err := requireAdminResourceContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminReviewResourceReportReq
		if err := parseAdminResourceRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewResourceReportLogic(store).ReviewResourceReport(r.Context(), pathvar.Vars(r)["reportId"], adminlogic.ReviewResourceReportReq{
			Action: req.Action, ResourceAction: req.ResourceAction, Reason: req.Reason,
			ReviewerID: admin.OperatorID, RefundPublishQuota: req.RefundPublishQuota,
		})
		response.JSON(w, resp, err)
	}
}

type adminResourceStore interface {
	adminlogic.PendingResourceStore
	adminlogic.ReviewResourceStore
	adminlogic.ResourceReportAdminStore
}

func requireAdminResourceContext(r *http.Request, svcCtx *svc.ServiceContext) (session.AdminTokenSubject, adminResourceStore, error) {
	admin, err := handlerx.AdminFromContext(adminResourceRequestContext(r))
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.ResourceModel == nil {
		logx.WithContext(adminResourceRequestContext(r)).Errorw("后台资源 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
		return session.AdminTokenSubject{}, nil, errx.New(errx.CodeInternalError, "资源管理服务暂不可用，请稍后重试")
	}
	return admin, svcCtx.APIStore, nil
}

func parseAdminResourceRequest(r *http.Request, target any) error {
	if err := httpx.Parse(r, target); err != nil {
		return errx.New(errx.CodeValidationFailed, "资源管理参数格式不正确")
	}
	return nil
}

func adminResourceRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
