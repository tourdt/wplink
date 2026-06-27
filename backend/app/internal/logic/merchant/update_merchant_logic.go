package merchant

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type UpdateMerchantStore interface {
	UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error)
}

type UpdateMerchantReq struct {
	MainCategories []string
	Description    string
	Images         []string
}

type UpdateMerchantResp struct {
	ID        string
	UpdatedAt string
}

type UpdateMerchantLogic struct {
	store UpdateMerchantStore
}

func NewUpdateMerchantLogic(store UpdateMerchantStore) *UpdateMerchantLogic {
	return &UpdateMerchantLogic{store: store}
}

func (l *UpdateMerchantLogic) UpdateMerchant(ctx context.Context, merchantID string, req UpdateMerchantReq) (UpdateMerchantResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或已停用")
	}

	updatedAt, err := l.store.UpdateMerchant(ctx, merchantID, model.UpdateMerchantPatch{
		MainCategories: append([]string(nil), req.MainCategories...),
		Description:    strings.TrimSpace(req.Description),
		Images:         append([]string(nil), req.Images...),
	})
	if err != nil {
		return UpdateMerchantResp{}, err
	}
	return UpdateMerchantResp{ID: merchantID, UpdatedAt: updatedAt}, nil
}
