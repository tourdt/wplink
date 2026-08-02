package server

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMerchantMapEventRateLimiterUsesFixedWindowWithControlledTime(t *testing.T) {
	now := time.Date(2026, time.August, 2, 10, 0, 0, 0, time.UTC)
	limiter := newMerchantMapEventRateLimiter(2, time.Minute, 3, func() time.Time { return now })

	if !limiter.Allow("198.51.100.1", "visitor-1") || !limiter.Allow("198.51.100.1", "visitor-1") {
		t.Fatal("first two requests should be allowed")
	}
	if limiter.Allow("198.51.100.1", "visitor-1") {
		t.Fatal("third request in the fixed window should be rejected")
	}

	now = now.Add(time.Minute)
	if !limiter.Allow("198.51.100.1", "visitor-1") {
		t.Fatal("request after the fixed window should be allowed")
	}
}

func TestMerchantMapEventRateLimiterStaysBoundedAndCleansExpiredKeys(t *testing.T) {
	now := time.Date(2026, time.August, 2, 10, 0, 0, 0, time.UTC)
	limiter := newMerchantMapEventRateLimiter(60, time.Minute, 2, func() time.Time { return now })

	if !limiter.Allow("198.51.100.1", "visitor-1") || !limiter.Allow("198.51.100.2", "visitor-2") {
		t.Fatal("first two active keys should be allowed")
	}
	if limiter.Allow("198.51.100.3", "visitor-3") {
		t.Fatal("new active key should be rejected when bounded storage is full")
	}
	if len(limiter.windows) != 2 {
		t.Fatalf("active key count = %d, want bounded at 2", len(limiter.windows))
	}

	now = now.Add(time.Minute)
	if !limiter.Allow("198.51.100.3", "visitor-3") {
		t.Fatal("new key should be allowed after expired keys are cleaned")
	}
	if len(limiter.windows) != 1 {
		t.Fatalf("active key count after cleanup = %d, want 1", len(limiter.windows))
	}
}

func TestMerchantMapEventRateLimiterIsConcurrencySafe(t *testing.T) {
	now := time.Date(2026, time.August, 2, 10, 0, 0, 0, time.UTC)
	limiter := newMerchantMapEventRateLimiter(60, time.Minute, 4096, func() time.Time { return now })

	var allowed atomic.Int64
	var workers sync.WaitGroup
	for index := 0; index < 200; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if limiter.Allow("198.51.100.1", "visitor-1") {
				allowed.Add(1)
			}
		}()
	}
	workers.Wait()

	if allowed.Load() != 60 {
		t.Fatalf("allowed requests = %d, want exactly 60", allowed.Load())
	}
	if len(limiter.windows) != 1 {
		t.Fatalf("active key count = %d, want 1", len(limiter.windows))
	}
}
