package task

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestCoordinatorIntegrationMultiInstanceMutualExclusionAndTakeover(t *testing.T) {
	dsn := os.Getenv("WPLINK_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("未设置 WPLINK_TEST_POSTGRES_DSN")
	}
	dbA := openCoordinatorIntegrationDB(t, dsn)
	dbB := openCoordinatorIntegrationDB(t, dsn)

	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseRunner := func() { releaseOnce.Do(func() { close(release) }) }
	firstDone := make(chan struct{})
	var firstErr error
	go func() {
		_, firstErr = NewPostgresCoordinator(dbA, "api-a").RunExclusive(
			context.Background(), TaskResourceLifecycle, time.Minute,
			func(context.Context) error {
				close(entered)
				<-release
				return nil
			},
		)
		close(firstDone)
	}()
	cleanupCoordinatorIntegrationRunner(t, releaseRunner, firstDone)
	waitCoordinatorIntegrationSignal(t, entered, "实例 A 未进入持锁任务")

	second, err := NewPostgresCoordinator(dbB, "api-b").RunExclusive(
		context.Background(), TaskResourceLifecycle, time.Second,
		func(context.Context) error {
			t.Fatal("持锁期间第二实例不应执行")
			return nil
		},
	)
	if err != nil || second.Acquired {
		t.Fatalf("第二实例应跳过: result=%+v err=%v", second, err)
	}
	releaseRunner()
	waitCoordinatorIntegrationSignal(t, firstDone, "实例 A 未在释放信号后结束")
	if firstErr != nil {
		t.Fatalf("实例 A 持锁任务执行失败: %v", firstErr)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		result, runErr := NewPostgresCoordinator(dbB, "api-b").RunExclusive(
			context.Background(), TaskResourceLifecycle, time.Second,
			func(context.Context) error { return nil },
		)
		if runErr == nil && result.Acquired {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("第一实例释放锁后第二实例未能接管")
}

func TestCoordinatorIntegrationTakeoverAfterBackendTermination(t *testing.T) {
	dsn := os.Getenv("WPLINK_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("未设置 WPLINK_TEST_POSTGRES_DSN")
	}
	dbA := openCoordinatorIntegrationDB(t, dsn)
	dbB := openCoordinatorIntegrationDB(t, dsn)
	connA, err := dbA.Conn(context.Background())
	if err != nil {
		t.Fatalf("申请实例 A 连接失败: %v", err)
	}
	defer connA.Close()

	var backendPID int
	var acquired bool
	if err := connA.QueryRowContext(
		context.Background(),
		"SELECT pg_backend_pid(), pg_try_advisory_lock($1)",
		int64(1001),
	).Scan(&backendPID, &acquired); err != nil || !acquired {
		t.Fatalf("实例 A 获取测试锁失败: acquired=%t err=%v", acquired, err)
	}

	var terminated bool
	if err := dbB.QueryRowContext(
		context.Background(),
		"SELECT pg_terminate_backend($1)",
		backendPID,
	).Scan(&terminated); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "42501" {
			t.Skipf("测试账号无权终止 PostgreSQL backend: %v", err)
		}
		t.Fatalf("终止 PostgreSQL backend 失败: %v", err)
	}
	if !terminated {
		t.Fatal("PostgreSQL 返回 backend 未终止")
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		result, runErr := NewPostgresCoordinator(dbB, "api-b").RunExclusive(
			context.Background(), TaskResourceLifecycle, time.Second,
			func(context.Context) error { return nil },
		)
		if runErr == nil && result.Acquired {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("实例 A 数据库会话终止后实例 B 未能接管")
}

func TestCoordinatorIntegrationDifferentLockKeysRunConcurrently(t *testing.T) {
	dsn := os.Getenv("WPLINK_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("未设置 WPLINK_TEST_POSTGRES_DSN")
	}
	dbA := openCoordinatorIntegrationDB(t, dsn)
	dbB := openCoordinatorIntegrationDB(t, dsn)

	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseRunner := func() { releaseOnce.Do(func() { close(release) }) }
	firstDone := make(chan struct{})
	var firstErr error
	go func() {
		_, firstErr = NewPostgresCoordinator(dbA, "api-a").RunExclusive(
			context.Background(), TaskResourceLifecycle, time.Minute,
			func(context.Context) error {
				close(entered)
				<-release
				return nil
			},
		)
		close(firstDone)
	}()
	cleanupCoordinatorIntegrationRunner(t, releaseRunner, firstDone)
	waitCoordinatorIntegrationSignal(t, entered, "资源生命周期任务未获取测试锁")

	result, err := NewPostgresCoordinator(dbB, "api-b").RunExclusive(
		context.Background(), TaskContentAuditRetry, time.Second,
		func(context.Context) error { return nil },
	)
	if err != nil || !result.Acquired {
		t.Fatalf("不同锁编号应能并行获取: result=%+v err=%v", result, err)
	}
	releaseRunner()
	waitCoordinatorIntegrationSignal(t, firstDone, "资源生命周期任务未在释放信号后结束")
	if firstErr != nil {
		t.Fatalf("资源生命周期任务执行失败: %v", firstErr)
	}
}

func openCoordinatorIntegrationDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 测试数据库失败: %v", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("关闭 PostgreSQL 测试数据库失败: %v", err)
		}
	})

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		t.Fatalf("连接 PostgreSQL 测试数据库失败: %v", err)
	}
	return db
}

func waitCoordinatorIntegrationSignal(t *testing.T, signal <-chan struct{}, timeoutMessage string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal(timeoutMessage)
	}
}

func cleanupCoordinatorIntegrationRunner(t *testing.T, release func(), done <-chan struct{}) {
	t.Helper()
	t.Cleanup(func() {
		release()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Errorf("等待持锁测试 goroutine 退出超时")
		}
	})
}
