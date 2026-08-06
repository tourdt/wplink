package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func MarkResourceDealHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return ownerActionHTTPHandler(svcCtx, "deal")
}
