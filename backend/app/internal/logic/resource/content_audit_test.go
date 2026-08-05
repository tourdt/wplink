package resource

import (
	"context"
	"errors"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestRetryResourceContentAuditUsesLeaseAwareAdapter(t *testing.T) {
	guard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	store := &fakeResourceAuditLeaseStore{
		snapshot: model.ResourceAuditSnapshot{
			ID:         guard.ResourceID,
			MerchantID: "merchant-1",
			TypeCode:   "supply",
			OpenID:     "openid-1",
		},
		publishResult: model.ReviewResourceResult{ID: guard.ResourceID, Status: model.ResourceStatusPublished},
	}
	auditor := &fakeRetryContentAuditor{result: ContentAuditResult{Decision: ContentAuditDecisionPass}}

	outcome, err := RetryResourceContentAudit(context.Background(), store, auditor, guard)
	if err != nil {
		t.Fatalf("执行租约内容审核重试失败: %v", err)
	}
	if outcome.ID != guard.ResourceID || outcome.Status != model.ResourceStatusPublished {
		t.Fatalf("审核重试结果错误: %+v", outcome)
	}
	if store.snapshotGuard != guard || store.publishGuard != guard {
		t.Fatalf("租约 guard 未贯穿读取和发布: snapshot=%+v publish=%+v", store.snapshotGuard, store.publishGuard)
	}
	if auditor.calls != 1 || auditor.input.ResourceID != guard.ResourceID {
		t.Fatalf("审核器调用错误: calls=%d input=%+v", auditor.calls, auditor.input)
	}
	if len(store.decisions) != 1 || store.decisions[0].ResourceID != guard.ResourceID || store.decisions[0].Decision != ContentAuditDecisionPass {
		t.Fatalf("审核决策未转发: %+v", store.decisions)
	}
}

func TestRetryResourceContentAuditCreatesMediaTasksWithLease(t *testing.T) {
	guard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	store := newFakeResourceAuditLeaseStore(guard)
	auditor := &fakeRetryContentAuditor{result: ContentAuditResult{
		Decision: ContentAuditDecisionPass,
		MediaTasks: []ContentAuditMediaTask{{
			TraceID: "trace-1", MediaURL: "https://example.com/image.png",
		}},
	}}

	outcome, err := RetryResourceContentAudit(context.Background(), store, auditor, guard)
	if err != nil {
		t.Fatalf("创建租约媒体审核任务失败: %v", err)
	}
	if outcome.Status != model.ResourceStatusPending {
		t.Fatalf("媒体任务创建后状态错误: got=%s want=%s", outcome.Status, model.ResourceStatusPending)
	}
	if store.createTasksGuard != guard || len(store.createdTasks) != 1 || store.createdTasks[0].TraceID != "trace-1" {
		t.Fatalf("媒体任务租约转发错误: guard=%+v tasks=%+v", store.createTasksGuard, store.createdTasks)
	}
	if store.publishGuard != (model.ResourceAuditGuard{}) {
		t.Fatalf("存在媒体任务时不应直接发布: %+v", store.publishGuard)
	}
}

func TestRetryResourceContentAuditRejectsRiskyResultWithLease(t *testing.T) {
	guard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	store := newFakeResourceAuditLeaseStore(guard)
	store.rejectResult = model.ReviewResourceResult{ID: guard.ResourceID, Status: model.ResourceStatusRejected}
	auditor := &fakeRetryContentAuditor{result: ContentAuditResult{Decision: ContentAuditDecisionRisky, Reason: "命中风险"}}

	outcome, err := RetryResourceContentAudit(context.Background(), store, auditor, guard)
	if err != nil {
		t.Fatalf("租约审核拒绝失败: %v", err)
	}
	if outcome.Status != model.ResourceStatusRejected || store.rejectGuard != guard || store.rejectReason != "命中风险" {
		t.Fatalf("租约拒绝结果错误: outcome=%+v guard=%+v reason=%s", outcome, store.rejectGuard, store.rejectReason)
	}
}

func TestRetryResourceContentAuditMarksDependencyRetryWithLease(t *testing.T) {
	guard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	store := newFakeResourceAuditLeaseStore(guard)
	store.retryCount = 2
	auditor := &fakeRetryContentAuditor{err: errors.New("微信审核请求超时")}

	outcome, err := RetryResourceContentAudit(context.Background(), store, auditor, guard)
	if err != nil {
		t.Fatalf("租约审核再次进入重试失败: %v", err)
	}
	if outcome.Status != model.ResourceStatusAuditRetry || store.retryGuard != guard || store.retryReason != "微信审核请求超时" {
		t.Fatalf("再次重试的租约结果错误: outcome=%+v guard=%+v reason=%s", outcome, store.retryGuard, store.retryReason)
	}
}

func TestRetryResourceContentAuditMovesMissingIdentityToManualReviewWithLease(t *testing.T) {
	guard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	store := newFakeResourceAuditLeaseStore(guard)
	store.snapshot.OpenID = ""
	auditor := &fakeRetryContentAuditor{}

	outcome, err := RetryResourceContentAudit(context.Background(), store, auditor, guard)
	if err != nil {
		t.Fatalf("缺少审核身份转人工失败: %v", err)
	}
	if outcome.Status != model.ResourceStatusManualReview || store.manualGuard != guard {
		t.Fatalf("转人工租约结果错误: outcome=%+v guard=%+v", outcome, store.manualGuard)
	}
	if auditor.calls != 0 {
		t.Fatalf("缺少审核身份不应调用审核器: calls=%d", auditor.calls)
	}
}

func TestRetryResourceContentAuditValidatesGuard(t *testing.T) {
	tests := []struct {
		name  string
		guard model.ResourceAuditGuard
		want  string
	}{
		{name: "empty resource", guard: model.ResourceAuditGuard{ProcessingBy: "api-a"}, want: "资源标识不能为空"},
		{name: "empty owner", guard: model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "  "}, want: "实例标识不能为空"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeResourceAuditLeaseStore{}
			auditor := &fakeRetryContentAuditor{}

			_, err := RetryResourceContentAudit(context.Background(), store, auditor, tt.guard)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("guard 校验错误不符合预期: err=%v wantContains=%s", err, tt.want)
			}
			if store.snapshotCalls != 0 || auditor.calls != 0 {
				t.Fatalf("无效 guard 不应访问存储或审核器: snapshotCalls=%d auditorCalls=%d", store.snapshotCalls, auditor.calls)
			}
		})
	}
}

