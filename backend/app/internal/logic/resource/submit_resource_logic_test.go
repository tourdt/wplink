package resource

import (
	"context"
	"database/sql"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestSubmitResourceRejectsEmptyID(t *testing.T) {
	logic := NewSubmitResourceLogic(&fakeSubmitResourceStore{})

	_, err := logic.SubmitResource(context.Background(), " ")

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestSubmitResourceSetsPending(t *testing.T) {
	store := &fakeSubmitResourceStore{result: model.SubmitResourceResult{ID: "resource-1", Status: "pending"}}
	logic := NewSubmitResourceLogic(store)

	resp, err := logic.SubmitResource(context.Background(), " resource-1 ")
	if err != nil {
		t.Fatalf("SubmitResource() error = %v", err)
	}

	if store.resourceID != "resource-1" {
		t.Fatalf("resourceID = %q, want trimmed resource-1", store.resourceID)
	}
	if resp.Status != "pending" || resp.Message != "已提交，等待系统安全检测" {
		t.Fatalf("resp = %#v, want pending submit message", resp)
	}
}

func TestSubmitResourceBlocksRiskyContentAudit(t *testing.T) {
	store := &fakeSubmitResourceStore{
		result: model.SubmitResourceResult{ID: "resource-1", Status: "pending"},
		snapshot: model.ResourceAuditSnapshot{
			ID:          "resource-1",
			MerchantID:  "merchant-1",
			TypeCode:    "inventory",
			OpenID:      "openid-1",
			Title:       "待提交库存",
			Description: "草稿内容",
		},
	}
	auditor := &fakeContentAuditor{result: ContentAuditResult{Decision: ContentAuditDecisionRisky, Labels: []string{"20002"}}}
	logic := NewSubmitResourceLogic(store, auditor)

	resp, err := logic.SubmitResource(context.Background(), " resource-1 ")

	if err != nil {
		t.Fatalf("SubmitResource() error = %v", err)
	}
	if store.resourceID != "resource-1" || resp.Status != model.ResourceStatusRejected || store.rejectedResourceID != "resource-1" {
		t.Fatalf("resourceID = %q resp = %#v rejectedResourceID = %q, want submitted then rejected", store.resourceID, resp, store.rejectedResourceID)
	}
	if auditor.input.ResourceID != "resource-1" || auditor.input.OpenID != "openid-1" {
		t.Fatalf("audit input = %#v, want draft snapshot", auditor.input)
	}
	if store.rejectReason != "内容疑似包含色情低俗信息，请修改后重新提交" {
		t.Fatalf("rejectReason = %q, want mapped label reason", store.rejectReason)
	}
}

func TestSubmitResourcePublishesWhenAuditPasses(t *testing.T) {
	store := &fakeSubmitResourceStore{
		result: model.SubmitResourceResult{ID: "resource-1", Status: "pending"},
		snapshot: model.ResourceAuditSnapshot{
			ID:          "resource-1",
			MerchantID:  "merchant-1",
			TypeCode:    "inventory",
			OpenID:      "openid-1",
			Title:       "待提交库存",
			Description: "草稿内容",
		},
	}
	auditor := &fakeContentAuditor{result: ContentAuditResult{Decision: ContentAuditDecisionPass}}
	logic := NewSubmitResourceLogic(store, auditor)

	resp, err := logic.SubmitResource(context.Background(), " resource-1 ")
	if err != nil {
		t.Fatalf("SubmitResource() error = %v", err)
	}

	if resp.Status != model.ResourceStatusPublished || store.publishedResourceID != "resource-1" {
		t.Fatalf("resp = %#v publishedResourceID = %q, want auto published", resp, store.publishedResourceID)
	}
}

func TestSubmitResourceCreatesImageAuditTasks(t *testing.T) {
	store := &fakeSubmitResourceStore{
		result: model.SubmitResourceResult{ID: "resource-1", Status: "pending"},
		snapshot: model.ResourceAuditSnapshot{
			ID:         "resource-1",
			MerchantID: "merchant-1",
			TypeCode:   "inventory",
			OpenID:     "openid-1",
			Title:      "待提交库存",
		},
	}
	auditor := &fakeContentAuditor{result: ContentAuditResult{
		Decision: ContentAuditDecisionPass,
		MediaTasks: []ContentAuditMediaTask{{
			TraceID:  "trace-media",
			MediaURL: "https://cdn.example.com/a.png",
		}},
	}}
	logic := NewSubmitResourceLogic(store, auditor)

	resp, err := logic.SubmitResource(context.Background(), "resource-1")
	if err != nil {
		t.Fatalf("SubmitResource() error = %v", err)
	}

	if resp.Status != model.ResourceStatusPending || len(store.auditTasks) != 1 || store.auditTasks[0].TraceID != "trace-media" {
		t.Fatalf("resp = %#v auditTasks = %#v, want pending image audit task", resp, store.auditTasks)
	}
}

func TestSubmitResourceQueuesRetryWhenOpenIDIsMissing(t *testing.T) {
	store := &fakeSubmitResourceStore{
		result: model.SubmitResourceResult{ID: "resource-1", Status: model.ResourceStatusPending},
		snapshot: model.ResourceAuditSnapshot{
			ID:         "resource-1",
			MerchantID: "merchant-1",
			TypeCode:   "inventory",
			Title:      "待提交库存",
		},
	}
	logic := NewSubmitResourceLogic(store, &fakeContentAuditor{})

	resp, err := logic.SubmitResource(context.Background(), "resource-1")
	if err != nil {
		t.Fatalf("SubmitResource() error = %v", err)
	}
	if resp.Status != model.ResourceStatusAuditRetry || store.auditRetryResourceID != "resource-1" {
		t.Fatalf("resp = %#v retryResourceID = %q, want automatic audit retry", resp, store.auditRetryResourceID)
	}
}

func TestSubmitResourceMapsUnavailableResourceToStateConflict(t *testing.T) {
	store := &fakeSubmitResourceStore{err: sql.ErrNoRows}
	logic := NewSubmitResourceLogic(store)

	_, err := logic.SubmitResource(context.Background(), "resource-1")

	if errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("error code = %q, want state conflict", errx.CodeOf(err))
	}
}

func TestSubmitResourceRequiresSavedDraft(t *testing.T) {
	store := &fakeSubmitResourceStore{err: sql.ErrNoRows}
	logic := NewSubmitResourceLogic(store)

	_, err := logic.SubmitResource(context.Background(), "rejected-resource")

	if errx.CodeOf(err) != errx.CodeStateConflict {
		t.Fatalf("error code = %q, want state conflict", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "请先编辑并保存草稿后再提交发布" {
		t.Fatalf("message = %q, want save draft first", errx.PublicMessage(err))
	}
}

func TestSubmitResourceMapsPublishQuotaInsufficient(t *testing.T) {
	store := &fakeSubmitResourceStore{err: model.ErrPublishQuotaInsufficient}
	logic := NewSubmitResourceLogic(store)

	_, err := logic.SubmitResource(context.Background(), "resource-1")

	if errx.CodeOf(err) != errx.CodeQuotaNotEnough {
		t.Fatalf("error code = %q, want quota not enough", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "本月免费发布次数已用完，可购买发布次数后继续发布" {
		t.Fatalf("message = %q, want publish quota upsell message", errx.PublicMessage(err))
	}
}

func TestSubmitResourceMapsDisabledPublishCategory(t *testing.T) {
	store := &fakeSubmitResourceStore{err: model.ErrPublishDisabled}
	logic := NewSubmitResourceLogic(store)

	_, err := logic.SubmitResource(context.Background(), "resource-1")

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "该分类暂不开放发布" {
		t.Fatalf("message = %q, want disabled publish message", errx.PublicMessage(err))
	}
}

type fakeSubmitResourceStore struct {
	resourceID           string
	result               model.SubmitResourceResult
	snapshot             model.ResourceAuditSnapshot
	err                  error
	auditTasks           []model.ResourceContentAuditTaskInput
	publishedResourceID  string
	rejectedResourceID   string
	rejectReason         string
	publishErr           error
	rejectErr            error
	auditRetryResourceID string
	auditRetryReason     string
}

func (s *fakeSubmitResourceStore) SubmitResourceForReview(ctx context.Context, resourceID string) (model.SubmitResourceResult, error) {
	s.resourceID = resourceID
	return s.result, s.err
}

func (s *fakeSubmitResourceStore) GetResourceAuditSnapshot(ctx context.Context, resourceID string) (model.ResourceAuditSnapshot, error) {
	if s.snapshot.ID == "" {
		return model.ResourceAuditSnapshot{ID: resourceID}, nil
	}
	return s.snapshot, nil
}

func (s *fakeSubmitResourceStore) CreateResourceContentAuditTasks(ctx context.Context, resourceID string, tasks []model.ResourceContentAuditTaskInput) error {
	s.auditTasks = append([]model.ResourceContentAuditTaskInput(nil), tasks...)
	return nil
}

func (s *fakeSubmitResourceStore) PublishResourceAfterAudit(ctx context.Context, resourceID string) (model.ReviewResourceResult, error) {
	s.publishedResourceID = resourceID
	if s.publishErr != nil {
		return model.ReviewResourceResult{}, s.publishErr
	}
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusPublished}, nil
}

func (s *fakeSubmitResourceStore) RejectResourceAfterAudit(ctx context.Context, resourceID string, reason string) (model.ReviewResourceResult, error) {
	s.rejectedResourceID = resourceID
	s.rejectReason = reason
	if s.rejectErr != nil {
		return model.ReviewResourceResult{}, s.rejectErr
	}
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusRejected}, nil
}

func (s *fakeSubmitResourceStore) MarkResourceAuditRetry(ctx context.Context, resourceID string, reason string) (int64, error) {
	s.auditRetryResourceID = resourceID
	s.auditRetryReason = reason
	return 1, nil
}

func (s *fakeSubmitResourceStore) MarkResourceManualReview(ctx context.Context, resourceID string, reason string) error {
	return nil
}
