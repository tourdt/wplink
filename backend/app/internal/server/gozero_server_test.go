package server

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"wplink/backend/app/internal/config"
	"wplink/backend/app/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func TestNewGoZeroServerMountsHealthAPIAndAdmin(t *testing.T) {
	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("admin:" + r.URL.Path))
	})
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api:" + r.URL.Path))
	})

	svcCtx := &svc.ServiceContext{CityStore: &fakeCityAPIStore{}}
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, svcCtx, adminHandler, apiHandler)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()
	if hasRoute(srv.Routes(), http.MethodGet, "/api/v1/city-stations") {
		t.Fatalf("routes = %#v, city stations should use single api fallback instead of dedicated go-zero route", srv.Routes())
	}
	if hasRoute(srv.Routes(), http.MethodGet, "/api/v1/me/resources/:resourceId/detail") {
		t.Fatalf("routes = %#v, API routes should use single api fallback instead of compat go-zero routes", srv.Routes())
	}

	healthRec := httptest.NewRecorder()
	srv.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthRec.Code != http.StatusOK || healthRec.Body.String() != "ok" {
		t.Fatalf("health response = %d %q, want 200 ok", healthRec.Code, healthRec.Body.String())
	}
	if !hasRoute(srv.Routes(), http.MethodGet, "/readyz") {
		t.Fatalf("routes = %#v, want readyz route", srv.Routes())
	}

	apiRec := httptest.NewRecorder()
	srv.ServeHTTP(apiRec, httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1", nil))
	if apiRec.Code != http.StatusOK || apiRec.Body.String() != "api:/api/v1/merchants/merchant-1" {
		t.Fatalf("api response = %d %q, want api handler", apiRec.Code, apiRec.Body.String())
	}

	ownResourceDetailRec := httptest.NewRecorder()
	srv.ServeHTTP(ownResourceDetailRec, httptest.NewRequest(http.MethodGet, "/api/v1/me/resources/resource-1/detail?merchantId=merchant-1", nil))
	if ownResourceDetailRec.Code != http.StatusOK || ownResourceDetailRec.Body.String() != "api:/api/v1/me/resources/resource-1/detail" {
		t.Fatalf("own resource detail response = %d %q, want api handler", ownResourceDetailRec.Code, ownResourceDetailRec.Body.String())
	}

	adminRec := httptest.NewRecorder()
	srv.ServeHTTP(adminRec, httptest.NewRequest(http.MethodGet, "/admin/assets/index.js", nil))
	if adminRec.Code != http.StatusOK || adminRec.Body.String() != "admin:/admin/assets/index.js" {
		t.Fatalf("admin response = %d %q, want admin handler", adminRec.Code, adminRec.Body.String())
	}
}

func TestGoZeroReadyzChecksDatabase(t *testing.T) {
	db := openReadyzTestDB(t, nil)
	defer db.Close()
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, &svc.ServiceContext{DB: db}, nil, nil)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("readyz response = %d %q, want 200 ok", rec.Code, rec.Body.String())
	}
}

func TestGoZeroReadyzReturnsUnavailableWhenDatabasePingFails(t *testing.T) {
	db := openReadyzTestDB(t, errors.New("database unavailable"))
	defer db.Close()
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, &svc.ServiceContext{DB: db}, nil, nil)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable || rec.Body.String() != "not ready" {
		t.Fatalf("readyz response = %d %q, want 503 not ready", rec.Code, rec.Body.String())
	}
}

func TestGoZeroAdminLoginRouteUsesSingleAPIHandler(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api:" + r.URL.Path))
	})
	svcCtx := &svc.ServiceContext{CityStore: &fakeCityAPIStore{}}
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, svcCtx, nil, apiHandler)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"loginName":"operator","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "api:/api/v1/admin/auth/login" {
		t.Fatalf("admin login response = %d %q, want single api handler", rec.Code, rec.Body.String())
	}
}

