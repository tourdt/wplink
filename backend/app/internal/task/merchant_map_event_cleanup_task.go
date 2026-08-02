package task

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type MerchantMapEventCleanupStore interface {
	DeleteMerchantMapEventsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type MerchantMapEventCleanupResult struct {
	DeletedCount int64
}

type MerchantMapEventCleanupTask struct {
	store         MerchantMapEventCleanupStore
	retentionDays int
	now           func() time.Time
}

func NewMerchantMapEventCleanupTask(store MerchantMapEventCleanupStore, retentionDays int, now func() time.Time) *MerchantMapEventCleanupTask {
	if now == nil {
		now = time.Now
	}
	return &MerchantMapEventCleanupTask{store: store, retentionDays: retentionDays, now: now}
}

func (t *MerchantMapEventCleanupTask) Run(ctx context.Context) (MerchantMapEventCleanupResult, error) {
	if t == nil || t.store == nil || t.retentionDays <= 0 || t.now == nil {
		return MerchantMapEventCleanupResult{}, errors.New("商家地图行为清理任务配置无效")
	}

	// 按自然日计算 90 天保留边界，避免夏令时或时区变化造成小时级漂移。
	cutoff := t.now().AddDate(0, 0, -t.retentionDays)
	deleted, err := t.store.DeleteMerchantMapEventsBefore(ctx, cutoff)
	if err != nil {
		return MerchantMapEventCleanupResult{}, fmt.Errorf("清理商家地图行为失败: %w", err)
	}
	return MerchantMapEventCleanupResult{DeletedCount: deleted}, nil
}
