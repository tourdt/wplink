package task

import (
	"context"
	"time"

	resourcelogic "wplink/backend/app/internal/logic/resource"
	"wplink/backend/app/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContentAuditRetryStore interface {
	resourcelogic.ResourceAuditRetryStore
	MarkStaleContentAuditTasksForRetry(ctx context.Context, staleBefore time.Time) (int64, error)
	ClaimDueResourceAuditRetries(ctx context.Context, batchSize int64, maxRetries int64) ([]model.ResourceAuditRetryClaim, error)
}

type ContentAuditRetryResult struct {
	StaleCount        int64
	RetriedCount      int64
	ManualReviewCount int64
}

type ContentAuditRetryTask struct {
	store        ContentAuditRetryStore
	auditor      resourcelogic.ContentAuditor
	batchSize    int64
	maxRetries   int64
	mediaTimeout time.Duration
}

func NewContentAuditRetryTask(store ContentAuditRetryStore, auditor resourcelogic.ContentAuditor, batchSize int64) *ContentAuditRetryTask {
	if batchSize <= 0 {
		batchSize = 20
	}
	return &ContentAuditRetryTask{
		store:        store,
		auditor:      auditor,
		batchSize:    batchSize,
		maxRetries:   5,
		mediaTimeout: 15 * time.Minute,
	}
}

func (t *ContentAuditRetryTask) Run(ctx context.Context) (ContentAuditRetryResult, error) {
	if t == nil || t.store == nil || t.auditor == nil {
		return ContentAuditRetryResult{}, nil
	}
	staleCount, err := t.store.MarkStaleContentAuditTasksForRetry(ctx, time.Now().UTC().Add(-t.mediaTimeout))
	if err != nil {
		return ContentAuditRetryResult{}, err
	}
	claims, err := t.store.ClaimDueResourceAuditRetries(ctx, t.batchSize, t.maxRetries)
	if err != nil {
		return ContentAuditRetryResult{}, err
	}
	result := ContentAuditRetryResult{StaleCount: staleCount}
	for _, claim := range claims {
		if claim.ManualReview {
			result.ManualReviewCount++
			logx.Errorf("资源内容审核连续重试失败，已转人工复核告警: resourceId=%s retryCount=%d", claim.ResourceID, claim.RetryCount)
			continue
		}
		if _, err := resourcelogic.RetryResourceContentAudit(ctx, t.store, t.auditor, claim.ResourceID); err != nil {
			logx.Errorf("自动重试资源内容审核失败: resourceId=%s retryCount=%d err=%+v", claim.ResourceID, claim.RetryCount, err)
			return result, err
		}
		result.RetriedCount++
	}
	return result, nil
}
