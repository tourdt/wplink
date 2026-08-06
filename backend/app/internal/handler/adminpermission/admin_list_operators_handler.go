package adminpermission

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListOperatorsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListOperatorsHTTPHandler(svcCtx)
}
