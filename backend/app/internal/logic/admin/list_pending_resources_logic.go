package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type PendingResourceStore interface {
	ListPendingResources(ctx context.Context, filter model.ListPendingResourcesFilter) (model.ListPendingResourcesResult, error)
}

type ListPendingResourcesReq struct {
	CityCode string
	TypeCode string
	Page     int64
	PageSize int64
}

type PendingResourceItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	TypeCode     string `json:"typeCode"`
	MerchantName string `json:"merchantName"`
	CreatedAt    string `json:"createdAt"`
}

type ListPendingResourcesResp struct {
	Items    []PendingResourceItem `json:"items"`
	Page     int64                 `json:"page"`
	PageSize int64                 `json:"pageSize"`
	Total    int64                 `json:"total"`
}

type ListPendingResourcesLogic struct {
	store PendingResourceStore
}

func NewListPendingResourcesLogic(store PendingResourceStore) *ListPendingResourcesLogic {
	return &ListPendingResourcesLogic{store: store}
}

func (l *ListPendingResourcesLogic) ListPendingResources(ctx context.Context, req ListPendingResourcesReq) (ListPendingResourcesResp, error) {
	result, err := l.store.ListPendingResources(ctx, model.ListPendingResourcesFilter{
		CityCode: strings.TrimSpace(req.CityCode),
		TypeCode: strings.TrimSpace(req.TypeCode),
		Status:   model.ResourceStatusPending,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ListPendingResourcesResp{}, err
	}

	items := make([]PendingResourceItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, PendingResourceItem{
			ID:           item.ID,
			Title:        item.Title,
			TypeCode:     item.TypeCode,
			MerchantName: item.MerchantName,
			CreatedAt:    item.CreatedAt,
		})
	}
	return ListPendingResourcesResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}
