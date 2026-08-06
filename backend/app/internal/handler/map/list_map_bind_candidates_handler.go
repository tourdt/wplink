package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListMapBindCandidatesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listMapBindCandidatesHTTPHandler(svcCtx)
}
