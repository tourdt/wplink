package resource

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func TakeDownOwnResourceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return ownerActionHTTPHandler(svcCtx, "take_down")
}
