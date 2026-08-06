package adminhotsearchkeyword

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func AdminListHotSearchKeywordsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return adminListHotSearchKeywordsHTTPHandler(svcCtx)
}
