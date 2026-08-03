package resource

import (
	"context"
	"strconv"
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
	resourceID, err := NormalizeRelatedResourceID(resourceID)
	if err != nil {
		return RelatedResourcesResp{}, err
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

// NormalizeRelatedResourceID 在任何 bigint 查询前统一校验资源 ID，避免非法路径参数触发数据库类型转换错误。
func NormalizeRelatedResourceID(resourceID string) (string, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return "", errx.New(errx.CodeResourceNotFound, "资源不存在或暂不可查看")
	}
	for index := 0; index < len(resourceID); index++ {
		if resourceID[index] < '0' || resourceID[index] > '9' {
			return "", errx.New(errx.CodeResourceNotFound, "资源不存在或暂不可查看")
		}
	}
	parsed, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil || parsed <= 0 {
		return "", errx.New(errx.CodeResourceNotFound, "资源不存在或暂不可查看")
	}
	return strconv.FormatInt(parsed, 10), nil
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
