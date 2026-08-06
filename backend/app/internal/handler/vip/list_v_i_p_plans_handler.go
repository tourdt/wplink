package vip

import (
	"net/http"

	"wplink/backend/app/internal/svc"
)

func ListVIPPlansHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listVIPPlansHTTPHandler(vipStoreFromServiceContext(svcCtx))
}
