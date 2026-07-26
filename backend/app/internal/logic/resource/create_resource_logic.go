package resource

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateResourceStore interface {
	GetMerchantPublishStatus(ctx context.Context, merchantID string) (string, error)
	GetMerchantContactPhone(ctx context.Context, merchantID string) (string, error)
	GetResourcePublishConfig(ctx context.Context, cityCode string, typeCode string) (model.ResourcePublishConfig, error)
	CreateResource(ctx context.Context, input model.CreateResourceInput) (model.CreateResourceResult, error)
	UpdateResourceDraft(ctx context.Context, resourceID string, input model.CreateResourceInput) (model.CreateResourceResult, error)
	RecordOperationLog(ctx context.Context, input model.OperationLogInput) error
}

type ResourceContactReq struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Wechat string `json:"wechat,omitempty"`
}

type CreateResourceReq struct {
	MerchantID        string
	CityCode          string
	TypeCode          string
	Direction         string
	Title             string
	Category          string
	District          string
	PriceText         string
	QuantityText      string
	Description       string
	Attributes        model.JSONMap
	Tags              []string
	Images            []string
	Contact           ResourceContactReq
	CreatedByUser     string
	CreatedByOperator string
	CreatedByRole     string
}

type CreateResourceResp struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type autoAuditOutcome struct {
	ID      string
	Status  string
	Message string
}

type CreateResourceLogic struct {
	store   CreateResourceStore
	auditor ContentAuditor
}

func NewCreateResourceLogic(store CreateResourceStore, auditors ...ContentAuditor) *CreateResourceLogic {
	var auditor ContentAuditor
	if len(auditors) > 0 {
		auditor = auditors[0]
	}
	return &CreateResourceLogic{store: store, auditor: auditor}
}

func (l *CreateResourceLogic) CreateResource(ctx context.Context, req CreateResourceReq) (CreateResourceResp, error) {
	return l.create(ctx, req, model.ResourceStatusPending, "已提交审核，审核通过后将展示给买家")
}

func (l *CreateResourceLogic) CreateResourceDraft(ctx context.Context, req CreateResourceReq) (CreateResourceResp, error) {
	return l.create(ctx, req, model.ResourceStatusDraft, "草稿已保存")
}

func (l *CreateResourceLogic) create(ctx context.Context, req CreateResourceReq, status string, message string) (CreateResourceResp, error) {
	input, typeCode, err := l.buildResourceInput(ctx, req, status)
	if err != nil {
		return CreateResourceResp{}, err
	}
	result, err := l.store.CreateResource(ctx, input)
	if err != nil {
		if errors.Is(err, model.ErrPublishQuotaInsufficient) {
			logx.Infof("创建资源被拦截: merchantId=%s typeCode=%s reason=publish_quota_insufficient", strings.TrimSpace(req.MerchantID), typeCode)
			return CreateResourceResp{}, errx.New(errx.CodeQuotaNotEnough, "本月发布次数已用完，可开通 VIP 或购买发布包")
		}
		logx.Errorf("创建资源失败: merchantId=%s typeCode=%s targetStatus=%s err=%+v", strings.TrimSpace(req.MerchantID), typeCode, status, err)
		return CreateResourceResp{}, err
	}
	if isOperatorProxy(req.CreatedByRole) && strings.TrimSpace(req.CreatedByOperator) != "" {
		if err := l.store.RecordOperationLog(ctx, model.OperationLogInput{
			OperatorID:     strings.TrimSpace(req.CreatedByOperator),
			OperatorRole:   strings.TrimSpace(req.CreatedByRole),
			Action:         "proxy_create_resource",
			ObjectType:     "resource",
			ObjectID:       result.ID,
			BeforeSnapshot: model.JSONMap{},
			AfterSnapshot:  model.JSONMap{"status": result.Status, "typeCode": typeCode},
		}); err != nil {
			logx.Errorf("记录代发布资源操作日志失败: operatorId=%s resourceId=%s status=%s err=%+v", strings.TrimSpace(req.CreatedByOperator), result.ID, result.Status, err)
			return CreateResourceResp{}, err
		}
	}
	if status != model.ResourceStatusDraft {
		return l.auditCreatedResource(ctx, result, input, req)
	}
	// 新建资源可能直接进入审核队列，也可能只是草稿；日志记录目标状态，避免排查时只看到写入成功但不知道用户路径。
	logx.Infof("创建资源成功: merchantId=%s resourceId=%s typeCode=%s status=%s createdByRole=%s", strings.TrimSpace(req.MerchantID), result.ID, typeCode, result.Status, strings.TrimSpace(req.CreatedByRole))
	return CreateResourceResp{ID: result.ID, Status: result.Status, Message: message}, nil
}

