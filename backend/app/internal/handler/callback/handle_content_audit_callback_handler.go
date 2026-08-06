package callback

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func HandleContentAuditCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return handleContentAuditCallbackHTTPHandler(svcCtx)
}
