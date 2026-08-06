package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListVIPPromotionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListVIPConfigHTTPHandler(svcCtx, vipConfigPromotions)
}
