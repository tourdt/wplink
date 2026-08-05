package handlerx

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	authlogic "wplink/backend/app/internal/logic/auth"
	"wplink/backend/app/internal/session"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminTokenService interface {
	ParseAdminToken(ctx context.Context, token string) (session.AdminTokenSubject, error)
}

type adminSubjectContextKey struct{}

// RequiredUser 校验 Bearer Token 并返回服务端确认的用户身份。
func RequiredUser(r *http.Request, service authlogic.TokenService) (session.UserTokenSubject, error) {
	ctx := requestContext(r)
	if dependencyMissing(service) {
		logx.WithContext(ctx).Error("用户身份校验依赖未配置")
		return session.UserTokenSubject{}, errx.New(errx.CodeInternalError, "登录服务暂不可用，请稍后重试")
	}
	token, ok := bearerToken(r)
	if !ok {
		return session.UserTokenSubject{}, errx.New(errx.CodeUnauthorized, "请先登录")
	}
	subject, err := service.ParseUserToken(ctx, token)
	if err != nil {
		// 不记录 Token 和原始错误文本，避免认证信息或下游敏感细节进入日志。
		logx.WithContext(ctx).Infow("用户 Token 校验未通过", logx.Field("errorType", fmt.Sprintf("%T", err)))
		return session.UserTokenSubject{}, errx.New(errx.CodeUnauthorized, "登录已过期，请重新登录")
	}
	return subject, nil
}

// OptionalUser 允许真正的匿名请求；一旦客户端携带 Authorization，就必须通过校验。
func OptionalUser(r *http.Request, service authlogic.TokenService) (session.UserTokenSubject, bool, error) {
	if dependencyMissing(service) {
		logx.WithContext(requestContext(r)).Error("可选用户身份校验依赖未配置")
		return session.UserTokenSubject{}, false, errx.New(errx.CodeInternalError, "登录服务暂不可用，请稍后重试")
	}
	if strings.TrimSpace(authorizationHeader(r)) == "" {
		return session.UserTokenSubject{}, false, nil
	}
	subject, err := RequiredUser(r, service)
	if err != nil {
		return session.UserTokenSubject{}, false, err
	}
	return subject, true, nil
}

// OptionalAdmin 尝试解析后台身份，供同时支持用户和管理员身份的业务边界使用。
func OptionalAdmin(r *http.Request, service AdminTokenService) (session.AdminTokenSubject, bool) {
	if dependencyMissing(service) {
		logx.WithContext(requestContext(r)).Error("可选管理员身份校验依赖未配置")
		return session.AdminTokenSubject{}, false
	}
	token, ok := bearerToken(r)
	if !ok {
		return session.AdminTokenSubject{}, false
	}
	subject, err := service.ParseAdminToken(requestContext(r), token)
	if err != nil {
		return session.AdminTokenSubject{}, false
	}
	return subject, true
}

// ContextWithAdmin 仅供认证中间件写入已验证的管理员身份。
func ContextWithAdmin(ctx context.Context, subject session.AdminTokenSubject) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, adminSubjectContextKey{}, subject)
}

// AdminFromContext 读取认证中间件写入的管理员身份，避免 Handler 再次解析 Header。
func AdminFromContext(ctx context.Context) (session.AdminTokenSubject, error) {
	if ctx == nil {
		return session.AdminTokenSubject{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	subject, ok := ctx.Value(adminSubjectContextKey{}).(session.AdminTokenSubject)
	if !ok || strings.TrimSpace(subject.OperatorID) == "" {
		return session.AdminTokenSubject{}, errx.New(errx.CodeUnauthorized, "请先登录管理后台")
	}
	return subject, nil
}

func bearerToken(r *http.Request) (string, bool) {
	header := strings.TrimSpace(authorizationHeader(r))
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return token, token != ""
}

func authorizationHeader(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Header.Get("Authorization")
}

func requestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

// dependencyMissing 同时识别 nil interface 和装入 interface 的 typed nil，
// 避免依赖配置错误继续进入方法分派并造成越权、panic 或错误的 401 响应。
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
