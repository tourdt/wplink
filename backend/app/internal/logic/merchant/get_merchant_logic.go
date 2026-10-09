package merchant

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type GetMerchantStore interface {
	GetMerchantDetail(ctx context.Context, merchantID string) (model.MerchantDetail, error)
}

type MerchantPermissionStore interface {
	UserCanManageMerchant(ctx context.Context, userID string, merchantID string) (bool, error)
}

type CreditTagInfo struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type MerchantContactInfo struct {
	Name         string `json:"name"`
	Phone        string `json:"phone,omitempty"`
	Wechat       string `json:"wechat,omitempty"`
	PhoneMasked  string `json:"phoneMasked"`
	WechatMasked string `json:"wechatMasked,omitempty"`
}

type MerchantResourcesSummary struct {
	PublishedCount int64 `json:"publishedCount"`
	DealtCount     int64 `json:"dealtCount"`
}

type MerchantLocationInfo struct {
	MarketName   string   `json:"marketName,omitempty"`
	BuildingName string   `json:"buildingName,omitempty"`
	FloorNo      string   `json:"floorNo,omitempty"`
	BoothNo      string   `json:"boothNo,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	LocationName string   `json:"locationName,omitempty"`
	Status       string   `json:"status"`
	UpdatedAt    string   `json:"updatedAt,omitempty"`
	Available    bool     `json:"available"`
}

type MerchantDetailResp struct {
	ID               string                   `json:"id"`
	MerchantNo       string                   `json:"merchantNo"`
	Name             string                   `json:"name"`
	MerchantType     string                   `json:"merchantType"`
	CityCode         string                   `json:"cityCode"`
	MainCategories   []string                 `json:"mainCategories"`
	ProfileStatus    string                   `json:"profileStatus"`
	VIPStatus        string                   `json:"vipStatus"`
	CreditTags       []CreditTagInfo          `json:"creditTags"`
	Contact          *MerchantContactInfo     `json:"contact,omitempty"`
	ResourcesSummary MerchantResourcesSummary `json:"resourcesSummary"`
	HeatScore        int64                    `json:"heatScore"`
	AddressText      string                   `json:"addressText,omitempty"`
	Location         MerchantLocationInfo     `json:"location"`
	Description      string                   `json:"description,omitempty"`
	LogoURL          string                   `json:"logoUrl,omitempty"`
	Images           []string                 `json:"images,omitempty"`
	LastActiveAt     string                   `json:"lastActiveAt,omitempty"`
	OwnerVisible     bool                     `json:"ownerVisible"`
	PublicPhone      bool                     `json:"publicPhone"`
	PublicWechat     bool                     `json:"publicWechat"`
	BusinessProfile  model.BusinessProfile    `json:"businessProfile"`
	MapVisible       bool                     `json:"mapVisible"`
}

type GetMerchantLogic struct {
	store GetMerchantStore
}

func NewGetMerchantLogic(store GetMerchantStore) *GetMerchantLogic {
	return &GetMerchantLogic{store: store}
}

func (l *GetMerchantLogic) GetMerchant(ctx context.Context, merchantID string, viewerUserID ...string) (MerchantDetailResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return MerchantDetailResp{}, errx.New(errx.CodeValidationFailed, "商家不存在或已停用")
	}

	detail, err := l.store.GetMerchantDetail(ctx, merchantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MerchantDetailResp{}, errx.New(errx.CodeMerchantNotFound, "商家不存在或已停用")
		}
		logMerchantFailure(ctx, "merchant_profile_read", merchantID, err)
		return MerchantDetailResp{}, errx.New(errx.CodeInternalError, "商家资料加载失败，请稍后重试")
	}

	tags := make([]CreditTagInfo, 0, len(detail.CreditTags))
	for _, tag := range detail.CreditTags {
		tags = append(tags, CreditTagInfo{Code: tag.Code, Label: tag.Label})
	}
	canManageMerchant, err := canExposeEditableContact(ctx, l.store, merchantID, firstViewerUserID(viewerUserID))
	if err != nil {
		logMerchantFailure(ctx, "merchant_profile_permission", merchantID, err)
		return MerchantDetailResp{}, errx.New(errx.CodeInternalError, "资料加载失败，请稍后重试")
	}
	// 暂停展示后的资料仅管理者可读，避免分享链接绕过目录隐藏。
	if !detail.OwnerVisible && !canManageMerchant {
		return MerchantDetailResp{}, errx.New(errx.CodeMerchantNotFound, "店铺暂不可查看")
	}
	var contact *MerchantContactInfo
	if canManageMerchant {
		// 管理者取得完整联系方式用于编辑；公开读取会在下方按单项授权裁剪。
		contact = &MerchantContactInfo{
			Name:         detail.ContactName,
			Phone:        detail.ContactPhone,
			Wechat:       detail.ContactWechat,
			PhoneMasked:  detail.PhoneMasked,
			WechatMasked: detail.WechatMasked,
		}
	}
	if !canManageMerchant && (detail.PublicPhone || detail.PublicWechat) {
		contact = &MerchantContactInfo{}
		if detail.PublicPhone {
			contact.Phone = detail.ContactPhone
		}
		if detail.PublicWechat {
			contact.Wechat = detail.ContactWechat
		}
	}
	addressText := detail.AddressText
	if !canManageMerchant && !detail.MapVisible {
		addressText = ""
	}
	location := visibleMerchantLocation(detail.Location, canManageMerchant)
	if !canManageMerchant && !detail.MapVisible {
		location = MerchantLocationInfo{Status: model.MerchantLocationStatusHidden, Available: false}
	}
	return MerchantDetailResp{
		ID:             detail.ID,
		MerchantNo:     detail.MerchantNo,
		Name:           detail.Name,
		MerchantType:   detail.MerchantType,
		CityCode:       detail.CityCode,
		MainCategories: append([]string(nil), detail.MainCategories...),
		ProfileStatus:  normalizeMerchantProfileStatus(detail.ProfileStatus),
		VIPStatus:      normalizeMerchantVIPStatus(detail.VIPStatus),
		CreditTags:     tags,
		Contact:        contact,
		ResourcesSummary: MerchantResourcesSummary{
			PublishedCount: detail.PublishedCount,
			DealtCount:     detail.DealtCount,
		},
		HeatScore:       calculateMerchantHeatScore(detail),
		AddressText:     addressText,
		Location:        location,
		Description:     detail.Description,
		LogoURL:         detail.LogoURL,
		Images:          append([]string(nil), detail.Images...),
		LastActiveAt:    detail.LastActiveAt,
		OwnerVisible:    detail.OwnerVisible,
		PublicPhone:     detail.PublicPhone,
		PublicWechat:    detail.PublicWechat,
		BusinessProfile: detail.BusinessProfile,
		MapVisible:      detail.MapVisible,
	}, nil
}

func visibleMerchantLocation(location model.MerchantLocation, canManage bool) MerchantLocationInfo {
	status := strings.TrimSpace(location.Status)
	if status == "" {
		status = model.MerchantLocationStatusHidden
	}
	coordinatesAvailable := location.Latitude != nil && location.Longitude != nil && model.ValidMerchantCoordinates(*location.Latitude, *location.Longitude)
	publiclyVisible := status == model.MerchantLocationStatusVisible && coordinatesAvailable
	managerCanCorrectRisk := canManage && status == model.MerchantLocationStatusRiskHidden && coordinatesAvailable
	if !publiclyVisible && !managerCanCorrectRisk {
		// 公开访问统一收敛为 hidden，既不泄露坐标和档口文字，也不能通过状态推断商家曾被风险隐藏。
		return MerchantLocationInfo{Status: model.MerchantLocationStatusHidden, Available: false}
	}

	return MerchantLocationInfo{
		MarketName:   location.MarketName,
		BuildingName: location.BuildingName,
		FloorNo:      location.FloorNo,
		BoothNo:      location.BoothNo,
		Latitude:     copyFloat64Pointer(location.Latitude),
		Longitude:    copyFloat64Pointer(location.Longitude),
		LocationName: location.LocationName,
		Status:       status,
		UpdatedAt:    location.UpdatedAt,
		Available:    true,
	}
}

func copyFloat64Pointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func firstViewerUserID(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func canExposeEditableContact(ctx context.Context, store GetMerchantStore, merchantID string, userID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, nil
	}
	permissionStore, ok := store.(MerchantPermissionStore)
	if !ok {
		return false, nil
	}
	allowed, err := permissionStore.UserCanManageMerchant(ctx, userID, merchantID)
	if err != nil {
		return false, err
	}
	return allowed, nil
}

func normalizeMerchantVIPStatus(status string) string {
	if strings.TrimSpace(status) == model.VIPStatusActive {
		return model.VIPStatusActive
	}
	return model.VIPStatusNone
}

func normalizeMerchantProfileStatus(status string) string {
	switch strings.TrimSpace(status) {
	case model.MerchantProfileStatusIncomplete:
		return model.MerchantProfileStatusIncomplete
	default:
		return model.MerchantProfileStatusCompleted
	}
}

func calculateMerchantHeatScore(detail model.MerchantDetail) int64 {
	// 商家热度只做 0-100 的展示分，后续调整权重时不影响小程序展示范围。
	score := detail.PublishedCount*8 + int64(len(detail.MainCategories))*2 + int64(len(detail.Images))*3
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
