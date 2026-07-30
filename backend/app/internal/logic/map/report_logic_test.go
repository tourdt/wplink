package maplogic

import (
	"context"
	"database/sql"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestReportLogicSubmitsTrimmedLocationCorrection(t *testing.T) {
	store := &fakeMapReportStore{
		result: model.MapObjectReportResult{
			ID:                "report-1",
			Status:            model.MapObjectReportStatusPending,
			ActiveReportCount: 2,
		},
	}
	logic := NewReportLogic(store)

	resp, err := logic.Submit(context.Background(), " object-1 ", model.MapObjectReportKindLocationCorrection, SubmitMapObjectReportReq{
		ReporterUserID: " user-1 ",
		ReasonCode:     " navigation_inaccurate ",
		Description:    " 导航落点在马路对面 ",
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if store.input.ObjectID != "object-1" || store.input.ReporterUserID != "user-1" {
		t.Fatalf("input = %#v, want trimmed object/user", store.input)
	}
	if store.input.Kind != model.MapObjectReportKindLocationCorrection || store.input.ReasonCode != "navigation_inaccurate" {
		t.Fatalf("input = %#v, want location correction reason", store.input)
	}
	if store.input.Description != "导航落点在马路对面" {
		t.Fatalf("description = %q, want trimmed description", store.input.Description)
	}
	if resp.Item.ID != "report-1" || resp.Item.ActiveReportCount != 2 || resp.Item.WarningTriggered {
		t.Fatalf("resp = %#v, want pending correction without warning", resp)
	}
}

func TestReportLogicReturnsAutomaticWarningState(t *testing.T) {
	store := &fakeMapReportStore{
		result: model.MapObjectReportResult{
			ID:                "report-3",
			Status:            model.MapObjectReportStatusPending,
			ActiveReportCount: 3,
			WarningTriggered:  true,
		},
	}

	resp, err := NewReportLogic(store).Submit(context.Background(), "object-1", model.MapObjectReportKindRiskReport, SubmitMapObjectReportReq{
		ReporterUserID: "user-3",
		ReasonCode:     "false_information",
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if !resp.Item.WarningTriggered || resp.Message != "反馈已记录，该点位已增加风险提醒" {
		t.Fatalf("resp = %#v, want automatic warning message", resp)
	}
}

func TestReportLogicRejectsUnknownReasonForKind(t *testing.T) {
	logic := NewReportLogic(&fakeMapReportStore{})

	_, err := logic.Submit(context.Background(), "object-1", model.MapObjectReportKindLocationCorrection, SubmitMapObjectReportReq{
		ReporterUserID: "user-1",
		ReasonCode:     "impersonation",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("Submit() error = %v, want validation failed", err)
	}
}

func TestReportLogicRequiresDescriptionForOtherRisk(t *testing.T) {
	logic := NewReportLogic(&fakeMapReportStore{})

	_, err := logic.Submit(context.Background(), "object-1", model.MapObjectReportKindRiskReport, SubmitMapObjectReportReq{
		ReporterUserID: "user-1",
		ReasonCode:     "other",
	})
	if err == nil || errx.PublicMessage(err) != "请补充说明具体问题" {
		t.Fatalf("Submit() error = %v, want description validation", err)
	}
}

func TestReportLogicRejectsDescriptionOver500Runes(t *testing.T) {
	description := make([]rune, 501)
	for index := range description {
		description[index] = '错'
	}
	logic := NewReportLogic(&fakeMapReportStore{})

	_, err := logic.Submit(context.Background(), "object-1", model.MapObjectReportKindRiskReport, SubmitMapObjectReportReq{
		ReporterUserID: "user-1",
		ReasonCode:     "false_information",
		Description:    string(description),
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("Submit() error = %v, want validation failed", err)
	}
}

func TestReportLogicMapsMissingObjectToFriendlyError(t *testing.T) {
	logic := NewReportLogic(&fakeMapReportStore{err: sql.ErrNoRows})

	_, err := logic.Submit(context.Background(), "object-missing", model.MapObjectReportKindRiskReport, SubmitMapObjectReportReq{
		ReporterUserID: "user-1",
		ReasonCode:     "false_information",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeResourceNotFound {
		t.Fatalf("Submit() error = %v, want resource not found", err)
	}
}

type fakeMapReportStore struct {
	input  model.MapObjectReportInput
	result model.MapObjectReportResult
	err    error
}

func (s *fakeMapReportStore) CreateMapObjectReport(ctx context.Context, input model.MapObjectReportInput) (model.MapObjectReportResult, error) {
	s.input = input
	return s.result, s.err
}
