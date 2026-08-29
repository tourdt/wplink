package metrics

import (
	"context"
	"strconv"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

var allowedMerchantDirectoryEventTypes = map[string]struct{}{
	"directory_exposure":       {},
	"directory_merchant_click": {},
	"location_view":            {},
	"nearby_marker_click":      {},
	"nearby_list_item_click":   {},
	"nearby_merchant_click":    {},
	"nearby_drawer_open":       {},
	"navigation_click":         {},
	"location_report_submit":   {},
}

var allowedMerchantDirectoryEventSources = map[string]struct{}{
	"directory":         {},
	"merchant_detail":   {},
	"merchant_location": {},
}

var merchantDirectoryEventsRequiringTarget = map[string]struct{}{
	"nearby_marker_click":    {},
	"nearby_list_item_click": {},
	"nearby_merchant_click":  {},
}

type MerchantDirectoryEventStore interface {
	RecordMerchantDirectoryEvent(ctx context.Context, input model.MerchantDirectoryEventInput) error
}

type RecordMerchantDirectoryEventReq struct {
	UserID           string `json:"-"`
	MerchantID       string `json:"merchantId"`
	TargetMerchantID string `json:"targetMerchantId,optional"`
	VisitorKey       string `json:"visitorKey"`
	SessionID        string `json:"sessionId"`
	EventType        string `json:"eventType"`
	Source           string `json:"source"`
}

type RecordMerchantDirectoryEventResp struct {
	Recorded bool `json:"recorded"`
}

type RecordMerchantDirectoryEventLogic struct {
	store MerchantDirectoryEventStore
}

func NewRecordMerchantDirectoryEventLogic(store MerchantDirectoryEventStore) *RecordMerchantDirectoryEventLogic {
	return &RecordMerchantDirectoryEventLogic{store: store}
}

func (l *RecordMerchantDirectoryEventLogic) RecordMerchantDirectoryEvent(ctx context.Context, req RecordMerchantDirectoryEventReq) (RecordMerchantDirectoryEventResp, error) {
	input := model.MerchantDirectoryEventInput{
		UserID:           strings.TrimSpace(req.UserID),
		MerchantID:       strings.TrimSpace(req.MerchantID),
		TargetMerchantID: strings.TrimSpace(req.TargetMerchantID),
		VisitorKey:       strings.TrimSpace(req.VisitorKey),
		SessionID:        strings.TrimSpace(req.SessionID),
		EventType:        strings.TrimSpace(req.EventType),
		Source:           strings.TrimSpace(req.Source),
	}
	if !isPositiveBigintID(input.MerchantID) ||
		(input.TargetMerchantID != "" && !isPositiveBigintID(input.TargetMerchantID)) {
		// 商家 ID 在进入带 ::bigint 转换的 SQL 前完成校验，避免匿名接口被非法值制造数据库错误日志。
		return RecordMerchantDirectoryEventResp{}, errx.New(errx.CodeValidationFailed, "目录行为商家参数无效")
	}
	if _, ok := allowedMerchantDirectoryEventTypes[input.EventType]; !ok {
		return RecordMerchantDirectoryEventResp{}, errx.New(errx.CodeValidationFailed, "目录行为类型无效")
	}
	if _, ok := allowedMerchantDirectoryEventSources[input.Source]; !ok {
		return RecordMerchantDirectoryEventResp{}, errx.New(errx.CodeValidationFailed, "目录行为来源无效")
	}
	if input.VisitorKey == "" || len(input.VisitorKey) > 96 || input.SessionID == "" || len(input.SessionID) > 96 {
		return RecordMerchantDirectoryEventResp{}, errx.New(errx.CodeValidationFailed, "目录行为会话参数无效")
	}
	// 只有点击具体周边点位或商家时才要求目标商家，抽屉打开等事件不应伪造目标归因。
	if _, required := merchantDirectoryEventsRequiringTarget[input.EventType]; required && input.TargetMerchantID == "" {
		return RecordMerchantDirectoryEventResp{}, errx.New(errx.CodeValidationFailed, "目录行为会话参数无效")
	}

	if err := l.store.RecordMerchantDirectoryEvent(ctx, input); err != nil {
		// 根错误只进入服务端日志，避免把 SQL、表名或数据库状态暴露给小程序。
		logx.WithContext(ctx).Errorw("记录商家目录行为失败",
			logx.Field("userId", input.UserID), logx.Field("merchantId", input.MerchantID),
			logx.Field("targetMerchantId", input.TargetMerchantID), logx.Field("eventType", input.EventType),
			logx.Field("source", input.Source), logx.Field("err", err))
		return RecordMerchantDirectoryEventResp{}, errx.New(errx.CodeInternalError, "目录行为记录失败，请稍后重试")
	}

	return RecordMerchantDirectoryEventResp{Recorded: true}, nil
}

func isPositiveBigintID(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return err == nil && parsed > 0
}
