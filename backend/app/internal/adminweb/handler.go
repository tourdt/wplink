package adminweb

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed dist
var embeddedDist embed.FS

func EmbeddedHandler(basePath string) (http.Handler, error) {
	dist, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return nil, err
	}
	return NewHandler(dist, basePath), nil
}

func NewHandler(dist fs.FS, basePath string) http.Handler {
	basePath = normalizeBasePath(basePath)
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, basePath) {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == strings.TrimSuffix(basePath, "/") {
			http.Redirect(w, r, basePath, http.StatusMovedPermanently)
			return
		}

		assetPath := strings.TrimPrefix(r.URL.Path, basePath)
		assetPath = strings.TrimPrefix(path.Clean("/"+assetPath), "/")
		if assetPath == "" {
			serveIndex(w, r, dist)
			return
		}
		if fileExists(dist, assetPath) {
			req := r.Clone(r.Context())
			req.URL.Path = "/" + assetPath
			fileServer.ServeHTTP(w, req)
			return
		}
		// 管理后台是 Vue history 模式，页面路由刷新时必须回退到 index.html。
		if path.Ext(assetPath) == "" {
			serveIndex(w, r, dist)
			return
		}
		http.NotFound(w, r)
	})
}

func normalizeBasePath(basePath string) string {
	basePath = "/" + strings.Trim(basePath, "/")
	if basePath == "/" {
		return "/"
	}
	return basePath + "/"
}

func serveIndex(w http.ResponseWriter, r *http.Request, dist fs.FS) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		http.Error(w, "admin web dist is not embedded", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", fileModTime(dist, "index.html"), bytes.NewReader(index))
}

func fileExists(dist fs.FS, name string) bool {
	info, err := fs.Stat(dist, name)
	return err == nil && !info.IsDir()
}

func fileModTime(dist fs.FS, name string) time.Time {
	if info, err := fs.Stat(dist, name); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}
