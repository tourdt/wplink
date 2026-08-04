package task

import (
	"context"
	"errors"
	"testing"
	"time"

	resourcelogic "wplink/backend/app/internal/logic/resource"
	"wplink/backend/app/internal/model"
)

func TestContentAuditRetryTaskClaimsAndProcessesWithInstanceLease(t *testing.T) {
	store := &fakeContentAuditRetryStore{
		claims: []model.ResourceAuditRetryClaim{{ResourceID: "resource-1", RetryCount: 1, ProcessingBy: "api-a"}},
		snapshots: map[model.ResourceAuditGuard]model.ResourceAuditSnapshot{
			{ResourceID: "resource-1", ProcessingBy: "api-a"}: auditRetrySnapshot("resource-1"),
		},
	}
	auditor := &fakeContentAuditRetryAuditor{result: resourcelogic.ContentAuditResult{Decision: resourcelogic.ContentAuditDecisionPass}}
	task := NewContentAuditRetryTask(store, auditor, 20, 3, "api-a", 2*time.Minute)

	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("执行内容审核重试失败: %v", err)
	}
	wantGuard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	if store.claimProcessingBy != "api-a" || store.claimLeaseDuration != 2*time.Minute || store.processedGuards[0] != wantGuard {
		t.Fatalf("租约参数传递错误: owner=%s duration=%s guards=%+v", store.claimProcessingBy, store.claimLeaseDuration, store.processedGuards)
	}
	if store.claimBatchSize != 20 || store.claimMaxRetries != 3 {
		t.Fatalf("领取参数错误: batch=%d maxRetries=%d", store.claimBatchSize, store.claimMaxRetries)
	}
	if result.RetriedCount != 1 {
		t.Fatalf("重试数量错误: got=%d want=1", result.RetriedCount)
	}
}

func TestContentAuditRetryTaskReturnsClaimFailure(t *testing.T) {
	wantErr := errors.New("领取审核重试失败")
	store := &fakeContentAuditRetryStore{claimErr: wantErr}
	task := NewContentAuditRetryTask(store, &fakeContentAuditRetryAuditor{}, 20, 3, "api-a", 2*time.Minute)

	_, err := task.Run(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("领取失败错误未返回: got=%v want=%v", err, wantErr)
	}
}

func TestContentAuditRetryTaskReturnsCanceledWhenClaimReturnsEmptyAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &fakeContentAuditRetryStore{
		staleCount: 2,
		onClaim:    cancel,
	}
	task := NewContentAuditRetryTask(store, &fakeContentAuditRetryAuditor{}, 20, 3, "api-a", 2*time.Minute)

	result, err := task.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("领取返回空集合时仍应感知 context 取消: %v", err)
	}
	if result.StaleCount != 2 || result.RetriedCount != 0 || result.ManualReviewCount != 0 {
		t.Fatalf("取消时应保留已完成的过期任务统计: %+v", result)
	}
}

func TestContentAuditRetryTaskContinuesAfterSingleLeaseLost(t *testing.T) {
	oldGuard := model.ResourceAuditGuard{ResourceID: "resource-old", ProcessingBy: "api-a"}
	validGuard := model.ResourceAuditGuard{ResourceID: "resource-valid", ProcessingBy: "api-a"}
	store := &fakeContentAuditRetryStore{
		claims: []model.ResourceAuditRetryClaim{
			{ResourceID: oldGuard.ResourceID, RetryCount: 1, ProcessingBy: oldGuard.ProcessingBy},
			{ResourceID: validGuard.ResourceID, RetryCount: 1, ProcessingBy: validGuard.ProcessingBy},
		},
		snapshots: map[model.ResourceAuditGuard]model.ResourceAuditSnapshot{
			validGuard: auditRetrySnapshot(validGuard.ResourceID),
		},
		snapshotErrors: map[model.ResourceAuditGuard]error{oldGuard: model.ErrResourceAuditLeaseLost},
	}
	auditor := &fakeContentAuditRetryAuditor{result: resourcelogic.ContentAuditResult{Decision: resourcelogic.ContentAuditDecisionPass}}
	task := NewContentAuditRetryTask(store, auditor, 20, 3, "api-a", 2*time.Minute)

	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("单条租约丢失不应中断批次: %v", err)
	}
	if result.RetriedCount != 1 || auditor.calls != 1 {
		t.Fatalf("租约丢失后的继续处理结果错误: result=%+v auditorCalls=%d", result, auditor.calls)
	}
	if len(store.processedGuards) != 1 || store.processedGuards[0] != validGuard {
		t.Fatalf("旧租约不应覆盖结果: processed=%+v", store.processedGuards)
	}
}

