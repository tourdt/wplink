package task

import (
	"context"
	"log"
	"time"
)

type MerchantDirectoryEventCleanupRunner interface {
	Run(ctx context.Context) (MerchantDirectoryEventCleanupResult, error)
}

type MerchantDirectoryEventCleanupScheduler struct {
	runner   MerchantDirectoryEventCleanupRunner
	interval time.Duration
	logger   *log.Logger
	runtime  *schedulerRuntime
}

func NewMerchantDirectoryEventCleanupScheduler(
	runner MerchantDirectoryEventCleanupRunner,
	interval time.Duration,
	logger *log.Logger,
	coordinator Coordinator,
	timeout time.Duration,
) *MerchantDirectoryEventCleanupScheduler {
	return &MerchantDirectoryEventCleanupScheduler{
		runner:   runner,
		interval: interval,
		logger:   logger,
		runtime:  newSchedulerRuntime(TaskMerchantDirectoryEventCleanup, coordinator, timeout),
	}
}

func (s *MerchantDirectoryEventCleanupScheduler) Enabled() bool {
	return s != nil && s.runner != nil && s.interval > 0
}

func (s *MerchantDirectoryEventCleanupScheduler) RunOnce(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	result, err := s.runner.Run(ctx)
	if err != nil {
		return err
	}
	if s.logger != nil && result.DeletedCount > 0 {
		s.logger.Printf("商家目录行为清理任务执行完成: deleted=%d", result.DeletedCount)
	}
	return nil
}

func (s *MerchantDirectoryEventCleanupScheduler) Start(ctx context.Context) {
	if !s.Enabled() {
		return
	}
	s.runtime.start(ctx, s.interval, s.RunOnce)
}

func (s *MerchantDirectoryEventCleanupScheduler) Wait() {
	s.runtime.wait()
}
