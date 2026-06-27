package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ResourceTypeConfigStore interface {
	ListResourceTypeConfigs(ctx context.Context, cityCode string, status string) ([]model.AdminResourceTypeConfig, error)
	UpdateResourceTypeConfig(ctx context.Context, configID string, patch model.ResourceTypeConfigPatch) (string, error)
}

type ListResourceTypeConfigsReq struct {
	CityCode string
	Status   string
}

type ResourceTypeConfigItem struct {
	ID               string                 `json:"id"`
	CityCode         string                 `json:"cityCode,omitempty"`
	TypeCode         string                 `json:"typeCode"`
	TypeName         string                 `json:"typeName"`
	FieldSchema      map[string]interface{} `json:"fieldSchema"`
	RequiredFields   []string               `json:"requiredFields"`
	FilterFields     []string               `json:"filterFields"`
	DisplayTemplate  map[string]interface{} `json:"displayTemplate"`
	ReviewRules      map[string]interface{} `json:"reviewRules"`
	SortWeights      map[string]interface{} `json:"sortWeights"`
	MessageRules     map[string]interface{} `json:"messageRules"`
	DefaultValidDays int64                  `json:"defaultValidDays"`
	Status           string                 `json:"status"`
}

type ListResourceTypeConfigsResp struct {
	Items []ResourceTypeConfigItem `json:"items"`
}

type UpdateResourceTypeConfigReq struct {
	FieldSchema      map[string]interface{}
	RequiredFields   []string
	FilterFields     []string
	DisplayTemplate  map[string]interface{}
	ReviewRules      map[string]interface{}
	SortWeights      map[string]interface{}
	MessageRules     map[string]interface{}
	DefaultValidDays int64
	Status           string
}

type UpdateResourceTypeConfigResp struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
}

type ResourceTypeConfigLogic struct {
	store ResourceTypeConfigStore
}

func NewResourceTypeConfigLogic(store ResourceTypeConfigStore) *ResourceTypeConfigLogic {
	return &ResourceTypeConfigLogic{store: store}
}

func (l *ResourceTypeConfigLogic) ListResourceTypeConfigs(ctx context.Context, req ListResourceTypeConfigsReq) (ListResourceTypeConfigsResp, error) {
	configs, err := l.store.ListResourceTypeConfigs(ctx, strings.TrimSpace(req.CityCode), strings.TrimSpace(req.Status))
	if err != nil {
		return ListResourceTypeConfigsResp{}, err
	}

	items := make([]ResourceTypeConfigItem, 0, len(configs))
	for _, config := range configs {
		items = append(items, ResourceTypeConfigItem{
			ID:               config.ID,
			CityCode:         config.CityCode,
			TypeCode:         config.TypeCode,
			TypeName:         config.TypeName,
			FieldSchema:      map[string]interface{}(config.FieldSchema),
			RequiredFields:   append([]string(nil), config.RequiredFields...),
			FilterFields:     append([]string(nil), config.FilterFields...),
			DisplayTemplate:  map[string]interface{}(config.DisplayTemplate),
			ReviewRules:      map[string]interface{}(config.ReviewRules),
			SortWeights:      map[string]interface{}(config.SortWeights),
			MessageRules:     map[string]interface{}(config.MessageRules),
			DefaultValidDays: config.DefaultValidDays,
			Status:           config.Status,
		})
	}
	return ListResourceTypeConfigsResp{Items: items}, nil
}

func (l *ResourceTypeConfigLogic) UpdateResourceTypeConfig(ctx context.Context, configID string, req UpdateResourceTypeConfigReq) (UpdateResourceTypeConfigResp, error) {
	configID = strings.TrimSpace(configID)
	if configID == "" {
		return UpdateResourceTypeConfigResp{}, errx.New(errx.CodeValidationFailed, "资源类型配置不存在")
	}
	if req.DefaultValidDays <= 0 {
		return UpdateResourceTypeConfigResp{}, errx.New(errx.CodeValidationFailed, "默认有效期必须大于 0")
	}
	if req.Status != "active" && req.Status != "disabled" {
		return UpdateResourceTypeConfigResp{}, errx.New(errx.CodeValidationFailed, "资源类型状态不正确")
	}

	updatedAt, err := l.store.UpdateResourceTypeConfig(ctx, configID, model.ResourceTypeConfigPatch{
		FieldSchema:      model.JSONMap(req.FieldSchema),
		RequiredFields:   append([]string(nil), req.RequiredFields...),
		FilterFields:     append([]string(nil), req.FilterFields...),
		DisplayTemplate:  model.JSONMap(req.DisplayTemplate),
		ReviewRules:      model.JSONMap(req.ReviewRules),
		SortWeights:      model.JSONMap(req.SortWeights),
		MessageRules:     model.JSONMap(req.MessageRules),
		DefaultValidDays: req.DefaultValidDays,
		Status:           req.Status,
	})
	if err != nil {
		return UpdateResourceTypeConfigResp{}, err
	}
	return UpdateResourceTypeConfigResp{ID: configID, UpdatedAt: updatedAt}, nil
}
