package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func CreateContactUnlockOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return contactUnlockOrderHTTPHandler(svcCtx)
}
