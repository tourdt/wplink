package task

import (
	"context"
	"sync"
	"time"
)

type schedulerRuntime struct {
	taskName    TaskName
	coordinator Coordinator
	timeout     time.Duration
	wg          sync.WaitGroup
}

func newSchedulerRuntime(taskName TaskName, coordinator Coordinator, timeout time.Duration) *schedulerRuntime {
	return &schedulerRuntime{
		taskName:    taskName,
		coordinator: coordinator,
		timeout:     timeout,
	}
}

func (r *schedulerRuntime) start(ctx context.Context, interval time.Duration, run func(context.Context) error) {
	if interval <= 0 {
		return
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		// 首轮和后续 tick 始终在同一个 goroutine 内串行执行，避免慢任务跨周期重入。
		r.runOnce(ctx, run)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 当前轮执行期间可能已有 tick 积压；取消后不能因 select 随机选中该 tick
				// 而启动新一轮，退出前仅等待正在执行的这一轮自然结束。
				select {
				case <-ctx.Done():
					return
				default:
				}
				r.runOnce(ctx, run)
			}
		}
	}()
}

func (r *schedulerRuntime) runOnce(ctx context.Context, run func(context.Context) error) {
	if r.coordinator == nil {
		// nil 仅用于既有 Scheduler 单元测试；生产装配必须注入 Coordinator。
		_ = run(ctx)
		return
	}
	// Coordinator 已统一记录锁、超时和 runner 错误，本层只维持调度循环，避免重复日志。
	_, _ = r.coordinator.RunExclusive(ctx, r.taskName, r.timeout, run)
}

func (r *schedulerRuntime) wait() {
	r.wg.Wait()
}
