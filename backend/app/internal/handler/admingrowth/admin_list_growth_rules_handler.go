package admingrowth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListGrowthRulesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListGrowthRulesHTTPHandler(svcCtx)
}
