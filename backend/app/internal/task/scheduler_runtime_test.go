package task

import (
	"bytes"
	"context"
	"errors"
	"log"
	"sync"
	"testing"
	"time"
)

type fakeCoordinator struct {
	mu                   sync.Mutex
	calls                int
	acquired             bool
	called               chan struct{}
	received             chan coordinatorCall
	coordinationErr      error
	coordinationFailures int
}

type coordinatorCall struct {
	taskName TaskName
	timeout  time.Duration
}

func (f *fakeCoordinator) RunExclusive(
	ctx context.Context,
	taskName TaskName,
	timeout time.Duration,
	run func(context.Context) error,
) (CoordinationResult, error) {
	f.mu.Lock()
	f.calls++
	acquired := f.acquired
	called := f.called
	received := f.received
	coordinationErr := error(nil)
	if f.coordinationFailures > 0 {
		f.coordinationFailures--
		coordinationErr = f.coordinationErr
	}
	f.mu.Unlock()
	if called != nil {
		select {
		case called <- struct{}{}:
		default:
		}
	}
	if received != nil {
		received <- coordinatorCall{taskName: taskName, timeout: timeout}
	}
	if coordinationErr != nil {
		return CoordinationResult{}, coordinationErr
	}
	if !acquired {
		return CoordinationResult{Acquired: false}, nil
	}
	return CoordinationResult{Acquired: true}, run(ctx)
}

func TestSchedulersUseConfiguredCoordinatorTaskAndTimeout(t *testing.T) {
	const timeout = 37 * time.Second
	tests := []struct {
		name     string
		wantTask TaskName
		new      func(Coordinator) scheduler
	}{
		{
			name:     "resource lifecycle",
			wantTask: TaskResourceLifecycle,
			new: func(coordinator Coordinator) scheduler {
				return NewResourceLifecycleScheduler(&fakeLifecycleRunner{}, time.Hour, nil, coordinator, timeout)
			},
		},
		{
			name:     "content audit retry",
			wantTask: TaskContentAuditRetry,
			new: func(coordinator Coordinator) scheduler {
				return NewContentAuditRetryScheduler(fakeContentAuditRetryRunner{}, time.Hour, nil, coordinator, timeout)
			},
		},
		{
			name:     "payment reconciliation",
			wantTask: TaskPaymentReconciliation,
			new: func(coordinator Coordinator) scheduler {
				return NewPaymentReconciliationScheduler(fakePaymentReconciliationRunner{}, time.Hour, nil, coordinator, timeout)
			},
		},
		{
			name:     "merchant map event cleanup",
			wantTask: TaskMerchantMapEventCleanup,
			new: func(coordinator Coordinator) scheduler {
				return NewMerchantMapEventCleanupScheduler(&fakeMerchantMapEventCleanupRunner{}, time.Hour, nil, coordinator, timeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			received := make(chan coordinatorCall, 1)
			coordinator := &fakeCoordinator{acquired: false, received: received}
			scheduler := tt.new(coordinator)
			ctx, cancel := context.WithCancel(context.Background())
			scheduler.Start(ctx)
			select {
			case call := <-received:
				if call.taskName != tt.wantTask {
					t.Fatalf("Coordinator taskName = %q, want %q", call.taskName, tt.wantTask)
				}
				if call.timeout != timeout {
					t.Fatalf("Coordinator timeout = %s, want %s", call.timeout, timeout)
				}
			case <-time.After(time.Second):
				t.Fatal("Scheduler 启动后未调用 Coordinator")
			}
			cancel()
			scheduler.Wait()
		})
	}
}

type scheduler interface {
	Start(context.Context)
	Wait()
}

type fakeContentAuditRetryRunner struct {
	err error
}

func (r fakeContentAuditRetryRunner) Run(context.Context) (ContentAuditRetryResult, error) {
	return ContentAuditRetryResult{}, r.err
}

type fakePaymentReconciliationRunner struct {
	err error
}

func (r fakePaymentReconciliationRunner) Run(context.Context) (PaymentReconciliationResult, error) {
	return PaymentReconciliationResult{}, r.err
}

func TestSchedulersLeaveRunnerErrorLoggingToCoordinator(t *testing.T) {
	runnerErr := errors.New("runner failed")
	tests := []struct {
		name string
		new  func(*log.Logger) runOnceScheduler
	}{
		{
			name: "resource lifecycle",
			new: func(logger *log.Logger) runOnceScheduler {
				return NewResourceLifecycleScheduler(
					&fakeLifecycleRunner{err: runnerErr}, time.Hour, logger, nil, 0,
				)
			},
		},
		{
			name: "content audit retry",
			new: func(logger *log.Logger) runOnceScheduler {
				return NewContentAuditRetryScheduler(
					fakeContentAuditRetryRunner{err: runnerErr}, time.Hour, logger, nil, 0,
				)
			},
		},
		{
			name: "payment reconciliation",
			new: func(logger *log.Logger) runOnceScheduler {
				return NewPaymentReconciliationScheduler(
					fakePaymentReconciliationRunner{err: runnerErr}, time.Hour, logger, nil, 0,
				)
			},
		},
		{
			name: "merchant map event cleanup",
			new: func(logger *log.Logger) runOnceScheduler {
				return NewMerchantMapEventCleanupScheduler(
					&fakeMerchantMapEventCleanupRunner{err: runnerErr}, time.Hour, logger, nil, 0,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			scheduler := tt.new(log.New(&output, "", 0))

			err := scheduler.RunOnce(context.Background())
			if !errors.Is(err, runnerErr) {
				t.Fatalf("RunOnce() error = %v, want %v", err, runnerErr)
			}
			if got := output.String(); got != "" {
				t.Fatalf("Scheduler 记录了应由 Coordinator 统一处理的 runner 错误日志: %q", got)
			}
		})
	}
}

type runOnceScheduler interface {
	RunOnce(context.Context) error
}

type notifyingResourceLifecycleRunner struct {
	called chan struct{}
}

func (r notifyingResourceLifecycleRunner) Run(context.Context) (ResourceLifecycleResult, error) {
	r.called <- struct{}{}
	return ResourceLifecycleResult{}, nil
}

func TestResourceLifecycleSchedulerStartRunsWithoutCoordinator(t *testing.T) {
	called := make(chan struct{}, 1)
	scheduler := NewResourceLifecycleScheduler(
		notifyingResourceLifecycleRunner{called: called}, time.Hour, nil, nil, 0,
	)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		scheduler.Wait()
	})

	scheduler.Start(ctx)
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("nil Coordinator 下 Scheduler Start 未直接执行 runner")
	}
	cancel()
	scheduler.Wait()
}

