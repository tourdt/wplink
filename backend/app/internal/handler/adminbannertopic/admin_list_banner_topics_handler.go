package adminbannertopic

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListBannerTopicsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListBannerTopicsHTTPHandler(svcCtx)
}
