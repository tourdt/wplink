package admin

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strconv"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type MerchantAdminStore interface {
	ListMerchants(ctx context.Context, filter model.ListMerchantsFilter) (model.ListMerchantsResult, error)
	ListAdminMerchantLocationReports(ctx context.Context, filter model.AdminMerchantLocationReportFilter) (model.ListAdminMerchantLocationReportsResult, error)
	ReviewMerchantLocationReport(ctx context.Context, input model.ReviewMerchantLocationReportInput) (model.ReviewMerchantLocationReportResult, error)
	UpdateMerchantLocationStatus(ctx context.Context, input model.UpdateMerchantLocationStatusInput) (model.UpdateMerchantLocationStatusResult, error)
}

type ListMerchantsReq struct {
	CityCode       string
	MerchantType   string
	Status         string
	LocationStatus string
	Keyword        string
	Page           int64
	PageSize       int64
}

type MerchantListItem struct {
	ID                         string `json:"id"`
	Name                       string `json:"name"`
	MerchantType               string `json:"merchantType"`
	Status                     string `json:"status"`
	LocationStatus             string `json:"locationStatus"`
	MarketName                 string `json:"marketName,omitempty"`
	BuildingName               string `json:"buildingName,omitempty"`
	FloorNo                    string `json:"floorNo,omitempty"`
	BoothNo                    string `json:"boothNo,omitempty"`
	PendingLocationReportCount int64  `json:"pendingLocationReportCount"`
	LastActiveAt               string `json:"lastActiveAt,omitempty"`
}

type ListMerchantsResp struct {
	Items    []MerchantListItem `json:"items"`
	Page     int64              `json:"page"`
	PageSize int64              `json:"pageSize"`
	Total    int64              `json:"total"`
}

type ListMerchantLocationReportsReq struct {
	Status   string
	Page     int64
	PageSize int64
}

type MerchantLocationReportItem struct {
	ReportID         string        `json:"reportId"`
	Status           string        `json:"status"`
	MerchantID       string        `json:"merchantId"`
	MerchantName     string        `json:"merchantName"`
	ReporterUserID   string        `json:"reporterUserId"`
	ReasonCode       string        `json:"reasonCode"`
	Description      string        `json:"description,omitempty"`
	LocationSnapshot model.JSONMap `json:"locationSnapshot"`
	ReviewedBy       string        `json:"reviewedBy,omitempty"`
	ReviewNote       string        `json:"reviewNote,omitempty"`
	CreatedAt        string        `json:"createdAt"`
	ReviewedAt       string        `json:"reviewedAt,omitempty"`
}

type ListMerchantLocationReportsResp struct {
	Items    []MerchantLocationReportItem `json:"items"`
	Page     int64                        `json:"page"`
	PageSize int64                        `json:"pageSize"`
	Total    int64                        `json:"total"`
}

type ReviewMerchantLocationReportReq struct {
	Status     string `json:"status"`
	ReviewNote string `json:"reviewNote,omitempty"`
}

