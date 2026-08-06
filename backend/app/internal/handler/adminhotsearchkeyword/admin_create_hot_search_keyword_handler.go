package adminhotsearchkeyword

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminCreateHotSearchKeywordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminSaveHotSearchKeywordHTTPHandler(svcCtx, false)
}
