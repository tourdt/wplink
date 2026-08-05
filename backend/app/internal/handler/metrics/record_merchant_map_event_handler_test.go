package metrics

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMerchantMapEventLimiterRejectsNewKeyAtCapacityUntilWindowExpires(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	limiter := newMerchantMapEventLimiter(60, time.Minute, 2, func() time.Time { return now })
	if !limiter.Allow("198.51.100.1", "visitor-1") || !limiter.Allow("198.51.100.2", "visitor-2") {
		t.Fatal("first two distinct keys should fit capacity")
	}
	if limiter.Allow("198.51.100.3", "visitor-3") {
		t.Fatal("third active key should be rejected at hard capacity")
	}

	now = now.Add(time.Minute)
	if !limiter.Allow("198.51.100.3", "visitor-3") {
		t.Fatal("expired windows should be reclaimed for a new key")
	}
}

func TestMerchantMapEventLimiterAllowsExactlySixtyUnderConcurrency(t *testing.T) {
	limiter := newMerchantMapEventLimiter(60, time.Minute, 4096, func() time.Time {
		return time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	})

	start := make(chan struct{})
	var wg sync.WaitGroup
	var allowed int64
	for index := 0; index < 200; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if limiter.Allow("198.51.100.7", "visitor-shared") {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if allowed != 60 {
		t.Fatalf("allowed=%d, want exactly 60", allowed)
	}
	if limiter.Allow("198.51.100.7", "visitor-shared") {
		t.Fatal("request after the first 60 should remain rate limited")
	}
}
