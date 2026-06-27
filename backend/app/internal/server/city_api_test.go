package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestAPIRouterListsCityStations(t *testing.T) {
	router := NewAPIRouter(&fakeCityAPIStore{
		stations: []model.CityStation{{
			ID:              "city-1",
			Code:            "zhili",
			Name:            "织里",
			PrimaryCategory: "童装",
			Status:          "active",
		}},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/city-stations", nil)
	router.ServeHTTP(rec, req)

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

func TestAPIRouterListsResourceTypesByCity(t *testing.T) {
	router := NewAPIRouter(&fakeCityAPIStore{
		resourceTypes: []model.ResourceTypeConfig{{
			ID:               "type-1",
			TypeCode:         "inventory",
			TypeName:         "库存",
			DefaultValidDays: 30,
			RequiredFields:   []string{"title", "category"},
			FilterFields:     []string{"category"},
			DisplayTemplate:  model.JSONMap{"title": "title"},
		}},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/city-stations/zhili/resource-types", nil)
	router.ServeHTTP(rec, req)

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
	if first["typeCode"] != "inventory" || first["typeName"] != "库存" {
		t.Fatalf("first type = %#v, want inventory", first)
	}
}

func TestAPIRouterReturnsNotFoundForUnsupportedCitySubPath(t *testing.T) {
	router := NewAPIRouter(&fakeCityAPIStore{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/city-stations/zhili/unknown", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for malformed path", rec.Code)
	}
}

type fakeCityAPIStore struct {
	stations      []model.CityStation
	resourceTypes []model.ResourceTypeConfig
	cityCode      string
}

func (s *fakeCityAPIStore) ListActiveCityStations(ctx context.Context) ([]model.CityStation, error) {
	return append([]model.CityStation(nil), s.stations...), nil
}

func (s *fakeCityAPIStore) ListActiveResourceTypesByCityCode(ctx context.Context, cityCode string) ([]model.ResourceTypeConfig, error) {
	s.cityCode = cityCode
	return append([]model.ResourceTypeConfig(nil), s.resourceTypes...), nil
}
