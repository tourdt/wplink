package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxResourceReportReviewReasonLength = 300

type ResourceReportAdminStore interface {
	ListAdminResourceReports(ctx context.Context, filter model.AdminResourceReportFilter) (model.ListAdminResourceReportsResult, error)
	ReviewResourceReport(ctx context.Context, input model.ReviewResourceReportInput) (model.ReviewResourceReportResult, error)
}

type ListResourceReportsReq struct {
	Status   string
	Page     int64
	PageSize int64
}

type AdminResourceReportItem struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	ResourceID       string `json:"resourceId"`
	ResourceTitle    string `json:"resourceTitle"`
	ResourceStatus   string `json:"resourceStatus"`
	MerchantID       string `json:"merchantId"`
	MerchantName     string `json:"merchantName"`
	ReportCount      int64  `json:"reportCount"`
	ReasonCode       string `json:"reasonCode"`
	ReasonText       string `json:"reasonText,omitempty"`
	LatestReportedAt string `json:"latestReportedAt"`
}

type ListResourceReportsResp struct {
	Items    []AdminResourceReportItem `json:"items"`
	Page     int64                     `json:"page"`
	PageSize int64                     `json:"pageSize"`
	Total    int64                     `json:"total"`
}

type ReviewResourceReportReq struct {
	Action             string
	ResourceAction     string
	Reason             string
	ReviewerID         string
	RefundPublishQuota bool
}

type ReviewResourceReportResp struct {
	ID                  string `json:"id"`
	Status              string `json:"status"`
	ResourceID          string `json:"resourceId"`
	ResourceStatus      string `json:"resourceStatus"`
	ResolvedReportCount int64  `json:"resolvedReportCount"`
	RefundPublishQuota  bool   `json:"refundPublishQuota"`
	Message             string `json:"message"`
}

type ResourceReportLogic struct {
	store ResourceReportAdminStore
}

func NewResourceReportLogic(store ResourceReportAdminStore) *ResourceReportLogic {
	return &ResourceReportLogic{store: store}
}

func (l *ResourceReportLogic) ListResourceReports(ctx context.Context, req ListResourceReportsReq) (ListResourceReportsResp, error) {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = model.ResourceReportStatusPending
	}
	if status != model.ResourceReportStatusPending && status != model.ResourceReportStatusValid && status != model.ResourceReportStatusInvalid {
		return ListResourceReportsResp{}, errx.New(errx.CodeValidationFailed, "举报状态不正确")
	}
	result, err := l.store.ListAdminResourceReports(ctx, model.AdminResourceReportFilter{
		Status:   status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		LogAdminFailure(ctx, "加载资源举报列表失败", "list_resource_reports", err,
			logx.Field("statusFiltered", status != ""), logx.Field("page", req.Page), logx.Field("pageSize", req.PageSize))
		return ListResourceReportsResp{}, errx.New(errx.CodeInternalError, "举报列表加载失败，请稍后重试")
	}
	items := make([]AdminResourceReportItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, AdminResourceReportItem{
			ID:               item.ID,
			Status:           item.Status,
			ResourceID:       item.ResourceID,
			ResourceTitle:    item.ResourceTitle,
			ResourceStatus:   item.ResourceStatus,
			MerchantID:       item.MerchantID,
			MerchantName:     item.MerchantName,
			ReportCount:      item.ReportCount,
			ReasonCode:       item.ReasonCode,
			ReasonText:       item.ReasonText,
			LatestReportedAt: item.LatestReportedAt,
		})
	}
	return ListResourceReportsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func (l *ResourceReportLogic) ReviewResourceReport(ctx context.Context, reportID string, req ReviewResourceReportReq) (ReviewResourceReportResp, error) {
	input, err := normalizeReviewResourceReportInput(reportID, req)
	if err != nil {
		return ReviewResourceReportResp{}, err
	}
	result, err := l.store.ReviewResourceReport(ctx, input)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewResourceReportResp{}, errx.New(errx.CodeStateConflict, "举报已被处理，请刷新后查看")
	}
	if err != nil {
		LogAdminFailure(ctx, "审核资源举报失败", "review_resource_report", err,
			logx.Field("reportId", input.ReportID), logx.Field("operatorId", input.ReviewerID),
			logx.Field("action", input.Action), logx.Field("resourceAction", input.ResourceAction),
			logx.Field("refundPublishQuota", input.RefundPublishQuota))
		return ReviewResourceReportResp{}, errx.New(errx.CodeInternalError, "举报处理失败，请稍后重试")
	}
	logx.Infof("审核资源举报成功: reportId=%s resourceId=%s reviewerId=%s action=%s resourceAction=%s resolvedReportCount=%d refundPublishQuota=%t", result.ID, result.ResourceID, input.ReviewerID, input.Action, input.ResourceAction, result.ResolvedReportCount, result.RefundPublishQuota)
	return ReviewResourceReportResp{
		ID:                  result.ID,
		Status:              result.Status,
		ResourceID:          result.ResourceID,
		ResourceStatus:      result.ResourceStatus,
		ResolvedReportCount: result.ResolvedReportCount,
		RefundPublishQuota:  result.RefundPublishQuota,
		Message:             resourceReportReviewMessage(result),
	}, nil
}

