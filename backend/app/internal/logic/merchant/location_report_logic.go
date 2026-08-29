package merchant

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

var allowedMerchantLocationReportReasons = map[string]struct{}{
	"navigation_inaccurate": {},
	"wrong_market":          {},
	"wrong_booth":           {},
	"other":                 {},
}

type MerchantLocationReportStore interface {
	CreateMerchantLocationReport(context.Context, model.CreateMerchantLocationReportInput) (model.CreateMerchantLocationReportResult, error)
}

type CreateMerchantLocationReportReq struct {
	ReasonCode  string `json:"reasonCode"`
	Description string `json:"description,omitempty"`
}

type CreateMerchantLocationReportResp struct {
	ReportID            string `json:"reportId"`
	Status              string `json:"status"`
	ActiveReporterCount int64  `json:"activeReporterCount"`
	AutoHidden          bool   `json:"autoHidden"`
	Idempotent          bool   `json:"idempotent"`
}

type MerchantLocationReportLogic struct {
	store MerchantLocationReportStore
}

func NewMerchantLocationReportLogic(store MerchantLocationReportStore) *MerchantLocationReportLogic {
	return &MerchantLocationReportLogic{store: store}
}

func (l *MerchantLocationReportLogic) CreateMerchantLocationReport(ctx context.Context, merchantID string, reporterUserID string, req CreateMerchantLocationReportReq) (CreateMerchantLocationReportResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	reporterUserID = strings.TrimSpace(reporterUserID)
	reasonCode := strings.TrimSpace(req.ReasonCode)
	description := strings.TrimSpace(req.Description)
	if !model.IsValidMerchantDirectoryID(merchantID) {
		return CreateMerchantLocationReportResp{}, errx.New(errx.CodeValidationFailed, "商家编号格式不正确")
	}
	if _, allowed := allowedMerchantLocationReportReasons[reasonCode]; !allowed {
		return CreateMerchantLocationReportResp{}, errx.New(errx.CodeValidationFailed, "请选择正确的位置问题类型")
	}
	if len([]rune(description)) > 200 {
		return CreateMerchantLocationReportResp{}, errx.New(errx.CodeValidationFailed, "补充说明最多可填写 200 个字")
	}

	result, err := l.store.CreateMerchantLocationReport(ctx, model.CreateMerchantLocationReportInput{
		MerchantID: merchantID, ReporterUserID: reporterUserID, ReasonCode: reasonCode, Description: description,
	})
	if err != nil {
		if errors.Is(err, model.ErrMerchantLocationNotReportable) || errors.Is(err, sql.ErrNoRows) {
			return CreateMerchantLocationReportResp{}, errx.New(errx.CodeResourceNotFound, "该商家位置待完善")
		}
		// 报告写入与自动隐藏共用事务，记录根错误及安全业务标识，外部只返回稳定中文提示。
		logx.WithContext(ctx).Errorw("提交商家位置反馈失败",
			logx.Field("merchantId", merchantID),
			logx.Field("reportId", result.ID),
			logx.Field("reporterUserId", reporterUserID),
			logx.Field("action", "create_location_report"),
			logx.Field("reasonCode", reasonCode),
			logx.Field("err", err),
		)
		return CreateMerchantLocationReportResp{}, errx.New(errx.CodeInternalError, "位置反馈提交失败，请稍后重试")
	}
	if result.AutoHidden {
		logx.WithContext(ctx).Infow("商家位置因多人反馈已自动隐藏",
			logx.Field("merchantId", merchantID),
			logx.Field("activeReporterCount", result.ActiveReporterCount),
			logx.Field("oldStatus", model.MerchantLocationStatusVisible),
			logx.Field("newStatus", model.MerchantLocationStatusRiskHidden),
		)
	}
	return CreateMerchantLocationReportResp{
		ReportID: result.ID, Status: result.Status, ActiveReporterCount: result.ActiveReporterCount,
		AutoHidden: result.AutoHidden, Idempotent: result.Idempotent,
	}, nil
}
