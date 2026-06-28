package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type SearchLogStore interface {
	ListSearchLogs(ctx context.Context, filter model.SearchLogFilter) (model.ListSearchLogsResult, error)
}

type SearchLogsReq struct {
	CityCode string
	Keyword  string
	Page     int64
	PageSize int64
}

type SearchLogItem struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"userId,omitempty"`
	CityCode    string                 `json:"cityCode,omitempty"`
	CityName    string                 `json:"cityName,omitempty"`
	Keyword     string                 `json:"keyword"`
	Filters     map[string]interface{} `json:"filters"`
	ResultCount int64                  `json:"resultCount"`
	CreatedAt   string                 `json:"createdAt"`
}

type SearchLogsResp struct {
	Items    []SearchLogItem `json:"items"`
	Page     int64           `json:"page"`
	PageSize int64           `json:"pageSize"`
	Total    int64           `json:"total"`
}

type SearchLogLogic struct {
	store SearchLogStore
}

func NewSearchLogLogic(store SearchLogStore) *SearchLogLogic {
	return &SearchLogLogic{store: store}
}

func (l *SearchLogLogic) ListSearchLogs(ctx context.Context, req SearchLogsReq) (SearchLogsResp, error) {
	result, err := l.store.ListSearchLogs(ctx, model.SearchLogFilter{
		CityCode: strings.TrimSpace(req.CityCode),
		Keyword:  strings.TrimSpace(req.Keyword),
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return SearchLogsResp{}, err
	}
	items := make([]SearchLogItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, SearchLogItem{
			ID:          item.ID,
			UserID:      item.UserID,
			CityCode:    item.CityCode,
			CityName:    item.CityName,
			Keyword:     item.Keyword,
			Filters:     item.Filters,
			ResultCount: item.ResultCount,
			CreatedAt:   item.CreatedAt,
		})
	}
	return SearchLogsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}