func normalizeReviewResourceReportInput(reportID string, req ReviewResourceReportReq) (model.ReviewResourceReportInput, error) {
	input := model.ReviewResourceReportInput{
		ReportID:           strings.TrimSpace(reportID),
		Action:             strings.TrimSpace(req.Action),
		ResourceAction:     strings.TrimSpace(req.ResourceAction),
		Reason:             strings.TrimSpace(req.Reason),
		ReviewerID:         strings.TrimSpace(req.ReviewerID),
		RefundPublishQuota: req.RefundPublishQuota,
	}
	if input.ReportID == "" {
		return model.ReviewResourceReportInput{}, errx.New(errx.CodeValidationFailed, "举报不存在或已处理")
	}
	if input.Action != model.ResourceReportActionValid && input.Action != model.ResourceReportActionInvalid {
		return model.ReviewResourceReportInput{}, errx.New(errx.CodeValidationFailed, "举报处理结论不正确")
	}
	if input.ResourceAction == "" {
		if input.Action == model.ResourceReportActionValid {
			input.ResourceAction = model.ResourceReportResourceActionTakeDown
		} else {
			input.ResourceAction = model.ResourceReportResourceActionNone
		}
	}
	if input.Action == model.ResourceReportActionInvalid && input.ResourceAction != model.ResourceReportResourceActionNone {
		return model.ReviewResourceReportInput{}, errx.New(errx.CodeValidationFailed, "举报不成立时无需处理资源")
	}
	if input.Action == model.ResourceReportActionValid && input.ResourceAction != model.ResourceReportResourceActionTakeDown {
		return model.ReviewResourceReportInput{}, errx.New(errx.CodeValidationFailed, "举报成立时请选择资源处理方式")
	}
	if input.RefundPublishQuota && input.Action != model.ResourceReportActionValid {
		return model.ReviewResourceReportInput{}, errx.New(errx.CodeValidationFailed, "只有举报成立并下架时才能退还发布次数")
	}
	if utf8.RuneCountInString(input.Reason) > maxResourceReportReviewReasonLength {
		return model.ReviewResourceReportInput{}, errx.New(errx.CodeValidationFailed, "处理原因请控制在 300 字以内")
	}
	if input.Reason == "" {
		input.Reason = defaultResourceReportReviewReason(input.Action)
	}
	return input, nil
}

func defaultResourceReportReviewReason(action string) string {
	if action == model.ResourceReportActionValid {
		return "举报成立，资源已按规则处理"
	}
	return "举报不成立，资源维持原状态"
}

func resourceReportReviewMessage(result model.ReviewResourceReportResult) string {
	message := "举报已处理"
	if result.ResolvedReportCount > 1 {
		message = fmt.Sprintf("举报已处理，已同步处理同资源 %d 条待处理举报", result.ResolvedReportCount)
	}
	if result.RefundPublishQuota {
		message += "，已退还 1 次发布次数"
	}
	return message
}
