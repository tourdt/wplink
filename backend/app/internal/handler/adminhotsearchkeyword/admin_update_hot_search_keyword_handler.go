package adminhotsearchkeyword

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminUpdateHotSearchKeywordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveHotSearchKeywordHTTPHandler(svcCtx, true)
}
