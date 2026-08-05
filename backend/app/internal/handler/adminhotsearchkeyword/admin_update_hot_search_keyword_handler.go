package adminhotsearchkeyword

import (
	"net/http"

	"wplink/backend/app/internal/handler/handlerx"
	"wplink/backend/app/internal/svc"
)

func AdminUpdateHotSearchKeywordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return handlerx.NotMigrated("AdminUpdateHotSearchKeywordHandler")
}
