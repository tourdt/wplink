package resource

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type SubmitResourceStore interface {
	SubmitResourceForReview(ctx context.Context, resourceID string) (model.SubmitResourceResult, error)
}

type SubmitResourceResp struct {
	ID      string
	Status  string
	Message string
}

type SubmitResourceLogic struct {
	store SubmitResourceStore
}

func NewSubmitResourceLogic(store SubmitResourceStore) *SubmitResourceLogic {
	return &SubmitResourceLogic{store: store}
}

func (l *SubmitResourceLogic) SubmitResource(ctx context.Context, resourceID string) (SubmitResourceResp, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return SubmitResourceResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}

	result, err := l.store.SubmitResourceForReview(ctx, resourceID)
	if err != nil {
		return SubmitResourceResp{}, err
	}
	return SubmitResourceResp{
		ID:      result.ID,
		Status:  result.Status,
		Message: "已提交审核，审核通过后将展示给买家",
	}, nil
}
