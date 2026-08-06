package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"wplink/backend/app/internal/handler"
	"wplink/backend/app/internal/middleware"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/rest"
)

func TestGeneratedAPIServerFailsClosedWhenAdminAuthDependencyIsMissing(t *testing.T) {
	server := newGeneratedAPIServer(t, &svc.ServiceContext{})

	publicBody := assertTask6GeneratedStatus(t, server, httptest.NewRequest(http.MethodGet, "/api/v1/growth-campaigns/active", nil), http.StatusInternalServerError)
	if publicBody["errorCode"] != errx.CodeInternalError || publicBody["msg"] == "接口暂不可用，请稍后重试" {
		t.Fatalf("public body=%#v, want registered real public Handler", publicBody)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/growth-campaigns", nil)
	adminReq.Header.Set("Authorization", "Bearer admin-token")
	adminBody := assertTask6GeneratedStatus(t, server, adminReq, http.StatusInternalServerError)
	if adminBody["errorCode"] != errx.CodeInternalError || adminBody["msg"] != "后台认证服务暂不可用，请稍后重试" {
		t.Fatalf("admin body=%#v, want fail-closed missing AdminAuth dependency", adminBody)
	}
}

// newGeneratedAPIServer 只注册 goctl 生成的真实路由表，避免测试误走旧 Router 的兼容兜底。
func newGeneratedAPIServer(t *testing.T, svcCtx *svc.ServiceContext) *rest.Server {
	t.Helper()
	if svcCtx == nil {
		svcCtx = &svc.ServiceContext{}
	}
	if svcCtx.AdminAuth == nil {
		// 生成路由注册要求 middleware 函数非 nil；测试缺少认证依赖时仍使用生产中间件的默认拒绝语义，绝不放行后台请求。
		svcCtx.AdminAuth = middleware.NewAdminAuthMiddleware(nil).Handle
	}

	server := rest.MustNewServer(rest.RestConf{
		Host: "127.0.0.1",
		Port: 4000,
	})
	handler.RegisterHandlers(server, svcCtx)
	t.Cleanup(server.Stop)
	return server
}
