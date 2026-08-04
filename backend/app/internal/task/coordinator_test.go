package task

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestTaskAdvisoryLockKeysAreStable(t *testing.T) {
	want := map[TaskName]int64{
		TaskResourceLifecycle:       1001,
		TaskContentAuditRetry:       1002,
		TaskPaymentReconciliation:   1003,
		TaskMerchantMapEventCleanup: 1004,
	}
	for taskName, lockKey := range want {
		if got := taskAdvisoryLockKeys[taskName]; got != lockKey {
			t.Fatalf("任务 %s 锁编号错误: got=%d want=%d", taskName, got, lockKey)
		}
	}
}

func TestCoordinatorSkipsWhenLockIsHeld(t *testing.T) {
	db, mock := newCoordinatorMockDB(t)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(int64(1001)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))

	called := false
	result, err := NewPostgresCoordinator(db, "api-a").RunExclusive(
		context.Background(), TaskResourceLifecycle, time.Second,
		func(context.Context) error {
			called = true
			return nil
		},
	)

	if err != nil || result.Acquired || called {
		t.Fatalf("锁占用时结果不正确: result=%+v called=%t err=%v", result, called, err)
	}
	assertCoordinatorSQLExpectations(t, mock)
}

func TestCoordinatorRunsAndUnlocksOnDedicatedConnection(t *testing.T) {
	db, mock := newCoordinatorMockDB(t)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(int64(1002)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(\$1\)`).
		WithArgs(int64(1002)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))

	base := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	times := []time.Time{base, base.Add(125 * time.Millisecond)}
	nextTime := 0
	coordinator := NewPostgresCoordinator(db, "api-a")
	coordinator.now = func() time.Time {
		value := times[nextTime]
		nextTime++
		return value
	}

	called := false
	result, err := coordinator.RunExclusive(
		context.Background(), TaskContentAuditRetry, time.Second,
		func(context.Context) error {
			// runner 执行期间专用连接必须仍处于占用状态；若实现退回 db.QueryRowContext，
			// 查询结束后连接会回池，此处 InUse 将变成 0。
			if inUse := db.Stats().InUse; inUse != 1 {
				return fmt.Errorf("runner 执行时专用连接数错误: got=%d want=1", inUse)
			}
			called = true
			return nil
		},
	)

	if err != nil || !result.Acquired || !called || result.Duration != 125*time.Millisecond {
		t.Fatalf("持锁执行结果不正确: result=%+v called=%t err=%v", result, called, err)
	}
	assertCoordinatorSQLExpectations(t, mock)
}

func TestCoordinatorKeepsRunnerErrorWhenUnlockFails(t *testing.T) {
	db, mock := newCoordinatorMockDB(t)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(int64(1003)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(\$1\)`).
		WithArgs(int64(1003)).
		WillReturnError(errors.New("unlock failed"))

	runnerErr := errors.New("payment provider unavailable")
	result, err := NewPostgresCoordinator(db, "api-a").RunExclusive(
		context.Background(), TaskPaymentReconciliation, time.Second,
		func(context.Context) error { return runnerErr },
	)

	if !result.Acquired || !errors.Is(err, runnerErr) {
		t.Fatalf("解锁失败不应覆盖业务错误: result=%+v err=%v", result, err)
	}
	assertCoordinatorSQLExpectations(t, mock)
}

func TestCoordinatorDiscardsPhysicalConnectionWhenUnlockQueryFails(t *testing.T) {
	db, mock := newCoordinatorMockDB(t)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(int64(1003)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(\$1\)`).
		WithArgs(int64(1003)).
		WillReturnError(errors.New("unlock connection state unknown"))

	result, err := NewPostgresCoordinator(db, "api-a").RunExclusive(
		context.Background(), TaskPaymentReconciliation, time.Second,
		func(context.Context) error { return nil },
	)

	if err != nil || !result.Acquired {
		t.Fatalf("解锁失败不应改变业务执行结果: result=%+v err=%v", result, err)
	}
	stats := db.Stats()
	if stats.OpenConnections != 0 || stats.Idle != 0 {
		t.Fatalf("锁状态未知的物理连接不得回池复用: open=%d idle=%d", stats.OpenConnections, stats.Idle)
	}
	assertCoordinatorSQLExpectations(t, mock)
}

func TestCoordinatorReturnsDeadlineExceededAndUnlocks(t *testing.T) {
	db, mock := newCoordinatorMockDB(t)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(int64(1004)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(\$1\)`).
		WithArgs(int64(1004)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))

	result, err := NewPostgresCoordinator(db, "api-a").RunExclusive(
		context.Background(), TaskMerchantMapEventCleanup, 20*time.Millisecond,
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	)

	if !result.Acquired || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("任务超时结果不正确: result=%+v err=%v", result, err)
	}
	assertCoordinatorSQLExpectations(t, mock)
}

