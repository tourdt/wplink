package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

	healthRec := httptest.NewRecorder()
	srv.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthRec.Code != http.StatusOK || healthRec.Body.String() != "ok" {
		t.Fatalf("health response = %d %q, want 200 ok", healthRec.Code, healthRec.Body.String())
	}

	apiRec := httptest.NewRecorder()
	srv.ServeHTTP(apiRec, httptest.NewRequest(http.MethodGet, "/api/v1/merchants/merchant-1", nil))
	if apiRec.Code != http.StatusOK || apiRec.Body.String() != "api:/api/v1/merchants/merchant-1" {
		t.Fatalf("api response = %d %q, want api handler", apiRec.Code, apiRec.Body.String())
	}

	adminRec := httptest.NewRecorder()
	srv.ServeHTTP(adminRec, httptest.NewRequest(http.MethodGet, "/admin/assets/index.js", nil))
	if adminRec.Code != http.StatusOK || adminRec.Body.String() != "admin:/admin/assets/index.js" {
		t.Fatalf("admin response = %d %q, want admin handler", adminRec.Code, adminRec.Body.String())
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
