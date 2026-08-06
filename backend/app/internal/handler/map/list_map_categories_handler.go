package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListMapCategoriesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listMapCategoriesHTTPHandler(svcCtx)
}