func TestRetryResourceContentAuditPreservesLeaseLostFromResultTransitions(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*fakeResourceAuditLeaseStore, *fakeRetryContentAuditor)
	}{
		{
			name: "publish",
			prepare: func(store *fakeResourceAuditLeaseStore, auditor *fakeRetryContentAuditor) {
				store.publishErr = model.ErrResourceAuditLeaseLost
				auditor.result = ContentAuditResult{Decision: ContentAuditDecisionPass}
			},
		},
		{
			name: "reject",
			prepare: func(store *fakeResourceAuditLeaseStore, auditor *fakeRetryContentAuditor) {
				store.rejectErr = model.ErrResourceAuditLeaseLost
				auditor.result = ContentAuditResult{Decision: ContentAuditDecisionRisky}
			},
		},
		{
			name: "media tasks",
			prepare: func(store *fakeResourceAuditLeaseStore, auditor *fakeRetryContentAuditor) {
				store.createTasksErr = model.ErrResourceAuditLeaseLost
				auditor.result = ContentAuditResult{MediaTasks: []ContentAuditMediaTask{{TraceID: "trace-1"}}}
			},
		},
		{
			name: "dependency retry",
			prepare: func(store *fakeResourceAuditLeaseStore, auditor *fakeRetryContentAuditor) {
				store.retryErr = model.ErrResourceAuditLeaseLost
				auditor.err = errors.New("微信审核请求超时")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
			store := newFakeResourceAuditLeaseStore(guard)
			auditor := &fakeRetryContentAuditor{}
			tt.prepare(store, auditor)

			_, err := RetryResourceContentAudit(context.Background(), store, auditor, guard)
			if !errors.Is(err, model.ErrResourceAuditLeaseLost) {
				t.Fatalf("结果转换丢失租约后应保留可识别错误: %v", err)
			}
		})
	}
}

