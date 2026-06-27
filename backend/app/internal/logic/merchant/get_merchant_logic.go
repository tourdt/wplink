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
	Code  string
	Label string
}

type MerchantContactInfo struct {
	Name         string
	PhoneMasked  string
	WechatMasked string
}

type MerchantResourcesSummary struct {
	PublishedCount int64
	DealtCount     int64
}

type MerchantDetailResp struct {
	ID                 string
	Name               string
	MerchantType       string
	CityCode           string
	MainCategories     []string
	VerificationStatus string
	CreditTags         []CreditTagInfo
	Contact            MerchantContactInfo
	ResourcesSummary   MerchantResourcesSummary
	Description        string
	Images             []string
	LastActiveAt       string
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
		Description:  detail.Description,
		Images:       append([]string(nil), detail.Images...),
		LastActiveAt: detail.LastActiveAt,
	}, nil
}
