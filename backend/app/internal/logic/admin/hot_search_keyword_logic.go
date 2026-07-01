package admin

import (
	"context"
	"strings"
	"unicode/utf8"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type HotSearchKeywordAdminStore interface {
	ListHotSearchKeywords(ctx context.Context, filter model.HotSearchKeywordFilter) ([]model.HotSearchKeywordConfig, error)
	CreateHotSearchKeyword(ctx context.Context, input model.SaveHotSearchKeywordInput) (model.SaveHotSearchKeywordResult, error)
	UpdateHotSearchKeyword(ctx context.Context, configID string, input model.SaveHotSearchKeywordInput) (model.SaveHotSearchKeywordResult, error)
}

type ListHotSearchKeywordsReq struct {
	CityCode string
	Status   string
}

type SaveHotSearchKeywordReq struct {
	CityCode  string
	Keyword   string
	SortOrder int64
	Status    string
	StartAt   string
	EndAt     string
}

type AdminHotSearchKeywordItem struct {
	ID        string `json:"id"`
	CityCode  string `json:"cityCode,omitempty"`
	Keyword   string `json:"keyword"`
	SortOrder int64  `json:"sortOrder"`
	Status    string `json:"status"`
	StartAt   string `json:"startAt,omitempty"`
	EndAt     string `json:"endAt,omitempty"`
	UpdatedAt string `json:"updatedAt"`
}

type ListHotSearchKeywordsResp struct {
	Items []AdminHotSearchKeywordItem `json:"items"`
}

type SaveHotSearchKeywordResp struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
}

type HotSearchKeywordAdminLogic struct {
	store HotSearchKeywordAdminStore
}

func NewHotSearchKeywordAdminLogic(store HotSearchKeywordAdminStore) *HotSearchKeywordAdminLogic {
	return &HotSearchKeywordAdminLogic{store: store}
}

func (l *HotSearchKeywordAdminLogic) ListHotSearchKeywords(ctx context.Context, req ListHotSearchKeywordsReq) (ListHotSearchKeywordsResp, error) {
	items, err := l.store.ListHotSearchKeywords(ctx, model.HotSearchKeywordFilter{
		CityCode: strings.TrimSpace(req.CityCode),
		Status:   strings.TrimSpace(req.Status),
	})
	if err != nil {
		return ListHotSearchKeywordsResp{}, err
	}
	return ListHotSearchKeywordsResp{Items: mapHotSearchKeywordItems(items)}, nil
}

func (l *HotSearchKeywordAdminLogic) CreateHotSearchKeyword(ctx context.Context, req SaveHotSearchKeywordReq) (SaveHotSearchKeywordResp, error) {
	input, err := normalizeHotSearchKeywordInput(req)
	if err != nil {
		return SaveHotSearchKeywordResp{}, err
	}
	result, err := l.store.CreateHotSearchKeyword(ctx, input)
	if err != nil {
		return SaveHotSearchKeywordResp{}, err
	}
	return SaveHotSearchKeywordResp{ID: result.ID, UpdatedAt: result.UpdatedAt}, nil
}

func (l *HotSearchKeywordAdminLogic) UpdateHotSearchKeyword(ctx context.Context, configID string, req SaveHotSearchKeywordReq) (SaveHotSearchKeywordResp, error) {
	configID = strings.TrimSpace(configID)
	if configID == "" {
		return SaveHotSearchKeywordResp{}, errx.New(errx.CodeValidationFailed, "热门搜索词配置不存在")
	}
	input, err := normalizeHotSearchKeywordInput(req)
	if err != nil {
		return SaveHotSearchKeywordResp{}, err
	}
	result, err := l.store.UpdateHotSearchKeyword(ctx, configID, input)
	if err != nil {
		return SaveHotSearchKeywordResp{}, err
	}
	return SaveHotSearchKeywordResp{ID: result.ID, UpdatedAt: result.UpdatedAt}, nil
}

func normalizeHotSearchKeywordInput(req SaveHotSearchKeywordReq) (model.SaveHotSearchKeywordInput, error) {
	input := model.SaveHotSearchKeywordInput{
		CityCode:  strings.TrimSpace(req.CityCode),
		Keyword:   strings.TrimSpace(req.Keyword),
		SortOrder: req.SortOrder,
		Status:    strings.TrimSpace(req.Status),
		StartAt:   strings.TrimSpace(req.StartAt),
		EndAt:     strings.TrimSpace(req.EndAt),
	}
	if input.Keyword == "" {
		return model.SaveHotSearchKeywordInput{}, errx.New(errx.CodeValidationFailed, "请填写热门搜索词")
	}
	if utf8.RuneCountInString(input.Keyword) > 64 {
		return model.SaveHotSearchKeywordInput{}, errx.New(errx.CodeValidationFailed, "热门搜索词不能超过 64 个字")
	}
	if input.Status == "" {
		input.Status = "draft"
	}
	if input.Status != "draft" && input.Status != "active" && input.Status != "disabled" {
		return model.SaveHotSearchKeywordInput{}, errx.New(errx.CodeValidationFailed, "配置状态不正确")
	}
	return input, nil
}

func mapHotSearchKeywordItems(configs []model.HotSearchKeywordConfig) []AdminHotSearchKeywordItem {
	items := make([]AdminHotSearchKeywordItem, 0, len(configs))
	for _, config := range configs {
		items = append(items, AdminHotSearchKeywordItem{
			ID:        config.ID,
			CityCode:  config.CityCode,
			Keyword:   config.Keyword,
			SortOrder: config.SortOrder,
			Status:    config.Status,
			StartAt:   config.StartAt,
			EndAt:     config.EndAt,
			UpdatedAt: config.UpdatedAt,
		})
	}
	return items
}
