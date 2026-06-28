package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

type ReviewResourceStore interface {
	ReviewResource(ctx context.Context, resourceID string, input model.ReviewResourceInput) (model.ReviewResourceResult, error)
}

type ReviewResourceReq struct {
	Action     string
	Reason     string
	ReviewerID string
}

type ReviewResourceResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ReviewResourceLogic struct {
	store ReviewResourceStore
}

func NewReviewResourceLogic(store ReviewResourceStore) *ReviewResourceLogic {
	return &ReviewResourceLogic{store: store}
}

func (l *ReviewResourceLogic) ReviewResource(ctx context.Context, resourceID string, req ReviewResourceReq) (ReviewResourceResp, error) {
	resourceID = strings.TrimSpace(resourceID)
	action := strings.TrimSpace(req.Action)
	reason := strings.TrimSpace(req.Reason)
	if resourceID == "" {
		return ReviewResourceResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	if action != "approve" && action != "reject" && action != "take_down" {
		return ReviewResourceResp{}, errx.New(errx.CodeValidationFailed, "审核动作不正确")
	}
	if (action == "reject" || action == "take_down") && reason == "" {
		return ReviewResourceResp{}, errx.New(errx.CodeValidationFailed, "请填写处理原因")
	}

	result, err := l.store.ReviewResource(ctx, resourceID, model.ReviewResourceInput{
		Action:     action,
		Reason:     reason,
		ReviewerID: strings.TrimSpace(req.ReviewerID),
	})
	if err != nil {
		return ReviewResourceResp{}, err
	}
	return ReviewResourceResp{ID: result.ID, Status: result.Status, Message: reviewMessage(action)}, nil
}

func reviewMessage(action string) string {
	switch action {
	case "approve":
		return "资源已审核通过"
	case "reject":
		return "资源已驳回"
	default:
		return "资源已下架"
	}
}
