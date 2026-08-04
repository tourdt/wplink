package task

import (
	"context"
	"errors"
	"time"

	resourcelogic "wplink/backend/app/internal/logic/resource"
	"wplink/backend/app/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContentAuditRetryStore interface {
	resourcelogic.ResourceAuditLeaseStore
	MarkStaleContentAuditTasksForRetry(ctx context.Context, staleBefore time.Time) (int64, error)
	ClaimDueResourceAuditRetries(ctx context.Context, batchSize int64, maxRetries int64, processingBy string, leaseDuration time.Duration) ([]model.ResourceAuditRetryClaim, error)
}

const DefaultContentAuditRetryLeaseDuration = 2 * time.Minute

type ContentAuditRetryResult struct {
	StaleCount        int64
	RetriedCount      int64
	ManualReviewCount int64
}

type ContentAuditRetryTask struct {
	store         ContentAuditRetryStore
	auditor       resourcelogic.ContentAuditor
	batchSize     int64
	maxRetries    int64
	instanceID    string
	leaseDuration time.Duration
	mediaTimeout  time.Duration
}

func NewContentAuditRetryTask(
	store ContentAuditRetryStore,
	auditor resourcelogic.ContentAuditor,
	batchSize int64,
	maxRetryCount int64,
	instanceID string,
	leaseDuration time.Duration,
) *ContentAuditRetryTask {
	if batchSize <= 0 {
		batchSize = 20
	}
	return &ContentAuditRetryTask{
		store:         store,
		auditor:       auditor,
		batchSize:     batchSize,
		maxRetries:    maxRetryCount,
		instanceID:    instanceID,
		leaseDuration: leaseDuration,
		mediaTimeout:  15 * time.Minute,
	}
}

func (t *ContentAuditRetryTask) Run(ctx context.Context) (ContentAuditRetryResult, error) {
	if t == nil || t.store == nil || t.auditor == nil {
		return ContentAuditRetryResult{}, nil
	}
	if err := ctx.Err(); err != nil {
		return ContentAuditRetryResult{}, err
	}
	staleCount, err := t.store.MarkStaleContentAuditTasksForRetry(ctx, time.Now().UTC().Add(-t.mediaTimeout))
	if err != nil {
		return ContentAuditRetryResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ContentAuditRetryResult{StaleCount: staleCount}, err
	}
	claims, err := t.store.ClaimDueResourceAuditRetries(ctx, t.batchSize, t.maxRetries, t.instanceID, t.leaseDuration)
	if err != nil {
		return ContentAuditRetryResult{}, err
	}
	result := ContentAuditRetryResult{StaleCount: staleCount}
	for _, claim := range claims {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if claim.ManualReview {
			result.ManualReviewCount++
			logx.WithContext(ctx).Errorw(
				"资源内容审核连续重试失败，已转人工复核",
				logx.Field("event", "content_audit_retry_manual_review"),
				logx.Field("resource_id", claim.ResourceID),
				logx.Field("instance_id", t.instanceID),
				logx.Field("retry_count", claim.RetryCount),
				logx.Field("retry_status", "manual_review"),
				logx.Field("root_cause", "retry_limit_reached"),
			)
			continue
		}
		guard := model.ResourceAuditGuard{ResourceID: claim.ResourceID, ProcessingBy: claim.ProcessingBy}
		if _, err := resourcelogic.RetryResourceContentAudit(ctx, t.store, t.auditor, guard); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return result, ctxErr
			}
			if errors.Is(err, model.ErrResourceAuditLeaseLost) {
				// 租约丢失表示其他实例已经接管；旧实例只记录安全告警，绝不能再做无 guard 的补偿写入。
				logx.WithContext(ctx).Infow(
					"内容审核重试租约已失效，跳过旧实例结果",
					logx.Field("event", "content_audit_retry_lease_lost"),
					logx.Field("resource_id", claim.ResourceID),
					logx.Field("instance_id", t.instanceID),
					logx.Field("retry_count", claim.RetryCount),
					logx.Field("retry_status", "lease_lost"),
					logx.Field("root_cause", "lease_expired_or_reassigned"),
				)
				continue
			}
			// 审核供应商错误可能包含响应描述；批次日志仅记录安全分类，详细根因由调用边界的脱敏日志负责。
			logx.WithContext(ctx).Errorw(
				"自动重试资源内容审核失败",
				logx.Field("event", "content_audit_retry_failed"),
				logx.Field("resource_id", claim.ResourceID),
				logx.Field("instance_id", t.instanceID),
				logx.Field("retry_count", claim.RetryCount),
				logx.Field("retry_status", "processing_failed"),
				logx.Field("root_cause", contentAuditRetryRootCause(err)),
			)
			return result, err
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		result.RetriedCount++
	}
	return result, nil
}

func contentAuditRetryRootCause(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	default:
		return "audit_processing_failed"
	}
}
