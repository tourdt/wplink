package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func NewRouter(adminHandler http.Handler, apiHandler ...http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/admin/", adminHandler)
	if len(apiHandler) > 0 && apiHandler[0] != nil {
		mux.Handle("/api/", apiHandler[0])
	} else {
		mux.HandleFunc("/api/", apiNotConnected)
	}
	return mux
}

func apiNotConnected(w http.ResponseWriter, r *http.Request) {
	// 当前阶段先让服务可启动和后台可访问，避免未接入的 API 被误认为 404 静态资源问题。
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    "API_NOT_CONNECTED",
		"message": "后端 API 路由尚未接入，请先完成 handler 接线",
		"path":    strings.TrimSpace(r.URL.Path),
	})
}
