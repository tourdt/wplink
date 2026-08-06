package contentaudit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
		logMediaCallbackInfo(ctx, "图片审核回调未匹配任务", "complete_audit_task", "", traceID)
		return MediaCheckCallbackResp{}, errx.New(errx.CodeStateConflict, "图片审核任务不存在")
	}
	if err != nil {
		logMediaCallbackFailure(ctx, "记录图片审核回调失败", "complete_audit_task", "", traceID, err)
		return MediaCheckCallbackResp{}, err
	}

	if input.Status == resourceAuditTaskStatusFailed {
		retryCount, retryErr := l.store.MarkResourceAuditRetryAfterMediaAudit(ctx, completion.ResourceID, traceID, input.Reason)
		if errors.Is(retryErr, sql.ErrNoRows) {
			logMediaCallbackInfo(ctx, "图片审核依赖失败但资源状态已流转", "mark_audit_retry", completion.ResourceID, traceID)
			return MediaCheckCallbackResp{ResourceID: completion.ResourceID, Status: "unchanged", Message: "图片审核回调已记录"}, nil
		}
		if retryErr != nil {
			logMediaCallbackFailure(ctx, "图片审核依赖失败后进入重试队列失败", "mark_audit_retry", completion.ResourceID, traceID, retryErr)
			return MediaCheckCallbackResp{}, errx.New(errx.CodeInternalError, "图片审核处理失败，请稍后重试")
		}
		l.recordAuditDecision(ctx, completion.ResourceID, "dependency_error", input.Reason, traceID)
		logMediaCallbackInfo(ctx, "图片审核依赖失败，已进入自动重试", "mark_audit_retry", completion.ResourceID, traceID, logx.Field("retryCount", retryCount))
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
		logMediaCallbackInfo(ctx, "图片审核回调已记录，仍有待完成任务", "complete_audit_task", completion.ResourceID, traceID, logx.Field("pendingCount", completion.PendingCount))
		return MediaCheckCallbackResp{ResourceID: completion.ResourceID, Status: model.ResourceStatusPending, Message: "图片审核回调已记录"}, nil
	}
	if completion.RejectedCount > 0 || completion.FailedCount > 0 {
		logMediaCallbackInfo(ctx, "图片审核回调已记录，资源已有未通过图片", "complete_audit_task", completion.ResourceID, traceID,
			logx.Field("rejectedCount", completion.RejectedCount), logx.Field("failedCount", completion.FailedCount))
		return MediaCheckCallbackResp{ResourceID: completion.ResourceID, Status: model.ResourceStatusPending, Message: "图片审核回调已记录"}, nil
	}

	published, err := l.store.PublishResourceAfterMediaAudit(ctx, completion.ResourceID, traceID)
	if err != nil {
		return l.handlePublishFailure(ctx, completion.ResourceID, traceID, err)
	}
	l.recordAuditDecision(ctx, completion.ResourceID, "pass", "", traceID)
	logMediaCallbackInfo(ctx, "图片全部审核通过，资源已自动发布", "publish_resource", completion.ResourceID, traceID)
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
		logMediaCallbackFailure(ctx, "记录图片内容审核决策失败", "record_audit_decision", resourceID, traceID, err,
			logx.Field("decision", decision))
	}
}

func (l *MediaCheckCallbackLogic) rejectResource(ctx context.Context, resourceID string, reason string, traceID string) (MediaCheckCallbackResp, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "图片疑似包含违规信息，请修改后重新提交"
	}
	rejected, err := l.store.RejectResourceAfterMediaAudit(ctx, resourceID, traceID, reason)
	if errors.Is(err, sql.ErrNoRows) {
		logMediaCallbackInfo(ctx, "图片审核未通过但资源状态已流转", "reject_resource", resourceID, traceID)
		return MediaCheckCallbackResp{ResourceID: resourceID, Status: "unchanged", Message: "图片审核回调已记录"}, nil
	}
	if err != nil {
		logMediaCallbackFailure(ctx, "图片审核未通过后自动驳回资源失败", "reject_resource", resourceID, traceID, err)
		return MediaCheckCallbackResp{}, errx.New(errx.CodeInternalError, "图片审核处理失败，请稍后重试")
	}
	logMediaCallbackInfo(ctx, "图片审核未通过，资源已自动驳回", "reject_resource", resourceID, traceID)
	return MediaCheckCallbackResp{ResourceID: rejected.ID, Status: rejected.Status, Message: reason}, nil
}

