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
	store       UpdateMerchantStore
	smsVerifier interface {
		VerifySMSCode(ctx context.Context, phone string, code string) error
	}
}

func NewUpdateMerchantLogic(store UpdateMerchantStore, verifier ...interface {
	VerifySMSCode(ctx context.Context, phone string, code string) error
}) *UpdateMerchantLogic {
	var smsVerifier interface {
		VerifySMSCode(ctx context.Context, phone string, code string) error
	}
	if len(verifier) > 0 {
		smsVerifier = verifier[0]
	}
	return &UpdateMerchantLogic{store: store, smsVerifier: smsVerifier}
}

func (l *UpdateMerchantLogic) UpdateMerchant(ctx context.Context, merchantID string, req UpdateMerchantReq) (UpdateMerchantResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或已停用")
	}
	contactPhone := strings.TrimSpace(req.ContactPhone)
	smsCode := strings.TrimSpace(req.SmsCode)
	if contactPhone != "" {
		if smsCode == "" {
			return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, "请填写手机号和短信验证码")
		}
		if len(contactPhone) < 6 || len(contactPhone) > 20 {
			return UpdateMerchantResp{}, errx.New(errx.CodeValidationFailed, "手机号格式不正确")
		}
		if l.smsVerifier == nil {
			return UpdateMerchantResp{}, errx.New(errx.CodeInternalError, "短信服务未配置，请稍后重试")
		}
		// 商家主页联系电话会对买家展示和联系行为产生影响，保存前必须证明当前操作者能接收该手机号验证码。
		if err := l.smsVerifier.VerifySMSCode(ctx, contactPhone, smsCode); err != nil {
			return UpdateMerchantResp{}, err
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
