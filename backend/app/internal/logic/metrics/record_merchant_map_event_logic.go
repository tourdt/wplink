package metrics

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

var allowedMerchantMapEventTypes = map[string]struct{}{
	"location_entry_click":  {},
	"location_view":         {},
	"navigation_click":      {},
	"nearby_drawer_open":    {},
	"nearby_marker_click":   {},
	"nearby_merchant_click": {},
}

var allowedMerchantMapEventSources = map[string]struct{}{
	"directory":         {},
	"merchant_detail":   {},
	"merchant_location": {},
}

var merchantMapEventsRequiringTarget = map[string]struct{}{
	"nearby_marker_click":   {},
	"nearby_merchant_click": {},
}

type MerchantMapEventStore interface {
	RecordMerchantMapEvent(ctx context.Context, input model.MerchantMapEventInput) error
}

type RecordMerchantMapEventReq struct {
	UserID           string `json:"-"`
	MerchantID       string `json:"merchantId"`
	TargetMerchantID string `json:"targetMerchantId,optional"`
	VisitorKey       string `json:"visitorKey"`
	SessionID        string `json:"sessionId"`
	EventType        string `json:"eventType"`
	Source           string `json:"source"`
}

type RecordMerchantMapEventResp struct {
	Recorded bool `json:"recorded"`
}

type RecordMerchantMapEventLogic struct {
	store MerchantMapEventStore
}

func NewRecordMerchantMapEventLogic(store MerchantMapEventStore) *RecordMerchantMapEventLogic {
	return &RecordMerchantMapEventLogic{store: store}
}

func (l *RecordMerchantMapEventLogic) RecordMerchantMapEvent(ctx context.Context, req RecordMerchantMapEventReq) (RecordMerchantMapEventResp, error) {
	input := model.MerchantMapEventInput{
		UserID:           strings.TrimSpace(req.UserID),
		MerchantID:       strings.TrimSpace(req.MerchantID),
		TargetMerchantID: strings.TrimSpace(req.TargetMerchantID),
		VisitorKey:       strings.TrimSpace(req.VisitorKey),
		SessionID:        strings.TrimSpace(req.SessionID),
		EventType:        strings.TrimSpace(req.EventType),
		Source:           strings.TrimSpace(req.Source),
	}
	if _, ok := allowedMerchantMapEventTypes[input.EventType]; !ok {
		return RecordMerchantMapEventResp{}, errx.New(errx.CodeValidationFailed, "地图行为类型无效")
	}
	if _, ok := allowedMerchantMapEventSources[input.Source]; !ok {
		return RecordMerchantMapEventResp{}, errx.New(errx.CodeValidationFailed, "地图行为来源无效")
	}
	if input.VisitorKey == "" || len(input.VisitorKey) > 96 || input.SessionID == "" || len(input.SessionID) > 96 {
		return RecordMerchantMapEventResp{}, errx.New(errx.CodeValidationFailed, "地图行为会话参数无效")
	}
	// 只有点击具体周边点位或商家时才要求目标商家，抽屉打开等事件不应伪造目标归因。
	if _, required := merchantMapEventsRequiringTarget[input.EventType]; required && input.TargetMerchantID == "" {
		return RecordMerchantMapEventResp{}, errx.New(errx.CodeValidationFailed, "地图行为会话参数无效")
	}

	if err := l.store.RecordMerchantMapEvent(ctx, input); err != nil {
		// 根错误只进入服务端日志，避免把 SQL、表名或数据库状态暴露给小程序。
		logx.Errorf("记录商家地图行为失败: userId=%s merchantId=%s targetMerchantId=%s eventType=%s source=%s err=%+v", input.UserID, input.MerchantID, input.TargetMerchantID, input.EventType, input.Source, err)
		return RecordMerchantMapEventResp{}, errx.New(errx.CodeInternalError, "地图行为记录失败，请稍后重试")
	}

	return RecordMerchantMapEventResp{Recorded: true}, nil
}
