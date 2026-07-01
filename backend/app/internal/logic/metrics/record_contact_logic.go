package metrics

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ContactStore interface {
	GetResourceContactUnlockInfo(ctx context.Context, resourceID string) (model.ResourceContactUnlockInfo, error)
	UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error)
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
	Action  string `json:"action,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Wechat  string `json:"wechat,omitempty"`
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
	contactResp, err := l.validateContactUnlock(ctx, input)
	if err != nil {
		return RecordContactResp{}, err
	}
	if _, err := l.store.RecordResourceContactEvent(ctx, input); err != nil {
		return RecordContactResp{}, err
	}
	if err := l.store.UpsertResourceMetric(ctx, contactMetricDelta(input.ResourceID, input.Action)); err != nil {
		return RecordContactResp{}, err
	}
	if contactResp.Message != "" {
		return contactResp, nil
	}
	return RecordContactResp{Message: "联系行为已记录", Action: input.Action}, nil
}

func (l *RecordContactLogic) validateContactUnlock(ctx context.Context, input model.ResourceContactEventInput) (RecordContactResp, error) {
	if !isContactUnlockAction(input.Action) {
		return RecordContactResp{}, nil
	}
	if input.UserID == "" {
		return RecordContactResp{}, errx.New(errx.CodeUnauthorized, "请先登录后联系商家")
	}
	info, err := l.store.GetResourceContactUnlockInfo(ctx, input.ResourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RecordContactResp{}, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
		}
		return RecordContactResp{}, err
	}
	if info.Status != model.ResourceStatusPublished || isExpired(info.ExpiresAt) {
		return RecordContactResp{}, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
	}
	canManage, err := l.store.UserCanManageMerchant(ctx, input.UserID, info.MerchantID)
	if err != nil {
		return RecordContactResp{}, err
	}
	if canManage {
		// 商家查看自己资源时不能制造联系指标，避免发布效果数据被误刷高。
		return RecordContactResp{}, errx.New(errx.CodeValidationFailed, "不能联系自己发布的资源")
	}
	switch input.Action {
	case "phone":
		if strings.TrimSpace(info.Phone) == "" {
			return RecordContactResp{}, errx.New(errx.CodeValidationFailed, "商家暂未填写电话")
		}
		return RecordContactResp{Message: "电话已解锁", Action: input.Action, Phone: strings.TrimSpace(info.Phone)}, nil
	case "wechat":
		if strings.TrimSpace(info.Wechat) == "" {
			return RecordContactResp{}, errx.New(errx.CodeValidationFailed, "商家暂未填写微信，可电话联系")
		}
		return RecordContactResp{Message: "微信号已解锁", Action: input.Action, Wechat: strings.TrimSpace(info.Wechat)}, nil
	default:
		return RecordContactResp{}, nil
	}
}

func isExpired(expiresAt sql.NullTime) bool {
	return expiresAt.Valid && !expiresAt.Time.After(time.Now().UTC())
}

func isSupportedContactAction(action string) bool {
	switch action {
	case "phone", "wechat", "merchant_home", "merchant_profile", "share":
		return true
	default:
		return false
	}
}

func isContactUnlockAction(action string) bool {
	return action == "phone" || action == "wechat"
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
