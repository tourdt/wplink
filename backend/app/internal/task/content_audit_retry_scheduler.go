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
	runtime  *schedulerRuntime
}

func NewContentAuditRetryScheduler(
	runner ContentAuditRetryRunner,
	interval time.Duration,
	logger *log.Logger,
	coordinator Coordinator,
	timeout time.Duration,
) *ContentAuditRetryScheduler {
	return &ContentAuditRetryScheduler{
		runner:   runner,
		interval: interval,
		logger:   logger,
		runtime:  newSchedulerRuntime(TaskContentAuditRetry, coordinator, timeout),
	}
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
	s.runtime.start(ctx, s.interval, s.RunOnce)
}

func (s *ContentAuditRetryScheduler) Wait() {
	s.runtime.wait()
}
