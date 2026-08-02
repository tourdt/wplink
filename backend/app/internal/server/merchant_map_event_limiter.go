package server

import (
	"crypto/sha256"
	"strings"
	"sync"
	"time"
)

const (
	merchantMapEventRequestBodyLimit = 4096
	merchantMapEventRateLimit        = 60
	merchantMapEventRateWindowSize   = time.Minute
	merchantMapEventRateMaxKeys      = 4096
)

type merchantMapEventRateWindow struct {
	startedAt time.Time
	count     int
}

type merchantMapEventRateLimiter struct {
	mu        sync.Mutex
	windows   map[[sha256.Size]byte]merchantMapEventRateWindow
	limit     int
	window    time.Duration
	maxKeys   int
	now       func() time.Time
	lastSweep time.Time
}

func newMerchantMapEventRateLimiter(limit int, window time.Duration, maxKeys int, now func() time.Time) *merchantMapEventRateLimiter {
	if now == nil {
		now = time.Now
	}
	return &merchantMapEventRateLimiter{
		windows: make(map[[sha256.Size]byte]merchantMapEventRateWindow),
		limit:   limit,
		window:  window,
		maxKeys: maxKeys,
		now:     now,
	}
}

func (l *merchantMapEventRateLimiter) Allow(clientIP string, visitorKey string) bool {
	if l == nil || l.limit <= 0 || l.window <= 0 || l.maxKeys <= 0 || l.now == nil {
		return false
	}

	now := l.now()
	key := sha256.Sum256([]byte(strings.TrimSpace(clientIP) + "\x00" + strings.TrimSpace(visitorKey)))

	l.mu.Lock()
	defer l.mu.Unlock()

	// 定期清理过期窗口；容量已满时也立即清理，确保活跃 key 永远不会超过硬上限。
	if l.lastSweep.IsZero() || !now.Before(l.lastSweep.Add(l.window)) || len(l.windows) >= l.maxKeys {
		l.cleanupExpiredLocked(now)
		l.lastSweep = now
	}

	current, exists := l.windows[key]
	if exists && !now.Before(current.startedAt.Add(l.window)) {
		delete(l.windows, key)
		exists = false
	}
	if exists {
		if current.count >= l.limit {
			return false
		}
		current.count++
		l.windows[key] = current
		return true
	}
	if len(l.windows) >= l.maxKeys {
		// 不驱逐仍活跃的窗口，避免攻击者通过不断换 key 重置已有请求计数。
		return false
	}
	l.windows[key] = merchantMapEventRateWindow{startedAt: now, count: 1}
	return true
}

func (l *merchantMapEventRateLimiter) cleanupExpiredLocked(now time.Time) {
	for key, current := range l.windows {
		if !now.Before(current.startedAt.Add(l.window)) {
			delete(l.windows, key)
		}
	}
}
