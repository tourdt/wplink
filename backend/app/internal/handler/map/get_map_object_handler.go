package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	"wplink/backend/app/internal/svc"
)

func GetMapObjectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return handlerx.NotMigrated("GetMapObjectHandler")
}
