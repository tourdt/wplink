package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMerchantDirectoryEventCleanupTaskDeletesEventsOlderThanNinetyDays(t *testing.T) {
	now := time.Date(2026, time.August, 2, 10, 0, 0, 0, time.UTC)
	store := &fakeMerchantDirectoryEventCleanupStore{deleted: 7}
	cleanup := NewMerchantDirectoryEventCleanupTask(store, 90, func() time.Time { return now })

	result, err := cleanup.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.DeletedCount != 7 {
		t.Fatalf("DeletedCount = %d, want 7", result.DeletedCount)
	}
	wantCutoff := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)
	if !store.cutoff.Equal(wantCutoff) {
		t.Fatalf("cutoff = %v, want %v", store.cutoff, wantCutoff)
	}
}

func TestMerchantDirectoryEventCleanupTaskReturnsSafeDiagnosticOnDatabaseFailure(t *testing.T) {
	store := &fakeMerchantDirectoryEventCleanupStore{err: errors.New("pq: delete failed")}
	cleanup := NewMerchantDirectoryEventCleanupTask(store, 90, func() time.Time {
		return time.Date(2026, time.August, 2, 10, 0, 0, 0, time.UTC)
	})

	_, err := cleanup.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "清理商家目录行为失败") {
		t.Fatalf("Run() error = %v, want safe Chinese diagnostic", err)
	}
}

func TestMerchantDirectoryEventCleanupSchedulerRunsConfiguredTask(t *testing.T) {
	runner := &fakeMerchantDirectoryEventCleanupRunner{result: MerchantDirectoryEventCleanupResult{DeletedCount: 3}}
	scheduler := NewMerchantDirectoryEventCleanupScheduler(runner, 24*time.Hour, nil, nil, 0)

	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestMerchantDirectoryEventCleanupSchedulerIsDisabledWithoutInterval(t *testing.T) {
	runner := &fakeMerchantDirectoryEventCleanupRunner{}
	scheduler := NewMerchantDirectoryEventCleanupScheduler(runner, 0, nil, nil, 0)

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

type fakeMerchantDirectoryEventCleanupStore struct {
	cutoff  time.Time
	deleted int64
	err     error
}

func (s *fakeMerchantDirectoryEventCleanupStore) DeleteMerchantDirectoryEventsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.cutoff = cutoff
	return s.deleted, s.err
}

type fakeMerchantDirectoryEventCleanupRunner struct {
	calls  int
	result MerchantDirectoryEventCleanupResult
	err    error
}

func (r *fakeMerchantDirectoryEventCleanupRunner) Run(context.Context) (MerchantDirectoryEventCleanupResult, error) {
	r.calls++
	return r.result, r.err
}
