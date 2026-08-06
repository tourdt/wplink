package admingrowth

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateGrowthRuleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveGrowthRuleHTTPHandler(svcCtx, true)
}