func TestContentAuditRetryTaskStopsBatchWhenContextTimesOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	guardOne := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-a"}
	guardTwo := model.ResourceAuditGuard{ResourceID: "resource-2", ProcessingBy: "api-a"}
	store := &fakeContentAuditRetryStore{
		claims: []model.ResourceAuditRetryClaim{
			{ResourceID: guardOne.ResourceID, RetryCount: 1, ProcessingBy: guardOne.ProcessingBy},
			{ResourceID: guardTwo.ResourceID, RetryCount: 1, ProcessingBy: guardTwo.ProcessingBy},
		},
		snapshots: map[model.ResourceAuditGuard]model.ResourceAuditSnapshot{
			guardOne: auditRetrySnapshot(guardOne.ResourceID),
			guardTwo: auditRetrySnapshot(guardTwo.ResourceID),
		},
	}
	auditor := &fakeContentAuditRetryAuditor{
		err: context.Canceled,
		onAudit: func(resourcelogic.ContentAuditInput) {
			cancel()
		},
	}
	task := NewContentAuditRetryTask(store, auditor, 20, 3, "api-a", 2*time.Minute)

	result, err := task.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("context 结束后应立即停止批次: %v", err)
	}
	if result.RetriedCount != 0 || auditor.calls != 1 {
		t.Fatalf("context 结束后不应处理后续资源: result=%+v auditorCalls=%d", result, auditor.calls)
	}
}

func TestContentAuditRetryTaskOnlyNewLeaseOwnerCanComplete(t *testing.T) {
	oldGuard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-old"}
	newGuard := model.ResourceAuditGuard{ResourceID: "resource-1", ProcessingBy: "api-new"}
	store := &fakeContentAuditRetryStore{
		claims: []model.ResourceAuditRetryClaim{
			{ResourceID: oldGuard.ResourceID, RetryCount: 1, ProcessingBy: oldGuard.ProcessingBy},
			{ResourceID: newGuard.ResourceID, RetryCount: 2, ProcessingBy: newGuard.ProcessingBy},
		},
		snapshots: map[model.ResourceAuditGuard]model.ResourceAuditSnapshot{
			newGuard: auditRetrySnapshot(newGuard.ResourceID),
		},
		snapshotErrors: map[model.ResourceAuditGuard]error{oldGuard: model.ErrResourceAuditLeaseLost},
	}
	auditor := &fakeContentAuditRetryAuditor{result: resourcelogic.ContentAuditResult{Decision: resourcelogic.ContentAuditDecisionPass}}
	task := NewContentAuditRetryTask(store, auditor, 20, 3, "api-new", 2*time.Minute)

	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("新租约持有者处理失败: %v", err)
	}
	if result.RetriedCount != 1 || len(store.processedGuards) != 1 || store.processedGuards[0] != newGuard {
		t.Fatalf("只能由新租约持有者落库: result=%+v processed=%+v", result, store.processedGuards)
	}
}

func TestContentAuditRetryTaskManualClaimDoesNotCallAuditor(t *testing.T) {
	store := &fakeContentAuditRetryStore{
		claims: []model.ResourceAuditRetryClaim{{ResourceID: "resource-manual", RetryCount: 5, ManualReview: true}},
	}
	auditor := &fakeContentAuditRetryAuditor{}
	task := NewContentAuditRetryTask(store, auditor, 20, 5, "api-a", 2*time.Minute)

	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("处理人工复核 claim 失败: %v", err)
	}
	if result.ManualReviewCount != 1 || result.RetriedCount != 0 || auditor.calls != 0 {
		t.Fatalf("人工复核 claim 不应外呼: result=%+v auditorCalls=%d", result, auditor.calls)
	}
	if store.snapshotCalls != 0 {
		t.Fatalf("人工复核 claim 不应进入审核逻辑: snapshotCalls=%d", store.snapshotCalls)
	}
}

