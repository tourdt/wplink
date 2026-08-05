package adminvipconfig

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	"wplink/backend/app/internal/svc"
)

func AdminListQuotaPacksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return handlerx.NotMigrated("AdminListQuotaPacksHandler")
}
