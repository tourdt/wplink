package admin

import (
	"context"
	"errors"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListDemandsPassesFiltersToStore(t *testing.T) {
	store := &fakeAdminDemandStore{
		result: model.ListDemandsResult{
			Items: []model.DemandListItem{{ID: "demand-1", Title: "找童装库存", DemandType: "inventory", Category: "童装", ContactName: "王老板", Status: "pending"}},
			Page:  1, PageSize: 20, Total: 1,
		},
	}
	logic := NewDemandAdminLogic(store)

	resp, err := logic.ListDemands(context.Background(), ListDemandsReq{CityCode: " zhili ", DemandType: "inventory", Status: "pending"})
	if err != nil {
		t.Fatalf("ListDemands() error = %v", err)
	}

	if store.filter.CityCode != "zhili" || store.filter.Status != "pending" {
		t.Fatalf("filter = %#v, want zhili pending", store.filter)
	}
	if len(resp.Items) != 1 || resp.Items[0].ContactName != "王老板" {
		t.Fatalf("items = %#v, want demand item", resp.Items)
	}
}

func TestGetDemandRequiresID(t *testing.T) {
	logic := NewDemandAdminLogic(&fakeAdminDemandStore{})

	_, err := logic.GetDemand(context.Background(), GetDemandReq{DemandID: " "})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("GetDemand() error = %v, want validation error", err)
	}
}

func TestGetDemandMapsModelDetail(t *testing.T) {
	store := &fakeAdminDemandStore{
		detail: model.DemandDetail{
			ID: "demand-1", Title: "找童装库存", DemandType: "inventory", Category: "童装",
			PriceRange:          model.JSONMap{"min": float64(20), "max": float64(40)},
			QuantityRequirement: model.JSONMap{"quantity": float64(1000), "unit": "件"},
			Attributes:          model.JSONMap{"season": "夏季"},
			ContactName:         "王老板", ContactPhone: "13800000000", ContactWechat: "wx-1",
			Status: "pending", CreatedAt: "2026-06-27T09:10:00Z",
		},
	}
	logic := NewDemandAdminLogic(store)

	resp, err := logic.GetDemand(context.Background(), GetDemandReq{DemandID: " demand-1 "})
	if err != nil {
		t.Fatalf("GetDemand() error = %v", err)
	}

	if store.detailID != "demand-1" || resp.Contact.Phone != "13800000000" || resp.QuantityRequirement["unit"] != "件" {
		t.Fatalf("resp = %#v, detailID = %q", resp, store.detailID)
	}
}

func TestUpdateDemandStatusRejectsUnsupportedStatus(t *testing.T) {
	logic := NewDemandAdminLogic(&fakeAdminDemandStore{})

	_, err := logic.UpdateDemandStatus(context.Background(), UpdateDemandStatusReq{DemandID: "demand-1", Status: "done"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("UpdateDemandStatus() error = %v, want validation error", err)
	}
}

func TestUpdateDemandStatusAcceptsContacted(t *testing.T) {
	store := &fakeAdminDemandStore{updated: model.UpdateDemandStatusResult{ID: "demand-1", Status: "contacted"}}
	logic := NewDemandAdminLogic(store)

	resp, err := logic.UpdateDemandStatus(context.Background(), UpdateDemandStatusReq{DemandID: "demand-1", Status: "contacted"})
	if err != nil {
		t.Fatalf("UpdateDemandStatus() error = %v", err)
	}
	if resp.Status != "contacted" {
		t.Fatalf("status = %q, want contacted", resp.Status)
	}
}

func TestUpdateDemandStatusPassesTrimmedInput(t *testing.T) {
	store := &fakeAdminDemandStore{updated: model.UpdateDemandStatusResult{ID: "demand-1", Status: "matching"}}
	logic := NewDemandAdminLogic(store)

	resp, err := logic.UpdateDemandStatus(context.Background(), UpdateDemandStatusReq{DemandID: " demand-1 ", Status: " matching "})
	if err != nil {
		t.Fatalf("UpdateDemandStatus() error = %v", err)
	}

	if store.updateID != "demand-1" || store.updateStatus != "matching" || resp.Status != "matching" {
		t.Fatalf("updateID = %q, status = %q, resp = %#v", store.updateID, store.updateStatus, resp)
	}
}

type fakeAdminDemandStore struct {
	filter       model.ListDemandsFilter
	result       model.ListDemandsResult
	detailID     string
	detail       model.DemandDetail
	updateID     string
	updateStatus string
	updated      model.UpdateDemandStatusResult
}

func (s *fakeAdminDemandStore) ListDemands(ctx context.Context, filter model.ListDemandsFilter) (model.ListDemandsResult, error) {
	s.filter = filter
	return s.result, nil
}

func (s *fakeAdminDemandStore) GetDemand(ctx context.Context, demandID string) (model.DemandDetail, error) {
	s.detailID = demandID
	if s.detail.ID == "" {
		return model.DemandDetail{}, errors.New("not found")
	}
	return s.detail, nil
}

func (s *fakeAdminDemandStore) UpdateDemandStatus(ctx context.Context, demandID string, status string) (model.UpdateDemandStatusResult, error) {
	s.updateID = demandID
	s.updateStatus = status
	return s.updated, nil
}
