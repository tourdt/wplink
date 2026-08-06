package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateVIPPromotionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveVIPPromotionHTTPHandler(svcCtx, false)
}
