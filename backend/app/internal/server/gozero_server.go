package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	adminauthhandler "wplink/backend/app/internal/handler/adminauth"
	cityhandler "wplink/backend/app/internal/handler/city"
	"wplink/backend/app/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func NewGoZeroServer(cfg config.Config, svcCtx *svc.ServiceContext, adminHandler http.Handler, apiHandler http.Handler) (*rest.Server, error) {
	srv, err := rest.NewServer(
		restConfFromConfig(cfg),
		rest.WithNotFoundHandler(fallbackHandler(adminHandler, apiHandler)),
		rest.WithCors(),
	)
	if err != nil {
		return nil, err
	}
	srv.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/healthz",
		Handler: healthzHandler,
	})
	srv.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/readyz",
		Handler: readyzHandler(svcCtx),
	})
	registerGoZeroAdminAuthRoutes(srv, svcCtx)
	registerCityRoutes(srv, svcCtx)
	registerCompatAPIRoutes(srv, apiHandler)
	return srv, nil
}

func registerGoZeroAdminAuthRoutes(srv *rest.Server, svcCtx *svc.ServiceContext) {
	if svcCtx == nil || svcCtx.AdminLoginService == nil {
		return
	}
	srv.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/admin/auth/login", Handler: adminauthhandler.AdminLoginHandler(svcCtx)},
	})
}

func registerCityRoutes(srv *rest.Server, svcCtx *svc.ServiceContext) {
	if svcCtx == nil || svcCtx.CityStore == nil {
		return
	}
	srv.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/city-stations", Handler: cityhandler.ListCityStationsHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/v1/city-stations/:cityCode/resource-types", Handler: cityhandler.ListCityResourceTypesHandler(svcCtx)},
	})
}

func restConfFromConfig(cfg config.Config) rest.RestConf {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 4000
	}
	restConf := rest.RestConf{
		Host: host,
		Port: port,
	}
	restConf.Name = cfg.Name
	restConf.Log = cfg.Log
	restConf.Middlewares.Log = true
	return restConf
}

func fallbackHandler(adminHandler http.Handler, apiHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case isAdminPath(r.URL.Path) && adminHandler != nil:
			adminHandler.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/") && apiHandler != nil:
			apiHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func isAdminPath(path string) bool {
	return path == "/admin" || strings.HasPrefix(path, "/admin/")
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func readyzHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svcCtx == nil || svcCtx.DB == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := svcCtx.DB.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
