package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// SetMerchantVisibility 只改变店铺公开状态，不重写经营资料和地图点位。
func (m *MerchantModel) SetMerchantVisibility(ctx context.Context, merchantID string, visible bool) error {
	result, err := m.db.ExecContext(ctx, `UPDATE merchants SET owner_visible = $2, updated_at = $3 WHERE id = $1 AND deleted_at IS NULL`, merchantID, visible, time.Now().UTC())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	logx.WithContext(ctx).Infow("店铺展示状态已更新", logx.Field("merchantId", merchantID), logx.Field("ownerVisible", visible))
	return nil
}
