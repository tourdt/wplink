package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"wplink/backend/app/internal/config"

	_ "github.com/lib/pq"
)

func TestSQLSMSSendLimiterIntegrationConcurrentReserve(t *testing.T) {
	db, phone := openSMSLimiterIntegrationDB(t)
	limiter := NewSQLSMSSendLimiter(db)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

	const workers = 8
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := limiter.Reserve(context.Background(), phone, now, time.Minute, 10)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes, limited := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrSMSSendTooFrequent):
			limited++
		default:
			t.Fatalf("Reserve() error = %v", err)
		}
	}
	if successes != 1 || limited != workers-1 {
		t.Fatalf("successes/limited = %d/%d, want 1/%d", successes, limited, workers-1)
	}
}

func TestSQLSMSSendLimiterIntegrationDailyLimitAndTokenGuard(t *testing.T) {
	db, phone := openSMSLimiterIntegrationDB(t)
	limiter := NewSQLSMSSendLimiter(db)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

	oldToken, err := limiter.Reserve(context.Background(), phone, now, time.Minute, 2)
	if err != nil {
		t.Fatalf("first Reserve() error = %v", err)
	}
	newToken, err := limiter.Reserve(context.Background(), phone, now.Add(2*time.Minute), time.Minute, 2)
	if err != nil {
		t.Fatalf("second Reserve() error = %v", err)
	}
	if err := limiter.Rollback(context.Background(), phone, now, oldToken); err != nil {
		t.Fatalf("old Rollback() error = %v", err)
	}
	var count int
	var storedToken string
	if err := db.QueryRow(`SELECT send_count, reservation_token FROM sms_send_limits WHERE phone=$1`, phone).Scan(&count, &storedToken); err != nil {
		t.Fatalf("query limit row: %v", err)
	}
	if count != 2 || storedToken != newToken {
		t.Fatalf("count/token = %d/%q, want 2/%q", count, storedToken, newToken)
	}
	if _, err := limiter.Reserve(context.Background(), phone, now.Add(4*time.Minute), time.Minute, 2); !errors.Is(err, ErrSMSDailyLimit) {
		t.Fatalf("third Reserve() error = %v, want daily limit", err)
	}
}

func TestSQLSMSSendLimiterIntegrationProviderFailureRollsBack(t *testing.T) {
	db, phone := openSMSLimiterIntegrationDB(t)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	cfg := config.SMSConfig{
		Provider:        "http",
		SendURL:         "https://sms.test/send",
		SendMinInterval: time.Minute,
		DailySendLimit:  1,
	}
	failing := NewConfiguredSMSVerifierWithLimiter(cfg, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})}, NewSQLSMSSendLimiter(db))
	failing.now = func() time.Time { return now }
	if err := failing.SendSMSCode(context.Background(), phone); err == nil {
		t.Fatal("first SendSMSCode() error = nil, want provider failure")
	}

	success := NewConfiguredSMSVerifierWithLimiter(cfg, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	})}, NewSQLSMSSendLimiter(db))
	success.now = func() time.Time { return now }
	if err := success.SendSMSCode(context.Background(), phone); err != nil {
		t.Fatalf("second SendSMSCode() error = %v, want rollback to release reservation", err)
	}
}

func TestSQLSMSSendLimiterIntegrationConcurrentRollbackNeverNegative(t *testing.T) {
	db, phone := openSMSLimiterIntegrationDB(t)
	limiter := NewSQLSMSSendLimiter(db)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

	token, err := limiter.Reserve(context.Background(), phone, now, time.Minute, 10)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}

	const workers = 2
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- limiter.Rollback(context.Background(), phone, now, token)
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("Rollback() error = %v", err)
		}
	}

	var count int
	if err := db.QueryRow(`SELECT send_count FROM sms_send_limits WHERE phone=$1`, phone).Scan(&count); err != nil {
		t.Fatalf("query limit row: %v", err)
	}
	if count != 0 {
		t.Fatalf("send_count = %d, want 0", count)
	}
}

func openSMSLimiterIntegrationDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("WPLINK_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("未设置 WPLINK_TEST_POSTGRES_DSN")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 测试库失败: %v", err)
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(16)
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		t.Fatalf("连接 PostgreSQL 测试库失败: %v", err)
	}
	phone := fmt.Sprintf("it-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := db.ExecContext(ctx, `DELETE FROM sms_send_limits WHERE phone=$1`, phone); err != nil {
			t.Errorf("清理短信限流测试数据失败: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Errorf("关闭 PostgreSQL 测试连接失败: %v", err)
		}
	})
	return db, phone
}
