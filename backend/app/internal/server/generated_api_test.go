package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/handler"
	"wplink/backend/app/internal/middleware"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/rest"
)

const missingGeneratedAdminAuthMessage = "newGeneratedAPIServer requires explicit AdminAuth; use newGeneratedAPIServerWithFailClosedAdminAuth only when testing missing auth dependencies"

func TestGeneratedRouteParity(t *testing.T) {
	server, err := NewGoZeroServer(
		config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000},
		&svc.ServiceContext{AdminAuth: generatedRouteParityAdminAuth()},
		nil,
	)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	t.Cleanup(server.Stop)

	expected := make([]string, 0, len(handler.ContractRoutes())+2)
	for _, route := range handler.ContractRoutes() {
		expected = append(expected, fmt.Sprintf("%s %s", route.Method, route.Path))
	}
	expected = append(expected, "GET /healthz", "GET /readyz")
	actual := make([]string, 0, len(server.Routes()))
	for _, route := range server.Routes() {
		actual = append(actual, fmt.Sprintf("%s %s", route.Method, route.Path))
	}

	if difference := generatedRouteParityDifference(expected, actual); difference != "" {
		t.Fatalf("生产 Server 路由与 ContractRoutes 及平台路由不一致:\n%s", difference)
	}
}

func TestGeneratedRouteParityDifferenceReportsSortedMultisetChanges(t *testing.T) {
	difference := generatedRouteParityDifference(
		[]string{"POST /z", "GET /a", "POST /z"},
		[]string{"DELETE /b", "GET /a", "GET /a"},
	)
	const expected = "缺失:\n- POST /z\n- POST /z\n" +
		"多余:\n- DELETE /b\n- GET /a\n" +
		"ContractRoutes 重复:\n- POST /z（2 次）\n" +
		"生成 Server 重复:\n- GET /a（2 次）"
	if difference != expected {
		t.Fatalf("路由多重集差异输出 = %q，期望 %q", difference, expected)
	}
}

// generatedRouteParityAdminAuth 显式提供只用于路由注册测试的中间件，避免夹具因依赖缺失误走默认拒绝分支。
func generatedRouteParityAdminAuth() rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return next
	}
}

func generatedRouteParityDifference(expected, actual []string) string {
	expectedCount := routeFingerprintCounts(expected)
	actualCount := routeFingerprintCounts(actual)
	missing := routeCountDifference(expectedCount, actualCount)
	extra := routeCountDifference(actualCount, expectedCount)
	expectedDuplicates := duplicateRouteFingerprints(expectedCount)
	actualDuplicates := duplicateRouteFingerprints(actualCount)
	if len(missing) == 0 && len(extra) == 0 && len(expectedDuplicates) == 0 && len(actualDuplicates) == 0 {
		return ""
	}

	sections := make([]string, 0, 4)
	if len(missing) > 0 {
		sections = append(sections, "缺失:\n- "+strings.Join(missing, "\n- "))
	}
	if len(extra) > 0 {
		sections = append(sections, "多余:\n- "+strings.Join(extra, "\n- "))
	}
	if len(expectedDuplicates) > 0 {
		sections = append(sections, "ContractRoutes 重复:\n- "+strings.Join(expectedDuplicates, "\n- "))
	}
	if len(actualDuplicates) > 0 {
		sections = append(sections, "生成 Server 重复:\n- "+strings.Join(actualDuplicates, "\n- "))
	}
	return strings.Join(sections, "\n")
}

func routeFingerprintCounts(routes []string) map[string]int {
	counts := make(map[string]int, len(routes))
	for _, route := range routes {
		counts[route]++
	}
	return counts
}

func routeCountDifference(left, right map[string]int) []string {
	difference := make([]string, 0)
	for fingerprint, count := range left {
		for index := right[fingerprint]; index < count; index++ {
			difference = append(difference, fingerprint)
		}
	}
	sort.Strings(difference)
	return difference
}

func duplicateRouteFingerprints(counts map[string]int) []string {
	duplicates := make([]string, 0)
	for fingerprint, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, fmt.Sprintf("%s（%d 次）", fingerprint, count))
		}
	}
	sort.Strings(duplicates)
	return duplicates
}

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
