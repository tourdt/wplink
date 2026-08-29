package merchant

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	merchantDirectoryCityCode        = "zhili"
	merchantDirectoryDefaultPageSize = int64(20)
	merchantDirectoryMaxPageSize     = int64(100)
	merchantLocationRadiusMeters     = int64(1000)
	merchantLocationNearbyLimit      = int64(20)
)

type MerchantDirectoryStore interface {
	ListMerchantDirectory(ctx context.Context, filter model.MerchantDirectoryFilter) ([]model.MerchantDirectoryItem, error)
	CountMerchantDirectory(ctx context.Context, filter model.MerchantDirectoryFilter) (int64, error)
	GetVisibleMerchantLocation(ctx context.Context, merchantID string) (model.MerchantDirectoryItem, error)
	ListNearbyVisibleMerchants(ctx context.Context, origin model.MerchantDirectoryItem, radiusMeters int64, limit int64) ([]model.MerchantDirectoryItem, error)
}

type ListMerchantDirectoryReq struct {
	CityCode   string
	Keyword    string
	Categories string
	Page       int64
	PageSize   int64
}

type MerchantDirectoryPublicItem struct {
	MerchantID           string   `json:"merchantId"`
	MerchantNo           string   `json:"merchantNo"`
	Name                 string   `json:"name"`
	MerchantType         string   `json:"merchantType"`
	MainCategories       []string `json:"mainCategories"`
	LogoURL              string   `json:"logoUrl,omitempty"`
	AddressText          string   `json:"addressText,omitempty"`
	MarketName           string   `json:"marketName,omitempty"`
	BuildingName         string   `json:"buildingName,omitempty"`
	FloorNo              string   `json:"floorNo,omitempty"`
	BoothNo              string   `json:"boothNo,omitempty"`
	Latitude             float64  `json:"latitude"`
	Longitude            float64  `json:"longitude"`
	LocationName         string   `json:"locationName,omitempty"`
	LastResourceActiveAt string   `json:"lastResourceActiveAt,omitempty"`
	DistanceMeters       float64  `json:"distanceMeters,omitempty"`
}

type ListMerchantDirectoryResp struct {
	Items    []MerchantDirectoryPublicItem `json:"items"`
	Total    int64                         `json:"total"`
	Page     int64                         `json:"page"`
	PageSize int64                         `json:"pageSize"`
}

type MerchantLocationContextResp struct {
	Current         MerchantDirectoryPublicItem   `json:"current"`
	Nearby          []MerchantDirectoryPublicItem `json:"nearby"`
	RadiusMeters    int64                         `json:"radiusMeters"`
	NearbyAvailable bool                          `json:"nearbyAvailable"`
}

type MerchantDirectoryLogic struct {
	store MerchantDirectoryStore
}

func NewMerchantDirectoryLogic(store MerchantDirectoryStore) *MerchantDirectoryLogic {
	return &MerchantDirectoryLogic{store: store}
}

