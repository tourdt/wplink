package task

import (
	"context"
	"log"
	"time"
)

type PaymentReconciliationRunner interface {
	Run(ctx context.Context) (PaymentReconciliationResult, error)
}

type PaymentReconciliationScheduler struct {
	runner   PaymentReconciliationRunner
	interval time.Duration
	logger   *log.Logger
	runtime  *schedulerRuntime
}

func NewPaymentReconciliationScheduler(
	runner PaymentReconciliationRunner,
	interval time.Duration,
	logger *log.Logger,
	coordinator Coordinator,
	timeout time.Duration,
) *PaymentReconciliationScheduler {
	return &PaymentReconciliationScheduler{
		runner:   runner,
		interval: interval,
		logger:   logger,
		runtime:  newSchedulerRuntime(TaskPaymentReconciliation, coordinator, timeout),
	}
}

func (s *PaymentReconciliationScheduler) Enabled() bool {
	return s != nil && s.runner != nil && s.interval > 0
}

func (s *PaymentReconciliationScheduler) RunOnce(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	result, err := s.runner.Run(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("支付补偿任务执行失败: err=%v", err)
		}
		return err
	}
	if s.logger != nil {
		s.logger.Printf(
			"支付补偿任务执行完成: scanned=%d paid=%d closed=%d pending=%d failed=%d",
			result.ScannedCount,
			result.PaidCount,
			result.ClosedCount,
			result.PendingCount,
			result.FailedCount,
		)
	}
	return nil
}

func (s *PaymentReconciliationScheduler) Start(ctx context.Context) {
	if !s.Enabled() {
		return
	}
	s.runtime.start(ctx, s.interval, s.RunOnce)
}

func (s *PaymentReconciliationScheduler) Wait() {
	s.runtime.wait()
}
