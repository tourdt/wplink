package middleware

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"wplink/backend/app/internal/handler/handlerx"
	"wplink/backend/app/internal/permission"
	"wplink/backend/common/errx"
	"wplink/backend/common/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminAuthMiddleware struct {
	tokenService handlerx.AdminTokenService
}

func NewAdminAuthMiddleware(tokenService handlerx.AdminTokenService) *AdminAuthMiddleware {
	return &AdminAuthMiddleware{tokenService: tokenService}
}

func (m *AdminAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || dependencyMissing(m.tokenService) {
			logx.WithContext(r.Context()).Error("后台认证依赖未配置")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "后台认证服务暂不可用，请稍后重试"))
			return
		}
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录管理后台"))
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" {
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "请先登录管理后台"))
			return
		}
		subject, err := m.tokenService.ParseAdminToken(r.Context(), token)
		if err != nil || strings.TrimSpace(subject.OperatorID) == "" {
			// 不记录 Token 和原始错误文本，只保留错误类型用于区分验证器故障类别。
			logx.WithContext(r.Context()).Infow("后台 Token 校验未通过", logx.Field("errorType", fmt.Sprintf("%T", err)))
			response.JSON(w, nil, errx.New(errx.CodeUnauthorized, "登录已过期，请重新登录"))
			return
		}
		if !permission.CanAccessAdmin(subject.Roles) {
			logx.WithContext(r.Context()).Infow("管理员角色无后台访问权限", logx.Field("operatorId", subject.OperatorID))
			response.JSON(w, nil, errx.New(errx.CodeForbidden, "您没有权限访问管理后台"))
			return
		}

		module := permission.AdminModuleForPath(r.URL.Path)
		modules := append([]string(nil), subject.Modules...)
		if len(modules) == 0 {
			modules = permission.ResolveAdminModules(subject.Roles, nil)
		}
		if !permission.CanAccessAdminModule(subject.Roles, modules, module) {
			logx.WithContext(r.Context()).Infow(
				"管理员无后台模块访问权限",
				logx.Field("operatorId", subject.OperatorID),
				logx.Field("module", module),
			)
			response.JSON(w, nil, errx.New(errx.CodeForbidden, "您没有权限访问该后台功能"))
			return
		}
		if next == nil {
			logx.WithContext(r.Context()).Error("后台认证中间件未配置下游 Handler")
			response.JSON(w, nil, errx.New(errx.CodeInternalError, "后台服务暂不可用，请稍后重试"))
			return
		}
		next(w, r.WithContext(handlerx.ContextWithAdmin(r.Context(), subject)))
	}
}

func dependencyMissing(dependency any) bool {
	if dependency == nil {
		return true
	}
	value := reflect.ValueOf(dependency)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
