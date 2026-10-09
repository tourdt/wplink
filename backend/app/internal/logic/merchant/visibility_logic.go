package merchant

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/common/errx"
)

type MerchantVisibilityStore interface {
	SetMerchantVisibility(ctx context.Context, merchantID string, visible bool) error
}

func SetMerchantVisibility(ctx context.Context, store MerchantVisibilityStore, merchantID string, visible *bool) (string, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" || visible == nil {
		return "", errx.New(errx.CodeValidationFailed, "请选择店铺展示状态")
	}
	if err := store.SetMerchantVisibility(ctx, merchantID, *visible); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errx.New(errx.CodeResourceNotFound, "商家不存在或已停用")
		}
		logMerchantFailure(ctx, "set_merchant_visibility", merchantID, err)
		return "", errx.New(errx.CodeInternalError, "店铺展示状态保存失败，请稍后重试")
	}
	return merchantID, nil
}
