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
	if l.auditor != nil {
		return l.auditSubmittedResource(ctx, result)
	}
	// 未启用自动审核时仅保留 pending 状态；生产环境应开启微信内容安全，避免资源绕过审核。
	logx.Infof("提交资源审核成功: resourceId=%s newStatus=%s", result.ID, result.Status)
	return SubmitResourceResp{
		ID:      result.ID,
		Status:  result.Status,
		Message: "已提交审核，审核通过后将展示给买家",
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
		reason := "内容审核服务暂不可用，请稍后重新提交"
		if _, rejectErr := auditStore.RejectResourceAfterAudit(ctx, submitted.ID, reason); rejectErr != nil {
			logx.Errorf("提交资源读取审核快照失败后自动驳回失败: resourceId=%s err=%+v rejectErr=%+v", submitted.ID, err, rejectErr)
			return SubmitResourceResp{}, errx.New(errx.CodeInternalError, "内容审核失败，请稍后重试")
		}
		logx.Errorf("提交资源读取审核快照失败并已驳回: resourceId=%s err=%+v", submitted.ID, err)
		return SubmitResourceResp{ID: submitted.ID, Status: model.ResourceStatusRejected, Message: reason}, nil
	}
	if strings.TrimSpace(snapshot.OpenID) == "" {
		reason := "内容审核需要微信登录信息，请重新登录后编辑提交"
		if _, err := auditStore.RejectResourceAfterAudit(ctx, submitted.ID, reason); err != nil {
			logx.Errorf("提交资源缺少 openid 自动驳回失败: resourceId=%s merchantId=%s typeCode=%s err=%+v", submitted.ID, snapshot.MerchantID, snapshot.TypeCode, err)
			return SubmitResourceResp{}, errx.New(errx.CodeInternalError, "内容审核失败，请稍后重试")
		}
		logx.Infof("提交资源内容审核拒绝发布: resourceId=%s merchantId=%s typeCode=%s reason=empty_openid", submitted.ID, snapshot.MerchantID, snapshot.TypeCode)
		return SubmitResourceResp{ID: submitted.ID, Status: model.ResourceStatusRejected, Message: reason}, nil
	}
	auditInput := contentAuditInputFromSnapshot(snapshot)
	result, err := l.auditor.AuditResource(ctx, auditInput)
	outcome, err := applyResourceAutoAuditResult(ctx, auditStore, "submit_resource", submitted.ID, auditInput, result, err)
	if err != nil {
		return SubmitResourceResp{}, err
	}
	return SubmitResourceResp{ID: outcome.ID, Status: outcome.Status, Message: outcome.Message}, nil
}
