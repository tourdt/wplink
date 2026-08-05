package task

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type TaskName string

const (
	TaskResourceLifecycle       TaskName = "resource_lifecycle"
	TaskContentAuditRetry       TaskName = "content_audit_retry"
	TaskPaymentReconciliation   TaskName = "payment_reconciliation"
	TaskMerchantMapEventCleanup TaskName = "merchant_map_event_cleanup"
)

var taskAdvisoryLockKeys = map[TaskName]int64{
	TaskResourceLifecycle:       1001,
	TaskContentAuditRetry:       1002,
	TaskPaymentReconciliation:   1003,
	TaskMerchantMapEventCleanup: 1004,
}

type CoordinationResult struct {
	Acquired bool
	Duration time.Duration
}

type Coordinator interface {
	RunExclusive(context.Context, TaskName, time.Duration, func(context.Context) error) (CoordinationResult, error)
}

type PostgresCoordinator struct {
	db         *sql.DB
	instanceID string
	now        func() time.Time
}

func NewPostgresCoordinator(db *sql.DB, instanceID string) *PostgresCoordinator {
	return &PostgresCoordinator{
		db:         db,
		instanceID: instanceID,
		now:        time.Now,
	}
}

func CurrentInstanceID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown-host"
	}
	instanceID := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	if len(instanceID) > 128 {
		return instanceID[:128]
	}
	return instanceID
}

func (c *PostgresCoordinator) RunExclusive(
	ctx context.Context,
	taskName TaskName,
	timeout time.Duration,
	run func(context.Context) error,
) (CoordinationResult, error) {
	lockKey, known := taskAdvisoryLockKeys[taskName]
	if !known {
		err := fmt.Errorf("未知自动任务 %s，无法获取协调锁", taskName)
		logx.WithContext(ctx).Errorw("自动任务协调配置错误", coordinationLogFields(
			"task_coordination_invalid_task", taskName, 0, c.instanceID, 0,
			logx.Field("error", err),
		)...)
		return CoordinationResult{}, err
	}
	if timeout <= 0 {
		err := fmt.Errorf("任务 %s 执行超时配置无效: %s", taskName, timeout)
		logx.WithContext(ctx).Errorw("自动任务协调配置错误", coordinationLogFields(
			"task_coordination_invalid_timeout", taskName, lockKey, c.instanceID, 0,
			logx.Field("error", err),
		)...)
		return CoordinationResult{}, err
	}

	// PostgreSQL session-level advisory lock 绑定数据库会话，因此加锁、业务执行和解锁期间
	// 始终独占同一个 *sql.Conn，不能退回连接池后再通过其他会话解锁。
	conn, err := c.db.Conn(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("任务 %s 申请专用数据库连接失败: %w", taskName, err)
		logx.WithContext(ctx).Errorw("自动任务申请专用数据库连接失败", coordinationLogFields(
			"task_coordination_connection_error", taskName, lockKey, c.instanceID, 0,
			logx.Field("error", err),
		)...)
		return CoordinationResult{}, wrappedErr
	}
	defer conn.Close()

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&acquired); err != nil {
		wrappedErr := fmt.Errorf("任务 %s 获取协调锁失败: %w", taskName, err)
		logx.WithContext(ctx).Errorw("自动任务获取协调锁失败", coordinationLogFields(
			"task_coordination_lock_error", taskName, lockKey, c.instanceID, 0,
			logx.Field("error", err),
		)...)
		return CoordinationResult{}, wrappedErr
	}
	if !acquired {
		logx.WithContext(ctx).Infow("自动任务因锁被占用而跳过", coordinationLogFields(
			"task_skipped_lock_held", taskName, lockKey, c.instanceID, 0,
		)...)
		return CoordinationResult{Acquired: false}, nil
	}

	startedAt := c.now()
	duration := time.Duration(0)
	// 即使业务上下文已经超时或取消，也要用独立短上下文释放会话锁；释放失败只记录并告警，
	// 不覆盖业务错误，便于调用方准确判断本次任务的实际执行结果。
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var unlocked bool
		unlockErr := conn.QueryRowContext(unlockCtx, "SELECT pg_advisory_unlock($1)", lockKey).Scan(&unlocked)
		connectionDiscarded := false
		var discardErr error
		if unlockErr != nil {
			// 解锁查询报错时无法判断 session 是否仍持锁。将底层连接标记为坏连接，避免它
			// 回池后因 advisory lock 可重入而误判获取成功，并长期阻塞其他实例接管。
			discardErr = conn.Raw(func(any) error { return driver.ErrBadConn })
			if discardErr == nil || errors.Is(discardErr, driver.ErrBadConn) || errors.Is(discardErr, sql.ErrConnDone) {
				connectionDiscarded = true
				discardErr = nil
			}
		}
		if unlockErr != nil || !unlocked {
			fields := coordinationLogFields(
				"task_coordination_unlock_failed", taskName, lockKey, c.instanceID, duration,
				logx.Field("unlocked", unlocked),
				logx.Field("connection_discarded", connectionDiscarded),
				logx.Field("error", unlockErr),
			)
			if discardErr != nil {
				fields = append(fields, logx.Field("connection_discard_error", discardErr))
			}
			logx.Errorw("自动任务协调锁释放失败", fields...)
		}
	}()

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// runner panic 必须保持原语义继续向上抛出；该 defer 先记录诊断信息，随后栈展开仍会
	// 执行上面的解锁 defer，避免一个异常任务长期阻塞其他实例。
	defer func() {
		if panicValue := recover(); panicValue != nil {
			duration = c.now().Sub(startedAt)
			logx.WithContext(ctx).Errorw("自动任务执行发生 panic", coordinationLogFields(
				"task_coordination_runner_panicked", taskName, lockKey, c.instanceID, duration,
				// panic 原值可能携带 token、DSN 或请求数据；日志只记录类型，原值仍继续抛出。
				logx.Field("panic_type", fmt.Sprintf("%T", panicValue)),
			)...)
			panic(panicValue)
		}
	}()

	runErr := run(runCtx)
	duration = c.now().Sub(startedAt)
	result := CoordinationResult{Acquired: true, Duration: duration}
	if runErr != nil {
		event := "task_coordination_runner_failed"
		if runCtx.Err() != nil {
			event = "task_coordination_runner_context_done"
		}
		logx.WithContext(ctx).Errorw("自动任务执行失败", coordinationLogFields(
			event, taskName, lockKey, c.instanceID, duration,
			logx.Field("error", runErr),
		)...)
		return result, runErr
	}
	if err := runCtx.Err(); err != nil {
		logx.WithContext(ctx).Errorw("自动任务执行上下文已结束", coordinationLogFields(
			"task_coordination_runner_context_done", taskName, lockKey, c.instanceID, duration,
			logx.Field("error", err),
		)...)
		return result, err
	}
	logx.WithContext(ctx).Infow("自动任务协调执行完成", coordinationLogFields(
		"task_coordination_completed", taskName, lockKey, c.instanceID, duration,
	)...)
	return result, nil
}

func coordinationLogFields(
	event string,
	taskName TaskName,
	lockKey int64,
	instanceID string,
	duration time.Duration,
	extra ...logx.LogField,
) []logx.LogField {
	fields := []logx.LogField{
		logx.Field("event", event),
		logx.Field("task", taskName),
		logx.Field("lock_key", lockKey),
		logx.Field("instance_id", instanceID),
		logx.Field("duration_ms", duration.Milliseconds()),
	}
	return append(fields, extra...)
}
