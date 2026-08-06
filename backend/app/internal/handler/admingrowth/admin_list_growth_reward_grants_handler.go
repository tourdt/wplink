package admingrowth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListGrowthRewardGrantsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListGrowthRewardGrantsHTTPHandler(svcCtx)
}
