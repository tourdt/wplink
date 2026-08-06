package admin

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReviewResourceStore interface {
	ReviewResource(ctx context.Context, resourceID string, input model.ReviewResourceInput) (model.ReviewResourceResult, error)
}

type ResourceReviewGrowthStore interface {
	TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error)
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
		LogAdminFailure(ctx, "管理员审核资源失败", "review_resource", err,
			logx.Field("resourceId", resourceID), logx.Field("operatorId", strings.TrimSpace(req.ReviewerID)), logx.Field("action", action))
		return ReviewResourceResp{}, err
	}
	// 资源审核会改变前台可见性，记录状态流转方便上线后追溯误审、下架和驳回问题。
	logx.Infof("管理员审核资源成功: resourceId=%s reviewerId=%s action=%s newStatus=%s", result.ID, strings.TrimSpace(req.ReviewerID), action, result.Status)
	l.triggerResourceApprovedGrowthReward(ctx, result, strings.TrimSpace(req.ReviewerID), action)
	return ReviewResourceResp{ID: result.ID, Status: result.Status, Message: reviewMessage(action)}, nil
}

func (l *ReviewResourceLogic) triggerResourceApprovedGrowthReward(ctx context.Context, result model.ReviewResourceResult, reviewerID string, action string) {
	if action != "approve" || result.Status != model.ResourceStatusPublished {
		return
	}
	growthStore, ok := l.store.(ResourceReviewGrowthStore)
	if !ok {
		return
	}
	for _, eventType := range []string{model.GrowthEventResourceFirstApproved, model.GrowthEventResourceApprovedCountReached} {
		// 审核通过后的成长奖励是运营激励，失败不能改变审核结果，只记录日志待后台排查或补发。
		if _, err := growthStore.TriggerGrowthEvent(ctx, model.GrowthEventInput{
			EventType:  eventType,
			ResourceID: result.ID,
		}); err != nil {
			LogAdminFailure(ctx, "资源审核通过后触发成长权益失败", "grant_growth_reward_after_resource_review", err,
				logx.Field("resourceId", result.ID), logx.Field("operatorId", reviewerID), logx.Field("eventType", eventType))
		}
	}
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
