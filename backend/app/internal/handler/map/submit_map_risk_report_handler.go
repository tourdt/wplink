package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/svc"
)

func SubmitMapRiskReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return submitMapObjectReportHTTPHandler(svcCtx, model.MapObjectReportKindRiskReport)
}
