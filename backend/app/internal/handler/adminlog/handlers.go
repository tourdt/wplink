package adminlog

import (
	"context"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/task"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func adminListOperationLogsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		operatorID, err := requireAdminLogIdentity(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.OperationLogModel == nil {
			logAdminLogDependencyFailure(r, "operation_logs")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "操作日志服务暂不可用，请稍后重试"))
			return
		}
		var req types.AdminOperationLogsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "操作日志筛选参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewOperationLogLogic(svcCtx.APIStore).ListOperationLogs(r.Context(), adminlogic.OperationLogsReq{
			ObjectType: req.ObjectType, ObjectID: req.ObjectId, OperatorID: req.OperatorId, Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			adminlogic.LogAdminFailure(r.Context(), "加载后台操作日志失败", "list_operation_logs", err,
				logx.Field("operatorId", operatorID), logx.Field("objectTypeFiltered", req.ObjectType != ""),
				logx.Field("objectIdFiltered", req.ObjectId != ""), logx.Field("operatorFiltered", req.OperatorId != ""),
				logx.Field("page", req.Page), logx.Field("pageSize", req.PageSize))
		}
		response.JSON(w, resp, err)
	}
}

func adminListSearchLogsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		operatorID, err := requireAdminLogIdentity(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.SearchLogModel == nil {
			logAdminLogDependencyFailure(r, "search_logs")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "搜索日志服务暂不可用，请稍后重试"))
			return
		}
		var req types.AdminSearchLogsReq
		if err := httpx.Parse(r, &req); err != nil {
			response.JSON(w, nil, errx.New(errx.CodeValidationFailed, "搜索日志筛选参数格式不正确"))
			return
		}
		resp, err := adminlogic.NewSearchLogLogic(svcCtx.APIStore).ListSearchLogs(r.Context(), adminlogic.SearchLogsReq{
			CityCode: req.CityCode, Keyword: req.Keyword, Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			adminlogic.LogAdminFailure(r.Context(), "加载后台搜索日志失败", "list_search_logs", err,
				logx.Field("operatorId", operatorID), logx.Field("cityFiltered", req.CityCode != ""),
				logx.Field("keywordFiltered", req.Keyword != ""), logx.Field("page", req.Page), logx.Field("pageSize", req.PageSize))
		}
		response.JSON(w, resp, err)
	}
}

func adminRunResourceLifecycleHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		operatorID, err := requireAdminLogIdentity(r)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		// 生命周期任务依赖资源和消息两套持久化能力；任一缺失均默认拒绝，避免 typed-nil 嵌入模型在任务中 panic。
		if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.ResourceModel == nil || svcCtx.APIStore.MessageModel == nil {
			logAdminLogDependencyFailure(r, "resource_lifecycle")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "资源生命周期服务暂不可用，请稍后重试"))
			return
		}
		result, err := task.NewResourceLifecycleTask(svcCtx.APIStore).Run(r.Context())
		if err != nil {
			adminlogic.LogAdminFailure(r.Context(), "执行资源生命周期任务失败", "run_resource_lifecycle", err,
				logx.Field("operatorId", operatorID), logx.Field("stage", task.ResourceLifecycleErrorStage(err)))
		}
		response.JSON(w, map[string]int64{
			"expiredCount": result.ExpiredCount, "expiringReminderCount": result.ExpiringReminderCount,
		}, err)
	}
}

func requireAdminLogIdentity(r *http.Request) (string, error) {
	admin, err := handlerx.AdminFromContext(adminLogRequestContext(r))
	return admin.OperatorID, err
}

func logAdminLogDependencyFailure(r *http.Request, dependency string) {
	operatorID, _ := requireAdminLogIdentity(r)
	logx.WithContext(adminLogRequestContext(r)).Errorw("后台日志任务 Handler 存储依赖未配置",
		logx.Field("operatorId", operatorID), logx.Field("dependency", dependency))
}

func adminLogRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