type fakeResourceAuditLeaseStore struct {
	snapshot      model.ResourceAuditSnapshot
	snapshotErr   error
	snapshotGuard model.ResourceAuditGuard
	snapshotCalls int

	createTasksGuard model.ResourceAuditGuard
	createdTasks     []model.ResourceContentAuditTaskInput
	createTasksErr   error

	publishGuard  model.ResourceAuditGuard
	publishResult model.ReviewResourceResult
	publishErr    error

	rejectGuard  model.ResourceAuditGuard
	rejectReason string
	rejectResult model.ReviewResourceResult
	rejectErr    error

	retryGuard  model.ResourceAuditGuard
	retryReason string
	retryCount  int64
	retryErr    error

	manualGuard  model.ResourceAuditGuard
	manualReason string
	manualErr    error

	decisions []model.ResourceAuditDecisionInput
}

func newFakeResourceAuditLeaseStore(guard model.ResourceAuditGuard) *fakeResourceAuditLeaseStore {
	return &fakeResourceAuditLeaseStore{
		snapshot: model.ResourceAuditSnapshot{
			ID:         guard.ResourceID,
			MerchantID: "merchant-1",
			TypeCode:   "supply",
			OpenID:     "openid-1",
		},
		publishResult: model.ReviewResourceResult{ID: guard.ResourceID, Status: model.ResourceStatusPublished},
	}
}

func (s *fakeResourceAuditLeaseStore) GetLeasedResourceAuditSnapshot(_ context.Context, guard model.ResourceAuditGuard) (model.ResourceAuditSnapshot, error) {
	s.snapshotCalls++
	s.snapshotGuard = guard
	return s.snapshot, s.snapshotErr
}

func (s *fakeResourceAuditLeaseStore) CreateLeasedResourceContentAuditTasks(_ context.Context, guard model.ResourceAuditGuard, tasks []model.ResourceContentAuditTaskInput) error {
	s.createTasksGuard = guard
	s.createdTasks = append([]model.ResourceContentAuditTaskInput(nil), tasks...)
	return s.createTasksErr
}

func (s *fakeResourceAuditLeaseStore) PublishLeasedResourceAfterAudit(_ context.Context, guard model.ResourceAuditGuard) (model.ReviewResourceResult, error) {
	s.publishGuard = guard
	return s.publishResult, s.publishErr
}

func (s *fakeResourceAuditLeaseStore) RejectLeasedResourceAfterAudit(_ context.Context, guard model.ResourceAuditGuard, reason string) (model.ReviewResourceResult, error) {
	s.rejectGuard = guard
	s.rejectReason = reason
	return s.rejectResult, s.rejectErr
}

func (s *fakeResourceAuditLeaseStore) MarkLeasedResourceAuditRetry(_ context.Context, guard model.ResourceAuditGuard, reason string) (int64, error) {
	s.retryGuard = guard
	s.retryReason = reason
	return s.retryCount, s.retryErr
}

func (s *fakeResourceAuditLeaseStore) MarkLeasedResourceManualReview(_ context.Context, guard model.ResourceAuditGuard, reason string) error {
	s.manualGuard = guard
	s.manualReason = reason
	return s.manualErr
}

func (s *fakeResourceAuditLeaseStore) RecordResourceAuditDecision(_ context.Context, input model.ResourceAuditDecisionInput) error {
	s.decisions = append(s.decisions, input)
	return nil
}

type fakeRetryContentAuditor struct {
	result ContentAuditResult
	err    error
	calls  int
	input  ContentAuditInput
}

func (a *fakeRetryContentAuditor) AuditResource(_ context.Context, input ContentAuditInput) (ContentAuditResult, error) {
	a.calls++
	a.input = input
	return a.result, a.err
}
