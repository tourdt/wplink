package adminbannertopic

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateBannerTopicHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveBannerTopicHTTPHandler(svcCtx, true)
}
