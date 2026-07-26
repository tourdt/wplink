package metrics

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
)

type ContactStore interface {
	GetResourceContactUnlockInfo(ctx context.Context, resourceID string) (model.ResourceContactUnlockInfo, error)
	UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error)
	HasActiveContactUnlock(ctx context.Context, resourceID string, userID string) (model.ContactUnlockState, error)
	FindActiveVIPManagedMerchant(ctx context.Context, userID string) (model.VIPManagedMerchant, error)
	UpsertContactUnlock(ctx context.Context, input model.ContactUnlockInput) (model.ContactUnlockResult, error)
	RecordResourceContactEvent(ctx context.Context, input model.ResourceContactEventInput) (model.ResourceContactEventResult, error)
	UpsertResourceMetric(ctx context.Context, delta model.ResourceMetricDelta) error
}

type ContactGrowthStore interface {
	TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error)
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
	contactResp, skipRecord, err := l.validateContactUnlock(ctx, input)
	if err != nil {
		return RecordContactResp{}, err
	}
	if skipRecord {
		return contactResp, nil
	}
	eventResult, err := l.store.RecordResourceContactEvent(ctx, input)
	if err != nil {
		return RecordContactResp{}, contactWriteError(err, "记录联系事件", input.ResourceID, input.UserID, input.Action)
	}
	if err := l.store.UpsertResourceMetric(ctx, contactMetricDelta(input.ResourceID, input.Action)); err != nil {
		return RecordContactResp{}, contactWriteError(err, "更新联系指标", input.ResourceID, input.UserID, input.Action)
	}
	l.triggerContactGrowthReward(ctx, input, eventResult)
	if contactResp.Message != "" {
		return contactResp, nil
	}
	return RecordContactResp{Message: "联系行为已记录", Action: input.Action}, nil
}

func (l *RecordContactLogic) triggerContactGrowthReward(ctx context.Context, input model.ResourceContactEventInput, eventResult model.ResourceContactEventResult) {
	growthStore, ok := l.store.(ContactGrowthStore)
	if !ok {
		return
	}
	eventType := ""
	switch input.Action {
	case "phone", "wechat":
		eventType = model.GrowthEventResourceShareEffectiveContact
	case "share_view":
		eventType = model.GrowthEventResourceShareEffectiveView
	default:
		return
	}
	// 联系/分享归因奖励失败不能影响用户查看联系方式或分享主流程，记录日志后由后台补偿。
	if _, err := growthStore.TriggerGrowthEvent(ctx, model.GrowthEventInput{
		EventType:  eventType,
		MerchantID: eventResult.MerchantID,
		UserID:     input.UserID,
		ResourceID: input.ResourceID,
		EventID:    eventResult.ID,
	}); err != nil {
		logx.Errorf("联系事件触发成长权益失败: resourceId=%s userId=%s action=%s eventId=%s err=%+v", input.ResourceID, input.UserID, input.Action, eventResult.ID, err)
	}
}

func (l *RecordContactLogic) validateContactUnlock(ctx context.Context, input model.ResourceContactEventInput) (RecordContactResp, bool, error) {
	if !isContactUnlockAction(input.Action) {
		return RecordContactResp{}, false, nil
	}
	if input.UserID == "" {
		return RecordContactResp{}, false, errx.New(errx.CodeUnauthorized, "请先登录后联系商家")
	}
	info, err := l.store.GetResourceContactUnlockInfo(ctx, input.ResourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RecordContactResp{}, false, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
		}
		logx.Errorf("加载联系方式解锁资源失败: resourceId=%s userId=%s action=%s err=%+v", input.ResourceID, input.UserID, input.Action, err)
		return RecordContactResp{}, false, errx.New(errx.CodeInternalError, "联系方式加载失败，请稍后重试")
	}
	if info.Status != model.ResourceStatusPublished || isExpired(info.ExpiresAt) {
		return RecordContactResp{}, false, errx.New(errx.CodeResourceNotFound, "资源不存在或已下架")
	}
	if info.DealtAt.Valid {
		return RecordContactResp{}, false, errx.New(errx.CodeStateConflict, "该供需已完成，暂不支持继续联系")
	}
	canManage, err := l.store.UserCanManageMerchant(ctx, input.UserID, info.MerchantID)
	if err != nil {
		logx.Errorf("校验联系方式查看者商家权限失败: resourceId=%s userId=%s merchantId=%s action=%s err=%+v", input.ResourceID, input.UserID, info.MerchantID, input.Action, err)
		return RecordContactResp{}, false, errx.New(errx.CodeInternalError, "联系方式加载失败，请稍后重试")
	}
	contactResp, err := unlockedContactResp(input.Action, info)
	if err != nil {
		return RecordContactResp{}, false, err
	}
	if contactResp.Message == "" {
		return RecordContactResp{}, false, nil
	}
	// 商家查看自己资源时允许复制联系方式，但不写联系事件和效果指标。
	if canManage {
		return contactResp, true, nil
	}
	unlockState, err := l.store.HasActiveContactUnlock(ctx, input.ResourceID, input.UserID)
	if err != nil {
		logx.Errorf("查询联系方式解锁状态失败: resourceId=%s userId=%s err=%+v", input.ResourceID, input.UserID, err)
		return RecordContactResp{}, false, errx.New(errx.CodeInternalError, "联系方式加载失败，请稍后重试")
	}
	if unlockState.Unlocked {
		return contactResp, false, nil
	}
	rules := model.ContactUnlockRulesFromCommercialRules(info.CommercialRules)
	switch rules.Mode {
	case model.ContactUnlockModeDisabled:
		return RecordContactResp{}, false, errx.New(errx.CodeValidationFailed, "该分类暂不开放查看联系方式")
	case model.ContactUnlockModeLoginFree, "":
		if err := l.persistContactUnlock(ctx, input, "", model.ContactUnlockSourceLoginFree, rules); err != nil {
			return RecordContactResp{}, false, err
		}
		return contactResp, false, nil
	case model.ContactUnlockModePaid, model.ContactUnlockModePaidOrVIP, model.ContactUnlockModeVIPOnly:
		vipMerchant, err := l.store.FindActiveVIPManagedMerchant(ctx, input.UserID)
		if err != nil {
			logx.Errorf("查询用户可用 VIP 商家失败: resourceId=%s userId=%s err=%+v", input.ResourceID, input.UserID, err)
			return RecordContactResp{}, false, errx.New(errx.CodeInternalError, "联系方式加载失败，请稍后重试")
		}
		if vipMerchant.MerchantID != "" && (rules.VIPFree || rules.Mode == model.ContactUnlockModePaidOrVIP || rules.Mode == model.ContactUnlockModeVIPOnly) {
			if err := l.persistContactUnlock(ctx, input, vipMerchant.MerchantID, model.ContactUnlockSourceVIP, rules); err != nil {
				return RecordContactResp{}, false, err
			}
			return contactResp, false, nil
		}
		if rules.Mode == model.ContactUnlockModeVIPOnly {
			return RecordContactResp{}, false, errx.New(errx.CodeForbidden, "该分类仅支持 VIP 查看联系方式")
		}
		return RecordContactResp{}, false, errx.New(errx.CodePaymentRequired, "该分类需付费后查看联系方式")
	default:
		return RecordContactResp{}, false, errx.New(errx.CodeValidationFailed, "该分类联系方式查看规则不正确")
	}
}

