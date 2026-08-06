package task

import (
	"context"
	"errors"

	"wplink/backend/app/internal/model"
)

type ResourceLifecycleStore interface {
	MarkExpiredResources(ctx context.Context) ([]model.LifecycleResource, error)
	ListResourcesExpiringSoon(ctx context.Context) ([]model.LifecycleResource, error)
	CreateMessage(ctx context.Context, input model.CreateMessageInput) (model.CreateMessageResult, error)
}

type ResourceLifecycleResult struct {
	ExpiredCount          int64
	ExpiringReminderCount int64
}

type ResourceLifecycleTask struct {
	store ResourceLifecycleStore
}

type resourceLifecycleStageError struct {
	stage string
	cause error
}

// Error 只返回稳定阶段，不拼接底层错误文本，避免上层误记录数据库或消息依赖中的敏感信息。
func (e *resourceLifecycleStageError) Error() string {
	return "resource lifecycle failed at " + e.stage
}

func (e *resourceLifecycleStageError) Unwrap() error {
	return e.cause
}

// ResourceLifecycleErrorStage 提供稳定、低基数的失败阶段供后台入口记录诊断日志。
func ResourceLifecycleErrorStage(err error) string {
	var stageErr *resourceLifecycleStageError
	if errors.As(err, &stageErr) {
		return stageErr.stage
	}
	return "unknown"
}

func resourceLifecycleFailure(stage string, err error) error {
	return &resourceLifecycleStageError{stage: stage, cause: err}
}

func NewResourceLifecycleTask(store ResourceLifecycleStore) *ResourceLifecycleTask {
	return &ResourceLifecycleTask{store: store}
}

func (t *ResourceLifecycleTask) Run(ctx context.Context) (ResourceLifecycleResult, error) {
	var result ResourceLifecycleResult
	expired, err := t.store.MarkExpiredResources(ctx)
	if err != nil {
		return ResourceLifecycleResult{}, resourceLifecycleFailure("mark_expired_resources", err)
	}
	for _, item := range expired {
		message, err := t.store.CreateMessage(ctx, model.CreateMessageInput{
			RecipientRoleCode: "merchant:" + item.MerchantID,
			MessageType:       "resource_expired",
			TriggerType:       "resource_expired",
			TriggerID:         item.ID,
			Title:             "资源已过期",
			Content:           item.Title + " 已过期，可再发类似资源继续获得曝光",
			TargetURL:         model.MerchantMyResourcesTargetURL(item.MerchantID),
		})
		if err != nil {
			return ResourceLifecycleResult{}, resourceLifecycleFailure("create_expired_message", err)
		}
		if message.Created {
			result.ExpiredCount++
		}
	}

	expiring, err := t.store.ListResourcesExpiringSoon(ctx)
	if err != nil {
		return ResourceLifecycleResult{}, resourceLifecycleFailure("list_expiring_resources", err)
	}
	for _, item := range expiring {
		message, err := t.store.CreateMessage(ctx, model.CreateMessageInput{
			RecipientRoleCode: "merchant:" + item.MerchantID,
			MessageType:       "resource_expiring",
			TriggerType:       "resource_expiring",
			TriggerID:         item.ID,
			Title:             "资源即将过期",
			Content:           item.Title + " 即将过期，请及时刷新或再发类似",
			TargetURL:         model.MerchantMyResourcesTargetURL(item.MerchantID),
		})
		if err != nil {
			return ResourceLifecycleResult{}, resourceLifecycleFailure("create_expiring_message", err)
		}
		if message.Created {
			result.ExpiringReminderCount++
		}
	}
	return result, nil
}
