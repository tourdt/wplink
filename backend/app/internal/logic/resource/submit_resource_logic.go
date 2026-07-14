package resource

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitResourceStore interface {
	SubmitResourceForReview(ctx context.Context, resourceID string) (model.SubmitResourceResult, error)
}

type SubmitResourceResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
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
		if errors.Is(err, model.ErrPublishQuotaInsufficient) {
			logx.Infof("提交资源审核被拦截: resourceId=%s reason=publish_quota_insufficient", resourceID)
			return SubmitResourceResp{}, errx.New(errx.CodeQuotaNotEnough, "本月发布次数已用完，可开通 VIP 或购买发布包")
		}
		if errors.Is(err, model.ErrPublishDisabled) {
			logx.Infof("提交资源审核被拦截: resourceId=%s reason=publish_disabled", resourceID)
			return SubmitResourceResp{}, errx.New(errx.CodeValidationFailed, "该分类暂不开放发布")
		}
		if errors.Is(err, sql.ErrNoRows) {
			logx.Infof("提交资源审核被拦截: resourceId=%s reason=draft_not_editable", resourceID)
			return SubmitResourceResp{}, errx.New(errx.CodeStateConflict, "请先编辑并保存草稿后再提交审核")
		}
		logx.Errorf("提交资源审核失败: resourceId=%s err=%+v", resourceID, err)
		return SubmitResourceResp{}, err
	}
	// 资源提交审核后会进入后台审核队列，记录新状态便于排查用户看到的审核进度。
	logx.Infof("提交资源审核成功: resourceId=%s newStatus=%s", result.ID, result.Status)
	return SubmitResourceResp{
		ID:      result.ID,
		Status:  result.Status,
		Message: "已提交审核，审核通过后将展示给买家",
	}, nil
}
