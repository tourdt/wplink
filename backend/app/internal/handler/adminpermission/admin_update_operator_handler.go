package adminpermission

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateOperatorHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveOperatorHTTPHandler(svcCtx, true)
}
