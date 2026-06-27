package demand

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type MyDemandStore interface {
	ListMyDemands(ctx context.Context, userID string, filter model.ListDemandsFilter) (model.ListDemandsResult, error)
}

type ListMyDemandsReq struct {
	Page     int64
	PageSize int64
}

type DemandListItem struct {
	ID        string
	Title     string
	Status    string
	CreatedAt string
}

type ListMyDemandsResp struct {
	Items    []DemandListItem
	Page     int64
	PageSize int64
	Total    int64
}

type ListMyDemandsLogic struct {
	store MyDemandStore
}

func NewListMyDemandsLogic(store MyDemandStore) *ListMyDemandsLogic {
	return &ListMyDemandsLogic{store: store}
}

func (l *ListMyDemandsLogic) ListMyDemands(ctx context.Context, userID string, req ListMyDemandsReq) (ListMyDemandsResp, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ListMyDemandsResp{}, errx.New(errx.CodeValidationFailed, "请先登录")
	}
	result, err := l.store.ListMyDemands(ctx, userID, model.ListDemandsFilter{Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		return ListMyDemandsResp{}, err
	}
	items := make([]DemandListItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, DemandListItem{ID: item.ID, Title: item.Title, Status: item.Status, CreatedAt: item.CreatedAt})
	}
	return ListMyDemandsResp{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
}
