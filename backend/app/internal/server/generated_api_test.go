package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"wplink/backend/app/internal/handler"
	"wplink/backend/app/internal/middleware"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/rest"
)

const missingGeneratedAdminAuthMessage = "newGeneratedAPIServer requires explicit AdminAuth; use newGeneratedAPIServerWithFailClosedAdminAuth only when testing missing auth dependencies"

func TestNewGeneratedAPIServerRequiresExplicitAdminAuth(t *testing.T) {
	if os.Getenv("WPLINK_TEST_MISSING_GENERATED_ADMIN_AUTH") == "1" {
		newGeneratedAPIServer(t, &svc.ServiceContext{})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestNewGeneratedAPIServerRequiresExplicitAdminAuth$")
	cmd.Env = append(os.Environ(), "WPLINK_TEST_MISSING_GENERATED_ADMIN_AUTH=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("newGeneratedAPIServer accepted a ServiceContext without AdminAuth")
	}
	subprocessOutput := string(output)
	if !strings.Contains(subprocessOutput, missingGeneratedAdminAuthMessage) {
		t.Fatalf("subprocess output=%q, want explicit failure %q", output, missingGeneratedAdminAuthMessage)
	}
	if strings.Contains(subprocessOutput, "panic:") {
		t.Fatalf("subprocess output=%q, want explicit AdminAuth failure without panic", output)
	}
}

func TestGeneratedAPIServerFailsClosedWhenAdminAuthDependencyIsMissing(t *testing.T) {
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{})

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
		t.Fatal("newGeneratedAPIServer requires a non-nil ServiceContext with explicit AdminAuth")
		return nil
	}
	if svcCtx.AdminAuth == nil {
		t.Fatal(missingGeneratedAdminAuthMessage)
		return nil
	}

	server := rest.MustNewServer(rest.RestConf{
		Host: "127.0.0.1",
		Port: 4000,
	})
	handler.RegisterHandlers(server, svcCtx)
	t.Cleanup(server.Stop)
	return server
}

// newGeneratedAPIServerWithFailClosedAdminAuth 仅用于调用方明确选择“认证依赖缺失”语义的测试。
// 它使用生产中间件的默认拒绝分支，后台请求返回 500，不能用于模拟已正确装配的生产服务。
func newGeneratedAPIServerWithFailClosedAdminAuth(t *testing.T, svcCtx *svc.ServiceContext) *rest.Server {
	t.Helper()
	if svcCtx == nil {
		svcCtx = &svc.ServiceContext{}
	}
	if svcCtx.AdminAuth != nil {
		t.Fatal("newGeneratedAPIServerWithFailClosedAdminAuth requires AdminAuth to be absent")
		return nil
	}
	svcCtx.AdminAuth = middleware.NewAdminAuthMiddleware(nil).Handle
	return newGeneratedAPIServer(t, svcCtx)
}
