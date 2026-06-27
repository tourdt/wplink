package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestDashboardOverviewReturnsCountsAndTasks(t *testing.T) {
	store := &fakeDashboardStore{
		overview: model.AdminDashboardOverview{
			PendingResourceCount:     2,
			PendingVerificationCount: 1,
			PendingDemandCount:       3,
			TodayContactCount:        8,
			Tasks:                    []model.AdminDashboardTask{{Type: "resource", Title: "待审核资源", CityName: "织里"}},
		},
	}
	logic := NewDashboardLogic(store)

	resp, err := logic.GetOverview(context.Background(), DashboardOverviewReq{CityCode: " zhili "})
	if err != nil {
		t.Fatalf("GetOverview() error = %v", err)
	}

	if store.cityCode != "zhili" || resp.Metrics.PendingResourceCount != 2 || len(resp.Tasks) != 1 {
		t.Fatalf("resp = %#v, cityCode = %q", resp, store.cityCode)
	}
}

type fakeDashboardStore struct {
	cityCode string
	overview model.AdminDashboardOverview
}

func (s *fakeDashboardStore) GetAdminDashboardOverview(ctx context.Context, cityCode string) (model.AdminDashboardOverview, error) {
	s.cityCode = cityCode
	return s.overview, nil
}
