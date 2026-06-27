package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/webview"
	"wplink/backend/common/errx"
)

type BannerTopicAdminStore interface {
	ListBannerTopics(ctx context.Context, filter model.BannerTopicFilter) ([]model.BannerTopicConfig, error)
	CreateBannerTopic(ctx context.Context, input model.SaveBannerTopicInput) (model.SaveBannerTopicResult, error)
	UpdateBannerTopic(ctx context.Context, configID string, input model.SaveBannerTopicInput) (model.SaveBannerTopicResult, error)
}

type ListBannerTopicsReq struct {
	CityCode string
	Kind     string
	Status   string
}

type SaveBannerTopicReq struct {
	CityCode   string
	Kind       string
	Title      string
	Subtitle   string
	CoverURL   string
	TypeScope  []string
	JumpType   string
	JumpTarget string
	Tags       []string
	StartAt    string
	EndAt      string
	SortOrder  int64
	Status     string
}

type BannerTopicItem struct {
	ID         string
	CityCode   string
	Kind       string
	Title      string
	Subtitle   string
	CoverURL   string
	TypeScope  []string
	JumpType   string
	JumpTarget string
	Tags       []string
	StartAt    string
	EndAt      string
	SortOrder  int64
	Status     string
	UpdatedAt  string
}

type ListBannerTopicsResp struct {
	Items []BannerTopicItem
}

type SaveBannerTopicResp struct {
	ID        string
	UpdatedAt string
}

type BannerTopicAdminLogic struct {
	store BannerTopicAdminStore
}

func NewBannerTopicAdminLogic(store BannerTopicAdminStore) *BannerTopicAdminLogic {
	return &BannerTopicAdminLogic{store: store}
}

func (l *BannerTopicAdminLogic) ListBannerTopics(ctx context.Context, req ListBannerTopicsReq) (ListBannerTopicsResp, error) {
	items, err := l.store.ListBannerTopics(ctx, model.BannerTopicFilter{
		CityCode: strings.TrimSpace(req.CityCode),
		Kind:     strings.TrimSpace(req.Kind),
		Status:   strings.TrimSpace(req.Status),
	})
	if err != nil {
		return ListBannerTopicsResp{}, err
	}
	return ListBannerTopicsResp{Items: mapBannerTopicItems(items)}, nil
}

func (l *BannerTopicAdminLogic) CreateBannerTopic(ctx context.Context, req SaveBannerTopicReq) (SaveBannerTopicResp, error) {
	input, err := normalizeBannerTopicInput(req)
	if err != nil {
		return SaveBannerTopicResp{}, err
	}
	result, err := l.store.CreateBannerTopic(ctx, input)
	if err != nil {
		return SaveBannerTopicResp{}, err
	}
	return SaveBannerTopicResp{ID: result.ID, UpdatedAt: result.UpdatedAt}, nil
}

func (l *BannerTopicAdminLogic) UpdateBannerTopic(ctx context.Context, configID string, req SaveBannerTopicReq) (SaveBannerTopicResp, error) {
	configID = strings.TrimSpace(configID)
	if configID == "" {
		return SaveBannerTopicResp{}, errx.New(errx.CodeValidationFailed, "配置不存在")
	}
	input, err := normalizeBannerTopicInput(req)
	if err != nil {
		return SaveBannerTopicResp{}, err
	}
	result, err := l.store.UpdateBannerTopic(ctx, configID, input)
	if err != nil {
		return SaveBannerTopicResp{}, err
	}
	return SaveBannerTopicResp{ID: result.ID, UpdatedAt: result.UpdatedAt}, nil
}

func normalizeBannerTopicInput(req SaveBannerTopicReq) (model.SaveBannerTopicInput, error) {
	input := model.SaveBannerTopicInput{
		CityCode:   strings.TrimSpace(req.CityCode),
		Kind:       strings.TrimSpace(req.Kind),
		Title:      strings.TrimSpace(req.Title),
		Subtitle:   strings.TrimSpace(req.Subtitle),
		CoverURL:   strings.TrimSpace(req.CoverURL),
		TypeScope:  trimStringSlice(req.TypeScope),
		JumpType:   strings.TrimSpace(req.JumpType),
		JumpTarget: strings.TrimSpace(req.JumpTarget),
		Tags:       trimStringSlice(req.Tags),
		StartAt:    strings.TrimSpace(req.StartAt),
		EndAt:      strings.TrimSpace(req.EndAt),
		SortOrder:  req.SortOrder,
		Status:     strings.TrimSpace(req.Status),
	}
	if input.Kind != "banner" && input.Kind != "topic" {
		return model.SaveBannerTopicInput{}, errx.New(errx.CodeValidationFailed, "配置类型不正确")
	}
	if input.Title == "" {
		return model.SaveBannerTopicInput{}, errx.New(errx.CodeValidationFailed, "请填写标题")
	}
	if input.JumpType != "topic" && input.JumpType != "resource" && input.JumpType != "merchant" && input.JumpType != "demand" && input.JumpType != "internal" && input.JumpType != "webview" {
		return model.SaveBannerTopicInput{}, errx.New(errx.CodeValidationFailed, "跳转类型不正确")
	}
	if input.JumpTarget == "" {
		return model.SaveBannerTopicInput{}, errx.New(errx.CodeValidationFailed, "请填写跳转目标")
	}
	if input.JumpType == "webview" && !webview.IsAllowedURL(input.JumpTarget) {
		return model.SaveBannerTopicInput{}, errx.New(errx.CodeValidationFailed, "活动链接不在允许访问范围内")
	}
	if input.Status == "" {
		input.Status = "draft"
	}
	if input.Status != "draft" && input.Status != "active" && input.Status != "disabled" {
		return model.SaveBannerTopicInput{}, errx.New(errx.CodeValidationFailed, "配置状态不正确")
	}
	return input, nil
}

func mapBannerTopicItems(configs []model.BannerTopicConfig) []BannerTopicItem {
	items := make([]BannerTopicItem, 0, len(configs))
	for _, config := range configs {
		items = append(items, BannerTopicItem{
			ID:         config.ID,
			CityCode:   config.CityCode,
			Kind:       config.Kind,
			Title:      config.Title,
			Subtitle:   config.Subtitle,
			CoverURL:   config.CoverURL,
			TypeScope:  append([]string(nil), config.TypeScope...),
			JumpType:   config.JumpType,
			JumpTarget: config.JumpTarget,
			Tags:       append([]string(nil), config.Tags...),
			StartAt:    config.StartAt,
			EndAt:      config.EndAt,
			SortOrder:  config.SortOrder,
			Status:     config.Status,
			UpdatedAt:  config.UpdatedAt,
		})
	}
	return items
}

func trimStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
