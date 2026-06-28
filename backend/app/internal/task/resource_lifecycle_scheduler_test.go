package task

import (
	"context"
	"testing"
	"time"
)

func TestResourceLifecycleSchedulerRunOnceExecutesRunner(t *testing.T) {
	runner := &fakeLifecycleRunner{result: ResourceLifecycleResult{ExpiredCount: 1, ExpiringReminderCount: 2}}
	scheduler := NewResourceLifecycleScheduler(runner, time.Hour, nil)

	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestResourceLifecycleSchedulerDisabledWhenIntervalMissing(t *testing.T) {
	runner := &fakeLifecycleRunner{}
	scheduler := NewResourceLifecycleScheduler(runner, 0, nil)

	if scheduler.Enabled() {
		t.Fatal("Enabled() = true, want disabled")
	}
	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("runner calls = %d, want 0", runner.calls)
	}
}

type fakeLifecycleRunner struct {
	calls  int
	result ResourceLifecycleResult
	err    error
}

func (r *fakeLifecycleRunner) Run(ctx context.Context) (ResourceLifecycleResult, error) {
	r.calls++
	return r.result, r.err
}
