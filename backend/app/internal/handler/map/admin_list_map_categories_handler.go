package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListMapCategoriesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListMapCategoriesHTTPHandler(svcCtx)
}
