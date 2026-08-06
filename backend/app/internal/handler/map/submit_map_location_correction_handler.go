package maphandler

import (
	"net/http"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/svc"
)

func SubmitMapLocationCorrectionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return submitMapObjectReportHTTPHandler(svcCtx, model.MapObjectReportKindLocationCorrection)
}
