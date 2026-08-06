package adminpermission

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateOperatorHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveOperatorHTTPHandler(svcCtx, false)
}
