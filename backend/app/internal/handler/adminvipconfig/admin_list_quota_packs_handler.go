package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListQuotaPacksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListVIPConfigHTTPHandler(svcCtx, vipConfigQuotaPacks)
}
