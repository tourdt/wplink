package contentaudit

import (
	"context"
	"database/sql"
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

	if resp.Status != model.ResourceStatusRejected || store.rejectedResourceID != "resource-1" || store.rejectedTraceID != "trace-media" {
		t.Fatalf("resp = %#v rejectedResourceID = %q, want rejected resource", resp, store.rejectedResourceID)
	}
	if store.completedInput.Status != resourceAuditTaskStatusRejected || store.rejectReason != "内容疑似包含色情低俗信息，请修改后重新提交" {
		t.Fatalf("completedInput = %#v rejectReason = %q, want risky image rejection", store.completedInput, store.rejectReason)
	}
	if len(store.auditDecisions) != 1 || store.auditDecisions[0].Decision != "risky" || len(store.auditDecisions[0].TraceIDs) != 1 || store.auditDecisions[0].TraceIDs[0] != "trace-media" {
		t.Fatalf("当前 trace 驳回成功后应记录风险决策: %+v", store.auditDecisions)
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

	if resp.Status != model.ResourceStatusPublished || store.publishedResourceID != "resource-1" || store.publishedTraceID != "trace-media" {
		t.Fatalf("resp = %#v publishedResourceID = %q, want published resource", resp, store.publishedResourceID)
	}
	if store.completedInput.Status != resourceAuditTaskStatusPass {
		t.Fatalf("completedInput = %#v, want pass task status", store.completedInput)
	}
}

func TestMediaCheckCallbackUsesCurrentTraceWhenQuotaFailureFallsBackToReject(t *testing.T) {
	store := &fakeMediaCheckCallbackStore{
		completion: model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"},
		publishErr: model.ErrPublishQuotaInsufficient,
	}
	logic := NewMediaCheckCallbackLogic(store)

	resp, err := logic.Handle(context.Background(), model.JSONMap{
		"trace_id": "trace-media",
		"errcode":  float64(0),
		"result":   map[string]interface{}{"suggest": "pass"},
	})
	if err != nil {
		t.Fatalf("额度不足后政策性驳回失败: %v", err)
	}
	if resp.Status != model.ResourceStatusRejected || store.publishedTraceID != "trace-media" || store.rejectedTraceID != "trace-media" {
		t.Fatalf("发布与回退驳回必须校验同一个当前 trace: resp=%+v publishTrace=%q rejectTrace=%q", resp, store.publishedTraceID, store.rejectedTraceID)
	}
	if store.rejectReason != "本月发布次数已用完，可开通 VIP 或购买发布包后重新提交" {
		t.Fatalf("政策性驳回原因错误: %q", store.rejectReason)
	}
}

func TestMediaCheckCallbackReturnsUnchangedWhenPolicyRejectMissesCurrentGeneration(t *testing.T) {
	tests := []struct {
		name       string
		publishErr error
	}{
		{name: "发布额度不足", publishErr: model.ErrPublishQuotaInsufficient},
		{name: "分类禁止发布", publishErr: model.ErrPublishDisabled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeMediaCheckCallbackStore{
				completion: model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"},
				publishErr: tt.publishErr,
				rejectErr:  sql.ErrNoRows,
			}
			logic := NewMediaCheckCallbackLogic(store)

			resp, err := logic.Handle(context.Background(), model.JSONMap{
				"trace_id": "trace-old",
				"errcode":  float64(0),
				"result":   map[string]interface{}{"suggest": "pass"},
			})
			if err != nil {
				t.Fatalf("政策性驳回代际 miss 不应返回错误: %v", err)
			}
			if resp.Status != "unchanged" || resp.Message != "图片审核回调已记录" {
				t.Fatalf("政策性驳回代际 miss 应保持当前资源状态: %+v", resp)
			}
		})
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

func TestMediaCheckCallbackDoesNotPublishTasksRebuiltByNewRetryGeneration(t *testing.T) {
	store := &fakeMediaCheckCallbackStore{
		completion:       model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"},
		completeReturned: make(chan struct{}),
		finalizeEntered:  make(chan struct{}),
		allowFinalize:    make(chan struct{}),
	}
	logic := NewMediaCheckCallbackLogic(store)
	result := make(chan struct {
		resp MediaCheckCallbackResp
		err  error
	}, 1)
	go func() {
		resp, err := logic.Handle(context.Background(), model.JSONMap{
			"trace_id": "trace-old",
			"errcode":  float64(0),
			"result":   map[string]interface{}{"suggest": "pass"},
		})
		result <- struct {
			resp MediaCheckCallbackResp
			err  error
		}{resp: resp, err: err}
	}()

	// 固定时序：旧任务已完成，finalize 尚未取得资源锁；此时新租约删除旧 trace 并重建 pending 任务集。
	<-store.completeReturned
	<-store.finalizeEntered
	store.generationRebuilt = true
	close(store.allowFinalize)

	got := <-result
	if got.err != nil {
		t.Fatalf("处理旧图片回调失败: %v", got.err)
	}
	if got.resp.Status != "unchanged" || store.publishedResourceID != "" {
		t.Fatalf("旧回调不得发布新一轮任务: resp=%+v publishedResourceID=%q", got.resp, store.publishedResourceID)
	}
}

func TestMediaCheckCallbackDoesNotRejectTasksRebuiltByNewRetryGeneration(t *testing.T) {
	store := &fakeMediaCheckCallbackStore{
		completion:       model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"},
		completeReturned: make(chan struct{}),
		finalizeEntered:  make(chan struct{}),
		allowFinalize:    make(chan struct{}),
	}
	logic := NewMediaCheckCallbackLogic(store)
	result := make(chan struct {
		resp MediaCheckCallbackResp
		err  error
	}, 1)
	go func() {
		resp, err := logic.Handle(context.Background(), model.JSONMap{
			"trace_id": "trace-old",
			"errcode":  float64(0),
			"result": map[string]interface{}{
				"suggest": "risky",
				"label":   float64(20002),
			},
		})
		result <- struct {
			resp MediaCheckCallbackResp
			err  error
		}{resp: resp, err: err}
	}()

	<-store.completeReturned
	<-store.finalizeEntered
	store.generationRebuilt = true
	close(store.allowFinalize)

	got := <-result
	if got.err != nil {
		t.Fatalf("处理旧风险图片回调失败: %v", got.err)
	}
	if got.resp.Status != "unchanged" || store.rejectedResourceID != "" {
		t.Fatalf("旧回调不得拒绝新一轮任务: resp=%+v rejectedResourceID=%q", got.resp, store.rejectedResourceID)
	}
	if len(store.auditDecisions) != 0 {
		t.Fatalf("旧风险 trace 已不属于当前任务集，不得污染新一代审核决策: %+v", store.auditDecisions)
	}
}

func TestMediaCheckCallbackDoesNotRetryTasksRebuiltByNewRetryGeneration(t *testing.T) {
	store := &fakeMediaCheckCallbackStore{
		completion:       model.ResourceContentAuditTaskCompletion{ResourceID: "resource-1"},
		completeReturned: make(chan struct{}),
		finalizeEntered:  make(chan struct{}),
		allowFinalize:    make(chan struct{}),
	}
	logic := NewMediaCheckCallbackLogic(store)
	result := make(chan struct {
		resp MediaCheckCallbackResp
		err  error
	}, 1)
	go func() {
		resp, err := logic.Handle(context.Background(), model.JSONMap{
			"trace_id": "trace-old",
			"errcode":  float64(45009),
		})
		result <- struct {
			resp MediaCheckCallbackResp
			err  error
		}{resp: resp, err: err}
	}()

	<-store.completeReturned
	<-store.finalizeEntered
	store.generationRebuilt = true
	close(store.allowFinalize)

	got := <-result
	if got.err != nil {
		t.Fatalf("处理旧失败图片回调失败: %v", got.err)
	}
	if got.resp.Status != "unchanged" || store.retriedResourceID != "" {
		t.Fatalf("旧失败回调不得把新一轮 pending 资源改回 audit_retry: resp=%+v retriedResourceID=%q", got.resp, store.retriedResourceID)
	}
}

type fakeMediaCheckCallbackStore struct {
	completedInput      model.ResourceContentAuditTaskResultInput
	completion          model.ResourceContentAuditTaskCompletion
	publishedResourceID string
	publishedTraceID    string
	rejectedResourceID  string
	rejectedTraceID     string
	rejectReason        string
	retriedResourceID   string
	retryTraceID        string
	auditDecisions      []model.ResourceAuditDecisionInput
	completeErr         error
	publishErr          error
	rejectErr           error
	completeReturned    chan struct{}
	finalizeEntered     chan struct{}
	allowFinalize       chan struct{}
	generationRebuilt   bool
}

func (s *fakeMediaCheckCallbackStore) CompleteResourceContentAuditTask(ctx context.Context, input model.ResourceContentAuditTaskResultInput) (model.ResourceContentAuditTaskCompletion, error) {
	s.completedInput = input
	if s.completeErr != nil {
		return model.ResourceContentAuditTaskCompletion{}, s.completeErr
	}
	if s.completeReturned != nil {
		close(s.completeReturned)
	}
	return s.completion, nil
}

func (s *fakeMediaCheckCallbackStore) PublishResourceAfterMediaAudit(ctx context.Context, resourceID string, traceID string) (model.ReviewResourceResult, error) {
	s.publishedTraceID = traceID
	if s.finalizeEntered != nil {
		close(s.finalizeEntered)
	}
	if s.allowFinalize != nil {
		<-s.allowFinalize
	}
	if s.generationRebuilt {
		return model.ReviewResourceResult{}, sql.ErrNoRows
	}
	s.publishedResourceID = resourceID
	if s.publishErr != nil {
		return model.ReviewResourceResult{}, s.publishErr
	}
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusPublished}, nil
}

func (s *fakeMediaCheckCallbackStore) RejectResourceAfterMediaAudit(ctx context.Context, resourceID string, traceID string, reason string) (model.ReviewResourceResult, error) {
	s.rejectedTraceID = traceID
	if s.finalizeEntered != nil {
		close(s.finalizeEntered)
	}
	if s.allowFinalize != nil {
		<-s.allowFinalize
	}
	if s.generationRebuilt {
		return model.ReviewResourceResult{}, sql.ErrNoRows
	}
	s.rejectedResourceID = resourceID
	s.rejectReason = reason
	if s.rejectErr != nil {
		return model.ReviewResourceResult{}, s.rejectErr
	}
	return model.ReviewResourceResult{ID: resourceID, Status: model.ResourceStatusRejected}, nil
}

func (s *fakeMediaCheckCallbackStore) MarkResourceAuditRetryAfterMediaAudit(ctx context.Context, resourceID string, traceID string, _ string) (int64, error) {
	s.retryTraceID = traceID
	if s.finalizeEntered != nil {
		close(s.finalizeEntered)
	}
	if s.allowFinalize != nil {
		<-s.allowFinalize
	}
	if s.generationRebuilt {
		return 0, sql.ErrNoRows
	}
	s.retriedResourceID = resourceID
	return 1, nil
}

// MarkResourceAuditRetry 只用于回归测试演示旧实现的无 trace 写入会跨代流转资源。
func (s *fakeMediaCheckCallbackStore) MarkResourceAuditRetry(_ context.Context, resourceID string, _ string) (int64, error) {
	if s.finalizeEntered != nil {
		close(s.finalizeEntered)
	}
	if s.allowFinalize != nil {
		<-s.allowFinalize
	}
	s.retriedResourceID = resourceID
	return 1, nil
}

func (s *fakeMediaCheckCallbackStore) MarkResourceManualReview(context.Context, string, string) error {
	return nil
}

func (s *fakeMediaCheckCallbackStore) RecordResourceAuditDecision(_ context.Context, input model.ResourceAuditDecisionInput) error {
	s.auditDecisions = append(s.auditDecisions, input)
	return nil
}
