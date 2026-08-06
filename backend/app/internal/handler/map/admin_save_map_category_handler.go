package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminSaveMapCategoryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveMapCategoryHTTPHandler(svcCtx)
}
