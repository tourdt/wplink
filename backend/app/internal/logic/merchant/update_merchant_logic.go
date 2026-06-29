package merchant

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type UpdateMerchantStore interface {
	UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error)
}

type UpdateMerchantReq struct {
	MainCategories []string      `json:"mainCategories,omitempty"`
	MerchantType   string        `json:"merchantType,omitempty"`
	Description    string        `json:"description,omitempty"`
	LogoURL        string        `json:"logoUrl,omitempty"`
	Images         []string      `json:"images,omitempty"`
	ContactName    string        `json:"contactName,omitempty"`
	ContactPhone   string        `json:"contactPhone,omitempty"`
	ContactWechat  string        `json:"contactWechat,omitempty"`
	AddressText    string        `json:"addressText,omitempty"`
	Location       model.JSONMap `json:"location,omitempty"`
	SmsCode        string        `json:"smsCode,omitempty"`
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
	contactPhone := strings.TrimSpace(req.ContactPhone)
	if contactPhone != "" {
		if !isValidContactPhone(contactPhone) {
			return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, "手机号格式不正确")
		}
	}

	updatedAt, err := l.store.UpdateMerchant(ctx, merchantID, model.UpdateMerchantPatch{
		MainCategories: append([]string(nil), req.MainCategories...),
		MerchantType:   strings.TrimSpace(req.MerchantType),
		Description:    strings.TrimSpace(req.Description),
		LogoURL:        strings.TrimSpace(req.LogoURL),
		Images:         append([]string(nil), req.Images...),
		ContactName:    strings.TrimSpace(req.ContactName),
		ContactPhone:   contactPhone,
		ContactWechat:  strings.TrimSpace(req.ContactWechat),
		AddressText:    strings.TrimSpace(req.AddressText),
		Location:       cloneJSONMap(req.Location),
		LocationSet:    req.Location != nil,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateMerchantResp{}, errx.New(errx.CodeResourceNotFound, "商家不存在或已停用")
		}
		return UpdateMerchantResp{}, err
	}
	return UpdateMerchantResp{ID: merchantID, UpdatedAt: updatedAt}, nil
}

func isValidContactPhone(phone string) bool {
	if len(phone) < 6 || len(phone) > 20 {
		return false
	}
	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
