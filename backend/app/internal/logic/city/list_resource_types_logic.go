package city

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ResourceTypeStore interface {
	ListActiveResourceTypesByCityCode(ctx context.Context, cityCode string) ([]model.ResourceTypeConfig, error)
}

type ResourceTypeConfigInfo struct {
	ID               string                 `json:"id"`
	TypeCode         string                 `json:"typeCode"`
	TypeName         string                 `json:"typeName"`
	DefaultValidDays int64                  `json:"defaultValidDays"`
	RequiredFields   []string               `json:"requiredFields"`
	FilterFields     []string               `json:"filterFields"`
	DisplayTemplate  map[string]interface{} `json:"displayTemplate"`
}

type ListResourceTypesResp struct {
	Items []ResourceTypeConfigInfo `json:"items"`
}

type ListResourceTypesLogic struct {
	store ResourceTypeStore
}

func NewListResourceTypesLogic(store ResourceTypeStore) *ListResourceTypesLogic {
	return &ListResourceTypesLogic{store: store}
}

func (l *ListResourceTypesLogic) ListResourceTypes(ctx context.Context, cityCode string) (ListResourceTypesResp, error) {
	cityCode = strings.TrimSpace(cityCode)
	if cityCode == "" {
		return ListResourceTypesResp{}, errx.New(errx.CodeValidationFailed, "请选择城市站")
	}

	configs, err := l.store.ListActiveResourceTypesByCityCode(ctx, cityCode)
	if err != nil {
		return ListResourceTypesResp{}, err
	}

	items := make([]ResourceTypeConfigInfo, 0, len(configs))
	for _, config := range configs {
		items = append(items, ResourceTypeConfigInfo{
			ID:               config.ID,
			TypeCode:         config.TypeCode,
			TypeName:         config.TypeName,
			DefaultValidDays: config.DefaultValidDays,
			RequiredFields:   append([]string(nil), config.RequiredFields...),
			FilterFields:     append([]string(nil), config.FilterFields...),
			DisplayTemplate:  map[string]interface{}(config.DisplayTemplate),
		})
	}
	return ListResourceTypesResp{Items: items}, nil
}
