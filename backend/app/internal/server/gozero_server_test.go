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
	"wplink/backend/app/internal/logic/adminauth"
	"wplink/backend/app/internal/model"
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
	if !hasRoute(srv.Routes(), http.MethodGet, "/api/v1/city-stations") {
		t.Fatalf("routes = %#v, want go-zero registered city stations route", srv.Routes())
	}
	if !hasRoute(srv.Routes(), http.MethodGet, "/api/v1/me/resources/:resourceId/detail") {
		t.Fatalf("routes = %#v, want own resource detail compat route", srv.Routes())
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

func TestGoZeroAdminLoginRouteUsesServiceContext(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("admin login route should not fall back to APIRouter")
	})
	loginService := &fakeGoZeroAdminLoginService{}
	svcCtx := &svc.ServiceContext{
		CityStore:         &fakeCityAPIStore{},
		AdminLoginService: loginService,
	}
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, svcCtx, nil, apiHandler)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"loginName":"operator","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if loginService.req.LoginName != "operator" || loginService.req.Password != "secret123" {
		t.Fatalf("login request = %#v, want parsed body", loginService.req)
	}
	if data["token"] != "admin-token" || data["userId"] != "user-1" {
		t.Fatalf("login data = %#v, want token and userId", data)
	}
}

func TestGoZeroServerAllowsAdminLoginCORSPreflight(t *testing.T) {
	loginService := &fakeGoZeroAdminLoginService{}
	svcCtx := &svc.ServiceContext{AdminLoginService: loginService}
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, svcCtx, nil, nil)
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
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("allow origin = %q, want *", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("allow headers = %q, want Authorization included", rec.Header().Get("Access-Control-Allow-Headers"))
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

func TestGoZeroCityRoutesUseServiceContextStore(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("city route should not fall back to APIRouter")
	})
	svcCtx := &svc.ServiceContext{
		CityStore: &fakeCityAPIStore{
			stations: []model.CityStation{{
				ID:              "city-1",
				Code:            "zhili",
				Name:            "织里",
				PrimaryCategory: "童装",
				Status:          "active",
			}},
			resourceTypes: []model.ResourceTypeConfig{{
				ID:               "type-1",
				TypeCode:         "inventory",
				TypeName:         "库存",
				DefaultValidDays: 30,
				RequiredFields:   []string{"title"},
			}},
		},
	}
	srv, err := NewGoZeroServer(config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000}, svcCtx, nil, apiHandler)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	stationsRec := httptest.NewRecorder()
	srv.ServeHTTP(stationsRec, httptest.NewRequest(http.MethodGet, "/api/v1/city-stations", nil))
	stationsData := decodeEnvelopeData(t, stationsRec, http.StatusOK)
	if len(stationsData["items"].([]interface{})) != 1 {
		t.Fatalf("stations data = %#v, want one city", stationsData)
	}

	typesRec := httptest.NewRecorder()
	srv.ServeHTTP(typesRec, httptest.NewRequest(http.MethodGet, "/api/v1/city-stations/zhili/resource-types", nil))
	typesData := decodeEnvelopeData(t, typesRec, http.StatusOK)
	if svcCtx.CityStore.(*fakeCityAPIStore).cityCode != "zhili" || len(typesData["items"].([]interface{})) != 1 {
		t.Fatalf("cityCode = %q typesData = %#v, want zhili and one type", svcCtx.CityStore.(*fakeCityAPIStore).cityCode, typesData)
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

type fakeGoZeroAdminLoginService struct {
	req adminauth.LoginRequest
}

func (s *fakeGoZeroAdminLoginService) Login(ctx context.Context, req adminauth.LoginRequest) (adminauth.LoginResponse, error) {
	s.req = req
	return adminauth.LoginResponse{Token: "admin-token", UserID: "user-1", Roles: []string{adminauth.RolePlatformOperator}}, nil
}
