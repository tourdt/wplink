package city

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ResourceTypeStore interface {
	ListActiveResourceTypesByCityCode(ctx context.Context, cityCode string, direction string) ([]model.ResourceTypeConfig, error)
}

type ListResourceTypesReq struct {
	CityCode  string
	Direction string
}

type ResourceTypeConfigInfo struct {
	ID               string                 `json:"id"`
	TypeCode         string                 `json:"typeCode"`
	TypeName         string                 `json:"typeName"`
	Direction        string                 `json:"direction"`
	DefaultValidDays int64                  `json:"defaultValidDays"`
	FieldSchema      map[string]interface{} `json:"fieldSchema"`
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

func (l *ListResourceTypesLogic) ListResourceTypes(ctx context.Context, req ListResourceTypesReq) (ListResourceTypesResp, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	if cityCode == "" {
		return ListResourceTypesResp{}, errx.New(errx.CodeValidationFailed, "请选择城市站")
	}
	direction, err := normalizeOptionalResourceDirection(req.Direction)
	if err != nil {
		return ListResourceTypesResp{}, err
	}

	configs, err := l.store.ListActiveResourceTypesByCityCode(ctx, cityCode, direction)
	if err != nil {
		return ListResourceTypesResp{}, err
	}

	items := make([]ResourceTypeConfigInfo, 0, len(configs))
	for _, config := range configs {
		items = append(items, ResourceTypeConfigInfo{
			ID:               config.ID,
			TypeCode:         config.TypeCode,
			TypeName:         config.TypeName,
			Direction:        config.Direction,
			DefaultValidDays: config.DefaultValidDays,
			FieldSchema:      map[string]interface{}(config.FieldSchema),
			RequiredFields:   append([]string(nil), config.RequiredFields...),
			FilterFields:     append([]string(nil), config.FilterFields...),
			DisplayTemplate:  map[string]interface{}(config.DisplayTemplate),
		})
	}
	return ListResourceTypesResp{Items: items}, nil
}

func normalizeOptionalResourceDirection(direction string) (string, error) {
	direction = strings.TrimSpace(direction)
	if direction == "" {
		return "", nil
	}
	if direction == model.ResourceDirectionSupply || direction == model.ResourceDirectionDemand {
		return direction, nil
	}
	return "", errx.New(errx.CodeValidationFailed, "请选择正确的供需方向")
}
