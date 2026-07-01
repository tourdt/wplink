package task

import (
	"context"

	"wplink/backend/app/internal/model"
)

type ResourceLifecycleStore interface {
	MarkExpiredResources(ctx context.Context) ([]model.LifecycleResource, error)
	ListResourcesExpiringSoon(ctx context.Context) ([]model.LifecycleResource, error)
	MarkExpiredVerifications(ctx context.Context) ([]model.LifecycleResource, error)
	ListVerificationsExpiringSoon(ctx context.Context) ([]model.LifecycleResource, error)
	CreateMessage(ctx context.Context, input model.CreateMessageInput) (model.CreateMessageResult, error)
}

type ResourceLifecycleResult struct {
	ExpiredCount                      int64
	ExpiringReminderCount             int64
	VerificationExpiredCount          int64
	VerificationExpiringReminderCount int64
}

type ResourceLifecycleTask struct {
	store ResourceLifecycleStore
}

func NewResourceLifecycleTask(store ResourceLifecycleStore) *ResourceLifecycleTask {
	return &ResourceLifecycleTask{store: store}
}

func (t *ResourceLifecycleTask) Run(ctx context.Context) (ResourceLifecycleResult, error) {
	var result ResourceLifecycleResult
	expired, err := t.store.MarkExpiredResources(ctx)
	if err != nil {
		return ResourceLifecycleResult{}, err
	}
	for _, item := range expired {
		if _, err := t.store.CreateMessage(ctx, model.CreateMessageInput{
			RecipientRoleCode: "merchant:" + item.MerchantID,
			MessageType:       "resource_expired",
			TriggerType:       "resource_expired",
			TriggerID:         item.ID,
			Title:             "资源已过期",
			Content:           item.Title + " 已过期，可再发类似资源继续获得曝光",
			TargetURL:         model.MerchantMyResourcesTargetURL(item.MerchantID),
		}); err != nil {
			return ResourceLifecycleResult{}, err
		}
		result.ExpiredCount++
	}

	expiring, err := t.store.ListResourcesExpiringSoon(ctx)
	if err != nil {
		return ResourceLifecycleResult{}, err
	}
	for _, item := range expiring {
		if _, err := t.store.CreateMessage(ctx, model.CreateMessageInput{
			RecipientRoleCode: "merchant:" + item.MerchantID,
			MessageType:       "resource_expiring",
			TriggerType:       "resource_expiring",
			TriggerID:         item.ID,
			Title:             "资源即将过期",
			Content:           item.Title + " 即将过期，请及时刷新或再发类似",
			TargetURL:         model.MerchantMyResourcesTargetURL(item.MerchantID),
		}); err != nil {
			return ResourceLifecycleResult{}, err
		}
		result.ExpiringReminderCount++
	}
	expiredVerifications, err := t.store.MarkExpiredVerifications(ctx)
	if err != nil {
		return ResourceLifecycleResult{}, err
	}
	for _, item := range expiredVerifications {
		if _, err := t.store.CreateMessage(ctx, model.CreateMessageInput{
			RecipientRoleCode: "merchant:" + item.MerchantID,
			MessageType:       "verification_expired",
			TriggerType:       "verification_expired",
			TriggerID:         item.ID,
			Title:             "认证已到期",
			Content:           item.Title + " 已到期，请重新提交认证以恢复认证标识",
			TargetURL:         model.MerchantVerificationTargetURL(item.MerchantID),
		}); err != nil {
			return ResourceLifecycleResult{}, err
		}
		result.VerificationExpiredCount++
	}

	expiringVerifications, err := t.store.ListVerificationsExpiringSoon(ctx)
	if err != nil {
		return ResourceLifecycleResult{}, err
	}
	for _, item := range expiringVerifications {
		if _, err := t.store.CreateMessage(ctx, model.CreateMessageInput{
			RecipientRoleCode: "merchant:" + item.MerchantID,
			MessageType:       "verification_expiring",
			TriggerType:       "verification_expiring",
			TriggerID:         item.ID,
			Title:             "认证即将到期",
			Content:           item.Title + " 即将到期，请提前重新提交认证",
			TargetURL:         model.MerchantVerificationTargetURL(item.MerchantID),
		}); err != nil {
			return ResourceLifecycleResult{}, err
		}
		result.VerificationExpiringReminderCount++
	}
	return result, nil
}
