package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type DashboardStore interface {
	GetAdminDashboardOverview(ctx context.Context, cityCode string) (model.AdminDashboardOverview, error)
}

type DashboardOverviewReq struct {
	CityCode string
}

type DashboardMetrics struct {
	PendingResourceCount     int64 `json:"pendingResourceCount"`
	PendingVerificationCount int64 `json:"pendingVerificationCount"`
	PendingDemandCount       int64 `json:"pendingDemandCount"`
	TodayContactCount        int64 `json:"todayContactCount"`
}

type DashboardTask struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	CityName  string `json:"cityName"`
	CreatedAt string `json:"createdAt"`
}

type DashboardOverviewResp struct {
	Metrics DashboardMetrics `json:"metrics"`
	Tasks   []DashboardTask  `json:"tasks"`
}

type DashboardLogic struct {
	store DashboardStore
}

func NewDashboardLogic(store DashboardStore) *DashboardLogic {
	return &DashboardLogic{store: store}
}

func (l *DashboardLogic) GetOverview(ctx context.Context, req DashboardOverviewReq) (DashboardOverviewResp, error) {
	overview, err := l.store.GetAdminDashboardOverview(ctx, strings.TrimSpace(req.CityCode))
	if err != nil {
		return DashboardOverviewResp{}, err
	}
	resp := DashboardOverviewResp{
		Metrics: DashboardMetrics{
			PendingResourceCount:     overview.PendingResourceCount,
			PendingVerificationCount: overview.PendingVerificationCount,
			PendingDemandCount:       overview.PendingDemandCount,
			TodayContactCount:        overview.TodayContactCount,
		},
	}
	for _, task := range overview.Tasks {
		resp.Tasks = append(resp.Tasks, DashboardTask{
			Type: task.Type, Title: task.Title, CityName: task.CityName, CreatedAt: task.CreatedAt,
		})
	}
	return resp, nil
}
