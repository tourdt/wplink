package contentaudit

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestMediaCheckCallbackRejectsRiskyImage(t *testing.T) {
	store := &fakeMediaCheckCallbackStore{
		completion: model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"},
	}
	logic := NewMediaCheckCallbackLogic(store)

	resp, err := logic.Handle(context.Background(), model.JSONMap{
		"trace_id": "trace-media",
		"errcode":  float64(0),
		"result": map[string]interface{}{
			"suggest": "risky",
			"label":   float64(20002),
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if resp.Status != model.ResourceStatusRejected || store.rejectedResourceID != "resource-1" {
		t.Fatalf("resp = %#v rejectedResourceID = %q, want rejected resource", resp, store.rejectedResourceID)
	}
	if store.completedInput.Status != resourceAuditTaskStatusRejected || store.rejectReason != "内容疑似包含色情低俗信息，请修改后重新提交" {
		t.Fatalf("completedInput = %#v rejectReason = %q, want risky image rejection", store.completedInput, store.rejectReason)
	}
}

func TestMediaCheckCallbackPublishesWhenAllImagesPass(t *testing.T) {
	store := &fakeMediaCheckCallbackStore{
		completion: model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"},
	}
	logic := NewMediaCheckCallbackLogic(store)

	resp, err := logic.Handle(context.Background(), model.JSONMap{
		"trace_id": "trace-media",
		"errcode":  float64(0),
		"result": map[string]interface{}{
			"suggest": "pass",
			"label":   float64(100),
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if resp.Status != model.ResourceStatusPublished || store.publishedResourceID != "resource-1" {
		t.Fatalf("resp = %#v publishedResourceID = %q, want published resource", resp, store.publishedResourceID)
	}
	if store.completedInput.Status != resourceAuditTaskStatusPass {
		t.Fatalf("completedInput = %#v, want pass task status", store.completedInput)
	}
}

func TestMediaCheckCallbackKeepsPendingWhenOtherImagesRemain(t *testing.T) {
	store := &fakeMediaCheckCallbackStore{
		completion: model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1", PendingCount: 1},
	}
	logic := NewMediaCheckCallbackLogic(store)

	resp, err := logic.Handle(context.Background(), model.JSONMap{
		"trace_id": "trace-media",
		"errcode":  float64(0),
		"result": map[string]interface{}{
			"suggest": "pass",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if resp.Status != model.ResourceStatusPending || store.publishedResourceID != "" || store.rejectedResourceID != "" {
		t.Fatalf("resp = %#v publishedResourceID = %q rejectedResourceID = %q, want pending only", resp, store.publishedResourceID, store.rejectedResourceID)
	}
}

type fakeMediaCheckCallbackStore struct {
	completedInput      model.ResourceContentAuditTaskResultInput
	completion          model.ResourceContentAuditTaskCompletion
	publishedResourceID string
	rejectedResourceID  string
	rejectReason        string
	completeErr         error
	publishErr          error
	rejectErr           error
}

func (s *fakeMediaCheckCallbackStore) CompleteResourceContentAuditTask(ctx context.Context, input model.ResourceContentAuditTaskResultInput) (model.ResourceContentAuditTaskCompletion, error) {
	s.completedInput = input
	if s.completeErr != nil {
		return model.ResourceContentAuditTaskCompletion{}, s.completeErr
	}
	return s.completion, nil
}

func (s *fakeMediaCheckCallbackStore) PublishResourceAfterAudit(ctx context.Context, resourceID string) (model.ReviewResourceResult, error) {
	s.publishedResourceID = resourceID
	if s.publishErr != nil {
		return model.ReviewResourceResult{}, s.publishErr
	}
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusPublished}, nil
}

func (s *fakeMediaCheckCallbackStore) RejectResourceAfterAudit(ctx context.Context, resourceID string, reason string) (model.ReviewResourceResult, error) {
	s.rejectedResourceID = resourceID
	s.rejectReason = reason
	if s.rejectErr != nil {
		return model.ReviewResourceResult{}, s.rejectErr
	}
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusRejected}, nil
}
