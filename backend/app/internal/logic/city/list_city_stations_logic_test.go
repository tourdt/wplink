package city

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestListCityStationsReturnsActiveStations(t *testing.T) {
	store := &fakeCityStore{
		stations: []model.CityStation{
			{ID: "city-1", Code: "zhili", Name: "织里", PrimaryCategory: "童装", Status: "active"},
		},
	}
	logic := NewListCityStationsLogic(store)

	resp, err := logic.ListCityStations(context.Background())
	if err != nil {
		t.Fatalf("ListCityStations() error = %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].Code != "zhili" {
		t.Fatalf("city code = %q, want zhili", resp.Items[0].Code)
	}
	if !store.activeOnly {
		t.Fatal("store should query active stations only")
	}
}

type fakeCityStore struct {
	activeOnly bool
	stations   []model.CityStation
}

func (s *fakeCityStore) ListActiveCityStations(_ context.Context) ([]model.CityStation, error) {
	s.activeOnly = true
	return append([]model.CityStation(nil), s.stations...), nil
}
