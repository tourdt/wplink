package task

import (
	"context"
	"log"
	"time"
)

type MerchantMapEventCleanupRunner interface {
	Run(ctx context.Context) (MerchantMapEventCleanupResult, error)
}

type MerchantMapEventCleanupScheduler struct {
	runner   MerchantMapEventCleanupRunner
	interval time.Duration
	logger   *log.Logger
	runtime  *schedulerRuntime
}

func NewMerchantMapEventCleanupScheduler(
	runner MerchantMapEventCleanupRunner,
	interval time.Duration,
	logger *log.Logger,
	coordinator Coordinator,
	timeout time.Duration,
) *MerchantMapEventCleanupScheduler {
	return &MerchantMapEventCleanupScheduler{
		runner:   runner,
		interval: interval,
		logger:   logger,
		runtime:  newSchedulerRuntime(TaskMerchantMapEventCleanup, coordinator, timeout),
	}
}

func (s *MerchantMapEventCleanupScheduler) Enabled() bool {
	return s != nil && s.runner != nil && s.interval > 0
}

func (s *MerchantMapEventCleanupScheduler) RunOnce(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	result, err := s.runner.Run(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("商家地图行为清理任务执行失败: err=%v", err)
		}
		return err
	}
	if s.logger != nil && result.DeletedCount > 0 {
		s.logger.Printf("商家地图行为清理任务执行完成: deleted=%d", result.DeletedCount)
	}
	return nil
}

func (s *MerchantMapEventCleanupScheduler) Start(ctx context.Context) {
	if !s.Enabled() {
		return
	}
	s.runtime.start(ctx, s.interval, s.RunOnce)
}

func (s *MerchantMapEventCleanupScheduler) Wait() {
	s.runtime.wait()
}
