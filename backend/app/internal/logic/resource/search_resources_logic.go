package resource

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchResourceStore interface {
	ListResources(ctx context.Context, filter model.ListResourcesFilter) (model.ListResourcesResult, error)
	RecordSearchLog(ctx context.Context, input model.SearchLogInput) error
}

type SearchResourcesReq struct {
	UserID       string
	CityCode     string
	GroupCode    string
	TypeCode     string
	Direction    string
	Keyword      string
	Category     string
	Tags         []string
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
		CityCode: req.CityCode, GroupCode: req.GroupCode, TypeCode: req.TypeCode, Keyword: req.Keyword, Category: req.Category,
		Direction: req.Direction, Tags: req.Tags, VerifiedOnly: req.VerifiedOnly, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return ListResourcesResp{}, err
	}
	err = l.store.RecordSearchLog(ctx, model.SearchLogInput{
		UserID: req.UserID, CityCode: req.CityCode, Keyword: req.Keyword,
		Filters:     model.JSONMap{"groupCode": strings.TrimSpace(req.GroupCode), "typeCode": req.TypeCode, "direction": strings.TrimSpace(req.Direction), "category": req.Category, "tags": append([]string(nil), req.Tags...), "verifiedOnly": req.VerifiedOnly},
		ResultCount: resp.Total,
	})
	if err != nil {
		logx.Errorf("记录搜索日志失败: userId=%s cityCode=%s keyword=%s err=%+v", strings.TrimSpace(req.UserID), strings.TrimSpace(req.CityCode), strings.TrimSpace(req.Keyword), err)
	}
	return resp, nil
}
