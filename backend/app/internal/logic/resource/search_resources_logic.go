package resource

import (
	"context"

	"wplink/backend/app/internal/model"
)

type SearchResourceStore interface {
	ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error)
	RecordSearchLog(ctx context.Context, input model.SearchLogInput) error
}

type SearchResourcesReq struct {
	UserID       string
	CityCode     string
	TypeCode     string
	Keyword      string
	Category     string
	VerifiedOnly bool
	Page         int64
	PageSize     int64
}

type SearchResourcesLogic struct {
	listLogic *ListResourcesLogic
	store     SearchResourceStore
}

func NewSearchResourcesLogic(store SearchResourceStore) *SearchResourcesLogic {
	return &SearchResourcesLogic{store: store, listLogic: NewListResourcesLogic(store)}
}

func (l *SearchResourcesLogic) SearchResources(ctx context.Context, req SearchResourcesReq) (ListResourcesResp, error) {
	resp, err := l.listLogic.ListResources(ctx, ListResourcesReq{
		CityCode: req.CityCode, TypeCode: req.TypeCode, Keyword: req.Keyword, Category: req.Category,
		VerifiedOnly: req.VerifiedOnly, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return ListResourcesResp{}, err
	}
	err = l.store.RecordSearchLog(ctx, model.SearchLogInput{
		UserID: req.UserID, CityCode: req.CityCode, Keyword: req.Keyword,
		Filters:     model.JSONMap{"typeCode": req.TypeCode, "category": req.Category, "verifiedOnly": req.VerifiedOnly},
		ResultCount: resp.Total,
	})
	if err != nil {
		return ListResourcesResp{}, err
	}
	return resp, nil
}
