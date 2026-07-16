package resource

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestReportResourceRejectsInvalidReason(t *testing.T) {
	logic := NewReportResourceLogic(&fakeResourceReportStore{})

	_, err := logic.ReportResource(context.Background(), ReportResourceReq{
		ResourceID:     "resource-1",
		ReporterUserID: "user-1",
		ReasonCode:     "bad",
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestReportResourcePassesInputToStore(t *testing.T) {
	store := &fakeResourceReportStore{result: model.ResourceReportResult{ID: "report-1", Status: model.ResourceReportStatusPending}}
	logic := NewReportResourceLogic(store)

	resp, err := logic.ReportResource(context.Background(), ReportResourceReq{
		ResourceID:     " resource-1 ",
		ReporterUserID: " user-1 ",
		ReasonCode:     " fake_info ",
		ReasonText:     " 虚假库存 ",
		Evidence:       model.JSONMap{"source": "resource_detail"},
	})
	if err != nil {
		t.Fatalf("ReportResource() error = %v", err)
	}

	if store.input.ResourceID != "resource-1" || store.input.ReporterUserID != "user-1" || store.input.ReasonCode != "fake_info" || store.input.ReasonText != "虚假库存" {
		t.Fatalf("input = %#v, want trimmed report input", store.input)
	}
	if resp.ID != "report-1" || resp.Status != model.ResourceReportStatusPending || resp.Message != "举报已提交，平台会尽快核查" {
		t.Fatalf("resp = %#v, want pending report response", resp)
	}
}

type fakeResourceReportStore struct {
	input  model.CreateResourceReportInput
	result model.ResourceReportResult
}

func (s *fakeResourceReportStore) CreateResourceReport(ctx context.Context, input model.CreateResourceReportInput) (model.ResourceReportResult, error) {
	s.input = input
	return s.result, nil
}
