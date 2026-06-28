package task

import (
	"context"
	"log"
	"time"
)

type ResourceLifecycleRunner interface {
	Run(ctx context.Context) (ResourceLifecycleResult, error)
}

type ResourceLifecycleScheduler struct {
	runner   ResourceLifecycleRunner
	interval time.Duration
	logger   *log.Logger
}

func NewResourceLifecycleScheduler(runner ResourceLifecycleRunner, interval time.Duration, logger *log.Logger) *ResourceLifecycleScheduler {
	return &ResourceLifecycleScheduler{runner: runner, interval: interval, logger: logger}
}

func (s *ResourceLifecycleScheduler) Enabled() bool {
	return s != nil && s.runner != nil && s.interval > 0
}

func (s *ResourceLifecycleScheduler) RunOnce(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	result, err := s.runner.Run(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("资源生命周期任务执行失败: err=%v", err)
		}
		return err
	}
	if s.logger != nil {
		s.logger.Printf("资源生命周期任务执行完成: expired=%d expiring=%d", result.ExpiredCount, result.ExpiringReminderCount)
	}
	return nil
}

func (s *ResourceLifecycleScheduler) Start(ctx context.Context) {
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
