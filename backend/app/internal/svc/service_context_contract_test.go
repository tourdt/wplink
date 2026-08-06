package svc_test

import "wplink/backend/app/internal/svc"

// 生产启动门禁与 ServiceContext 同属 svc，不再依赖已删除的兼容 Router 接口。
var _ func(*svc.ServiceContext) error = svc.ValidateAPIServiceContext
