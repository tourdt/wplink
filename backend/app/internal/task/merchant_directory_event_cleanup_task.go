package task

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type MerchantDirectoryEventCleanupStore interface {
	DeleteMerchantDirectoryEventsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type MerchantDirectoryEventCleanupResult struct {
	DeletedCount int64
}

type MerchantDirectoryEventCleanupTask struct {
	store         MerchantDirectoryEventCleanupStore
	retentionDays int
	now           func() time.Time
}

func NewMerchantDirectoryEventCleanupTask(store MerchantDirectoryEventCleanupStore, retentionDays int, now func() time.Time) *MerchantDirectoryEventCleanupTask {
	if now == nil {
		now = time.Now
	}
	return &MerchantDirectoryEventCleanupTask{store: store, retentionDays: retentionDays, now: now}
}

func (t *MerchantDirectoryEventCleanupTask) Run(ctx context.Context) (MerchantDirectoryEventCleanupResult, error) {
	if t == nil || t.store == nil || t.retentionDays <= 0 || t.now == nil {
		return MerchantDirectoryEventCleanupResult{}, errors.New("商家目录行为清理任务配置无效")
	}

	// 按自然日计算 90 天保留边界，避免夏令时或时区变化造成小时级漂移。
	cutoff := t.now().AddDate(0, 0, -t.retentionDays)
	deleted, err := t.store.DeleteMerchantDirectoryEventsBefore(ctx, cutoff)
	if err != nil {
		return MerchantDirectoryEventCleanupResult{}, fmt.Errorf("清理商家目录行为失败: %w", err)
	}
	return MerchantDirectoryEventCleanupResult{DeletedCount: deleted}, nil
}
