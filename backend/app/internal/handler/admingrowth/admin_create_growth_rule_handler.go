package admingrowth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateGrowthRuleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveGrowthRuleHTTPHandler(svcCtx, false)
}
