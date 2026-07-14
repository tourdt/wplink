package server

import (
	"net/http"

	adminauthhandler "wplink/backend/app/internal/handler/adminauth"
	cityhandler "wplink/backend/app/internal/handler/city"
	"wplink/backend/app/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func registerGoctlHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	if server == nil || svcCtx == nil {
		return
	}
	registerGoctlAdminAuthHandlers(server, svcCtx)
	registerGoctlCityHandlers(server, svcCtx)
}

func registerGoctlAdminAuthHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	if svcCtx.AdminLoginService == nil {
		return
	}
	// 已迁移到 goctl handler 形态的后台登录路由直接注册到 go-zero，避免继续依赖兜底 ServeMux。
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/auth/login",
				Handler: adminauthhandler.AdminLoginHandler(svcCtx),
			},
		},
		rest.WithPrefix("/api/v1/admin"),
	)
}

func registerGoctlCityHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	if svcCtx.CityStore == nil {
		return
	}
	// 城市站是公共基础数据，先接入已存在的 goctl handler；未迁移端点仍由兼容 API router 处理。
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/city-stations",
				Handler: cityhandler.ListCityStationsHandler(svcCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/city-stations/:cityCode/resource-types",
				Handler: cityhandler.ListCityResourceTypesHandler(svcCtx),
			},
		},
		rest.WithPrefix("/api/v1"),
	)
}