func (l *MediaCheckCallbackLogic) handlePublishFailure(ctx context.Context, resourceID string, traceID string, err error) (MediaCheckCallbackResp, error) {
	if errors.Is(err, sql.ErrNoRows) {
		logMediaCallbackInfo(ctx, "图片审核通过但资源状态已流转", "publish_resource", resourceID, traceID)
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
		_, rejectErr := l.store.RejectResourceAfterMediaAudit(ctx, resourceID, traceID, reason)
		if errors.Is(rejectErr, sql.ErrNoRows) {
			// 当前 trace 已被新一轮审核替换时，政策性发布失败不能驳回新代际资源。
			logMediaCallbackInfo(ctx, "图片审核政策性驳回未命中当前审核代际，回调已记录且资源状态未变", "policy_reject_resource", resourceID, traceID)
			return MediaCheckCallbackResp{ResourceID: resourceID, Status: "unchanged", Message: "图片审核回调已记录"}, nil
		}
		if rejectErr != nil {
			logMediaCallbackFailure(ctx, "图片审核通过但自动发布失败，随后驳回资源也失败", "policy_reject_resource", resourceID, traceID, rejectErr,
				logx.Field("publishErrorCategory", mediaCallbackErrorCategory(ctx, err)))
			return MediaCheckCallbackResp{}, errx.New(errx.CodeInternalError, "发布失败，请稍后重试")
		}
		logMediaCallbackInfo(ctx, "图片审核通过但资源不满足发布条件，已自动驳回", "policy_reject_resource", resourceID, traceID)
		return MediaCheckCallbackResp{ResourceID: resourceID, Status: model.ResourceStatusRejected, Message: reason}, nil
	}
	logMediaCallbackFailure(ctx, "图片审核通过后自动发布失败", "publish_resource", resourceID, traceID, err)
	return MediaCheckCallbackResp{}, errx.New(errx.CodeInternalError, "发布失败，请稍后重试")
}

func logMediaCallbackInfo(ctx context.Context, message string, operation string, resourceID string, traceID string, fields ...logx.LogField) {
	baseFields := mediaCallbackLogFields(operation, resourceID, traceID)
	logx.WithContext(ctx).Infow(message, append(baseFields, fields...)...)
}

func logMediaCallbackFailure(ctx context.Context, message string, operation string, resourceID string, traceID string, err error, fields ...logx.LogField) {
	baseFields := append(mediaCallbackLogFields(operation, resourceID, traceID),
		logx.Field("errorCategory", mediaCallbackErrorCategory(ctx, err)))
	logx.WithContext(ctx).Errorw(message, append(baseFields, fields...)...)
}

func mediaCallbackLogFields(operation string, resourceID string, traceID string) []logx.LogField {
	fields := []logx.LogField{
		logx.Field("operation", operation),
		logx.Field("traceFingerprint", mediaCallbackTraceFingerprint(traceID)),
	}
	if resourceID = strings.TrimSpace(resourceID); resourceID != "" {
		fields = append(fields, logx.Field("resourceId", resourceID))
	}
	return fields
}

func mediaCallbackTraceFingerprint(traceID string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(traceID)))
	return fmt.Sprintf("%x", digest[:6])
}

// mediaCallbackErrorCategory 只基于稳定 sentinel 和错误类型分类，不读取 err.Error() 中可能携带的密钥或回调原文。
func mediaCallbackErrorCategory(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, context.Canceled) || (ctx != nil && errors.Is(ctx.Err(), context.Canceled)):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded) || (ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)):
		return "timeout"
	case errors.Is(err, sql.ErrNoRows):
		return "not_found"
	case errors.Is(err, sql.ErrConnDone) || errors.Is(err, driver.ErrBadConn):
		return "unavailable"
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "unavailable"
	}
	return "unknown"
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
