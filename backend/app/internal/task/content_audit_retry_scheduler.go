package task

import (
	"context"
	"log"
	"time"
)

type ContentAuditRetryRunner interface {
	Run(ctx context.Context) (ContentAuditRetryResult, error)
}

type ContentAuditRetryScheduler struct {
	runner   ContentAuditRetryRunner
	interval time.Duration
	logger   *log.Logger
}

func NewContentAuditRetryScheduler(runner ContentAuditRetryRunner, interval time.Duration, logger *log.Logger) *ContentAuditRetryScheduler {
	return &ContentAuditRetryScheduler{runner: runner, interval: interval, logger: logger}
}

func (s *ContentAuditRetryScheduler) Enabled() bool {
	return s != nil && s.runner != nil && s.interval > 0
}

func (s *ContentAuditRetryScheduler) RunOnce(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	result, err := s.runner.Run(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("内容审核自动重试任务执行失败: err=%v", err)
		}
		return err
	}
	if s.logger != nil && (result.StaleCount > 0 || result.RetriedCount > 0 || result.ManualReviewCount > 0) {
		s.logger.Printf(
			"内容审核自动重试任务执行完成: stale=%d retried=%d manualReview=%d",
			result.StaleCount,
			result.RetriedCount,
			result.ManualReviewCount,
		)
	}
	return nil
}

func (s *ContentAuditRetryScheduler) Start(ctx context.Context) {
	if !s.Enabled() {
		return
	}
	go func() {
		_ = s.RunOnce(ctx)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.RunOnce(ctx)
			}
		}
	}()
}
