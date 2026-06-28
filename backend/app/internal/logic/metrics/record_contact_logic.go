package metrics

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ContactStore interface {
	RecordResourceContactEvent(ctx context.Context, input model.ResourceContactEventInput) (model.ResourceContactEventResult, error)
	UpsertResourceMetric(ctx context.Context, delta model.ResourceMetricDelta) error
}

type RecordContactReq struct {
	ResourceID string
	UserID     string
	Action     string
}

type RecordContactResp struct {
	Message string `json:"message"`
}

type RecordContactLogic struct {
	store ContactStore
}

func NewRecordContactLogic(store ContactStore) *RecordContactLogic {
	return &RecordContactLogic{store: store}
}

func (l *RecordContactLogic) RecordContact(ctx context.Context, req RecordContactReq) (RecordContactResp, error) {
	input := model.ResourceContactEventInput{
		ResourceID: strings.TrimSpace(req.ResourceID),
		UserID:     strings.TrimSpace(req.UserID),
		Action:     strings.TrimSpace(req.Action),
	}
	if input.ResourceID == "" {
		return RecordContactResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	if !isSupportedContactAction(input.Action) {
		return RecordContactResp{}, errx.New(errx.CodeValidationFailed, "联系动作不正确")
	}
	if _, err := l.store.RecordResourceContactEvent(ctx, input); err != nil {
		return RecordContactResp{}, err
	}
	if err := l.store.UpsertResourceMetric(ctx, contactMetricDelta(input.ResourceID, input.Action)); err != nil {
		return RecordContactResp{}, err
	}
	return RecordContactResp{Message: "联系行为已记录"}, nil
}

func isSupportedContactAction(action string) bool {
	switch action {
	case "phone", "wechat", "merchant_home", "merchant_profile", "share":
		return true
	default:
		return false
	}
}

func contactMetricDelta(resourceID string, action string) model.ResourceMetricDelta {
	delta := model.ResourceMetricDelta{ResourceID: resourceID, ContactClickCount: 1}
	switch action {
	case "phone":
		delta.PhoneClickCount = 1
	case "wechat":
		delta.WechatCopyCount = 1
	case "share":
		delta.ShareCount = 1
	}
	return delta
}
