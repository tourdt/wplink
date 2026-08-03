package resource

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type RelatedResourcesStore interface {
	ListRelatedResources(ctx context.Context, resourceID string, limit int64) ([]model.ResourceListItem, error)
}

type RelatedResourcesReq struct {
	PageSize int64
}

type RelatedResourcesResp struct {
	Items []ResourceListItem `json:"items"`
}

type ListRelatedResourcesLogic struct {
	store RelatedResourcesStore
}

func NewListRelatedResourcesLogic(store RelatedResourcesStore) *ListRelatedResourcesLogic {
	return &ListRelatedResourcesLogic{store: store}
}

func (l *ListRelatedResourcesLogic) ListRelatedResources(ctx context.Context, resourceID string, req RelatedResourcesReq) (RelatedResourcesResp, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return RelatedResourcesResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或暂不可查看")
	}

	items, err := l.store.ListRelatedResources(ctx, resourceID, relatedResourcesPageSize(req.PageSize))
	if err != nil {
		logx.Errorf("加载相关推荐失败: resourceId=%s err=%+v", resourceID, err)
		return RelatedResourcesResp{}, errx.New(errx.CodeInternalError, "相关推荐加载失败，请稍后重试")
	}

	respItems := make([]ResourceListItem, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, resourceListItemFromModel(item))
	}
	return RelatedResourcesResp{Items: respItems}, nil
}

func relatedResourcesPageSize(pageSize int64) int64 {
	if pageSize <= 0 {
		return 3
	}
	if pageSize > 6 {
		return 6
	}
	return pageSize
}
