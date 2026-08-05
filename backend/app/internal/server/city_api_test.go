package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/svc"
)

func TestCityListsStationsThroughGeneratedRoutes(t *testing.T) {
	store := &fakeCityAPIStore{
		stations: []model.CityStation{{
			ID:              "city-1",
			Code:            "zhili",
			Name:            "织里",
			PrimaryCategory: "童装",
			Status:          "active",
		}},
	}
	server := newGeneratedAPIServer(t, &svc.ServiceContext{CityStore: store})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/city-stations", nil)
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	first := items[0].(map[string]interface{})
	if first["code"] != "zhili" || first["name"] != "织里" {
		t.Fatalf("first city = %#v, want zhili", first)
	}
}

func TestCityListsResourceTypesThroughGeneratedRoutes(t *testing.T) {
	store := &fakeCityAPIStore{
		resourceTypes: []model.ResourceTypeConfig{{
			ID:               "type-1",
			Version:          2,
			TypeCode:         "stock_clearance",
			TypeName:         "库存出售",
			Direction:        model.ResourceDirectionSupply,
			DefaultValidDays: 30,
			RequiredFields:   []string{"title", "category"},
			FilterFields:     []string{"category"},
			DisplayTemplate:  model.JSONMap{"title": "title"},
		}},
	}
	server := newGeneratedAPIServer(t, &svc.ServiceContext{CityStore: store})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/city-stations/zhili/resource-types", nil)
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	first := items[0].(map[string]interface{})
	if first["typeCode"] != "stock_clearance" || first["typeName"] != "库存出售" {
		t.Fatalf("first type = %#v, want stock_clearance with friendly display name", first)
	}
	if first["direction"] != model.ResourceDirectionSupply {
		t.Fatalf("direction = %#v, want supply", first["direction"])
	}
	if first["version"] != float64(2) {
		t.Fatalf("version = %#v, want 2", first["version"])
	}
}

func TestCityPassesResourceTypeDirectionThroughGeneratedRoutes(t *testing.T) {
	store := &fakeCityAPIStore{}
	server := newGeneratedAPIServer(t, &svc.ServiceContext{CityStore: store})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/city-stations/zhili/resource-types?direction=demand", nil)
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if store.cityCode != "zhili" || store.direction != model.ResourceDirectionDemand {
		t.Fatalf("cityCode = %q direction = %q, want zhili/demand", store.cityCode, store.direction)
	}
}

func TestCityReturnsNotFoundForUnsupportedGeneratedSubPath(t *testing.T) {
	server := newGeneratedAPIServer(t, &svc.ServiceContext{CityStore: &fakeCityAPIStore{}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/city-stations/zhili/unknown", nil)
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for malformed path", rec.Code)
	}
}

type fakeCityAPIStore struct {
	stations      []model.CityStation
	resourceTypes []model.ResourceTypeConfig
	cityCode      string
	direction     string
}

func (s *fakeCityAPIStore) ListActiveCityStations(ctx context.Context) ([]model.CityStation, error) {
	return append([]model.CityStation(nil), s.stations...), nil
}

func (s *fakeCityAPIStore) ListActiveResourceTypesByCityCode(ctx context.Context, cityCode string, direction string) ([]model.ResourceTypeConfig, error) {
	s.cityCode = cityCode
	s.direction = direction
	return append([]model.ResourceTypeConfig(nil), s.resourceTypes...), nil
}