type fakeContentAuditRetryStore struct {
	staleCount int64
	staleErr   error

	claims                []model.ResourceAuditRetryClaim
	claimErr              error
	claimBatchSize        int64
	claimMaxRetries       int64
	claimProcessingBy     string
	claimLeaseDuration    time.Duration
	onClaim               func()
	snapshots             map[model.ResourceAuditGuard]model.ResourceAuditSnapshot
	snapshotErrors        map[model.ResourceAuditGuard]error
	snapshotCalls         int
	processedGuards       []model.ResourceAuditGuard
	manualProcessedGuards []model.ResourceAuditGuard
}

func (s *fakeContentAuditRetryStore) MarkStaleContentAuditTasksForRetry(context.Context, time.Time) (int64, error) {
	return s.staleCount, s.staleErr
}

func (s *fakeContentAuditRetryStore) ClaimDueResourceAuditRetries(_ context.Context, batchSize int64, maxRetries int64, processingBy string, leaseDuration time.Duration) ([]model.ResourceAuditRetryClaim, error) {
	s.claimBatchSize = batchSize
	s.claimMaxRetries = maxRetries
	s.claimProcessingBy = processingBy
	s.claimLeaseDuration = leaseDuration
	if s.onClaim != nil {
		s.onClaim()
	}
	return append([]model.ResourceAuditRetryClaim(nil), s.claims...), s.claimErr
}

func (s *fakeContentAuditRetryStore) GetLeasedResourceAuditSnapshot(ctx context.Context, guard model.ResourceAuditGuard) (model.ResourceAuditSnapshot, error) {
	s.snapshotCalls++
	if err := ctx.Err(); err != nil {
		return model.ResourceAuditSnapshot{}, err
	}
	if err := s.snapshotErrors[guard]; err != nil {
		return model.ResourceAuditSnapshot{}, err
	}
	return s.snapshots[guard], nil
}

func (s *fakeContentAuditRetryStore) CreateLeasedResourceContentAuditTasks(ctx context.Context, guard model.ResourceAuditGuard, _ []model.ResourceContentAuditTaskInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.processedGuards = append(s.processedGuards, guard)
	return nil
}

func (s *fakeContentAuditRetryStore) PublishLeasedResourceAfterAudit(ctx context.Context, guard model.ResourceAuditGuard) (model.ReviewResourceResult, error) {
	if err := ctx.Err(); err != nil {
		return model.ReviewResourceResult{}, err
	}
	s.processedGuards = append(s.processedGuards, guard)
	return model.ReviewResourceResult{ID: guard.ResourceID, Status: model.ResourceStatusPublished}, nil
}

func (s *fakeContentAuditRetryStore) RejectLeasedResourceAfterAudit(ctx context.Context, guard model.ResourceAuditGuard, _ string) (model.ReviewResourceResult, error) {
	if err := ctx.Err(); err != nil {
		return model.ReviewResourceResult{}, err
	}
	s.processedGuards = append(s.processedGuards, guard)
	return model.ReviewResourceResult{ID: guard.ResourceID, Status: model.ResourceStatusRejected}, nil
}

func (s *fakeContentAuditRetryStore) MarkLeasedResourceAuditRetry(ctx context.Context, guard model.ResourceAuditGuard, _ string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.processedGuards = append(s.processedGuards, guard)
	return 1, nil
}

func (s *fakeContentAuditRetryStore) MarkLeasedResourceManualReview(ctx context.Context, guard model.ResourceAuditGuard, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.manualProcessedGuards = append(s.manualProcessedGuards, guard)
	return nil
}

type fakeContentAuditRetryAuditor struct {
	result  resourcelogic.ContentAuditResult
	err     error
	calls   int
	onAudit func(resourcelogic.ContentAuditInput)
}

func (a *fakeContentAuditRetryAuditor) AuditResource(_ context.Context, input resourcelogic.ContentAuditInput) (resourcelogic.ContentAuditResult, error) {
	a.calls++
	if a.onAudit != nil {
		a.onAudit(input)
	}
	return a.result, a.err
}

func auditRetrySnapshot(resourceID string) model.ResourceAuditSnapshot {
	return model.ResourceAuditSnapshot{
		ID:         resourceID,
		MerchantID: "merchant-1",
		TypeCode:   "supply",
		OpenID:     "openid-1",
	}
}