type ReviewMerchantLocationReportResp struct {
	ReportID   string `json:"reportId"`
	MerchantID string `json:"merchantId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type UpdateMerchantLocationStatusReq struct {
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}

type UpdateMerchantLocationStatusResp struct {
	MerchantID     string `json:"merchantId"`
	LocationStatus string `json:"locationStatus"`
	Message        string `json:"message"`
}

type MerchantAdminLogic struct {
	store MerchantAdminStore
}

func NewMerchantAdminLogic(store MerchantAdminStore) *MerchantAdminLogic {
	return &MerchantAdminLogic{store: store}
}

func (l *MerchantAdminLogic) ListMerchants(ctx context.Context, req ListMerchantsReq) (ListMerchantsResp, error) {
	result, err := l.store.ListMerchants(ctx, model.ListMerchantsFilter{
		CityCode:       strings.TrimSpace(req.CityCode),
		MerchantType:   strings.TrimSpace(req.MerchantType),
		Status:         strings.TrimSpace(req.Status),
		LocationStatus: strings.TrimSpace(req.LocationStatus),
		Keyword:        strings.TrimSpace(req.Keyword),
		Page:           req.Page,
		PageSize:       req.PageSize,
	})
	if err != nil {
		return ListMerchantsResp{}, err
	}

	items := make([]MerchantListItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, MerchantListItem{
			ID:                         item.ID,
			Name:                       item.Name,
			MerchantType:               item.MerchantType,
			Status:                     item.Status,
			LocationStatus:             item.LocationStatus,
			MarketName:                 item.MarketName,
			BuildingName:               item.BuildingName,
			FloorNo:                    item.FloorNo,
			BoothNo:                    item.BoothNo,
			PendingLocationReportCount: item.PendingLocationReportCount,
			LastActiveAt:               item.LastActiveAt,
		})
	}
	return ListMerchantsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func (l *MerchantAdminLogic) ListMerchantLocationReports(ctx context.Context, req ListMerchantLocationReportsReq) (ListMerchantLocationReportsResp, error) {
	status := strings.TrimSpace(req.Status)
	if status != "" && !isMerchantLocationReportStatus(status, true) {
		return ListMerchantLocationReportsResp{}, errx.New(errx.CodeValidationFailed, "位置反馈状态不正确")
	}
	if merchantLocationReportPaginationOverflows(req.Page, req.PageSize) {
		return ListMerchantLocationReportsResp{}, errx.New(errx.CodeValidationFailed, "分页参数超出范围，请重新选择页码")
	}
	result, err := l.store.ListAdminMerchantLocationReports(ctx, model.AdminMerchantLocationReportFilter{Status: status, Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		logx.WithContext(ctx).Errorw("加载商家位置反馈列表失败",
			logx.Field("action", "list_merchant_location_reports"), logx.Field("status", status),
			logx.Field("page", req.Page), logx.Field("pageSize", req.PageSize), logx.Field("err", err))
		return ListMerchantLocationReportsResp{}, errx.New(errx.CodeInternalError, "位置反馈列表加载失败，请稍后重试")
	}
	items := make([]MerchantLocationReportItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, MerchantLocationReportItem{
			ReportID: item.ID, Status: item.Status, MerchantID: item.MerchantID, MerchantName: item.MerchantName,
			ReporterUserID: item.ReporterUserID, ReasonCode: item.ReasonCode, Description: item.Description,
			LocationSnapshot: item.LocationSnapshot, ReviewedBy: item.ReviewedBy, ReviewNote: item.ReviewNote,
			CreatedAt: item.CreatedAt, ReviewedAt: item.ReviewedAt,
		})
	}
	return ListMerchantLocationReportsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}

func merchantLocationReportPaginationOverflows(page int64, pageSize int64) bool {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page > 1 && page-1 > math.MaxInt64/pageSize
}

func (l *MerchantAdminLogic) ReviewMerchantLocationReport(ctx context.Context, reportID string, req ReviewMerchantLocationReportReq, actor AdminPermissionActor) (ReviewMerchantLocationReportResp, error) {
	reportID = strings.TrimSpace(reportID)
	status := strings.TrimSpace(req.Status)
	reviewNote := strings.TrimSpace(req.ReviewNote)
	adminID := strings.TrimSpace(actor.OperatorID)
	if reportID == "" || adminID == "" {
		return ReviewMerchantLocationReportResp{}, errx.New(errx.CodeValidationFailed, "位置反馈处理参数不完整")
	}
	if !isPositiveAdminBigintID(reportID) {
		return ReviewMerchantLocationReportResp{}, errx.New(errx.CodeValidationFailed, "位置反馈编号格式不正确")
	}
	if !isMerchantLocationReportStatus(status, false) {
		return ReviewMerchantLocationReportResp{}, errx.New(errx.CodeValidationFailed, "位置反馈处理结果不正确")
	}
	if len([]rune(reviewNote)) > 200 {
		return ReviewMerchantLocationReportResp{}, errx.New(errx.CodeValidationFailed, "处理说明最多可填写 200 个字")
	}
	result, err := l.store.ReviewMerchantLocationReport(ctx, model.ReviewMerchantLocationReportInput{
		ReportID: reportID, Status: status, ReviewNote: reviewNote, AdminID: adminID, AdminRole: primaryAdminRole(actor.Roles),
	})
	if err != nil {
		if errors.Is(err, model.ErrMerchantLocationReportAlreadyReviewed) {
			return ReviewMerchantLocationReportResp{}, errx.New(errx.CodeStateConflict, "该位置反馈已处理，请刷新后查看")
		}
		if errors.Is(err, sql.ErrNoRows) {
			return ReviewMerchantLocationReportResp{}, errx.New(errx.CodeResourceNotFound, "位置反馈不存在或已删除")
		}
		logx.WithContext(ctx).Errorw("后台处理商家位置反馈失败",
			logx.Field("merchantId", result.MerchantID), logx.Field("reportId", reportID),
			logx.Field("adminId", adminID), logx.Field("action", status), logx.Field("err", err))
		return ReviewMerchantLocationReportResp{}, errx.New(errx.CodeInternalError, "位置反馈处理失败，请稍后重试")
	}
	return ReviewMerchantLocationReportResp{ReportID: result.ReportID, MerchantID: result.MerchantID, Status: result.Status, Message: "位置反馈已处理"}, nil
}

func (l *MerchantAdminLogic) UpdateMerchantLocationStatus(ctx context.Context, merchantID string, req UpdateMerchantLocationStatusReq, actor AdminPermissionActor) (UpdateMerchantLocationStatusResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	action := strings.TrimSpace(req.Action)
	reason := strings.TrimSpace(req.Reason)
	adminID := strings.TrimSpace(actor.OperatorID)
	if merchantID == "" || adminID == "" {
		return UpdateMerchantLocationStatusResp{}, errx.New(errx.CodeValidationFailed, "商家位置操作参数不完整")
	}
	if !isPositiveAdminBigintID(merchantID) {
		return UpdateMerchantLocationStatusResp{}, errx.New(errx.CodeValidationFailed, "商家编号格式不正确")
	}
	status := ""
	switch action {
	case "hide":
		status = model.MerchantLocationStatusHidden
	case "restore":
		status = model.MerchantLocationStatusVisible
	default:
		return UpdateMerchantLocationStatusResp{}, errx.New(errx.CodeValidationFailed, "商家位置操作不正确")
	}
	if len([]rune(reason)) > 200 {
		return UpdateMerchantLocationStatusResp{}, errx.New(errx.CodeValidationFailed, "操作原因最多可填写 200 个字")
	}
	result, err := l.store.UpdateMerchantLocationStatus(ctx, model.UpdateMerchantLocationStatusInput{
		MerchantID: merchantID, Status: status, Reason: reason, AdminID: adminID, AdminRole: primaryAdminRole(actor.Roles),
	})
	if err != nil {
		if errors.Is(err, model.ErrMerchantLocationCoordinatesMissing) {
			return UpdateMerchantLocationStatusResp{}, errx.New(errx.CodeValidationFailed, "商家尚未设置有效地图位置，无法恢复")
		}
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateMerchantLocationStatusResp{}, errx.New(errx.CodeResourceNotFound, "商家不存在或已停用")
		}
		logx.WithContext(ctx).Errorw("后台更新商家位置状态失败",
			logx.Field("merchantId", merchantID), logx.Field("reportId", ""),
			logx.Field("adminId", adminID), logx.Field("action", action), logx.Field("err", err))
		return UpdateMerchantLocationStatusResp{}, errx.New(errx.CodeInternalError, "商家位置状态更新失败，请稍后重试")
	}
	return UpdateMerchantLocationStatusResp{MerchantID: result.MerchantID, LocationStatus: result.NewStatus, Message: "商家位置状态已更新"}, nil
}

func isMerchantLocationReportStatus(status string, includePending bool) bool {
	if includePending && status == model.MerchantLocationReportStatusPending {
		return true
	}
	return status == model.MerchantLocationReportStatusValid || status == model.MerchantLocationReportStatusInvalid || status == model.MerchantLocationReportStatusResolved
}

func primaryAdminRole(roles []string) string {
	for _, role := range roles {
		if role == permission.RoleSuperAdmin {
			return permission.RoleSuperAdmin
		}
	}
	return permission.RolePlatformOperator
}

func isPositiveAdminBigintID(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return err == nil && parsed > 0
}
