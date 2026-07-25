package auth

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"
)

type memorySMSSendLimit struct {
	lastSentAt       time.Time
	count            int
	reservationToken string
}

type MemorySMSSendLimiter struct {
	mu     sync.Mutex
	limits map[string]memorySMSSendLimit
}

func NewMemorySMSSendLimiter() *MemorySMSSendLimiter {
	return &MemorySMSSendLimiter{limits: make(map[string]memorySMSSendLimit)}
}

func (l *MemorySMSSendLimiter) Reserve(_ context.Context, phone string, now time.Time, minInterval time.Duration, dailyLimit int) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := phone + ":" + now.Format("2006-01-02")
	limit := l.limits[key]
	if !limit.lastSentAt.IsZero() && now.Sub(limit.lastSentAt) < minInterval {
		return "", ErrSMSSendTooFrequent
	}
	if limit.count >= dailyLimit {
		return "", ErrSMSDailyLimit
	}
	token := newSMSSendReservationToken()
	limit.lastSentAt = now
	limit.count++
	limit.reservationToken = token
	l.limits[key] = limit
	return token, nil
}

func (l *MemorySMSSendLimiter) Rollback(_ context.Context, phone string, now time.Time, reservationToken string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := phone + ":" + now.Format("2006-01-02")
	limit, ok := l.limits[key]
	if !ok || limit.reservationToken != reservationToken {
		return nil
	}
	limit.count--
	if limit.count <= 0 {
		delete(l.limits, key)
		return nil
	}
	limit.lastSentAt = time.Time{}
	limit.reservationToken = ""
	l.limits[key] = limit
	return nil
}

// SQLSMSSendLimiter 把限流状态存入 PostgreSQL，确保多个 API 实例共享同一份计数。
// INSERT ... ON CONFLICT 的行锁让“检查间隔 + 增加次数”成为单条原子操作。
type SQLSMSSendLimiter struct {
	db *sql.DB
}

func NewSQLSMSSendLimiter(db *sql.DB) *SQLSMSSendLimiter {
	return &SQLSMSSendLimiter{db: db}
}

func (l *SQLSMSSendLimiter) Reserve(ctx context.Context, phone string, now time.Time, minInterval time.Duration, dailyLimit int) (string, error) {
	if l == nil || l.db == nil {
		return "", errors.New("短信限流数据库未配置")
	}
	token := newSMSSendReservationToken()
	var reservedToken string
	err := l.db.QueryRowContext(ctx, `
INSERT INTO sms_send_limits (
  phone,
  send_date,
  last_sent_at,
  send_count,
  reservation_token,
  updated_at
)
VALUES ($1, $2::date, $3, 1, $4, $3)
ON CONFLICT (phone, send_date) DO UPDATE
SET
  last_sent_at = EXCLUDED.last_sent_at,
  send_count = sms_send_limits.send_count + 1,
  reservation_token = EXCLUDED.reservation_token,
  updated_at = EXCLUDED.updated_at
WHERE sms_send_limits.last_sent_at <= EXCLUDED.last_sent_at - ($5 * interval '1 second')
  AND sms_send_limits.send_count < $6
RETURNING reservation_token
`, phone, now.Format("2006-01-02"), now, token, minInterval.Seconds(), dailyLimit).Scan(&reservedToken)
	if err == nil {
		return reservedToken, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	var lastSentAt time.Time
	var count int
	if err := l.db.QueryRowContext(ctx, `
SELECT last_sent_at, send_count
FROM sms_send_limits
WHERE phone = $1 AND send_date = $2::date
`, phone, now.Format("2006-01-02")).Scan(&lastSentAt, &count); err != nil {
		return "", err
	}
	if count >= dailyLimit {
		return "", ErrSMSDailyLimit
	}
	return "", ErrSMSSendTooFrequent
}

func (l *SQLSMSSendLimiter) Rollback(ctx context.Context, phone string, now time.Time, reservationToken string) error {
	if l == nil || l.db == nil {
		return errors.New("短信限流数据库未配置")
	}
	_, err := l.db.ExecContext(ctx, `
UPDATE sms_send_limits
SET
  send_count = GREATEST(send_count - 1, 0),
  last_sent_at = '-infinity'::timestamptz,
  reservation_token = '',
  updated_at = now()
WHERE phone = $1
  AND send_date = $2::date
  AND reservation_token = $3
`, phone, now.Format("2006-01-02"), reservationToken)
	return err
}