func (l *RecordContactLogic) persistContactUnlock(ctx context.Context, input model.ResourceContactEventInput, viewerMerchantID string, sourceType string, rules model.ContactUnlockRules) error {
	days := rules.RepeatUnlockDays
	if days <= 0 {
		days = model.DefaultContactRepeatUnlockDays
	}
	now := time.Now().UTC()
	if _, err := l.store.UpsertContactUnlock(ctx, model.ContactUnlockInput{
		ResourceID:       input.ResourceID,
		UserID:           input.UserID,
		ViewerMerchantID: viewerMerchantID,
		SourceType:       sourceType,
		StartsAt:         now,
		ExpiresAt:        now.AddDate(0, 0, int(days)),
	}); err != nil {
		return contactWriteError(err, "写入联系方式解锁记录", input.ResourceID, input.UserID, input.Action)
	}
	return nil
}

func unlockedContactResp(action string, info model.ResourceContactUnlockInfo) (RecordContactResp, error) {
	switch action {
	case "phone":
		if strings.TrimSpace(info.Phone) == "" {
			return RecordContactResp{}, errx.New(errx.CodeValidationFailed, "商家暂未填写电话")
		}
		return RecordContactResp{Message: "电话已解锁", Action: action, Phone: strings.TrimSpace(info.Phone)}, nil
	case "wechat":
		if strings.TrimSpace(info.Wechat) == "" {
			return RecordContactResp{}, errx.New(errx.CodeValidationFailed, "商家暂未填写微信，可电话联系")
		}
		return RecordContactResp{Message: "微信号已解锁", Action: action, Wechat: strings.TrimSpace(info.Wechat)}, nil
	default:
		return RecordContactResp{}, nil
	}
}

func isExpired(expiresAt sql.NullTime) bool {
	return expiresAt.Valid && !expiresAt.Time.After(time.Now().UTC())
}

func isSupportedContactAction(action string) bool {
	switch action {
	case "phone", "wechat", "merchant_home", "merchant_profile", "share", "share_view":
		return true
	default:
		return false
	}
}

func isContactUnlockAction(action string) bool {
	return action == "phone" || action == "wechat"
}

func contactWriteError(err error, operation string, resourceID string, userID string, action string) error {
	if isUserForeignKeyViolation(err) {
		logx.Errorf("%s失败，用户登录态已失效: resourceId=%s userId=%s action=%s err=%+v", operation, resourceID, userID, action, err)
		return errx.New(errx.CodeUnauthorized, "登录状态无效，请重新登录")
	}
	logx.Errorf("%s失败: resourceId=%s userId=%s action=%s err=%+v", operation, resourceID, userID, action, err)
	return errx.New(errx.CodeInternalError, "联系行为记录失败，请稍后重试")
}

func isUserForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23503" && strings.Contains(pqErr.Constraint, "user_id")
	}
	message := err.Error()
	return strings.Contains(message, "violates foreign key constraint") && strings.Contains(message, "user_id")
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