func (l *CreateResourceLogic) auditCreatedResource(ctx context.Context, created model.CreateResourceResult, input model.CreateResourceInput, req CreateResourceReq) (CreateResourceResp, error) {
	if l.auditor == nil {
		return CreateResourceResp{ID: created.ID, Status: created.Status, Message: "已提交审核，审核通过后将展示给买家"}, nil
	}
	auditStore, ok := l.store.(ResourceAutoAuditStore)
	if !ok {
		logx.Errorf("创建资源自动审核缺少状态更新能力: merchantId=%s resourceId=%s typeCode=%s", input.MerchantID, created.ID, input.TypeCode)
		return CreateResourceResp{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
	}
	userID := strings.TrimSpace(req.CreatedByUser)
	if isOperatorProxy(req.CreatedByRole) {
		reason := "后台代发布缺少微信审核身份，已转人工复核"
		return holdCreatedResourceForManualReview(ctx, auditStore, created.ID, reason, "operator_proxy", input)
	}
	if userID == "" {
		reason := "资源缺少微信审核身份，已转人工复核"
		return holdCreatedResourceForManualReview(ctx, auditStore, created.ID, reason, "missing_user", input)
	}
	userStore, ok := l.store.(ResourceAuditUserStore)
	if !ok {
		logx.Errorf("创建资源内容预审核缺少用户 openid 查询能力: merchantId=%s typeCode=%s userId=%s", input.MerchantID, input.TypeCode, userID)
		return CreateResourceResp{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
	}
	openID, err := userStore.GetUserWechatOpenID(ctx, userID)
	if err != nil {
		return scheduleCreatedResourceAuditRetry(ctx, auditStore, created.ID, "读取微信审核身份失败", input, err)
	}
	if strings.TrimSpace(openID) == "" {
		reason := "资源缺少微信审核身份，已转人工复核"
		return holdCreatedResourceForManualReview(ctx, auditStore, created.ID, reason, "empty_openid", input)
	}
	auditInput := contentAuditInputFromCreate(input, openID)
	auditInput.ResourceID = created.ID
	result, err := l.auditor.AuditResource(ctx, auditInput)
	return l.applyAutoAuditResult(ctx, auditStore, "create_resource", created.ID, auditInput, result, err)
}

func holdCreatedResourceForManualReview(ctx context.Context, auditStore ResourceAutoAuditStore, resourceID string, reason string, blockReason string, input model.CreateResourceInput) (CreateResourceResp, error) {
	stateStore, ok := auditStore.(ResourceAuditStateStore)
	if !ok {
		logx.Errorf("创建资源需要人工复核但缺少状态更新能力: merchantId=%s resourceId=%s typeCode=%s blockReason=%s", input.MerchantID, resourceID, input.TypeCode, blockReason)
		return CreateResourceResp{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
	}
	if err := stateStore.MarkResourceManualReview(ctx, resourceID, reason); err != nil {
		logx.Errorf("创建资源转人工复核失败: merchantId=%s resourceId=%s typeCode=%s blockReason=%s err=%+v", input.MerchantID, resourceID, input.TypeCode, blockReason, err)
		return CreateResourceResp{}, errx.New(errx.CodeInternalError, "内容审核失败，请稍后重试")
	}
	recordResourceAuditDecision(ctx, auditStore, model.ResourceAuditDecisionInput{
		ResourceID: resourceID,
		Action:     "create_resource",
		Decision:   "manual_review",
		Reason:     reason,
	})
	logx.Infof("创建资源已转人工复核: merchantId=%s resourceId=%s typeCode=%s blockReason=%s", input.MerchantID, resourceID, input.TypeCode, blockReason)
	return CreateResourceResp{ID: resourceID, Status: model.ResourceStatusManualReview, Message: "内容审核中"}, nil
}

func scheduleCreatedResourceAuditRetry(ctx context.Context, auditStore ResourceAutoAuditStore, resourceID string, reason string, input model.CreateResourceInput, cause error) (CreateResourceResp, error) {
	stateStore, ok := auditStore.(ResourceAuditStateStore)
	if !ok {
		logx.Errorf("创建资源内容审核依赖失败但缺少重试能力: merchantId=%s resourceId=%s typeCode=%s err=%+v", input.MerchantID, resourceID, input.TypeCode, cause)
		return CreateResourceResp{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
	}
	retryCount, err := stateStore.MarkResourceAuditRetry(ctx, resourceID, reason)
	if err != nil {
		logx.Errorf("创建资源进入审核重试队列失败: merchantId=%s resourceId=%s typeCode=%s err=%+v cause=%+v", input.MerchantID, resourceID, input.TypeCode, err, cause)
		return CreateResourceResp{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
	}
	recordResourceAuditDecision(ctx, auditStore, model.ResourceAuditDecisionInput{
		ResourceID: resourceID,
		Action:     "create_resource",
		Decision:   "dependency_error",
		Reason:     reason,
	})
	logx.Errorf("创建资源内容审核依赖失败，已进入自动重试: merchantId=%s resourceId=%s typeCode=%s retryCount=%d err=%+v", input.MerchantID, resourceID, input.TypeCode, retryCount, cause)
	return CreateResourceResp{ID: resourceID, Status: model.ResourceStatusAuditRetry, Message: "内容审核服务暂时不可用，系统将自动重试"}, nil
}

func (l *CreateResourceLogic) applyAutoAuditResult(ctx context.Context, auditStore ResourceAutoAuditStore, action string, resourceID string, input ContentAuditInput, result ContentAuditResult, err error) (CreateResourceResp, error) {
	outcome, err := applyResourceAutoAuditResult(ctx, auditStore, action, resourceID, input, result, err)
	if err != nil {
		return CreateResourceResp{}, err
	}
	return CreateResourceResp{ID: outcome.ID, Status: outcome.Status, Message: outcome.Message}, nil
}

func applyResourceAutoAuditResult(ctx context.Context, auditStore ResourceAutoAuditStore, action string, resourceID string, input ContentAuditInput, result ContentAuditResult, err error) (autoAuditOutcome, error) {
	if err != nil {
		reason := "内容审核服务暂时不可用，系统将自动重试"
		stateStore, ok := auditStore.(ResourceAuditStateStore)
		if !ok {
			logx.Errorf("内容审核失败但存储不支持自动重试: action=%s resourceId=%s err=%+v", action, resourceID, err)
			return autoAuditOutcome{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
		}
		retryCount, retryErr := stateStore.MarkResourceAuditRetry(ctx, resourceID, err.Error())
		if retryErr != nil {
			logx.Errorf("内容审核失败后进入自动重试队列失败: action=%s resourceId=%s err=%+v retryErr=%+v", action, resourceID, err, retryErr)
			return autoAuditOutcome{}, errx.New(errx.CodeInternalError, "内容审核服务暂不可用，请稍后重试")
		}
		recordResourceAuditDecision(ctx, auditStore, model.ResourceAuditDecisionInput{
			ResourceID: resourceID,
			Action:     action,
			Decision:   "dependency_error",
			Reason:     err.Error(),
		})
		logx.Errorf("资源内容审核依赖失败，已进入自动重试: action=%s merchantId=%s resourceId=%s typeCode=%s retryCount=%d err=%+v", action, input.MerchantID, resourceID, input.TypeCode, retryCount, err)
		return autoAuditOutcome{ID: resourceID, Status: model.ResourceStatusAuditRetry, Message: reason}, nil
	}
	decision := normalizeContentAuditDecision(result.Decision)
	recordedDecision := decision
	if decision == ContentAuditDecisionReview {
		recordedDecision = "review_relaxed"
	}
	recordResourceAuditDecision(ctx, auditStore, model.ResourceAuditDecisionInput{
		ResourceID: resourceID,
		Action:     action,
		Decision:   recordedDecision,
		Reason:     strings.TrimSpace(result.Reason),
		Labels:     append([]string(nil), result.Labels...),
		TraceIDs:   append([]string(nil), result.TraceIDs...),
	})
	if decision == ContentAuditDecisionRisky {
		reason := ResourceAuditRejectReason(result, "内容可能含有违规信息，请调整文字或图片后重新提交")
		if _, err := auditStore.RejectResourceAfterAudit(ctx, resourceID, reason); err != nil {
			logx.Errorf("资源内容审核命中风险后自动驳回失败: action=%s resourceId=%s labels=%s err=%+v", action, resourceID, strings.Join(result.Labels, ","), err)
			return autoAuditOutcome{}, errx.New(errx.CodeInternalError, "内容审核失败，请稍后重试")
		}
		logx.Infof("资源内容审核拒绝发布: action=%s merchantId=%s resourceId=%s typeCode=%s labels=%s reason=%s", action, input.MerchantID, resourceID, input.TypeCode, strings.Join(result.Labels, ","), reason)
		return autoAuditOutcome{ID: resourceID, Status: model.ResourceStatusRejected, Message: reason}, nil
	}
	if decision == ContentAuditDecisionReview {
		// 冷启动阶段采用宽松审核：微信建议复核但未判定风险时继续发布，并记录标签供后续抽检。
		// 确定高风险内容仍由上方 risky 分支直接拦截。
		logx.Infof("资源内容审核建议复核，按宽松策略继续自动发布: action=%s merchantId=%s resourceId=%s typeCode=%s labels=%s", action, input.MerchantID, resourceID, input.TypeCode, strings.Join(result.Labels, ","))
	}
	if len(result.MediaTasks) > 0 {
		tasks := make([]model.ResourceContentAuditTaskInput, 0, len(result.MediaTasks))
		for _, task := range result.MediaTasks {
			tasks = append(tasks, model.ResourceContentAuditTaskInput{
				TraceID:   task.TraceID,
				AuditType: "image",
				MediaURL:  task.MediaURL,
			})
		}
		if err := auditStore.CreateResourceContentAuditTasks(ctx, resourceID, tasks); err != nil {
			logx.Errorf("保存资源图片审核任务失败: action=%s resourceId=%s taskCount=%d err=%+v", action, resourceID, len(tasks), err)
			return autoAuditOutcome{}, errx.New(errx.CodeInternalError, "内容审核任务创建失败，请稍后重试")
		}
		logx.Infof("资源图片审核任务已创建: action=%s merchantId=%s resourceId=%s typeCode=%s taskCount=%d", action, input.MerchantID, resourceID, input.TypeCode, len(tasks))
		return autoAuditOutcome{ID: resourceID, Status: model.ResourceStatusPending, Message: "内容审核中，审核通过后将自动发布"}, nil
	}
	published, err := auditStore.PublishResourceAfterAudit(ctx, resourceID)
	if err != nil {
		logx.Errorf("资源内容审核通过后自动发布失败: action=%s merchantId=%s resourceId=%s typeCode=%s err=%+v", action, input.MerchantID, resourceID, input.TypeCode, err)
		return autoAuditOutcome{}, mapAutoPublishError(ctx, auditStore, resourceID, err)
	}
	logx.Infof("资源内容审核通过并自动发布: action=%s merchantId=%s resourceId=%s typeCode=%s", action, input.MerchantID, resourceID, input.TypeCode)
	return autoAuditOutcome{ID: published.ID, Status: published.Status, Message: "已发布"}, nil
}

func mapAutoPublishError(ctx context.Context, auditStore ResourceAutoAuditStore, resourceID string, err error) error {
	switch {
	case errors.Is(err, model.ErrPublishQuotaInsufficient):
		reason := "本月发布次数已用完，可开通 VIP 或购买发布包后重新提交"
		_, _ = auditStore.RejectResourceAfterAudit(ctx, resourceID, reason)
		return errx.New(errx.CodeQuotaNotEnough, reason)
	case errors.Is(err, model.ErrPublishDisabled):
		reason := "该分类暂不开放发布"
		_, _ = auditStore.RejectResourceAfterAudit(ctx, resourceID, reason)
		return errx.New(errx.CodeValidationFailed, reason)
	default:
		return errx.New(errx.CodeInternalError, "发布失败，请稍后重试")
	}
}

func (l *CreateResourceLogic) UpdateResourceDraft(ctx context.Context, resourceID string, req CreateResourceReq) (CreateResourceResp, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return CreateResourceResp{}, errx.New(errx.CodeValidationFailed, "资源不存在或已下架")
	}
	input, _, err := l.buildResourceInput(ctx, req, model.ResourceStatusDraft)
	if err != nil {
		return CreateResourceResp{}, err
	}
	result, err := l.store.UpdateResourceDraft(ctx, resourceID, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logx.Infof("更新资源草稿被拦截: merchantId=%s resourceId=%s reason=not_editable", strings.TrimSpace(req.MerchantID), resourceID)
			return CreateResourceResp{}, errx.New(errx.CodeStateConflict, "资源不存在或当前状态不可编辑")
		}
		logx.Errorf("更新资源草稿失败: merchantId=%s resourceId=%s err=%+v", strings.TrimSpace(req.MerchantID), resourceID, err)
		return CreateResourceResp{}, err
	}
	logx.Infof("更新资源草稿成功: merchantId=%s resourceId=%s status=%s", strings.TrimSpace(req.MerchantID), result.ID, result.Status)
	return CreateResourceResp{ID: result.ID, Status: result.Status, Message: "草稿已保存，请重新提交审核"}, nil
}

func (l *CreateResourceLogic) buildResourceInput(ctx context.Context, req CreateResourceReq, status string) (model.CreateResourceInput, string, error) {
	cityCode := strings.TrimSpace(req.CityCode)
	typeCode := strings.TrimSpace(req.TypeCode)
	config, err := l.store.GetResourcePublishConfig(ctx, cityCode, typeCode)
	if err != nil {
		return model.CreateResourceInput{}, "", err
	}

	values := map[string]string{
		"merchantId":    strings.TrimSpace(req.MerchantID),
		"cityCode":      cityCode,
		"typeCode":      typeCode,
		"title":         strings.TrimSpace(req.Title),
		"category":      strings.TrimSpace(req.Category),
		"quantityText":  strings.TrimSpace(req.QuantityText),
		"priceText":     strings.TrimSpace(req.PriceText),
		"district":      strings.TrimSpace(req.District),
		"contactName":   strings.TrimSpace(req.Contact.Name),
		"contactPhone":  strings.TrimSpace(req.Contact.Phone),
		"contactWechat": strings.TrimSpace(req.Contact.Wechat),
		"description":   strings.TrimSpace(req.Description),
	}
	if shouldUseMerchantContactPhone(values["contactPhone"]) {
		// 发布页只能从公开商家资料拿到脱敏手机号，保存资源前用商家资料真实电话兜底。
		merchantPhone, err := l.store.GetMerchantContactPhone(ctx, values["merchantId"])
		if err != nil {
			return model.CreateResourceInput{}, "", err
		}
		values["contactPhone"] = strings.TrimSpace(merchantPhone)
	}
	deriveResourceSummaryFields(config, values, req.Attributes)
	normalizedTags, err := normalizeResourceTags(config, req.Tags)
	if err != nil {
		return model.CreateResourceInput{}, "", err
	}
	if values["description"] == "" {
		return model.CreateResourceInput{}, "", errx.New(errx.CodeValidationFailed, resourceDescriptionRequiredMessage(config.Direction))
	}
	if values["title"] == "" {
		// 小程序发布页减少用户填写项，不再要求手动标题；后端兜底生成标题，保证列表、分享和审核消息仍有稳定展示文本。
		values["title"] = buildGeneratedResourceTitle(values)
	}
	if err := validateResourceRequiredFields(config, values, req.Attributes, normalizedTags, req.Images); err != nil {
		return model.CreateResourceInput{}, "", err
	}
	if err := validateResourceDynamicFieldValues(config.FieldSchema, req.Attributes); err != nil {
		return model.CreateResourceInput{}, "", err
	}
	publishMode := model.PublishModeFromCommercialRules(config.CommercialRules)
	if publishMode == model.ResourcePublishModeDisabled {
		logx.Infof("创建资源被拦截: merchantId=%s typeCode=%s reason=publish_disabled", values["merchantId"], typeCode)
		return model.CreateResourceInput{}, "", errx.New(errx.CodeValidationFailed, "该分类暂不开放发布")
	}
	merchantStatus, err := l.store.GetMerchantPublishStatus(ctx, values["merchantId"])
	if err != nil {
		return model.CreateResourceInput{}, "", err
	}
	if merchantStatus != model.MerchantStatusActive {
		return model.CreateResourceInput{}, "", errx.New(errx.CodeValidationFailed, "商家已停用，不能发布资源")
	}
	if values["category"] == "" {
		// 数据库仍使用 category 作为检索摘要列。类型没有可映射分类时写入稳定兜底值，避免把旧固定“品类”重新暴露给用户。
		values["category"] = "待沟通"
	}

	return model.CreateResourceInput{
		MerchantID:                values["merchantId"],
		CityCode:                  cityCode,
		ResourceTypeConfigID:      config.ID,
		ResourceTypeConfigVersion: max(config.Version, 1),
		ResourceTypeSnapshot:      model.ResourceTypeSnapshotFromPublishConfig(config),
		TypeCode:                  typeCode,
		Direction:                 normalizeConfigDirection(config.Direction),
		Status:                    status,
		Title:                     values["title"],
		Category:                  values["category"],
		District:                  strings.TrimSpace(req.District),
		PriceText:                 values["priceText"],
		QuantityText:              values["quantityText"],
		CoverURL:                  firstResourceImage(req.Images),
		Description:               values["description"],
		Attributes:                req.Attributes,
		Tags:                      normalizedTags,
		Images:                    append([]string(nil), req.Images...),
		ContactName:               values["contactName"],
		ContactPhone:              values["contactPhone"],
		ContactWechat:             strings.TrimSpace(req.Contact.Wechat),
		CreatedByUser:             strings.TrimSpace(req.CreatedByUser),
		CreatedByOperator:         strings.TrimSpace(req.CreatedByOperator),
		ConsumePublishQuota:       false,
	}, typeCode, nil
}

func resourceDescriptionRequiredMessage(direction string) string {
	if normalizeConfigDirection(direction) == model.ResourceDirectionDemand {
		return "请填写需求描述"
	}
	return "请填写供应描述"
}

func buildGeneratedResourceTitle(values map[string]string) string {
	summaryParts := make([]string, 0, 3)
	for _, field := range []string{"category", "quantityText", "priceText"} {
		if value := strings.TrimSpace(values[field]); value != "" {
			summaryParts = append(summaryParts, value)
		}
	}
	if title := truncateGeneratedResourceTitle(strings.Join(summaryParts, " ")); title != "" {
		return title
	}
	return truncateGeneratedResourceTitle(values["description"])
}

func truncateGeneratedResourceTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	const maxTitleRunes = 30
	runes := []rune(value)
	if len(runes) <= maxTitleRunes {
		return value
	}
	return string(runes[:maxTitleRunes]) + "..."
}

func deriveResourceSummaryFields(config model.ResourcePublishConfig, values map[string]string, attributes model.JSONMap) {
	summary, ok := config.DisplayTemplate["summary"].(map[string]interface{})
	if !ok {
		if typed, ok := config.DisplayTemplate["summary"].(model.JSONMap); ok {
			summary = map[string]interface{}(typed)
		}
	}
	if len(summary) == 0 {
		return
	}
	for _, target := range []string{"category", "quantityText", "priceText"} {
		source, _ := summary[target].(string)
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		if value := resourceSummarySourceValue(source, values, attributes); value != "" {
			values[target] = value
		}
	}
}

func resourceSummarySourceValue(source string, values map[string]string, attributes model.JSONMap) string {
	if value, ok := values[source]; ok {
		return strings.TrimSpace(value)
	}
	value, ok := attributes[source]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case bool:
		if typed {
			return "是"
		}
		return "否"
	case model.JSONMap, map[string]interface{}:
		return resourceAddressAttributeText(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func normalizeConfigDirection(direction string) string {
	direction = strings.TrimSpace(direction)
	if direction == model.ResourceDirectionDemand {
		return model.ResourceDirectionDemand
	}
	return model.ResourceDirectionSupply
}

func isOperatorProxy(role string) bool {
	role = strings.TrimSpace(role)
	return role == "platform_operator" || role == "super_admin"
}

func firstResourceImage(images []string) string {
	for _, image := range images {
		if trimmed := strings.TrimSpace(image); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func shouldUseMerchantContactPhone(phone string) bool {
	phone = strings.TrimSpace(phone)
	return phone == "" || strings.Contains(phone, "*")
}

var resourceBaseFieldLabels = map[string]string{
	"merchantId":    "商家",
	"cityCode":      "城市站",
	"typeCode":      "资源类型",
	"title":         "标题",
	"category":      "品类",
	"district":      "区域",
	"quantityText":  "数量/产能",
	"priceText":     "价格描述",
	"description":   "资源描述",
	"contactName":   "联系人",
	"contactPhone":  "联系电话",
	"contactWechat": "联系微信",
	"tags":          "资源标签",
	"images":        "资源图片",
}

const maxCustomSelectAttributeLength = 32
const maxAddressAttributeLength = 160
const maxAddressNameLength = 80
const maxResourceTagCount = 8
const maxResourceTagLength = 12

type resourceFieldSpec struct {
	Key         string
	Label       string
	Type        string
	Options     []string
	AllowCustom bool
}

func normalizeResourceTags(config model.ResourcePublishConfig, tags []string) ([]string, error) {
	allowedOptions := resourceTagOptionSet(config.FieldSchema)
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	label := resourceTagValidationLabel(config.Direction)
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !validResourceTagText(tag) {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("请正确填写%s", label))
		}
		if len(allowedOptions) > 0 {
			if _, ok := allowedOptions[tag]; !ok {
				return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("请选择正确的%s", label))
			}
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		if len(normalized) >= maxResourceTagCount {
			return nil, errx.New(errx.CodeValidationFailed, fmt.Sprintf("最多选择 %d 个%s", maxResourceTagCount, label))
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized, nil
}

func resourceTagValidationLabel(direction string) string {
	if normalizeConfigDirection(direction) == model.ResourceDirectionDemand {
		return "需求标签"
	}
	return "供应标签"
}

func resourceTagOptionSet(fieldSchema model.JSONMap) map[string]struct{} {
	options := stringOptionsFromInterface(fieldSchema["tagOptions"])
	if len(options) == 0 {
		return nil
	}
	optionSet := make(map[string]struct{}, len(options))
	for _, option := range options {
		if validResourceTagText(option) {
			optionSet[option] = struct{}{}
		}
	}
	return optionSet
}

func validResourceTagText(value string) bool {
	if len([]rune(value)) > maxResourceTagLength {
		return false
	}
	// 标签会进入列表卡片、详情和内容审核文本，拦截控制字符，避免不可见内容污染公开展示。
	return !strings.ContainsFunc(value, func(r rune) bool {
		return r < 32 || r == 127
	})
}

func validateResourceRequiredFields(config model.ResourcePublishConfig, values map[string]string, attributes model.JSONMap, tags []string, images []string) error {
	fieldLabels := resourceFieldLabels(config.FieldSchema)
	for key, label := range resourceBaseFieldLabels {
		if _, ok := fieldLabels[key]; !ok {
			fieldLabels[key] = label
		}
	}

	for _, field := range config.RequiredFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		missing := false
		if value, ok := values[field]; ok {
			missing = strings.TrimSpace(value) == ""
		} else if field == "tags" {
			missing = len(tags) == 0
		} else if field == "images" {
			missing = len(images) == 0
		} else {
			missing = resourceAttributeMissing(attributes[field])
		}
		if missing {
			label := fieldLabels[field]
			if label == "" {
				label = "配置字段"
			}
			return errx.New(errx.CodeValidationFailed, fmt.Sprintf("请补充%s", label))
		}
	}
	return nil
}

func validateResourceDynamicFieldValues(fieldSchema model.JSONMap, attributes model.JSONMap) error {
	for _, field := range resourceFieldSpecs(fieldSchema) {
		switch field.Type {
		case "select":
			if len(field.Options) == 0 {
				continue
			}
			if err := validateResourceSelectAttribute(field, attributes); err != nil {
				return err
			}
		case "address":
			if err := validateResourceAddressAttribute(field, attributes); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateResourceSelectAttribute(field resourceFieldSpec, attributes model.JSONMap) error {
	value, ok := attributes[field.Key]
	if !ok || resourceAttributeMissing(value) {
		return nil
	}
	text, ok := value.(string)
	if !ok {
		return errx.New(errx.CodeValidationFailed, fmt.Sprintf("请选择正确的%s", fieldLabelOrKey(field)))
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	attributes[field.Key] = text
	if stringSliceContains(field.Options, text) {
		return nil
	}
	if !field.AllowCustom {
		return errx.New(errx.CodeValidationFailed, fmt.Sprintf("请选择正确的%s", fieldLabelOrKey(field)))
	}
	if !validCustomSelectAttribute(text) {
		return errx.New(errx.CodeValidationFailed, fmt.Sprintf("请正确填写%s", fieldLabelOrKey(field)))
	}
	return nil
}

func validateResourceAddressAttribute(field resourceFieldSpec, attributes model.JSONMap) error {
	value, ok := attributes[field.Key]
	if !ok || resourceAttributeMissing(value) {
		return nil
	}
	normalized, message := normalizeResourceAddressAttribute(value, fieldLabelOrKey(field))
	if message != "" {
		return errx.New(errx.CodeValidationFailed, message)
	}
	attributes[field.Key] = normalized
	return nil
}

func normalizeResourceAddressAttribute(value interface{}, label string) (model.JSONMap, string) {
	switch typed := value.(type) {
	case string:
		address, ok := cleanResourceAddressText(typed, maxAddressAttributeLength)
		if !ok || address == "" {
			return nil, fmt.Sprintf("请正确填写%s", label)
		}
		return model.JSONMap{"address": address}, ""
	case model.JSONMap:
		return normalizeResourceAddressMap(map[string]interface{}(typed), label)
	case map[string]interface{}:
		return normalizeResourceAddressMap(typed, label)
	default:
		return nil, fmt.Sprintf("请正确填写%s", label)
	}
}

func normalizeResourceAddressMap(values map[string]interface{}, label string) (model.JSONMap, string) {
	address, addressOK := cleanResourceAddressText(resourceAddressMapString(values, "address"), maxAddressAttributeLength)
	name, nameOK := cleanResourceAddressText(resourceAddressMapString(values, "name"), maxAddressNameLength)
	if !addressOK || !nameOK {
		return nil, fmt.Sprintf("请正确填写%s", label)
	}
	if address == "" {
		address = name
	}
	if address == "" {
		return nil, fmt.Sprintf("请正确填写%s", label)
	}

	lat, latPresent, latOK := resourceCoordinateFromMap(values, "latitude", "lat")
	lng, lngPresent, lngOK := resourceCoordinateFromMap(values, "longitude", "lng")
	if latPresent != lngPresent || (latPresent && (!latOK || !lngOK || lat < -90 || lat > 90 || lng < -180 || lng > 180)) {
		return nil, fmt.Sprintf("请重新选择%s地图位置", label)
	}

	normalized := model.JSONMap{"address": address}
	if name != "" {
		normalized["name"] = name
	}
	if latPresent {
		normalized["latitude"] = lat
		normalized["longitude"] = lng
	}
	return normalized, ""
}

func resourceAddressMapString(values map[string]interface{}, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func cleanResourceAddressText(value string, maxRunes int) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if len([]rune(value)) > maxRunes {
		return "", false
	}
	// 地址会进入公开详情和导航标题，拦截控制字符，避免出现不可见内容影响展示和审核。
	if strings.ContainsFunc(value, func(r rune) bool {
		return r < 32 || r == 127
	}) {
		return "", false
	}
	return value, true
}

func resourceCoordinateFromMap(values map[string]interface{}, primaryKey string, fallbackKey string) (float64, bool, bool) {
	if value, ok := values[primaryKey]; ok {
		return parseResourceCoordinate(value)
	}
	if value, ok := values[fallbackKey]; ok {
		return parseResourceCoordinate(value)
	}
	return 0, false, true
}

func parseResourceCoordinate(value interface{}) (float64, bool, bool) {
	if value == nil {
		return 0, false, true
	}
	switch typed := value.(type) {
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return 0, false, true
		}
		parsed, err := strconv.ParseFloat(typed, 64)
		return parsed, true, err == nil && validResourceCoordinateNumber(parsed)
	case float64:
		return typed, true, validResourceCoordinateNumber(typed)
	case float32:
		parsed := float64(typed)
		return parsed, true, validResourceCoordinateNumber(parsed)
	case int:
		return float64(typed), true, true
	case int64:
		return float64(typed), true, true
	case int32:
		return float64(typed), true, true
	case uint:
		return float64(typed), true, true
	case uint64:
		return float64(typed), true, true
	case uint32:
		return float64(typed), true, true
	default:
		return 0, true, false
	}
}

func validResourceCoordinateNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func resourceAddressAttributeText(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case model.JSONMap:
		return resourceAddressMapDisplayText(map[string]interface{}(typed))
	case map[string]interface{}:
		return resourceAddressMapDisplayText(typed)
	default:
		return ""
	}
}

func resourceAddressMapDisplayText(values map[string]interface{}) string {
	if text := resourceAddressMapString(values, "address"); text != "" {
		return text
	}
	return resourceAddressMapString(values, "name")
}

func isResourceAddressLikeMap(value interface{}) bool {
	var values map[string]interface{}
	switch typed := value.(type) {
	case model.JSONMap:
		values = map[string]interface{}(typed)
	case map[string]interface{}:
		values = typed
	default:
		return false
	}
	for _, key := range []string{"address", "name", "latitude", "longitude", "lat", "lng"} {
		if _, ok := values[key]; ok {
			return true
		}
	}
	return false
}

func resourceFieldLabels(fieldSchema model.JSONMap) map[string]string {
	labels := make(map[string]string)
	for _, field := range resourceFieldSpecs(fieldSchema) {
		if field.Key != "" && field.Label != "" {
			labels[field.Key] = field.Label
		}
	}
	return labels
}

func resourceFieldSpecs(fieldSchema model.JSONMap) []resourceFieldSpec {
	fields, ok := fieldSchema["fields"].([]interface{})
	if !ok {
		return nil
	}
	specs := make([]resourceFieldSpec, 0, len(fields))
	for _, entry := range fields {
		field, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := field["key"].(string)
		label, _ := field["label"].(string)
		key = strings.TrimSpace(key)
		label = strings.TrimSpace(label)
		if key == "" {
			continue
		}
		fieldType, _ := field["type"].(string)
		allowCustom, _ := field["allowCustom"].(bool)
		specs = append(specs, resourceFieldSpec{
			Key:         key,
			Label:       label,
			Type:        strings.TrimSpace(fieldType),
			Options:     stringOptionsFromInterface(field["options"]),
			AllowCustom: allowCustom,
		})
	}
	return specs
}

func stringOptionsFromInterface(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		options := make([]string, 0, len(typed))
		for _, item := range typed {
			if option := strings.TrimSpace(item); option != "" {
				options = append(options, option)
			}
		}
		return options
	case []interface{}:
		options := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if ok {
				if option := strings.TrimSpace(text); option != "" {
					options = append(options, option)
				}
			}
		}
		return options
	default:
		return nil
	}
}

func stringSliceContains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func validCustomSelectAttribute(value string) bool {
	if len([]rune(value)) > maxCustomSelectAttributeLength {
		return false
	}
	// 自定义选项会进入公开展示和后续运营归并，先拦截控制字符，避免出现不可见内容影响审核和筛选。
	return !strings.ContainsFunc(value, func(r rune) bool {
		return r < 32 || r == 127
	})
}

func fieldLabelOrKey(field resourceFieldSpec) string {
	if field.Label != "" {
		return field.Label
	}
	return field.Key
}

func resourceAttributeMissing(value interface{}) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case bool:
		return false
	case model.JSONMap, map[string]interface{}:
		if isResourceAddressLikeMap(typed) {
			return resourceAddressAttributeText(typed) == ""
		}
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return rv.Len() == 0
	}
	return false
}
