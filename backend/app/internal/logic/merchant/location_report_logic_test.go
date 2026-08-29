package merchant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestCreateMerchantLocationReportAcceptsControlledReasons(t *testing.T) {
	for _, reasonCode := range []string{"navigation_inaccurate", "wrong_market", "wrong_booth", "other"} {
		t.Run(reasonCode, func(t *testing.T) {
			store := &fakeMerchantLocationReportStore{result: model.CreateMerchantLocationReportResult{
				ID: "9001", Status: model.MerchantLocationReportStatusPending, ActiveReporterCount: 1,
			}}
			logic := NewMerchantLocationReportLogic(store)

			resp, err := logic.CreateMerchantLocationReport(context.Background(), " 101 ", " 201 ", CreateMerchantLocationReportReq{
				ReasonCode: " " + reasonCode + " ", Description: " 位置说明 ",
			})
			if err != nil {
				t.Fatalf("CreateMerchantLocationReport() error = %v", err)
			}
			if store.input.MerchantID != "101" || store.input.ReporterUserID != "201" || store.input.ReasonCode != reasonCode || store.input.Description != "位置说明" {
				t.Fatalf("store input = %#v, want trimmed controlled report", store.input)
			}
			if resp.ReportID != "9001" || resp.Status != model.MerchantLocationReportStatusPending || resp.ActiveReporterCount != 1 {
				t.Fatalf("resp = %#v, want stored report", resp)
			}
		})
	}
}

func TestCreateMerchantLocationReportReturnsAutoHiddenResult(t *testing.T) {
	store := &fakeMerchantLocationReportStore{result: model.CreateMerchantLocationReportResult{
		ID: "9003", Status: model.MerchantLocationReportStatusPending, ActiveReporterCount: 3, AutoHidden: true,
	}}

	resp, err := NewMerchantLocationReportLogic(store).CreateMerchantLocationReport(context.Background(), "101", "203", CreateMerchantLocationReportReq{
		ReasonCode: "wrong_booth",
	})
	if err != nil {
		t.Fatalf("CreateMerchantLocationReport() error = %v", err)
	}
	if !resp.AutoHidden || resp.ActiveReporterCount != 3 {
		t.Fatalf("resp = %#v, want third report auto-hidden result", resp)
	}
}

func TestCreateMerchantLocationReportRejectsUnknownReason(t *testing.T) {
	store := &fakeMerchantLocationReportStore{}
	_, err := NewMerchantLocationReportLogic(store).CreateMerchantLocationReport(context.Background(), "101", "201", CreateMerchantLocationReportReq{
		ReasonCode: "free_text_reason",
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "请选择正确的位置问题类型" {
		t.Fatalf("error = %v, want controlled reason validation", err)
	}
	if store.called {
		t.Fatal("store called for unknown reason")
	}
}

func TestCreateMerchantLocationReportRejectsInvalidMerchantIDBeforeStore(t *testing.T) {
	for _, merchantID := range []string{"abc", "0", "-1", "9223372036854775808", "1 OR 1=1"} {
		t.Run(merchantID, func(t *testing.T) {
			store := &fakeMerchantLocationReportStore{}
			_, err := NewMerchantLocationReportLogic(store).CreateMerchantLocationReport(context.Background(), merchantID, "201", CreateMerchantLocationReportReq{
				ReasonCode: "wrong_booth",
			})
			if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "商家编号格式不正确" {
				t.Fatalf("error = %v, want invalid merchant ID validation", err)
			}
			if store.called {
				t.Fatal("store called for invalid merchant ID")
			}
		})
	}
}

func TestCreateMerchantLocationReportRejectsDescriptionOverTwoHundredCharacters(t *testing.T) {
	store := &fakeMerchantLocationReportStore{}
	_, err := NewMerchantLocationReportLogic(store).CreateMerchantLocationReport(context.Background(), "101", "201", CreateMerchantLocationReportReq{
		ReasonCode: "other", Description: strings.Repeat("织", 201),
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "补充说明最多可填写 200 个字" {
		t.Fatalf("error = %v, want 200-character validation", err)
	}
	if store.called {
		t.Fatal("store called for oversized description")
	}
}

func TestCreateMerchantLocationReportMapsUnavailableLocation(t *testing.T) {
	store := &fakeMerchantLocationReportStore{err: model.ErrMerchantLocationNotReportable}
	_, err := NewMerchantLocationReportLogic(store).CreateMerchantLocationReport(context.Background(), "101", "201", CreateMerchantLocationReportReq{
		ReasonCode: "navigation_inaccurate",
	})
	if errx.CodeOf(err) != errx.CodeResourceNotFound || errx.PublicMessage(err) != "该商家位置待完善" {
		t.Fatalf("error = %v, want unavailable location message", err)
	}
}

func TestCreateMerchantLocationReportHidesDatabaseError(t *testing.T) {
	store := &fakeMerchantLocationReportStore{err: errors.New("pq: relation merchant_location_reports does not exist")}
	_, err := NewMerchantLocationReportLogic(store).CreateMerchantLocationReport(context.Background(), "101", "201", CreateMerchantLocationReportReq{
		ReasonCode: "wrong_market",
	})
	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "位置反馈提交失败，请稍后重试" {
		t.Fatalf("error = %v, want safe report failure", err)
	}
	if strings.Contains(errx.PublicMessage(err), "merchant_location_reports") || strings.Contains(errx.PublicMessage(err), "pq:") {
		t.Fatalf("public error leaked database detail: %q", errx.PublicMessage(err))
	}
}

type fakeMerchantLocationReportStore struct {
	called bool
	input  model.CreateMerchantLocationReportInput
	result model.CreateMerchantLocationReportResult
	err    error
}

func (s *fakeMerchantLocationReportStore) CreateMerchantLocationReport(_ context.Context, input model.CreateMerchantLocationReportInput) (model.CreateMerchantLocationReportResult, error) {
	s.called = true
	s.input = input
	return s.result, s.err
}
