package adminconfig

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

func adminListResourceTypeConfigsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminConfigContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListResourceTypeConfigsReq
		if err := parseAdminConfigRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewResourceTypeConfigLogic(store).ListResourceTypeConfigs(r.Context(), adminlogic.ListResourceTypeConfigsReq{
			CityCode: req.CityCode, Status: req.Status,
		})
		response.JSON(w, resp, err)
	}
}

func adminCreateResourceTypeConfigHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminConfigContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminCreateResourceTypeConfigReq
		if err := parseAdminConfigRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewResourceTypeConfigLogic(store).CreateResourceTypeConfig(r.Context(), adminlogic.CreateResourceTypeConfigReq{
			CityCode: req.CityCode, TypeCode: req.TypeCode, TypeName: req.TypeName, Direction: req.Direction,
			GroupCode: req.GroupCode, GroupName: req.GroupName, GroupSort: req.GroupSort,
			FieldSchema: req.FieldSchema, RequiredFields: req.RequiredFields, FilterFields: req.FilterFields,
			DisplayTemplate: req.DisplayTemplate, ReviewRules: req.ReviewRules, SortWeights: req.SortWeights,
			MessageRules: req.MessageRules, CommercialRules: req.CommercialRules,
			DefaultValidDays: req.DefaultValidDays, Status: req.Status,
		})
		response.JSON(w, resp, err)
	}
}

func adminUpdateResourceTypeConfigHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, store, err := requireAdminConfigContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminUpdateResourceTypeConfigReq
		if err := parseAdminConfigRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		// 配置主键只取路由 path，optimistic version 仍由 Logic 校验并保留中文冲突语义。
		resp, err := adminlogic.NewResourceTypeConfigLogic(store).UpdateResourceTypeConfig(r.Context(), pathvar.Vars(r)["configId"], adminlogic.UpdateResourceTypeConfigReq{
			Version: req.Version, FieldSchema: req.FieldSchema, RequiredFields: req.RequiredFields,
			FilterFields: req.FilterFields, DisplayTemplate: req.DisplayTemplate, ReviewRules: req.ReviewRules,
			SortWeights: req.SortWeights, MessageRules: req.MessageRules, CommercialRules: req.CommercialRules,
			DefaultValidDays: req.DefaultValidDays, Status: req.Status,
		})
		response.JSON(w, resp, err)
	}
}

func requireAdminConfigContext(r *http.Request, svcCtx *svc.ServiceContext) (session.AdminTokenSubject, adminlogic.ResourceTypeConfigStore, error) {
	admin, err := handlerx.AdminFromContext(adminConfigRequestContext(r))
	if err != nil {
		return session.AdminTokenSubject{}, nil, err
	}
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.ResourceTypeConfigModel == nil {
		logx.WithContext(adminConfigRequestContext(r)).Errorw("后台资源类型配置 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
		return session.AdminTokenSubject{}, nil, errx.New(errx.CodeInternalError, "资源类型配置服务暂不可用，请稍后重试")
	}
	return admin, svcCtx.APIStore, nil
}

func parseAdminConfigRequest(r *http.Request, target any) error {
	if err := httpx.Parse(r, target); err != nil {
		return errx.New(errx.CodeValidationFailed, "资源类型配置参数格式不正确")
	}
	return nil
}

func adminConfigRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