func TestGoZeroServerAllowsAdminLoginCORSPreflight(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, &svc.ServiceContext{}, nil, apiHandler)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/auth/login", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight response code = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:5173" {
		t.Fatalf("allow origin = %q, want local admin dev origin", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("allow headers = %q, want Authorization included", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestGoZeroServerRestrictsProductionCORS(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", RuntimeMode: "production", Host: "127.0.0.1", Port: 4000}, &svc.ServiceContext{}, nil, apiHandler)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	localRec := httptest.NewRecorder()
	localReq := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/auth/login", nil)
	localReq.Header.Set("Origin", "http://127.0.0.1:5173")
	localReq.Header.Set("Access-Control-Request-Method", http.MethodPost)
	srv.ServeHTTP(localRec, localReq)
	if localRec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("local allow origin = %q, want blocked in production", localRec.Header().Get("Access-Control-Allow-Origin"))
	}

	prodRec := httptest.NewRecorder()
	prodReq := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/auth/login", nil)
	prodReq.Header.Set("Origin", "https://app.iweipi.cn")
	prodReq.Header.Set("Access-Control-Request-Method", http.MethodPost)
	srv.ServeHTTP(prodRec, prodReq)
	if prodRec.Header().Get("Access-Control-Allow-Origin") != "https://app.iweipi.cn" {
		t.Fatalf("production allow origin = %q, want app.iweipi.cn", prodRec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRestConfFromConfigEnablesSafeAccessLogs(t *testing.T) {
	cfg := config.Config{
		Name: "wplink-api",
		Host: "127.0.0.1",
		Port: 4000,
		Log: config.LogConfig{
			Mode:     "file",
			Encoding: "json",
			Path:     "logs",
			Level:    "info",
			Rotation: "daily",
			KeepDays: 7,
			Stat:     true,
		},
	}

	restConf := restConfFromConfig(cfg)
	if restConf.Log.Mode != "file" || restConf.Log.Path != "logs" || restConf.Log.Rotation != "daily" || restConf.Log.KeepDays != 7 {
		t.Fatalf("rest log config = %#v, want copied daily file log config", restConf.Log)
	}
	if !restConf.Middlewares.Log {
		t.Fatalf("middlewares = %#v, want access log middleware enabled", restConf.Middlewares)
	}
	if restConf.Verbose {
		t.Fatal("Verbose = true, want false to avoid dumping request bodies and sensitive headers")
	}
}

func openReadyzTestDB(t *testing.T, pingErr error) *sql.DB {
	t.Helper()
	driverName := "readyz-test-" + strconv.FormatInt(readyzTestDriverSeq.Add(1), 10)
	sql.Register(driverName, readyzTestDriver{pingErr: pingErr})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

var readyzTestDriverSeq atomic.Int64

type readyzTestDriver struct {
	pingErr error
}

func (d readyzTestDriver) Open(name string) (driver.Conn, error) {
	return readyzTestConn{pingErr: d.pingErr}, nil
}

type readyzTestConn struct {
	pingErr error
}

func (c readyzTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c readyzTestConn) Close() error {
	return nil
}

func (c readyzTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (c readyzTestConn) Ping(ctx context.Context) error {
	return c.pingErr
}

func TestGoZeroCityRoutesUseSingleAPIHandler(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api:" + r.URL.Path))
	})
	svcCtx := &svc.ServiceContext{CityStore: &fakeCityAPIStore{}}
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, svcCtx, nil, apiHandler)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	stationsRec := httptest.NewRecorder()
	srv.ServeHTTP(stationsRec, httptest.NewRequest(http.MethodGet, "/api/v1/city-stations", nil))
	if stationsRec.Code != http.StatusOK || stationsRec.Body.String() != "api:/api/v1/city-stations" {
		t.Fatalf("stations response = %d %q, want single api handler", stationsRec.Code, stationsRec.Body.String())
	}

	typesRec := httptest.NewRecorder()
	srv.ServeHTTP(typesRec, httptest.NewRequest(http.MethodGet, "/api/v1/city-stations/zhili/resource-types", nil))
	if typesRec.Code != http.StatusOK || typesRec.Body.String() != "api:/api/v1/city-stations/zhili/resource-types" {
		t.Fatalf("types response = %d %q, want single api handler", typesRec.Code, typesRec.Body.String())
	}
}

func hasRoute(routes []rest.Route, method string, path string) bool {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}
