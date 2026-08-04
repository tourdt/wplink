package contentaudit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	resourcelogic "wplink/backend/app/internal/logic/resource"
	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	resourceAuditTaskStatusPass     = "pass"
	resourceAuditTaskStatusRejected = "rejected"
	resourceAuditTaskStatusFailed   = "failed"
)

type MediaCheckCallbackStore interface {
	CompleteResourceContentAuditTask(ctx context.Context, input model.ResourceContentAuditTaskResultInput) (model.ResourceContentAuditTaskCompletion, error)
	PublishResourceAfterMediaAudit(ctx context.Context, resourceID string, traceID string) (model.ReviewResourceResult, error)
	RejectResourceAfterMediaAudit(ctx context.Context, resourceID string, traceID string, reason string) (model.ReviewResourceResult, error)
	MarkResourceAuditRetryAfterMediaAudit(ctx context.Context, resourceID string, traceID string, reason string) (int64, error)
}

type MediaCheckCallbackResp struct {
	ResourceID string `json:"resourceId,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type MediaCheckCallbackLogic struct {
	store MediaCheckCallbackStore
}

func NewMediaCheckCallbackLogic(store MediaCheckCallbackStore) *MediaCheckCallbackLogic {
	return &MediaCheckCallbackLogic{store: store}
}

func (l *MediaCheckCallbackLogic) Handle(ctx context.Context, payload model.JSONMap) (MediaCheckCallbackResp, error) {
	traceID := trimPayloadString(payload, "trace_id", "traceId")
	if traceID == "" {
		return MediaCheckCallbackResp{}, errx.New(errx.CodeValidationFailed, "图片审核回调缺少 trace_id")
	}
	input := mediaAuditTaskResultInput(traceID, payload)
	completion, err := l.store.CompleteResourceContentAuditTask(ctx, input)
	if errors.Is(err, sql.ErrNoRows) {
		logx.Infof("图片审核回调未匹配任务: traceId=%s", traceID)
		return MediaCheckCallbackResp{}, errx.New(errx.CodeStateConflict, "图片审核任务不存在")
	}
	if err != nil {
		logx.Errorf("记录图片审核回调失败: traceId=%s err=%+v", traceID, err)
		return MediaCheckCallbackResp{}, err
	}

	if input.Status == resourceAuditTaskStatusFailed {
		retryCount, retryErr := l.store.MarkResourceAuditRetryAfterMediaAudit(ctx, completion.ResourceID, traceID, input.Reason)
		if errors.Is(retryErr, sql.ErrNoRows) {
			logx.Infof("图片审核依赖失败但资源状态已流转: resourceId=%s traceId=%s", completion.ResourceID, traceID)
			return MediaCheckCallbackResp{ResourceID: completion.ResourceID, Status: "unchanged", Message: "图片审核回调已记录"}, nil
		}
		if retryErr != nil {
			logx.Errorf("图片审核依赖失败后进入重试队列失败: resourceId=%s traceId=%s err=%+v", completion.ResourceID, traceID, retryErr)
			return MediaCheckCallbackResp{}, errx.New(errx.CodeInternalError, "图片审核处理失败，请稍后重试")
		}
		l.recordAuditDecision(ctx, completion.ResourceID, "dependency_error", input.Reason, traceID)
		logx.Errorf("图片审核依赖失败，已进入自动重试: resourceId=%s traceId=%s retryCount=%d", completion.ResourceID, traceID, retryCount)
		return MediaCheckCallbackResp{ResourceID: completion.ResourceID, Status: model.ResourceStatusAuditRetry, Message: "图片审核服务暂时不可用，系统将自动重试"}, nil
	}
	if input.Status == resourceAuditTaskStatusRejected {
		resp, rejectErr := l.rejectResource(ctx, completion.ResourceID, input.Reason, traceID)
		if rejectErr == nil && resp.Status == model.ResourceStatusRejected {
			// 仅在 trace guard 真正完成当前代资源驳回后记录决策，避免旧回调污染新 attempt。
			l.recordAuditDecision(ctx, completion.ResourceID, "risky", input.Reason, traceID)
		}
		return resp, rejectErr
	}
	if completion.PendingCount > 0 {
		logx.Infof("图片审核回调已记录，仍有待完成任务: resourceId=%s traceId=%s pendingCount=%d", completion.ResourceID, traceID, completion.PendingCount)
		return MediaCheckCallbackResp{ResourceID: completion.ResourceID, Status: model.ResourceStatusPending, Message: "图片审核回调已记录"}, nil
	}
	if completion.RejectedCount > 0 || completion.FailedCount > 0 {
		logx.Infof("图片审核回调已记录，资源已有未通过图片: resourceId=%s traceId=%s rejectedCount=%d failedCount=%d", completion.ResourceID, traceID, completion.RejectedCount, completion.FailedCount)
		return MediaCheckCallbackResp{ResourceID: completion.ResourceID, Status: model.ResourceStatusPending, Message: "图片审核回调已记录"}, nil
	}

	published, err := l.store.PublishResourceAfterMediaAudit(ctx, completion.ResourceID, traceID)
	if err != nil {
		return l.handlePublishFailure(ctx, completion.ResourceID, traceID, err)
	}
	l.recordAuditDecision(ctx, completion.ResourceID, "pass", "", traceID)
	logx.Infof("图片全部审核通过，资源已自动发布: resourceId=%s traceId=%s", completion.ResourceID, traceID)
	return MediaCheckCallbackResp{ResourceID: published.ID, Status: published.Status, Message: "资源已发布"}, nil
}

func (l *MediaCheckCallbackLogic) recordAuditDecision(ctx context.Context, resourceID string, decision string, reason string, traceID string) {
	store, ok := l.store.(resourcelogic.ResourceAuditDecisionStore)
	if !ok {
		return
	}
	if err := store.RecordResourceAuditDecision(ctx, model.ResourceAuditDecisionInput{
		ResourceID: resourceID,
		Action:     "media_callback",
		Decision:   decision,
		Reason:     reason,
		TraceIDs:   []string{traceID},
	}); err != nil {
		logx.Errorf("记录图片内容审核决策失败: resourceId=%s traceId=%s decision=%s err=%+v", resourceID, traceID, decision, err)
	}
}

func (l *MediaCheckCallbackLogic) rejectResource(ctx context.Context, resourceID string, reason string, traceID string) (MediaCheckCallbackResp, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "图片疑似包含违规信息，请修改后重新提交"
	}
	rejected, err := l.store.RejectResourceAfterMediaAudit(ctx, resourceID, traceID, reason)
	if errors.Is(err, sql.ErrNoRows) {
		logx.Infof("图片审核未通过但资源状态已流转: resourceId=%s traceId=%s reason=%s", resourceID, traceID, reason)
		return MediaCheckCallbackResp{ResourceID: resourceID, Status: "unchanged", Message: "图片审核回调已记录"}, nil
	}
	if err != nil {
		logx.Errorf("图片审核未通过后自动驳回资源失败: resourceId=%s traceId=%s err=%+v", resourceID, traceID, err)
		return MediaCheckCallbackResp{}, errx.New(errx.CodeInternalError, "图片审核处理失败，请稍后重试")
	}
	logx.Infof("图片审核未通过，资源已自动驳回: resourceId=%s traceId=%s reason=%s", resourceID, traceID, reason)
	return MediaCheckCallbackResp{ResourceID: rejected.ID, Status: rejected.Status, Message: reason}, nil
}

func (l *MediaCheckCallbackLogic) handlePublishFailure(ctx context.Context, resourceID string, traceID string, err error) (MediaCheckCallbackResp, error) {
	if errors.Is(err, sql.ErrNoRows) {
		logx.Infof("图片审核通过但资源状态已流转: resourceId=%s traceId=%s", resourceID, traceID)
		return MediaCheckCallbackResp{ResourceID: resourceID, Status: "unchanged", Message: "图片审核回调已记录"}, nil
	}
	reason := ""
	switch {
	case errors.Is(err, model.ErrPublishQuotaInsufficient):
		reason = "本月发布次数已用完，可开通 VIP 或购买发布包后重新提交"
	case errors.Is(err, model.ErrPublishDisabled):
		reason = "该分类暂不开放发布"
	}
	if reason != "" {
		if _, rejectErr := l.store.RejectResourceAfterMediaAudit(ctx, resourceID, traceID, reason); rejectErr != nil && !errors.Is(rejectErr, sql.ErrNoRows) {
			logx.Errorf("图片审核通过但自动发布失败，随后驳回资源也失败: resourceId=%s traceId=%s err=%+v rejectErr=%+v", resourceID, traceID, err, rejectErr)
			return MediaCheckCallbackResp{}, errx.New(errx.CodeInternalError, "发布失败，请稍后重试")
		}
		logx.Infof("图片审核通过但资源不满足发布条件，已自动驳回: resourceId=%s traceId=%s reason=%s", resourceID, traceID, reason)
		return MediaCheckCallbackResp{ResourceID: resourceID, Status: model.ResourceStatusRejected, Message: reason}, nil
	}
	logx.Errorf("图片审核通过后自动发布失败: resourceId=%s traceId=%s err=%+v", resourceID, traceID, err)
	return MediaCheckCallbackResp{}, errx.New(errx.CodeInternalError, "发布失败，请稍后重试")
}

func mediaAuditTaskResultInput(traceID string, payload model.JSONMap) model.ResourceContentAuditTaskResultInput {
	errCode := payloadInt64(payload, "errcode", "errCode")
	result := payloadMap(payload, "result")
	suggest := trimPayloadString(result, "suggest")
	if suggest == "" {
		suggest = resourcelogic.ContentAuditDecisionPass
	}
	label := payloadInt64(result, "label")
	labels := mediaAuditPayloadLabels(payload, result)
	status := resourceAuditTaskStatusPass
	reason := ""
	if errCode != 0 {
		status = resourceAuditTaskStatusFailed
		reason = "图片审核失败，请重新上传图片后再提交"
	} else if normalizeDecision(suggest) == resourcelogic.ContentAuditDecisionRisky {
		status = resourceAuditTaskStatusRejected
		reason = resourcelogic.ResourceAuditRejectReason(resourcelogic.ContentAuditResult{Labels: labels}, "图片疑似包含违规信息，请修改后重新提交")
	}
	return model.ResourceContentAuditTaskResultInput{
		TraceID:    traceID,
		Status:     status,
		Suggest:    suggest,
		Label:      label,
		Reason:     reason,
		RawPayload: payload,
	}
}

func mediaAuditPayloadLabels(payload model.JSONMap, result model.JSONMap) []string {
	labels := make([]string, 0, 4)
	if label := payloadInt64(result, "label"); label != 0 {
		labels = append(labels, fmt.Sprintf("%d", label))
	}
	for _, item := range payloadSlice(payload, "detail") {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if label := payloadInt64(model.JSONMap(itemMap), "label"); label != 0 {
			labels = append(labels, fmt.Sprintf("%d", label))
		}
	}
	return trimNonEmpty(labels)
}

func trimPayloadString(payload model.JSONMap, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if trimmed := strings.TrimSpace(typed); trimmed != "" {
				return trimmed
			}
		case fmt.Stringer:
			if trimmed := strings.TrimSpace(typed.String()); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func payloadInt64(payload model.JSONMap, keys ...string) int64 {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case int64:
			return typed
		case int:
			return int64(typed)
		case float64:
			return int64(typed)
		case json.Number:
			parsed, _ := typed.Int64()
			return parsed
		}
	}
	return 0
}

func payloadMap(payload model.JSONMap, key string) model.JSONMap {
	value, ok := payload[key]
	if !ok {
		return model.JSONMap{}
	}
	switch typed := value.(type) {
	case model.JSONMap:
		return typed
	case map[string]interface{}:
		return model.JSONMap(typed)
	default:
		return model.JSONMap{}
	}
}

func payloadSlice(payload model.JSONMap, key string) []interface{} {
	value, ok := payload[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []interface{}:
		return typed
	default:
		return nil
	}
}
