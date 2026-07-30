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
	store   SubmitResourceStore
	auditor ContentAuditor
}

func NewSubmitResourceLogic(store SubmitResourceStore, auditors ...ContentAuditor) *SubmitResourceLogic {
	var auditor ContentAuditor
	if len(auditors) > 0 {
		auditor = auditors[0]
	}
	return &SubmitResourceLogic{store: store, auditor: auditor}
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
			return SubmitResourceResp{}, errx.New(errx.CodeQuotaNotEnough, "本月免费发布次数已用完，可购买发布次数后继续发布")
		}
		if errors.Is(err, model.ErrPublishDisabled) {
			logx.Infof("提交资源审核被拦截: resourceId=%s reason=publish_disabled", resourceID)
			return SubmitResourceResp{}, errx.New(errx.CodeValidationFailed, "该分类暂不开放发布")
		}
		if errors.Is(err, sql.ErrNoRows) {
			logx.Infof("提交资源审核被拦截: resourceId=%s reason=draft_not_editable", resourceID)
			return SubmitResourceResp{}, errx.New(errx.CodeStateConflict, "请先编辑并保存草稿后再提交发布")
		}
		logx.Errorf("提交资源审核失败: resourceId=%s err=%+v", resourceID, err)
		return SubmitResourceResp{}, err
	}
	if l.auditor != nil {
		return l.auditSubmittedResource(ctx, result)
	}
	// 未启用自动审核时仅保留 pending 状态；生产环境应开启微信内容安全，避免资源绕过审核。
	logx.Infof("提交资源审核成功: resourceId=%s newStatus=%s", result.ID, result.Status)
	return SubmitResourceResp{
		ID:      result.ID,
		Status:  result.Status,
		Message: "已提交，等待系统安全检测",
	}, nil
}

func (l *SubmitResourceLogic) auditSubmittedResource(ctx context.Context, submitted model.SubmitResourceResult) (SubmitResourceResp, error) {
	auditStore, ok := l.store.(ResourceAutoAuditStore)
	if !ok {
		logx.Errorf("提交资源自动审核缺少状态更新能力: resourceId=%s", submitted.ID)
		return SubmitResourceResp{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
	}
	snapshotStore, ok := l.store.(ResourceAuditSnapshotStore)
	if !ok {
		logx.Errorf("提交资源自动审核缺少资源快照查询能力: resourceId=%s", submitted.ID)
		return SubmitResourceResp{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
	}
	snapshot, err := snapshotStore.GetResourceAuditSnapshot(ctx, submitted.ID)
	if err != nil {
		stateStore, ok := auditStore.(ResourceAuditStateStore)
		if !ok {
			logx.Errorf("提交资源读取审核快照失败且缺少自动重试能力: resourceId=%s err=%+v", submitted.ID, err)
			return SubmitResourceResp{}, errx.New(errx.CodeInternalError, "内容审核失败，请稍后重试")
		}
		if _, retryErr := stateStore.MarkResourceAuditRetry(ctx, submitted.ID, "读取审核快照失败"); retryErr != nil {
			logx.Errorf("提交资源读取审核快照失败后进入重试队列失败: resourceId=%s err=%+v retryErr=%+v", submitted.ID, err, retryErr)
			return SubmitResourceResp{}, errx.New(errx.CodeInternalError, "内容审核失败，请稍后重试")
		}
		recordResourceAuditDecision(ctx, auditStore, model.ResourceAuditDecisionInput{
			ResourceID: submitted.ID,
			Action:     "submit_resource",
			Decision:   "dependency_error",
			Reason:     "读取审核快照失败",
		})
		logx.Errorf("提交资源读取审核快照失败，已进入自动重试: resourceId=%s err=%+v", submitted.ID, err)
		return SubmitResourceResp{ID: submitted.ID, Status: model.ResourceStatusAuditRetry, Message: "内容审核服务暂时不可用，系统将自动重试"}, nil
	}
	if strings.TrimSpace(snapshot.OpenID) == "" {
		stateStore, ok := auditStore.(ResourceAuditStateStore)
		if !ok {
			logx.Errorf("提交资源缺少 openid 且无法进入自动重试: resourceId=%s merchantId=%s typeCode=%s", submitted.ID, snapshot.MerchantID, snapshot.TypeCode)
			return SubmitResourceResp{}, errx.New(errx.CodeInternalError, "内容审核失败，请稍后重试")
		}
		retryCount, err := stateStore.MarkResourceAuditRetry(ctx, submitted.ID, "资源缺少自动安全检测身份")
		if err != nil {
			logx.Errorf("提交资源缺少 openid 后进入自动重试失败: resourceId=%s merchantId=%s typeCode=%s err=%+v", submitted.ID, snapshot.MerchantID, snapshot.TypeCode, err)
			return SubmitResourceResp{}, errx.New(errx.CodeInternalError, "内容审核失败，请稍后重试")
		}
		recordResourceAuditDecision(ctx, auditStore, model.ResourceAuditDecisionInput{
			ResourceID: submitted.ID,
			Action:     "submit_resource",
			Decision:   "dependency_error",
			Reason:     "资源缺少自动安全检测身份",
		})
		logx.Infof("提交资源缺少 openid，已进入自动重试: resourceId=%s merchantId=%s typeCode=%s retryCount=%d", submitted.ID, snapshot.MerchantID, snapshot.TypeCode, retryCount)
		return SubmitResourceResp{ID: submitted.ID, Status: model.ResourceStatusAuditRetry, Message: "自动安全检测暂时不可用，系统将自动重试"}, nil
	}
	auditInput := contentAuditInputFromSnapshot(snapshot)
	result, err := l.auditor.AuditResource(ctx, auditInput)
	outcome, err := applyResourceAutoAuditResult(ctx, auditStore, "submit_resource", submitted.ID, auditInput, result, err)
	if err != nil {
		return SubmitResourceResp{}, err
	}
	return SubmitResourceResp{ID: outcome.ID, Status: outcome.Status, Message: outcome.Message}, nil
}
