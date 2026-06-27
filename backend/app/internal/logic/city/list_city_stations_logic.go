package city

import (
	"context"

	"wplink/backend/app/internal/model"
)

type CityStationStore interface {
	ListActiveCityStations(ctx context.Context) ([]model.CityStation, error)
}

type CityStationInfo struct {
	ID              string `json:"id"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	PrimaryCategory string `json:"primaryCategory,omitempty"`
	Status          string `json:"status"`
}

type ListCityStationsResp struct {
	Items []CityStationInfo `json:"items"`
}

type ListCityStationsLogic struct {
	store CityStationStore
}

func NewListCityStationsLogic(store CityStationStore) *ListCityStationsLogic {
	return &ListCityStationsLogic{store: store}
}

func (l *ListCityStationsLogic) ListCityStations(ctx context.Context) (ListCityStationsResp, error) {
	stations, err := l.store.ListActiveCityStations(ctx)
	if err != nil {
		return ListCityStationsResp{}, err
	}

	items := make([]CityStationInfo, 0, len(stations))
	for _, station := range stations {
		items = append(items, CityStationInfo{
			ID:              station.ID,
			Code:            station.Code,
			Name:            station.Name,
			PrimaryCategory: station.PrimaryCategory,
			Status:          station.Status,
		})
	}
	return ListCityStationsResp{Items: items}, nil
}
