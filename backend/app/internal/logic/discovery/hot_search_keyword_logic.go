package discovery

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type HotSearchKeywordDiscoveryStore interface {
	ListActiveHotSearchKeywords(ctx context.Context, cityCode string) ([]model.HotSearchKeywordConfig, error)
}

type ListHotSearchKeywordsReq struct {
	CityCode string
}

type HotSearchKeywordItem struct {
	Keyword string `json:"keyword"`
}

type ListHotSearchKeywordsResp struct {
	Items []HotSearchKeywordItem `json:"items"`
}

type HotSearchKeywordDiscoveryLogic struct {
	store HotSearchKeywordDiscoveryStore
}

func NewHotSearchKeywordDiscoveryLogic(store HotSearchKeywordDiscoveryStore) *HotSearchKeywordDiscoveryLogic {
	return &HotSearchKeywordDiscoveryLogic{store: store}
}

func (l *HotSearchKeywordDiscoveryLogic) ListHotSearchKeywords(ctx context.Context, req ListHotSearchKeywordsReq) (ListHotSearchKeywordsResp, error) {
	configs, err := l.store.ListActiveHotSearchKeywords(ctx, strings.TrimSpace(req.CityCode))
	if err != nil {
		return ListHotSearchKeywordsResp{}, err
	}
	items := make([]HotSearchKeywordItem, 0, len(configs))
	for _, config := range configs {
		keyword := strings.TrimSpace(config.Keyword)
		if keyword == "" {
			continue
		}
		items = append(items, HotSearchKeywordItem{Keyword: keyword})
	}
	return ListHotSearchKeywordsResp{Items: items}, nil
}
