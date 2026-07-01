package admin

import (
	"context"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListHotSearchKeywordsPassesTrimmedFilters(t *testing.T) {
	store := &fakeHotSearchKeywordAdminStore{
		items: []model.HotSearchKeywordConfig{{ID: "keyword-1", Keyword: "夏款现货", Status: "active"}},
	}
	logic := NewHotSearchKeywordAdminLogic(store)

	resp, err := logic.ListHotSearchKeywords(context.Background(), ListHotSearchKeywordsReq{CityCode: " zhili ", Status: " active "})
	if err != nil {
		t.Fatalf("ListHotSearchKeywords() error = %v", err)
	}

	if store.filter.CityCode != "zhili" || store.filter.Status != "active" {
		t.Fatalf("filter = %#v, want trimmed filters", store.filter)
	}
	if len(resp.Items) != 1 || resp.Items[0].Keyword != "夏款现货" {
		t.Fatalf("items = %#v, want keyword item", resp.Items)
	}
}

func TestCreateHotSearchKeywordPassesNormalizedInput(t *testing.T) {
	store := &fakeHotSearchKeywordAdminStore{
		saved: model.SaveHotSearchKeywordResult{ID: "keyword-1", UpdatedAt: "2026-06-30T10:00:00Z"},
	}
	logic := NewHotSearchKeywordAdminLogic(store)

	resp, err := logic.CreateHotSearchKeyword(context.Background(), SaveHotSearchKeywordReq{
		CityCode:  " zhili ",
		Keyword:   " 夏款现货 ",
		SortOrder: 20,
		Status:    " active ",
	})
	if err != nil {
		t.Fatalf("CreateHotSearchKeyword() error = %v", err)
	}

	if store.input.CityCode != "zhili" || store.input.Keyword != "夏款现货" || store.input.Status != "active" {
		t.Fatalf("input = %#v, want normalized input", store.input)
	}
	if resp.ID != "keyword-1" {
		t.Fatalf("id = %q, want keyword-1", resp.ID)
	}
}

func TestCreateHotSearchKeywordRejectsEmptyKeyword(t *testing.T) {
	logic := NewHotSearchKeywordAdminLogic(&fakeHotSearchKeywordAdminStore{})

	_, err := logic.CreateHotSearchKeyword(context.Background(), SaveHotSearchKeywordReq{CityCode: "zhili", Keyword: " "})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("CreateHotSearchKeyword() error = %v, want validation error", err)
	}
}

func TestUpdateHotSearchKeywordRejectsEmptyConfigID(t *testing.T) {
	logic := NewHotSearchKeywordAdminLogic(&fakeHotSearchKeywordAdminStore{})

	_, err := logic.UpdateHotSearchKeyword(context.Background(), " ", SaveHotSearchKeywordReq{Keyword: "夏款现货"})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("UpdateHotSearchKeyword() error = %v, want validation error", err)
	}
}

type fakeHotSearchKeywordAdminStore struct {
	filter model.HotSearchKeywordFilter
	input  model.SaveHotSearchKeywordInput
	items  []model.HotSearchKeywordConfig
	saved  model.SaveHotSearchKeywordResult
}

func (s *fakeHotSearchKeywordAdminStore) ListHotSearchKeywords(ctx context.Context, filter model.HotSearchKeywordFilter) ([]model.HotSearchKeywordConfig, error) {
	s.filter = filter
	return append([]model.HotSearchKeywordConfig(nil), s.items...), nil
}

func (s *fakeHotSearchKeywordAdminStore) CreateHotSearchKeyword(ctx context.Context, input model.SaveHotSearchKeywordInput) (model.SaveHotSearchKeywordResult, error) {
	s.input = input
	return s.saved, nil
}

func (s *fakeHotSearchKeywordAdminStore) UpdateHotSearchKeyword(ctx context.Context, configID string, input model.SaveHotSearchKeywordInput) (model.SaveHotSearchKeywordResult, error) {
	s.input = input
	s.input.ID = configID
	return s.saved, nil
}
