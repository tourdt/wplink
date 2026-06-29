package merchant

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type GetMerchantStore interface {
	GetMerchantDetail(ctx context.Context, merchantID string) (model.MerchantDetail, error)
}

type CreditTagInfo struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type MerchantContactInfo struct {
	Name         string `json:"name"`
	PhoneMasked  string `json:"phoneMasked"`
	WechatMasked string `json:"wechatMasked,omitempty"`
}

type MerchantResourcesSummary struct {
	PublishedCount int64 `json:"publishedCount"`
	DealtCount     int64 `json:"dealtCount"`
}

type MerchantDetailResp struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	MerchantType       string                   `json:"merchantType"`
	CityCode           string                   `json:"cityCode"`
	MainCategories     []string                 `json:"mainCategories"`
	VerificationStatus string                   `json:"verificationStatus"`
	CreditTags         []CreditTagInfo          `json:"creditTags"`
	Contact            MerchantContactInfo      `json:"contact"`
	ResourcesSummary   MerchantResourcesSummary `json:"resourcesSummary"`
	AddressText        string                   `json:"addressText,omitempty"`
	Location           model.JSONMap            `json:"location,omitempty"`
	Description        string                   `json:"description,omitempty"`
	LogoURL            string                   `json:"logoUrl,omitempty"`
	Images             []string                 `json:"images,omitempty"`
	LastActiveAt       string                   `json:"lastActiveAt,omitempty"`
}

type GetMerchantLogic struct {
	store GetMerchantStore
}

func NewGetMerchantLogic(store GetMerchantStore) *GetMerchantLogic {
	return &GetMerchantLogic{store: store}
}

func (l *GetMerchantLogic) GetMerchant(ctx context.Context, merchantID string) (MerchantDetailResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return MerchantDetailResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或已停用")
	}

	detail, err := l.store.GetMerchantDetail(ctx, merchantID)
	if err != nil {
		return MerchantDetailResp{}, err
	}

	tags := make([]CreditTagInfo, 0, len(detail.CreditTags))
	for _, tag := range detail.CreditTags {
		tags = append(tags, CreditTagInfo{Code: tag.Code, Label: tag.Label})
	}
	return MerchantDetailResp{
		ID:                 detail.ID,
		Name:               detail.Name,
		MerchantType:       detail.MerchantType,
		CityCode:           detail.CityCode,
		MainCategories:     append([]string(nil), detail.MainCategories...),
		VerificationStatus: detail.VerificationStatus,
		CreditTags:         tags,
		Contact: MerchantContactInfo{
			Name:         detail.ContactName,
			PhoneMasked:  detail.PhoneMasked,
			WechatMasked: detail.WechatMasked,
		},
		ResourcesSummary: MerchantResourcesSummary{
			PublishedCount: detail.PublishedCount,
			DealtCount:     detail.DealtCount,
		},
		AddressText:  detail.AddressText,
		Location:     cloneJSONMap(detail.Location),
		Description:  detail.Description,
		LogoURL:      detail.LogoURL,
		Images:       append([]string(nil), detail.Images...),
		LastActiveAt: detail.LastActiveAt,
	}, nil
}

func cloneJSONMap(value model.JSONMap) model.JSONMap {
	if value == nil {
		return nil
	}
	cloned := make(model.JSONMap, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}
