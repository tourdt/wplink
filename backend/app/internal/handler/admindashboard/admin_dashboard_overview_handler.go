package admindashboard

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminDashboardOverviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminDashboardOverviewHTTPHandler(svcCtx)
}
