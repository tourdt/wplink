package maplogic

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

var mapObjectReportReasons = map[string]map[string]struct{}{
	model.MapObjectReportKindLocationCorrection: {
		"address_inaccurate":    {},
		"navigation_inaccurate": {},
		"moved_or_closed":       {},
		"information_expired":   {},
	},
	model.MapObjectReportKindRiskReport: {
		"impersonation":      {},
		"false_information":  {},
		"prohibited_content": {},
		"other":              {},
	},
}

type ReportStore interface {
	CreateMapObjectReport(ctx context.Context, input model.MapObjectReportInput) (model.MapObjectReportResult, error)
}

type ReportLogic struct {
	store ReportStore
}

func NewReportLogic(store ReportStore) *ReportLogic {
	return &ReportLogic{store: store}
}

type SubmitMapObjectReportReq struct {
	ReporterUserID string `json:"-"`
	ReasonCode     string `json:"reasonCode"`
	Description    string `json:"description"`
}

type MapObjectReportItem struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	ActiveReportCount int64  `json:"activeReportCount"`
	WarningTriggered  bool   `json:"warningTriggered"`
}

type SubmitMapObjectReportResp struct {
	Item    MapObjectReportItem `json:"item"`
	Message string              `json:"message"`
}

func (l *ReportLogic) Submit(ctx context.Context, objectID string, kind string, req SubmitMapObjectReportReq) (SubmitMapObjectReportResp, error) {
	objectID = strings.TrimSpace(objectID)
	kind = strings.TrimSpace(kind)
	reporterUserID := strings.TrimSpace(req.ReporterUserID)
	reasonCode := strings.TrimSpace(req.ReasonCode)
	description := strings.TrimSpace(req.Description)

	if objectID == "" {
		return SubmitMapObjectReportResp{}, errx.New(errx.CodeValidationFailed, "请选择要反馈的地图点位")
	}
	if reporterUserID == "" {
		return SubmitMapObjectReportResp{}, errx.New(errx.CodeUnauthorized, "请先登录")
	}
	allowedReasons, ok := mapObjectReportReasons[kind]
	if !ok {
		return SubmitMapObjectReportResp{}, errx.New(errx.CodeValidationFailed, "反馈类型不支持")
	}
	if _, ok := allowedReasons[reasonCode]; !ok {
		return SubmitMapObjectReportResp{}, errx.New(errx.CodeValidationFailed, "请选择有效的反馈原因")
	}
	if reasonCode == "other" && description == "" {
		return SubmitMapObjectReportResp{}, errx.New(errx.CodeValidationFailed, "请补充说明具体问题")
	}
	if utf8.RuneCountInString(description) > 500 {
		return SubmitMapObjectReportResp{}, errx.New(errx.CodeValidationFailed, "补充说明不能超过500个字")
	}

	result, err := l.store.CreateMapObjectReport(ctx, model.MapObjectReportInput{
		ObjectID:       objectID,
		ReporterUserID: reporterUserID,
		Kind:           kind,
		ReasonCode:     reasonCode,
		Description:    description,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SubmitMapObjectReportResp{}, errx.New(errx.CodeResourceNotFound, "地图点位不存在或已下线")
		}
		logx.Errorf("地图点位反馈入库失败: objectId=%s reporterUserId=%s kind=%s reasonCode=%s err=%+v", objectID, reporterUserID, kind, reasonCode, err)
		return SubmitMapObjectReportResp{}, errx.New(errx.CodeInternalError, "反馈提交失败，请稍后重试")
	}

	message := "反馈已记录，感谢帮助完善拿货地图"
	if result.WarningTriggered {
		message = "反馈已记录，该点位已增加风险提醒"
	}
	logx.Infof("地图点位反馈已入库: reportId=%s objectId=%s reporterUserId=%s kind=%s reasonCode=%s activeReportCount=%d warningTriggered=%t",
		result.ID, objectID, reporterUserID, kind, reasonCode, result.ActiveReportCount, result.WarningTriggered)
	return SubmitMapObjectReportResp{
		Item: MapObjectReportItem{
			ID:                result.ID,
			Status:            result.Status,
			ActiveReportCount: result.ActiveReportCount,
			WarningTriggered:  result.WarningTriggered,
		},
		Message: message,
	}, nil
}
