package resource

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"wplink/backend/app/internal/model"
)

const (
	ContentAuditDecisionPass   = "pass"
	ContentAuditDecisionReview = "review"
	ContentAuditDecisionRisky  = "risky"
)

type ContentAuditInput struct {
	ResourceID    string
	MerchantID    string
	TypeCode      string
	OpenID        string
	Title         string
	Category      string
	District      string
	PriceText     string
	QuantityText  string
	Description   string
	Attributes    model.JSONMap
	Tags          []string
	Images        []string
	ContactName   string
	ContactWechat string
}

type ContentAuditResult struct {
	Decision   string
	Reason     string
	Labels     []string
	TraceIDs   []string
	MediaTasks []ContentAuditMediaTask
}

type ContentAuditMediaTask struct {
	TraceID  string
	MediaURL string
}

type ContentAuditor interface {
	AuditResource(ctx context.Context, input ContentAuditInput) (ContentAuditResult, error)
}

type ResourceAuditUserStore interface {
	GetUserWechatOpenID(ctx context.Context, userID string) (string, error)
}

type ResourceAuditSnapshotStore interface {
	GetResourceAuditSnapshot(ctx context.Context, resourceID string) (model.ResourceAuditSnapshot, error)
}

type ResourceAutoAuditStore interface {
	CreateResourceContentAuditTasks(ctx context.Context, resourceID string, tasks []model.ResourceContentAuditTaskInput) error
	PublishResourceAfterAudit(ctx context.Context, resourceID string) (model.ReviewResourceResult, error)
	RejectResourceAfterAudit(ctx context.Context, resourceID string, reason string) (model.ReviewResourceResult, error)
}

func contentAuditInputFromCreate(input model.CreateResourceInput, openID string) ContentAuditInput {
	return ContentAuditInput{
		MerchantID:    input.MerchantID,
		TypeCode:      input.TypeCode,
		OpenID:        openID,
		Title:         input.Title,
		Category:      input.Category,
		District:      input.District,
		PriceText:     input.PriceText,
		QuantityText:  input.QuantityText,
		Description:   input.Description,
		Attributes:    input.Attributes,
		Tags:          append([]string(nil), input.Tags...),
		Images:        append([]string(nil), input.Images...),
		ContactName:   input.ContactName,
		ContactWechat: input.ContactWechat,
	}
}

func contentAuditInputFromSnapshot(snapshot model.ResourceAuditSnapshot) ContentAuditInput {
	return ContentAuditInput{
		ResourceID:    snapshot.ID,
		MerchantID:    snapshot.MerchantID,
		TypeCode:      snapshot.TypeCode,
		OpenID:        snapshot.OpenID,
		Title:         snapshot.Title,
		Category:      snapshot.Category,
		District:      snapshot.District,
		PriceText:     snapshot.PriceText,
		QuantityText:  snapshot.QuantityText,
		Description:   snapshot.Description,
		Attributes:    snapshot.Attributes,
		Tags:          append([]string(nil), snapshot.Tags...),
		Images:        append([]string(nil), snapshot.Images...),
		ContactName:   snapshot.ContactName,
		ContactWechat: snapshot.ContactWechat,
	}
}

func ResourceAuditRejectReason(result ContentAuditResult, fallback string) string {
	if reason := strings.TrimSpace(result.Reason); reason != "" {
		return reason
	}
	if len(result.Labels) == 0 {
		return fallback
	}
	for _, label := range result.Labels {
		switch strings.TrimSpace(label) {
		case "10001":
			return "内容疑似包含广告营销信息，请修改后重新提交"
		case "20001":
			return "内容疑似涉及敏感时政信息，请修改后重新提交"
		case "20002":
			return "内容疑似包含色情低俗信息，请修改后重新提交"
		case "20003":
			return "内容疑似包含辱骂攻击信息，请修改后重新提交"
		case "20006":
			return "内容疑似包含违法违规信息，请修改后重新提交"
		case "20008":
			return "内容疑似包含欺诈风险信息，请修改后重新提交"
		case "20012":
			return "内容疑似包含低俗信息，请修改后重新提交"
		case "20013":
			return "内容疑似涉及版权风险，请修改后重新提交"
		case "21000":
			return "内容疑似存在安全风险，请修改后重新提交"
		}
	}
	return fallback
}

func normalizeContentAuditDecision(decision string) string {
	switch strings.TrimSpace(strings.ToLower(decision)) {
	case ContentAuditDecisionRisky:
		return ContentAuditDecisionRisky
	case ContentAuditDecisionReview:
		return ContentAuditDecisionReview
	default:
		return ContentAuditDecisionPass
	}
}

func BuildResourceAuditText(input ContentAuditInput, maxRunes int) string {
	if maxRunes <= 0 || maxRunes > 2500 {
		maxRunes = 2500
	}
	parts := make([]string, 0, 10)
	addPart := func(label string, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, label+"："+value)
		}
	}
	addPart("标题", input.Title)
	addPart("分类", input.Category)
	addPart("地区", input.District)
	addPart("数量", input.QuantityText)
	addPart("价格", input.PriceText)
	addPart("描述", input.Description)
	addPart("联系人", input.ContactName)
	addPart("微信", input.ContactWechat)
	if len(input.Tags) > 0 {
		addPart("标签", strings.Join(trimNonEmptyStrings(input.Tags), "，"))
	}
	if len(input.Attributes) > 0 {
		addPart("补充信息", auditAttributeText(input.Attributes))
	}
	return limitRunes(strings.Join(parts, "\n"), maxRunes)
}

func auditAttributeText(attributes model.JSONMap) string {
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := auditScalarText(attributes[key])
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, "，")
}

func auditScalarText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case bool:
		if typed {
			return "是"
		}
		return "否"
	case float64, float32, int, int64, int32, uint, uint64, uint32:
		return strings.TrimSpace(fmt.Sprint(typed))
	default:
		return ""
	}
}

func trimNonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func limitRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
