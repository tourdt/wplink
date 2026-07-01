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

type MerchantVerificationInfo struct {
	Status       string   `json:"status"`
	Type         string   `json:"type"`
	ReviewedAt   string   `json:"reviewedAt,omitempty"`
	CheckedItems []string `json:"checkedItems"`
}

type MerchantDetailResp struct {
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	MerchantType       string                    `json:"merchantType"`
	CityCode           string                    `json:"cityCode"`
	MainCategories     []string                  `json:"mainCategories"`
	VerificationStatus string                    `json:"verificationStatus"`
	VerificationInfo   *MerchantVerificationInfo `json:"verificationInfo,omitempty"`
	CreditTags         []CreditTagInfo           `json:"creditTags"`
	Contact            MerchantContactInfo       `json:"contact"`
	ResourcesSummary   MerchantResourcesSummary  `json:"resourcesSummary"`
	HeatScore          int64                     `json:"heatScore"`
	AddressText        string                    `json:"addressText,omitempty"`
	Location           model.JSONMap             `json:"location,omitempty"`
	Description        string                    `json:"description,omitempty"`
	LogoURL            string                    `json:"logoUrl,omitempty"`
	Images             []string                  `json:"images,omitempty"`
	LastActiveAt       string                    `json:"lastActiveAt,omitempty"`
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
		VerificationInfo:   buildMerchantVerificationInfo(detail),
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
		HeatScore:    calculateMerchantHeatScore(detail),
		AddressText:  detail.AddressText,
		Location:     cloneJSONMap(detail.Location),
		Description:  detail.Description,
		LogoURL:      detail.LogoURL,
		Images:       append([]string(nil), detail.Images...),
		LastActiveAt: detail.LastActiveAt,
	}, nil
}

func buildMerchantVerificationInfo(detail model.MerchantDetail) *MerchantVerificationInfo {
	if detail.VerificationStatus != "verified" {
		return nil
	}
	// 商家主页只公开核验结论，不透出营业执照、信用代码、联系人等审核材料。
	return &MerchantVerificationInfo{
		Status:       detail.VerificationStatus,
		Type:         detail.MerchantType,
		ReviewedAt:   detail.VerificationReviewedAt,
		CheckedItems: []string{"主体资质", "经营场地"},
	}
}

func calculateMerchantHeatScore(detail model.MerchantDetail) int64 {
	// 商家热度只做 0-100 的展示分，后续调整权重时不影响小程序展示范围。
	score := detail.PublishedCount*8 + int64(len(detail.MainCategories))*2 + int64(len(detail.Images))*3
	if detail.VerificationStatus == "verified" {
		score += 15
	}
	followerScore := detail.FollowerCount * 2
	if followerScore > 20 {
		followerScore = 20
	}
	score += followerScore
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
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
