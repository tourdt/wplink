package server

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"wplink/backend/app/internal/config"
	adminauthlogic "wplink/backend/app/internal/logic/adminauth"
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
		t.Fatalf("routes = %#v, city stations should be registered as go-zero route", srv.Routes())
	}
	if !hasRoute(srv.Routes(), http.MethodGet, "/api/v1/city-stations/:cityCode/resource-types") {
		t.Fatalf("routes = %#v, city resource types should be registered as go-zero route", srv.Routes())
	}
	if hasRoute(srv.Routes(), http.MethodGet, "/api/v1/me/resources/:resourceId/detail") {
		t.Fatalf("routes = %#v, unmigrated API routes should still use the compatibility fallback", srv.Routes())
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

func TestGoZeroAdminLoginRouteUsesGoctlHandlerWhenDependencyReady(t *testing.T) {
	loginService := &goZeroAdminLoginService{
		resp: adminauthlogic.LoginResponse{
			Token:      "token-1",
			OperatorID: "operator-1",
			Roles:      []string{adminauthlogic.RoleSuperAdmin},
		},
	}
	srv := newGeneratedAPIServer(t, &svc.ServiceContext{AdminLoginService: loginService})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"loginName":"operator","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("admin login response code = %d body = %q, want 200", rec.Code, rec.Body.String())
	}
	if loginService.req.LoginName != "operator" || loginService.req.Password != "secret123" {
		t.Fatalf("login req = %#v, want parsed request from goctl handler", loginService.req)
	}
	var body struct {
		Data struct {
			Token      string   `json:"token"`
			OperatorID string   `json:"operatorId"`
			Roles      []string `json:"roles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode admin login response: %v", err)
	}
	if body.Data.Token != "token-1" || body.Data.OperatorID != "operator-1" || len(body.Data.Roles) != 1 || body.Data.Roles[0] != adminauthlogic.RoleSuperAdmin {
		t.Fatalf("admin login data = %#v, want fake service response", body.Data)
	}
}

func TestGoZeroAdminLoginGeneratedPreservesClientContextAndRateLimitError(t *testing.T) {
	loginService := &goZeroAdminLoginService{err: adminauthlogic.ErrLoginRateLimited}
	srv := newGeneratedAPIServer(t, &svc.ServiceContext{AdminLoginService: loginService})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"loginName":"operator","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	req.Header.Set("User-Agent", "wplink-admin-test")
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("admin login response = %d %q, want 429", rec.Code, rec.Body.String())
	}
	var body struct {
		ErrorCode string `json:"errorCode"`
		Message   string `json:"msg"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode rate limited response: %v", err)
	}
	if body.ErrorCode != "RATE_LIMITED" || body.Message != "当前网络登录尝试过多，请稍后重试" {
		t.Fatalf("rate limited error = (%q, %q)", body.ErrorCode, body.Message)
	}
	if loginService.req.ClientIP != "203.0.113.7" || loginService.req.UserAgent != "wplink-admin-test" {
		t.Fatalf("client context = ip %q ua %q", loginService.req.ClientIP, loginService.req.UserAgent)
	}
}

func TestGoZeroAdminLoginGeneratedHidesRawInternalError(t *testing.T) {
	loginService := &goZeroAdminLoginService{err: errors.New("sql: connection refused password=secret123")}
	srv := newGeneratedAPIServer(t, &svc.ServiceContext{AdminLoginService: loginService})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(`{"loginName":"operator","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusUnauthorized, "UNAUTHORIZED", "登录失败，请稍后重试")
	if strings.Contains(rec.Body.String(), "connection refused") || strings.Contains(rec.Body.String(), "secret123") {
		t.Fatalf("admin login response leaked internal error or password: %s", rec.Body.String())
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

func TestGoZeroCityRoutesUseGoctlHandlers(t *testing.T) {
	store := &fakeCityAPIStore{
		stations: []model.CityStation{{
			ID:              "city-1",
			Code:            "zhili",
			Name:            "织里",
			PrimaryCategory: "童装",
			Status:          "active",
		}},
		resourceTypes: []model.ResourceTypeConfig{{
			ID:               "type-1",
			TypeCode:         "stock_clearance",
			TypeName:         "库存出售",
			Direction:        model.ResourceDirectionDemand,
			DefaultValidDays: 30,
			RequiredFields:   []string{"title", "category"},
			FilterFields:     []string{"category"},
			DisplayTemplate:  model.JSONMap{"title": "title"},
		}},
	}
	svcCtx := &svc.ServiceContext{CityStore: store}
	srv, err := NewGoZeroServer(
		config.Config{Name: "wplink-api", Host: "127.0.0.1", Port: 4000},
		svcCtx,
		nil,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("city route should use goctl handler, got fallback path %s", r.URL.Path)
		}),
	)
	if err != nil {
		t.Fatalf("NewGoZeroServer() error = %v", err)
	}
	defer srv.Stop()

	stationsRec := httptest.NewRecorder()
	srv.ServeHTTP(stationsRec, httptest.NewRequest(http.MethodGet, "/api/v1/city-stations", nil))
	if stationsRec.Code != http.StatusOK {
		t.Fatalf("stations response = %d %q, want 200", stationsRec.Code, stationsRec.Body.String())
	}
	var stationsBody map[string]interface{}
	if err := json.Unmarshal(stationsRec.Body.Bytes(), &stationsBody); err != nil {
		t.Fatalf("decode stations response: %v", err)
	}
	stationsData := stationsBody["data"].(map[string]interface{})
	stationsItems := stationsData["items"].([]interface{})
	firstStation := stationsItems[0].(map[string]interface{})
	if firstStation["id"] != "city-1" || firstStation["code"] != "zhili" {
		t.Fatalf("first station = %#v, want generated types response", firstStation)
	}

	typesRec := httptest.NewRecorder()
	srv.ServeHTTP(typesRec, httptest.NewRequest(http.MethodGet, "/api/v1/city-stations/zhili/resource-types?direction=demand", nil))
	if typesRec.Code != http.StatusOK {
		t.Fatalf("types response = %d %q, want 200", typesRec.Code, typesRec.Body.String())
	}
	if store.cityCode != "zhili" || store.direction != model.ResourceDirectionDemand {
		t.Fatalf("cityCode = %q direction = %q, want zhili/demand", store.cityCode, store.direction)
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

type goZeroAdminLoginService struct {
	req  adminauthlogic.LoginRequest
	resp adminauthlogic.LoginResponse
	err  error
}

func (s *goZeroAdminLoginService) Login(ctx context.Context, req adminauthlogic.LoginRequest) (adminauthlogic.LoginResponse, error) {
	s.req = req
	return s.resp, s.err
}
