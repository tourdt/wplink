package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMerchantMapEventCleanupTaskDeletesEventsOlderThanNinetyDays(t *testing.T) {
	now := time.Date(2026, time.August, 2, 10, 0, 0, 0, time.UTC)
	store := &fakeMerchantMapEventCleanupStore{deleted: 7}
	cleanup := NewMerchantMapEventCleanupTask(store, 90, func() time.Time { return now })

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

func TestMerchantMapEventCleanupTaskReturnsSafeDiagnosticOnDatabaseFailure(t *testing.T) {
	store := &fakeMerchantMapEventCleanupStore{err: errors.New("pq: delete failed")}
	cleanup := NewMerchantMapEventCleanupTask(store, 90, func() time.Time {
		return time.Date(2026, time.August, 2, 10, 0, 0, 0, time.UTC)
	})

	_, err := cleanup.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "清理商家地图行为失败") {
		t.Fatalf("Run() error = %v, want safe Chinese diagnostic", err)
	}
}

func TestMerchantMapEventCleanupSchedulerRunsConfiguredTask(t *testing.T) {
	runner := &fakeMerchantMapEventCleanupRunner{result: MerchantMapEventCleanupResult{DeletedCount: 3}}
	scheduler := NewMerchantMapEventCleanupScheduler(runner, 24*time.Hour, nil)

	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestMerchantMapEventCleanupSchedulerIsDisabledWithoutInterval(t *testing.T) {
	runner := &fakeMerchantMapEventCleanupRunner{}
	scheduler := NewMerchantMapEventCleanupScheduler(runner, 0, nil)

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

type fakeMerchantMapEventCleanupStore struct {
	cutoff  time.Time
	deleted int64
	err     error
}

func (s *fakeMerchantMapEventCleanupStore) DeleteMerchantMapEventsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.cutoff = cutoff
	return s.deleted, s.err
}

type fakeMerchantMapEventCleanupRunner struct {
	calls  int
	result MerchantMapEventCleanupResult
	err    error
}

func (r *fakeMerchantMapEventCleanupRunner) Run(context.Context) (MerchantMapEventCleanupResult, error) {
	r.calls++
	return r.result, r.err
}
