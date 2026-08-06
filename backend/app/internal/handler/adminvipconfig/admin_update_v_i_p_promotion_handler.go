package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateVIPPromotionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveVIPPromotionHTTPHandler(svcCtx, true)
}
