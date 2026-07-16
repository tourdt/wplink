package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestReviewResourceReportDefaultsValidToTakeDown(t *testing.T) {
	store := &fakeResourceReportAdminStore{
		reviewResult: model.ReviewResourceReportResult{
			ID:                  "report-1",
			Status:              model.ResourceReportStatusValid,
			ResourceID:          "resource-1",
			ResourceStatus:      model.ResourceStatusTakenDown,
			ResolvedReportCount: 3,
			RefundPublishQuota:  true,
		},
	}
	logic := NewResourceReportLogic(store)

	resp, err := logic.ReviewResourceReport(context.Background(), " report-1 ", ReviewResourceReportReq{
		Action:             model.ResourceReportActionValid,
		ReviewerID:         " operator-1 ",
		RefundPublishQuota: true,
	})
	if err != nil {
		t.Fatalf("ReviewResourceReport() error = %v", err)
	}

	if store.reviewInput.ResourceAction != model.ResourceReportResourceActionTakeDown {
		t.Fatalf("resourceAction = %q, want take_down", store.reviewInput.ResourceAction)
	}
	if store.reviewInput.Reason != "举报成立，资源已按规则处理" {
		t.Fatalf("reason = %q, want default valid reason", store.reviewInput.Reason)
	}
	if !store.reviewInput.RefundPublishQuota {
		t.Fatal("refundPublishQuota = false, want true")
	}
	if resp.Message != "举报已处理，已同步处理同资源 3 条待处理举报，已退还 1 次发布次数" {
		t.Fatalf("message = %q, want batch refund message", resp.Message)
	}
}

func TestReviewResourceReportRejectsRefundWhenInvalid(t *testing.T) {
	logic := NewResourceReportLogic(&fakeResourceReportAdminStore{})

	_, err := logic.ReviewResourceReport(context.Background(), "report-1", ReviewResourceReportReq{
		Action:             model.ResourceReportActionInvalid,
		RefundPublishQuota: true,
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestListResourceReportsMapsItems(t *testing.T) {
	store := &fakeResourceReportAdminStore{
		listResult: model.ListAdminResourceReportsResult{
			Items: []model.AdminResourceReportItem{{
				ID: "report-1", Status: model.ResourceReportStatusPending, ResourceID: "resource-1", ResourceTitle: "女童春款卫衣库存",
				ResourceStatus: model.ResourceStatusPublished, MerchantID: "merchant-1", MerchantName: "织里云仓", ReportCount: 2,
				ReasonCode: "fake_info", ReasonText: "虚假库存", LatestReportedAt: "2026-07-16T10:00:00Z",
			}},
			Page: 1, PageSize: 20, Total: 1,
		},
	}
	logic := NewResourceReportLogic(store)

	resp, err := logic.ListResourceReports(context.Background(), ListResourceReportsReq{})
	if err != nil {
		t.Fatalf("ListResourceReports() error = %v", err)
	}

	if store.filter.Status != model.ResourceReportStatusPending {
		t.Fatalf("status filter = %q, want pending", store.filter.Status)
	}
	if len(resp.Items) != 1 || resp.Items[0].ReportCount != 2 || resp.Total != 1 {
		t.Fatalf("resp = %#v, want mapped report item", resp)
	}
}

type fakeResourceReportAdminStore struct {
	filter      model.AdminResourceReportFilter
	listResult  model.ListAdminResourceReportsResult
	reviewInput model.ReviewResourceReportInput
	reviewResult model.ReviewResourceReportResult
}

func (s *fakeResourceReportAdminStore) ListAdminResourceReports(ctx context.Context, filter model.AdminResourceReportFilter) (model.ListAdminResourceReportsResult, error) {
	s.filter = filter
	return s.listResult, nil
}

func (s *fakeResourceReportAdminStore) ReviewResourceReport(ctx context.Context, input model.ReviewResourceReportInput) (model.ReviewResourceReportResult, error) {
	s.reviewInput = input
	return s.reviewResult, nil
}
