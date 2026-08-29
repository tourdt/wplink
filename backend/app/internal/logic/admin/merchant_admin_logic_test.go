package admin

import (
	"context"
	"errors"
	"math"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/app/internal/permission"
	"wplink/backend/common/errx"
)

func TestListMerchantsPassesFiltersToStore(t *testing.T) {
	store := &fakeAdminMerchantStore{
		result: model.ListMerchantsResult{
			Items: []model.MerchantListItem{
				{ID: "merchant-1", Name: "织里样板童装厂", MerchantType: "factory", Status: "active", LocationStatus: model.MerchantLocationStatusRiskHidden, MarketName: "织里童装城", BuildingName: "一期", FloorNo: "1F", BoothNo: "A108", PendingLocationReportCount: 3},
			},
			Page:     1,
			PageSize: 20,
			Total:    1,
		},
	}
	logic := NewMerchantAdminLogic(store)

	resp, err := logic.ListMerchants(context.Background(), ListMerchantsReq{
		CityCode:       " zhili ",
		MerchantType:   "factory",
		Status:         " active ",
		LocationStatus: " risk_hidden ",
		Keyword:        " 样板 ",
		Page:           1,
		PageSize:       20,
	})
	if err != nil {
		t.Fatalf("ListMerchants() error = %v", err)
	}

	if store.filter.CityCode != "zhili" {
		t.Fatalf("cityCode = %q, want trimmed zhili", store.filter.CityCode)
	}
	if store.filter.Keyword != "样板" {
		t.Fatalf("keyword = %q, want trimmed 样板", store.filter.Keyword)
	}
	if store.filter.Status != model.MerchantStatusActive || store.filter.LocationStatus != model.MerchantLocationStatusRiskHidden {
		t.Fatalf("status filters = %q/%q, want independent business and location filters", store.filter.Status, store.filter.LocationStatus)
	}
	if len(resp.Items) != 1 || resp.Items[0].Name != "织里样板童装厂" {
		t.Fatalf("items = %#v, want merchant list item", resp.Items)
	}
	item := resp.Items[0]
	if item.LocationStatus != model.MerchantLocationStatusRiskHidden || item.MarketName != "织里童装城" || item.BuildingName != "一期" || item.FloorNo != "1F" || item.BoothNo != "A108" || item.PendingLocationReportCount != 3 {
		t.Fatalf("item = %#v, want complete location governance summary", item)
	}
}

