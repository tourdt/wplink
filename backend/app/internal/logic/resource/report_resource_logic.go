package resource

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

const maxResourceReportReasonTextLength = 200

type ResourceReportStore interface {
	CreateResourceReport(ctx context.Context, input model.CreateResourceReportInput) (model.ResourceReportResult, error)
}

type ReportResourceReq struct {
	ResourceID     string
	ReporterUserID string
	ReasonCode     string
	ReasonText     string
	Evidence       model.JSONMap
}

type ReportResourceResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ReportResourceLogic struct {
	store ResourceReportStore
}

func NewReportResourceLogic(store ResourceReportStore) *ReportResourceLogic {
	return &ReportResourceLogic{store: store}
}

func (l *ReportResourceLogic) ReportResource(ctx context.Context, req ReportResourceReq) (ReportResourceResp, error) {
	input := model.CreateResourceReportInput{
		ResourceID:     strings.TrimSpace(req.ResourceID),
		ReporterUserID: strings.TrimSpace(req.ReporterUserID),
		ReasonCode:     strings.TrimSpace(req.ReasonCode),
		ReasonText:     strings.TrimSpace(req.ReasonText),
		Evidence:       req.Evidence,
	}
	if input.ResourceID == "" {
		return ReportResourceResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	if input.ReporterUserID == "" {
		return ReportResourceResp{}, errx.New(errx.CodeUnauthorized, "请先登录后举报")
	}
	if !validResourceReportReasonCode(input.ReasonCode) {
		return ReportResourceResp{}, errx.New(errx.CodeValidationFailed, "请选择举报原因")
	}
	if utf8.RuneCountInString(input.ReasonText) > maxResourceReportReasonTextLength {
		return ReportResourceResp{}, errx.New(errx.CodeValidationFailed, "举报说明请控制在 200 字以内")
	}

	result, err := l.store.CreateResourceReport(ctx, input)
	if errors.Is(err, sql.ErrNoRows) {
		return ReportResourceResp{}, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
	}
	if err != nil {
		logx.Errorf("提交资源举报失败: resourceId=%s reporterUserId=%s reasonCode=%s err=%+v", input.ResourceID, input.ReporterUserID, input.ReasonCode, err)
		return ReportResourceResp{}, errx.New(errx.CodeInternalError, "举报提交失败，请稍后重试")
	}
	logx.Infof("提交资源举报成功: resourceId=%s reporterUserId=%s reportId=%s reasonCode=%s", input.ResourceID, input.ReporterUserID, result.ID, input.ReasonCode)
	return ReportResourceResp{ID: result.ID, Status: result.Status, Message: "举报已提交，平台会尽快核查"}, nil
}

func validResourceReportReasonCode(code string) bool {
	switch code {
	case "fake_info", "unreachable", "inaccurate_price_quantity", "image_infringement", "illegal_content", "malicious_redirect", "other":
		return true
	default:
		return false
	}
}
