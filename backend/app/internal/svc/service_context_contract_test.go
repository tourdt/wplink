package svc_test

import (
	"testing"

	"wplink/backend/app/internal/server"
	"wplink/backend/app/internal/svc"
)

var _ server.ResourceAPIStore = (*svc.APIStore)(nil)

func TestAPIStoreSatisfiesProductionResourceDependencies(t *testing.T) {
	// 生产路由启动前会校验 ResourceAPIStore；这里用编译期断言提前暴露缺失模型。
}
