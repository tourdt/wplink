package auth

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	"wplink/backend/app/internal/svc"
)

func WechatLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return handlerx.NotMigrated("WechatLoginHandler")
}