func TestSchedulerRuntimeContinuesAfterCoordinatorError(t *testing.T) {
	coordinator := &fakeCoordinator{
		acquired:             true,
		coordinationErr:      errors.New("coordinator unavailable"),
		coordinationFailures: 1,
	}
	runtime := newSchedulerRuntime(TaskResourceLifecycle, coordinator, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	runnerCalled := make(chan struct{}, 1)
	t.Cleanup(func() {
		cancel()
		runtime.wait()
	})

	runtime.start(ctx, 10*time.Millisecond, func(context.Context) error {
		runnerCalled <- struct{}{}
		return nil
	})
	select {
	case <-runnerCalled:
	case <-time.After(time.Second):
		t.Fatal("Coordinator 首轮返回错误后，下一轮未继续执行 runner")
	}
	cancel()
	runtime.wait()
}

func TestSchedulerRuntimeStartsImmediatelyAndWaitsForShutdown(t *testing.T) {
	coordinator := &fakeCoordinator{acquired: true}
	runtime := newSchedulerRuntime(TaskResourceLifecycle, coordinator, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	called := make(chan struct{}, 1)

	runtime.start(ctx, time.Hour, func(context.Context) error {
		called <- struct{}{}
		return nil
	})
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("Scheduler 启动后未立即执行")
	}
	cancel()
	waitForSchedulerRuntime(t, runtime)
}

func TestSchedulerRuntimeDoesNotOverlapRuns(t *testing.T) {
	coordinator := &fakeCoordinator{acquired: true}
	runtime := newSchedulerRuntime(TaskResourceLifecycle, coordinator, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	firstRelease := make(chan struct{})
	started := make(chan int, 2)
	var releaseOnce sync.Once
	var mu sync.Mutex
	active := 0
	maxActive := 0
	calls := 0
	t.Cleanup(func() {
		cancel()
		releaseOnce.Do(func() { close(firstRelease) })
		runtime.wait()
	})

	runtime.start(ctx, 10*time.Millisecond, func(context.Context) error {
		mu.Lock()
		calls++
		call := calls
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()
		started <- call
		if call == 1 {
			<-firstRelease
		}
		mu.Lock()
		active--
		mu.Unlock()
		return nil
	})

	select {
	case call := <-started:
		if call != 1 {
			t.Fatalf("首次执行序号 = %d, want 1", call)
		}
	case <-time.After(time.Second):
		t.Fatal("Scheduler 首次执行未启动")
	}
	select {
	case call := <-started:
		t.Fatalf("首次执行未结束时发生重入: call=%d", call)
	case <-time.After(50 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(firstRelease) })
	select {
	case call := <-started:
		if call != 2 {
			t.Fatalf("下一轮执行序号 = %d, want 2", call)
		}
	case <-time.After(time.Second):
		t.Fatal("首次执行结束后下一轮未继续")
	}
	cancel()
	waitForSchedulerRuntime(t, runtime)

	mu.Lock()
	defer mu.Unlock()
	if maxActive != 1 {
		t.Fatalf("并发执行数峰值 = %d, want 1", maxActive)
	}
}

func TestSchedulerRuntimeSkipsRunnerWhenLockIsNotAcquired(t *testing.T) {
	coordinatorCalled := make(chan struct{}, 1)
	coordinator := &fakeCoordinator{acquired: false, called: coordinatorCalled}
	runtime := newSchedulerRuntime(TaskResourceLifecycle, coordinator, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	runnerCalled := make(chan struct{}, 1)

	runtime.start(ctx, time.Hour, func(context.Context) error {
		runnerCalled <- struct{}{}
		return nil
	})
	select {
	case <-coordinatorCalled:
	case <-time.After(time.Second):
		t.Fatal("Scheduler 启动后未调用 Coordinator")
	}
	cancel()
	waitForSchedulerRuntime(t, runtime)
	select {
	case <-runnerCalled:
		t.Fatal("未获得协调锁时仍调用了 runner")
	default:
	}
}

func TestSchedulerRuntimeContinuesAfterRunnerError(t *testing.T) {
	coordinator := &fakeCoordinator{acquired: true}
	runtime := newSchedulerRuntime(TaskResourceLifecycle, coordinator, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	callNumbers := make(chan int, 2)
	var mu sync.Mutex
	calls := 0

	runtime.start(ctx, 10*time.Millisecond, func(context.Context) error {
		mu.Lock()
		calls++
		call := calls
		mu.Unlock()
		callNumbers <- call
		if call == 1 {
			return errors.New("first run failed")
		}
		return nil
	})
	for want := 1; want <= 2; want++ {
		select {
		case got := <-callNumbers:
			if got != want {
				t.Fatalf("执行序号 = %d, want %d", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("runner 第 %d 轮未执行", want)
		}
	}
	cancel()
	waitForSchedulerRuntime(t, runtime)
}

func TestSchedulerRuntimeWaitsForCurrentRunAfterCancellation(t *testing.T) {
	coordinator := &fakeCoordinator{acquired: true}
	runtime := newSchedulerRuntime(TaskResourceLifecycle, coordinator, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	waited := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		cancel()
		releaseOnce.Do(func() { close(release) })
		runtime.wait()
	})

	runtime.start(ctx, 10*time.Millisecond, func(context.Context) error {
		started <- struct{}{}
		<-release
		return nil
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("Scheduler 首次执行未启动")
	}
	cancel()
	go func() {
		runtime.wait()
		close(waited)
	}()
	select {
	case <-waited:
		t.Fatal("当前执行尚未结束时 Wait 已返回")
	case <-time.After(50 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case <-waited:
	case <-time.After(time.Second):
		t.Fatal("当前执行结束后 Wait 未返回")
	}
}

func TestSchedulerRuntimeDoesNotStartBufferedTickAfterCancellation(t *testing.T) {
	// 多轮覆盖 ctx.Done 与 ticker 同时就绪的选择分支；取消后即使已有积压 tick，
	// 也只能等待当前执行完成，不能再开始新一轮。
	for iteration := 0; iteration < 20; iteration++ {
		coordinator := &fakeCoordinator{acquired: true}
		runtime := newSchedulerRuntime(TaskResourceLifecycle, coordinator, time.Second)
		ctx, cancel := context.WithCancel(context.Background())
		secondStarted := make(chan struct{}, 1)
		release := make(chan struct{})
		var mu sync.Mutex
		calls := 0

		runtime.start(ctx, time.Millisecond, func(context.Context) error {
			mu.Lock()
			calls++
			call := calls
			mu.Unlock()
			if call == 2 {
				secondStarted <- struct{}{}
				<-release
			}
			return nil
		})
		select {
		case <-secondStarted:
		case <-time.After(time.Second):
			t.Fatal("Scheduler 第二轮执行未启动")
		}
		// 第二轮保持执行超过一个周期，确保 ticker 中已有待处理信号。
		time.Sleep(5 * time.Millisecond)
		cancel()
		close(release)
		waitForSchedulerRuntime(t, runtime)

		mu.Lock()
		gotCalls := calls
		mu.Unlock()
		if gotCalls != 2 {
			t.Fatalf("第 %d 轮取消后执行次数 = %d, want 2", iteration+1, gotCalls)
		}
	}
}

func waitForSchedulerRuntime(t *testing.T, runtime *schedulerRuntime) {
	t.Helper()
	waited := make(chan struct{})
	go func() {
		runtime.wait()
		close(waited)
	}()
	select {
	case <-waited:
	case <-time.After(time.Second):
		t.Fatal("Scheduler 取消后 Wait 未返回")
	}
}
