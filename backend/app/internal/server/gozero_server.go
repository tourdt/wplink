package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func NewGoZeroServer(cfg config.Config, svcCtx *svc.ServiceContext, adminHandler http.Handler, apiHandler http.Handler) (*rest.Server, error) {
	srv, err := rest.NewServer(
		restConfFromConfig(cfg),
		rest.WithNotFoundHandler(fallbackHandler(adminHandler, apiHandler)),
		rest.WithCors(corsAllowedOrigins(cfg)...),
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
	registerGoctlHandlers(srv, svcCtx)
	return srv, nil
}

func corsAllowedOrigins(cfg config.Config) []string {
	if config.IsProductionMode(cfg.RuntimeMode) {
		return []string{"https://app.iweipi.cn"}
	}
	return []string{
		"https://app.iweipi.cn",
		"http://127.0.0.1:5173",
		"http://localhost:5173",
	}
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
