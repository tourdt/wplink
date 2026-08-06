package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func SubmitMapBindRequestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return submitMapBindRequestHTTPHandler(svcCtx)
}
