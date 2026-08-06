package adminpermission

import (
	"context"
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	adminlogic "wplink/backend/app/internal/logic/admin"
	"wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/svc"
	"wplink/backend/app/internal/types"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func adminListOperatorsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, store, err := requireAdminPermissionContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminListOperatorsReq
		if err := parseAdminPermissionRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewAdminPermissionLogic(store, adminauth.BcryptPasswordHasher{}).ListOperators(r.Context(), adminlogic.ListAdminOperatorsReq{
			Keyword: req.Keyword, Role: req.Role, Status: req.Status, Page: req.Page, PageSize: req.PageSize,
		}, actor)
		response.JSON(w, resp, err)
	}
}

func adminSaveOperatorHTTPHandler(svcCtx *svc.ServiceContext, update bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, store, err := requireAdminPermissionContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminSaveOperatorReq
		if err := parseAdminPermissionRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		logic := adminlogic.NewAdminPermissionLogic(store, adminauth.BcryptPasswordHasher{})
		input := adminlogic.SaveAdminOperatorReq{
			LoginName: req.LoginName, RealName: req.RealName, Password: req.Password, Roles: req.Roles, Status: req.Status,
		}
		var resp adminlogic.SaveAdminOperatorResp
		if update {
			resp, err = logic.UpdateOperator(r.Context(), pathvar.Vars(r)["operatorId"], input, actor)
		} else {
			resp, err = logic.CreateOperator(r.Context(), input, actor)
		}
		response.JSON(w, resp, err)
	}
}

func adminUpdateOperatorStatusHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, store, err := requireAdminPermissionContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminUpdateOperatorStatusReq
		if err := parseAdminPermissionRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewAdminPermissionLogic(store, adminauth.BcryptPasswordHasher{}).UpdateOperatorStatus(
			r.Context(), pathvar.Vars(r)["operatorId"], adminlogic.UpdateAdminOperatorStatusReq{Status: req.Status}, actor,
		)
		response.JSON(w, resp, err)
	}
}

func adminListModulePermissionsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, store, err := requireAdminPermissionContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewAdminPermissionLogic(store, adminauth.BcryptPasswordHasher{}).ListModulePermissions(r.Context(), actor)
		response.JSON(w, resp, err)
	}
}

func adminUpdateRoleModulePermissionsHTTPHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, store, err := requireAdminPermissionContext(r, svcCtx)
		if err != nil {
			response.JSON(w, nil, err)
			return
		}
		var req types.AdminUpdateRoleModulePermissionsReq
		if err := parseAdminPermissionRequest(r, &req); err != nil {
			response.JSON(w, nil, err)
			return
		}
		resp, err := adminlogic.NewAdminPermissionLogic(store, adminauth.BcryptPasswordHasher{}).UpdateRoleModulePermissions(
			r.Context(), pathvar.Vars(r)["roleCode"], adminlogic.UpdateAdminRoleModulePermissionsReq{Modules: req.Modules}, actor,
		)
		response.JSON(w, resp, err)
	}
}

func requireAdminPermissionContext(r *http.Request, svcCtx *svc.ServiceContext) (adminlogic.AdminPermissionActor, adminlogic.AdminPermissionStore, error) {
	admin, err := handlerx.AdminFromContext(adminPermissionRequestContext(r))
	if err != nil {
		return adminlogic.AdminPermissionActor{}, nil, err
	}
	if svcCtx == nil || svcCtx.APIStore == nil || svcCtx.APIStore.AdminPermissionModel == nil {
		logx.WithContext(adminPermissionRequestContext(r)).Errorw("后台权限管理 Handler 存储依赖未配置", logx.Field("operatorId", admin.OperatorID))
		return adminlogic.AdminPermissionActor{}, nil, errx.New(errx.CodeInternalError, "管理员权限服务暂不可用，请稍后重试")
	}
	return adminlogic.AdminPermissionActor{OperatorID: admin.OperatorID, Roles: append([]string(nil), admin.Roles...)}, svcCtx.APIStore, nil
}

func parseAdminPermissionRequest(r *http.Request, target any) error {
	if err := httpx.Parse(r, target); err != nil {
		return errx.New(errx.CodeValidationFailed, "管理员权限参数格式不正确")
	}
	return nil
}

func adminPermissionRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
