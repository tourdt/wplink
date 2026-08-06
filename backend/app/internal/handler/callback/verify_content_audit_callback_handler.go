package callback

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func VerifyContentAuditCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return verifyContentAuditCallbackHTTPHandler(svcCtx)
}
