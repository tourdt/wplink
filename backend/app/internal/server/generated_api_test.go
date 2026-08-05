package server

import (
	"testing"

	"wplink/backend/app/internal/handler"
	"wplink/backend/app/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// newGeneratedAPIServer 只注册 goctl 生成的真实路由表，避免测试误走旧 Router 的兼容兜底。
func newGeneratedAPIServer(t *testing.T, svcCtx *svc.ServiceContext) *rest.Server {
	t.Helper()

	server := rest.MustNewServer(rest.RestConf{
		Host: "127.0.0.1",
		Port: 4000,
	})
	handler.RegisterHandlers(server, svcCtx)
	t.Cleanup(server.Stop)
	return server
}