func (l *MerchantDirectoryLogic) ListDirectory(ctx context.Context, req ListMerchantDirectoryReq) (ListMerchantDirectoryResp, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	if cityCode == "" {
		cityCode = merchantDirectoryCityCode
	}
	if cityCode != merchantDirectoryCityCode {
		return ListMerchantDirectoryResp{}, errx.New(errx.CodeValidationFailed, "当前仅支持织里商家目录")
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = merchantDirectoryDefaultPageSize
	}
	if pageSize > merchantDirectoryMaxPageSize {
		return ListMerchantDirectoryResp{}, errx.New(errx.CodeValidationFailed, "每页商家数量不能超过 100 条")
	}
	filter := model.MerchantDirectoryFilter{
		CityCode:   cityCode,
		Keyword:    strings.TrimSpace(req.Keyword),
		Categories: splitDirectoryCategories(req.Categories),
		Page:       page,
		PageSize:   pageSize,
	}
	items, err := l.store.ListMerchantDirectory(ctx, filter)
	if err != nil {
		logx.WithContext(ctx).Errorw("查询商家目录失败",
			logx.Field("operation", "list_merchant_directory"),
			logx.Field("page", page),
			logx.Field("pageSize", pageSize),
			logx.Field("err", err))
		return ListMerchantDirectoryResp{}, errx.New(errx.CodeInternalError, "商家列表加载失败，请稍后重试")
	}
	total, err := l.store.CountMerchantDirectory(ctx, filter)
	if err != nil {
		logx.WithContext(ctx).Errorw("统计商家目录失败",
			logx.Field("operation", "count_merchant_directory"),
			logx.Field("page", page),
			logx.Field("pageSize", pageSize),
			logx.Field("err", err))
		return ListMerchantDirectoryResp{}, errx.New(errx.CodeInternalError, "商家列表加载失败，请稍后重试")
	}
	return ListMerchantDirectoryResp{
		Items:    mapMerchantDirectoryPublicItems(items),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (l *MerchantDirectoryLogic) GetLocationContext(ctx context.Context, merchantID string) (MerchantLocationContextResp, error) {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return MerchantLocationContextResp{}, errx.New(errx.CodeValidationFailed, "该商家位置待完善")
	}
	current, err := l.store.GetVisibleMerchantLocation(ctx, merchantID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logx.WithContext(ctx).Errorw("查询当前商家公开位置失败",
				logx.Field("operation", "get_visible_merchant_location"),
				logx.Field("merchantId", merchantID),
				logx.Field("err", err))
			return MerchantLocationContextResp{}, errx.New(errx.CodeInternalError, "该商家位置待完善")
		}
		return MerchantLocationContextResp{}, errx.New(errx.CodeResourceNotFound, "该商家位置待完善")
	}
	if !validDirectoryItemCoordinates(current) {
		return MerchantLocationContextResp{}, errx.New(errx.CodeResourceNotFound, "该商家位置待完善")
	}

	currentItem := mapMerchantDirectoryPublicItem(current)
	nearby, err := l.store.ListNearbyVisibleMerchants(ctx, current, merchantLocationRadiusMeters, merchantLocationNearbyLimit)
	if err != nil {
		// 附近推荐不是导航前置条件；数据库异常时保留当前商家，避免扩大故障影响面。
		logx.WithContext(ctx).Errorw("查询附近公开商家失败",
			logx.Field("operation", "list_nearby_visible_merchants"),
			logx.Field("merchantId", merchantID),
			logx.Field("radiusMeters", merchantLocationRadiusMeters),
			logx.Field("limit", merchantLocationNearbyLimit),
			logx.Field("err", err))
		return MerchantLocationContextResp{
			Current:         currentItem,
			Nearby:          []MerchantDirectoryPublicItem{},
			RadiusMeters:    merchantLocationRadiusMeters,
			NearbyAvailable: false,
		}, nil
	}
	return MerchantLocationContextResp{
		Current:         currentItem,
		Nearby:          mapMerchantDirectoryPublicItems(nearby),
		RadiusMeters:    merchantLocationRadiusMeters,
		NearbyAvailable: true,
	}, nil
}

func splitDirectoryCategories(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		items = append(items, part)
	}
	return items
}

func mapMerchantDirectoryPublicItems(items []model.MerchantDirectoryItem) []MerchantDirectoryPublicItem {
	result := make([]MerchantDirectoryPublicItem, 0, len(items))
	for _, item := range items {
		result = append(result, mapMerchantDirectoryPublicItem(item))
	}
	return result
}

func mapMerchantDirectoryPublicItem(item model.MerchantDirectoryItem) MerchantDirectoryPublicItem {
	lastResourceActiveAt := ""
	if !item.LastResourceActiveAt.IsZero() {
		lastResourceActiveAt = item.LastResourceActiveAt.Format(timeRFC3339)
	}
	return MerchantDirectoryPublicItem{
		MerchantID:           item.MerchantID,
		MerchantNo:           item.MerchantNo,
		Name:                 item.Name,
		MerchantType:         item.MerchantType,
		MainCategories:       append([]string{}, item.MainCategories...),
		LogoURL:              item.LogoURL,
		AddressText:          item.AddressText,
		MarketName:           item.MarketName,
		BuildingName:         item.BuildingName,
		FloorNo:              item.FloorNo,
		BoothNo:              item.BoothNo,
		Latitude:             item.Latitude,
		Longitude:            item.Longitude,
		LocationName:         item.LocationName,
		LastResourceActiveAt: lastResourceActiveAt,
		DistanceMeters:       item.DistanceMeters,
	}
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func validDirectoryItemCoordinates(item model.MerchantDirectoryItem) bool {
	return !math.IsNaN(item.Latitude) && !math.IsInf(item.Latitude, 0) && item.Latitude >= -90 && item.Latitude <= 90 &&
		!math.IsNaN(item.Longitude) && !math.IsInf(item.Longitude, 0) && item.Longitude >= -180 && item.Longitude <= 180
}
