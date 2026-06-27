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
	PendingResourceCount     int64
	PendingVerificationCount int64
	PendingDemandCount       int64
	TodayContactCount        int64
}

type DashboardTask struct {
	Type      string
	Title     string
	CityName  string
	CreatedAt string
}

type DashboardOverviewResp struct {
	Metrics DashboardMetrics
	Tasks   []DashboardTask
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