func TestCoordinatorRejectsUnknownTaskBeforeOpeningConnection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	mock.ExpectClose()
	if err := db.Close(); err != nil {
		t.Fatalf("关闭 sqlmock 失败: %v", err)
	}

	called := false
	_, err = NewPostgresCoordinator(db, "api-a").RunExclusive(
		context.Background(), TaskName("not_registered"), time.Second,
		func(context.Context) error {
			called = true
			return nil
		},
	)

	if err == nil || !strings.Contains(err.Error(), "未知") || !strings.Contains(err.Error(), "not_registered") || called {
		t.Fatalf("未知任务校验结果不正确: called=%t err=%v", called, err)
	}
}

func TestCoordinatorRejectsNonPositiveTimeoutBeforeOpeningConnection(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("创建 sqlmock 失败: %v", err)
			}
			mock.ExpectClose()
			if err := db.Close(); err != nil {
				t.Fatalf("关闭 sqlmock 失败: %v", err)
			}

			called := false
			_, err = NewPostgresCoordinator(db, "api-a").RunExclusive(
				context.Background(), TaskResourceLifecycle, timeout,
				func(context.Context) error {
					called = true
					return nil
				},
			)

			if err == nil || !strings.Contains(err.Error(), "超时") || !strings.Contains(err.Error(), string(TaskResourceLifecycle)) || called {
				t.Fatalf("非法超时校验结果不正确: timeout=%s called=%t err=%v", timeout, called, err)
			}
		})
	}
}

func TestCoordinatorUnlocksAndRepanicsWhenRunnerPanics(t *testing.T) {
	db, mock := newCoordinatorMockDB(t)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(int64(1001)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(\$1\)`).
		WithArgs(int64(1001)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))

	var logBuffer bytes.Buffer
	previousWriter := logx.Reset()
	logx.SetWriter(logx.NewWriter(&logBuffer))
	t.Cleanup(func() {
		if currentWriter := logx.Reset(); currentWriter != nil {
			_ = currentWriter.Close()
		}
		if previousWriter != nil {
			logx.SetWriter(previousWriter)
		}
	})

	panicValue := "token=secret-do-not-log"
	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		_, _ = NewPostgresCoordinator(db, "api-a").RunExclusive(
			context.Background(), TaskResourceLifecycle, time.Second,
			func(context.Context) error { panic(panicValue) },
		)
	}()

	if recovered != panicValue {
		t.Fatalf("runner panic 未按原值继续抛出: got=%v want=%v", recovered, panicValue)
	}
	logText := logBuffer.String()
	if strings.Contains(logText, panicValue) || !strings.Contains(logText, `"panic_type":"string"`) {
		t.Fatalf("panic 日志应记录类型且不得包含原值: log=%q", logText)
	}
	assertCoordinatorSQLExpectations(t, mock)
}

func TestCoordinatorReturnsContextualLockQueryError(t *testing.T) {
	db, mock := newCoordinatorMockDB(t)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(int64(1001)).
		WillReturnError(errors.New("database unavailable"))

	_, err := NewPostgresCoordinator(db, "api-a").RunExclusive(
		context.Background(), TaskResourceLifecycle, time.Second,
		func(context.Context) error { return nil },
	)

	if err == nil || !strings.Contains(err.Error(), "获取协调锁失败") || !strings.Contains(err.Error(), string(TaskResourceLifecycle)) {
		t.Fatalf("获取锁错误缺少中文上下文: %v", err)
	}
	assertCoordinatorSQLExpectations(t, mock)
}

func TestCoordinatorCurrentInstanceIDUsesHostAndPIDWithinLimit(t *testing.T) {
	instanceID := CurrentInstanceID()
	pidSuffix := fmt.Sprintf(":%d", os.Getpid())
	if len(instanceID) == 0 || len(instanceID) > 128 || !strings.HasSuffix(instanceID, pidSuffix) {
		t.Fatalf("实例 ID 格式不正确: id=%q len=%d suffix=%q", instanceID, len(instanceID), pidSuffix)
	}
}

func newCoordinatorMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func assertCoordinatorSQLExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 期望未满足: %v", err)
	}
}
