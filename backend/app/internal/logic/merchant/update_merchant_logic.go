package merchant

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateMerchantStore interface {
	UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error)
}

type UpdateMerchantReq struct {
	Name            string                 `json:"name,omitempty"`
	MainCategories  []string               `json:"mainCategories,omitempty"`
	Description     string                 `json:"description,omitempty"`
	LogoURL         string                 `json:"logoUrl,omitempty"`
	Images          []string               `json:"images,omitempty"`
	ContactName     string                 `json:"contactName,omitempty"`
	ContactPhone    string                 `json:"contactPhone,omitempty"`
	ContactWechat   string                 `json:"contactWechat,omitempty"`
	PublicPhone     *bool                  `json:"publicPhone,omitempty"`
	PublicWechat    *bool                  `json:"publicWechat,omitempty"`
	BusinessProfile *model.BusinessProfile `json:"businessProfile,omitempty"`
	MapVisible      *bool                  `json:"mapVisible,omitempty"`
	AddressText     string                 `json:"addressText,omitempty"`
	LocationAction  string                 `json:"locationAction,omitempty"`
	MarketName      string                 `json:"marketName,omitempty"`
	BuildingName    string                 `json:"buildingName,omitempty"`
	FloorNo         string                 `json:"floorNo,omitempty"`
	BoothNo         string                 `json:"boothNo,omitempty"`
	Latitude        *float64               `json:"latitude,omitempty"`
	Longitude       *float64               `json:"longitude,omitempty"`
	LocationName    string                 `json:"locationName,omitempty"`
	SmsCode         string                 `json:"smsCode,omitempty"`
}

type UpdateMerchantResp struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
}

type UpdateMerchantLogic struct {
	store UpdateMerchantStore
}

func NewUpdateMerchantLogic(store UpdateMerchantStore, _ ...interface {
	VerifySMSCode(ctx context.Context, phone string, code string) error
}) *UpdateMerchantLogic {
	return &UpdateMerchantLogic{store: store}
}

func (l *UpdateMerchantLogic) UpdateMerchant(ctx context.Context, merchantID string, req UpdateMerchantReq) (UpdateMerchantResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或已停用")
	}
	patch, field, err := buildMerchantProfilePatch(req)
	if err != nil {
		logx.WithContext(ctx).Infow("商家资料校验未通过", logx.Field("operation", "merchant_profile_update"), logx.Field("merchantId", merchantID), logx.Field("stage", "validation"), logx.Field("field", field))
		return UpdateMerchantResp{}, err
	}
	updatedAt, err := l.store.UpdateMerchant(ctx, merchantID, patch)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateMerchantResp{}, errx.New(errx.CodeResourceNotFound, "商家不存在或已停用")
		}
		var validationErr *model.MerchantValidationError
		if errors.As(err, &validationErr) {
			logx.WithContext(ctx).Infow("商家资料合并校验未通过", logx.Field("operation", "merchant_profile_update"), logx.Field("merchantId", merchantID), logx.Field("stage", "merged_validation"))
			return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, validationErr.Message)
		}
		// 数据库根错误以安全分类进入工程日志，前端使用稳定中文提示，避免泄露 SQL 或内部表结构。
		logMerchantFailure(ctx, "merchant_profile_update", merchantID, err)
		return UpdateMerchantResp{}, errx.New(errx.CodeInternalError, "商家资料保存失败，请稍后重试")
	}
	return UpdateMerchantResp{ID: merchantID, UpdatedAt: updatedAt}, nil
}
