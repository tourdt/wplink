package adminbannertopic

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateBannerTopicHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveBannerTopicHTTPHandler(svcCtx, false)
}
