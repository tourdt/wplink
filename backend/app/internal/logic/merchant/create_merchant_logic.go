package merchant

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type CreateMerchantStore interface {
	CreateMerchant(ctx context.Context, input model.CreateMerchantInput) (model.CreateMerchantResult, error)
}

type CreateMerchantReq struct {
	CityCode       string
	Name           string
	MerchantType   string
	MainCategories []string
	ContactName    string
	ContactPhone   string
	ContactWechat  string
	AddressText    string
	Description    string
}

type CreateMerchantResp struct {
	ID                 string
	Name               string
	VerificationStatus string
	Status             string
}

type CreateMerchantLogic struct {
	store CreateMerchantStore
}

func NewCreateMerchantLogic(store CreateMerchantStore) *CreateMerchantLogic {
	return &CreateMerchantLogic{store: store}
}

func (l *CreateMerchantLogic) CreateMerchant(ctx context.Context, req CreateMerchantReq) (CreateMerchantResp, error) {
	input := model.CreateMerchantInput{
		CityCode:       strings.TrimSpace(req.CityCode),
		Name:           strings.TrimSpace(req.Name),
		MerchantType:   strings.TrimSpace(req.MerchantType),
		MainCategories: append([]string(nil), req.MainCategories...),
		ContactName:    strings.TrimSpace(req.ContactName),
		ContactPhone:   strings.TrimSpace(req.ContactPhone),
		ContactWechat:  strings.TrimSpace(req.ContactWechat),
		AddressText:    strings.TrimSpace(req.AddressText),
		Description:    strings.TrimSpace(req.Description),
	}
	if input.CityCode == "" || input.Name == "" || input.MerchantType == "" || len(input.MainCategories) == 0 || input.ContactName == "" || input.ContactPhone == "" {
		return CreateMerchantResp{}, errx.New(errx.CodeValidationFailed, "请补充商家名称、类型、主营品类和联系方式")
	}

	result, err := l.store.CreateMerchant(ctx, input)
	if err != nil {
		return CreateMerchantResp{}, err
	}
	return CreateMerchantResp{
		ID:                 result.ID,
		Name:               result.Name,
		VerificationStatus: result.VerificationStatus,
		Status:             result.Status,
	}, nil
}