func TestListMerchantLocationReportsPassesPendingFilter(t *testing.T) {
	store := &fakeAdminMerchantStore{reportList: model.ListAdminMerchantLocationReportsResult{
		Items: []model.AdminMerchantLocationReportItem{{ID: "9001", MerchantID: "101", MerchantName: "晨星童装", Status: model.MerchantLocationReportStatusPending}},
		Page:  1, PageSize: 20, Total: 1,
	}}
	logic := NewMerchantAdminLogic(store)

	resp, err := logic.ListMerchantLocationReports(context.Background(), ListMerchantLocationReportsReq{Status: " pending ", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListMerchantLocationReports() error = %v", err)
	}
	if store.reportFilter.Status != model.MerchantLocationReportStatusPending || len(resp.Items) != 1 || resp.Items[0].ReportID != "9001" {
		t.Fatalf("filter=%#v resp=%#v, want pending report", store.reportFilter, resp)
	}
}

func TestListMerchantLocationReportsRejectsOverflowingOffsetBeforeStore(t *testing.T) {
	store := &fakeAdminMerchantStore{}
	_, err := NewMerchantAdminLogic(store).ListMerchantLocationReports(context.Background(), ListMerchantLocationReportsReq{
		Page: math.MaxInt64, PageSize: 100,
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "分页参数超出范围，请重新选择页码" {
		t.Fatalf("error = %v, want pagination overflow validation", err)
	}
	if store.reportListCalls != 0 {
		t.Fatal("store must not be called when pagination offset overflows")
	}
}

func TestListMerchantLocationReportsHidesStoreCountError(t *testing.T) {
	store := &fakeAdminMerchantStore{reportErr: errors.New("pq: relation merchant_location_reports is unavailable")}
	_, err := NewMerchantAdminLogic(store).ListMerchantLocationReports(context.Background(), ListMerchantLocationReportsReq{
		Status: model.MerchantLocationReportStatusPending, Page: 2, PageSize: 20,
	})
	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "位置反馈列表加载失败，请稍后重试" {
		t.Fatalf("error = %v, want safe list failure", err)
	}
}

func TestReviewMerchantLocationReportPassesAuthenticatedAdminAndControlledStatus(t *testing.T) {
	store := &fakeAdminMerchantStore{reviewResult: model.ReviewMerchantLocationReportResult{ReportID: "9001", MerchantID: "101", Status: model.MerchantLocationReportStatusValid}}
	logic := NewMerchantAdminLogic(store)

	resp, err := logic.ReviewMerchantLocationReport(context.Background(), " 9001 ", ReviewMerchantLocationReportReq{
		Status: " valid ", ReviewNote: " 情况属实 ",
	}, AdminPermissionActor{OperatorID: " 701 ", Roles: []string{permission.RoleSuperAdmin}})
	if err != nil {
		t.Fatalf("ReviewMerchantLocationReport() error = %v", err)
	}
	if store.reviewInput.ReportID != "9001" || store.reviewInput.Status != model.MerchantLocationReportStatusValid || store.reviewInput.ReviewNote != "情况属实" || store.reviewInput.AdminID != "701" || store.reviewInput.AdminRole != permission.RoleSuperAdmin {
		t.Fatalf("review input = %#v, want authenticated admin review", store.reviewInput)
	}
	if resp.ReportID != "9001" || resp.Status != model.MerchantLocationReportStatusValid {
		t.Fatalf("resp = %#v, want reviewed report", resp)
	}
}

func TestReviewMerchantLocationReportRejectsUnknownStatus(t *testing.T) {
	store := &fakeAdminMerchantStore{}
	_, err := NewMerchantAdminLogic(store).ReviewMerchantLocationReport(context.Background(), "9001", ReviewMerchantLocationReportReq{
		Status: "pending",
	}, AdminPermissionActor{OperatorID: "701", Roles: []string{permission.RolePlatformOperator}})
	if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "位置反馈处理结果不正确" {
		t.Fatalf("error = %v, want controlled review status", err)
	}
	if store.reviewCalled {
		t.Fatal("store called for invalid review status")
	}
}

func TestReviewMerchantLocationReportRejectsInvalidReportIDBeforeStore(t *testing.T) {
	for _, reportID := range []string{"abc", "0", "-1", "9223372036854775808", "1 OR 1=1"} {
		t.Run(reportID, func(t *testing.T) {
			store := &fakeAdminMerchantStore{}
			_, err := NewMerchantAdminLogic(store).ReviewMerchantLocationReport(context.Background(), reportID, ReviewMerchantLocationReportReq{
				Status: model.MerchantLocationReportStatusValid,
			}, AdminPermissionActor{OperatorID: "701", Roles: []string{permission.RolePlatformOperator}})
			if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "位置反馈编号格式不正确" {
				t.Fatalf("error = %v, want invalid report ID validation", err)
			}
			if store.reviewCalled {
				t.Fatal("store called for invalid report ID")
			}
		})
	}
}

func TestReviewMerchantLocationReportMapsAlreadyReviewed(t *testing.T) {
	store := &fakeAdminMerchantStore{reviewErr: model.ErrMerchantLocationReportAlreadyReviewed}
	_, err := NewMerchantAdminLogic(store).ReviewMerchantLocationReport(context.Background(), "9001", ReviewMerchantLocationReportReq{
		Status: model.MerchantLocationReportStatusInvalid,
	}, AdminPermissionActor{OperatorID: "701", Roles: []string{permission.RolePlatformOperator}})
	if errx.CodeOf(err) != errx.CodeStateConflict || errx.PublicMessage(err) != "该位置反馈已处理，请刷新后查看" {
		t.Fatalf("error = %v, want already-reviewed conflict", err)
	}
}

func TestUpdateMerchantLocationStatusRestoresWithAuthenticatedAdmin(t *testing.T) {
	store := &fakeAdminMerchantStore{statusResult: model.UpdateMerchantLocationStatusResult{
		MerchantID: "101", OldStatus: model.MerchantLocationStatusRiskHidden, NewStatus: model.MerchantLocationStatusVisible,
	}}
	logic := NewMerchantAdminLogic(store)

	resp, err := logic.UpdateMerchantLocationStatus(context.Background(), " 101 ", UpdateMerchantLocationStatusReq{
		Action: " restore ", Reason: " 已核对坐标 ",
	}, AdminPermissionActor{OperatorID: " 701 ", Roles: []string{permission.RolePlatformOperator}})
	if err != nil {
		t.Fatalf("UpdateMerchantLocationStatus() error = %v", err)
	}
	if store.statusInput.MerchantID != "101" || store.statusInput.Status != model.MerchantLocationStatusVisible || store.statusInput.AdminID != "701" || store.statusInput.AdminRole != permission.RolePlatformOperator || store.statusInput.Reason != "已核对坐标" {
		t.Fatalf("status input = %#v, want authenticated restore", store.statusInput)
	}
	if resp.LocationStatus != model.MerchantLocationStatusVisible {
		t.Fatalf("resp = %#v, want visible status", resp)
	}
}

func TestUpdateMerchantLocationStatusRejectsRestoreWithoutCoordinates(t *testing.T) {
	store := &fakeAdminMerchantStore{statusErr: model.ErrMerchantLocationCoordinatesMissing}
	_, err := NewMerchantAdminLogic(store).UpdateMerchantLocationStatus(context.Background(), "101", UpdateMerchantLocationStatusReq{
		Action: "restore",
	}, AdminPermissionActor{OperatorID: "701", Roles: []string{permission.RolePlatformOperator}})
	if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "商家尚未设置有效地图位置，无法恢复" {
		t.Fatalf("error = %v, want missing coordinates message", err)
	}
}

func TestUpdateMerchantLocationStatusRejectsInvalidMerchantIDBeforeStore(t *testing.T) {
	for _, merchantID := range []string{"abc", "0", "-1", "9223372036854775808", "1 OR 1=1"} {
		t.Run(merchantID, func(t *testing.T) {
			store := &fakeAdminMerchantStore{}
			_, err := NewMerchantAdminLogic(store).UpdateMerchantLocationStatus(context.Background(), merchantID, UpdateMerchantLocationStatusReq{
				Action: "hide",
			}, AdminPermissionActor{OperatorID: "701", Roles: []string{permission.RolePlatformOperator}})
			if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "商家编号格式不正确" {
				t.Fatalf("error = %v, want invalid merchant ID validation", err)
			}
			if store.statusCalled {
				t.Fatal("store called for invalid merchant ID")
			}
		})
	}
}

func TestUpdateMerchantLocationStatusHidesDatabaseError(t *testing.T) {
	store := &fakeAdminMerchantStore{statusErr: errors.New("pq: update merchants failed")}
	_, err := NewMerchantAdminLogic(store).UpdateMerchantLocationStatus(context.Background(), "101", UpdateMerchantLocationStatusReq{
		Action: "hide",
	}, AdminPermissionActor{OperatorID: "701", Roles: []string{permission.RolePlatformOperator}})
	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "商家位置状态更新失败，请稍后重试" {
		t.Fatalf("error = %v, want safe location status failure", err)
	}
}

type fakeAdminMerchantStore struct {
	filter          model.ListMerchantsFilter
	result          model.ListMerchantsResult
	reportFilter    model.AdminMerchantLocationReportFilter
	reportList      model.ListAdminMerchantLocationReportsResult
	reportErr       error
	reportListCalls int
	reviewCalled    bool
	reviewInput     model.ReviewMerchantLocationReportInput
	reviewResult    model.ReviewMerchantLocationReportResult
	reviewErr       error
	statusInput     model.UpdateMerchantLocationStatusInput
	statusResult    model.UpdateMerchantLocationStatusResult
	statusErr       error
	statusCalled    bool
}

func (s *fakeAdminMerchantStore) ListMerchants(ctx context.Context, filter model.ListMerchantsFilter) (model.ListMerchantsResult, error) {
	s.filter = filter
	return s.result, nil
}

func (s *fakeAdminMerchantStore) ListAdminMerchantLocationReports(_ context.Context, filter model.AdminMerchantLocationReportFilter) (model.ListAdminMerchantLocationReportsResult, error) {
	s.reportListCalls++
	s.reportFilter = filter
	return s.reportList, s.reportErr
}

func (s *fakeAdminMerchantStore) ReviewMerchantLocationReport(_ context.Context, input model.ReviewMerchantLocationReportInput) (model.ReviewMerchantLocationReportResult, error) {
	s.reviewCalled = true
	s.reviewInput = input
	return s.reviewResult, s.reviewErr
}

func (s *fakeAdminMerchantStore) UpdateMerchantLocationStatus(_ context.Context, input model.UpdateMerchantLocationStatusInput) (model.UpdateMerchantLocationStatusResult, error) {
	s.statusCalled = true
	s.statusInput = input
	return s.statusResult, s.statusErr
}
