package merchant

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"
	"time"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestListMerchantDirectoryAppliesDefaultsAndNormalizesEmptyItems(t *testing.T) {
	store := &fakeDirectoryStore{}
	resp, err := NewMerchantDirectoryLogic(store).ListDirectory(context.Background(), ListMerchantDirectoryReq{
		Categories: " 童装, ,童装,卫衣 ",
	})

	if err != nil {
		t.Fatalf("ListDirectory() error = %v", err)
	}
	if store.filter.CityCode != "zhili" || store.filter.Page != 1 || store.filter.PageSize != 20 {
		t.Fatalf("filter = %#v, want zhili and default 1/20 pagination", store.filter)
	}
	if len(store.filter.Categories) != 2 || store.filter.Categories[0] != "童装" || store.filter.Categories[1] != "卫衣" {
		t.Fatalf("categories = %#v, want trimmed unique values", store.filter.Categories)
	}
	if resp.Items == nil || len(resp.Items) != 0 {
		t.Fatalf("items = %#v, want non-nil empty array", resp.Items)
	}
	if resp.Page != 1 || resp.PageSize != 20 {
		t.Fatalf("pagination = %d/%d, want 1/20", resp.Page, resp.PageSize)
	}
}

func TestListMerchantDirectoryRejectsUnsupportedCityAndOversizedPage(t *testing.T) {
	tests := []struct {
		name string
		req  ListMerchantDirectoryReq
	}{
		{name: "unsupported city", req: ListMerchantDirectoryReq{CityCode: "guangzhou"}},
		{name: "page size over 100", req: ListMerchantDirectoryReq{CityCode: "zhili", PageSize: 101}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeDirectoryStore{}
			_, err := NewMerchantDirectoryLogic(store).ListDirectory(context.Background(), tt.req)
			if errx.CodeOf(err) != errx.CodeValidationFailed {
				t.Fatalf("error = %v, want validation failed", err)
			}
			if store.listCalled || store.countCalled {
				t.Fatal("store must not be called for invalid filter")
			}
		})
	}
}

func TestListMerchantDirectoryReturnsSingleSafeErrorForDatabaseFailures(t *testing.T) {
	for _, tt := range []struct {
		name     string
		listErr  error
		countErr error
	}{
		{name: "list", listErr: errors.New("pq: relation merchants unavailable")},
		{name: "count", countErr: errors.New("pq: count timeout")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeDirectoryStore{listErr: tt.listErr, countErr: tt.countErr}
			_, err := NewMerchantDirectoryLogic(store).ListDirectory(context.Background(), ListMerchantDirectoryReq{Page: 2, PageSize: 30})
			if errx.CodeOf(err) != errx.CodeInternalError || err.Error() != "商家列表加载失败，请稍后重试" {
				t.Fatalf("error = %v, want stable safe directory error", err)
			}
			if err.Error() == "pq: relation merchants unavailable" || err.Error() == "pq: count timeout" {
				t.Fatalf("error = %v, must not expose database detail", err)
			}
		})
	}
}

func TestListMerchantDirectoryMapsPublicFieldsWithoutContacts(t *testing.T) {
	activeAt := time.Date(2026, 8, 28, 12, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	store := &fakeDirectoryStore{
		items: []model.MerchantDirectoryItem{{
			MerchantID: "merchant-1", MerchantNo: "M00000018", Name: "晨星童装", MerchantType: "factory",
			MainCategories: []string{"童装"}, LogoURL: "logo.png", AddressText: "利济路 88 号",
			MarketName: "织里童装城", BuildingName: "一期", FloorNo: "1F", BoothNo: "A108",
			Latitude: 30.8732, Longitude: 120.2255, LocationName: "织里童装城", LastResourceActiveAt: activeAt,
		}},
		total: 1,
	}
	resp, err := NewMerchantDirectoryLogic(store).ListDirectory(context.Background(), ListMerchantDirectoryReq{CityCode: "zhili"})
	if err != nil {
		t.Fatalf("ListDirectory() error = %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items = %#v, want one merchant", resp.Items)
	}
	item := resp.Items[0]
	if item.MerchantID != "merchant-1" || item.BoothNo != "A108" || item.LastResourceActiveAt != "2026-08-28T12:30:00+08:00" {
		t.Fatalf("item = %#v, want public directory fields", item)
	}
	if item.MainCategories == nil {
		t.Fatalf("item = %#v, mainCategories must be a JSON array", item)
	}
}

func TestGetMerchantLocationContextReturnsFriendlyErrorWhenCurrentIsUnavailable(t *testing.T) {
	for _, tt := range []struct {
		name       string
		merchantID string
		current    model.MerchantDirectoryItem
		currentErr error
	}{
		{name: "empty merchant", merchantID: " "},
		{name: "not visible", merchantID: "merchant-missing", currentErr: sql.ErrNoRows},
		{name: "database failure", merchantID: "merchant-1", currentErr: errors.New("db unavailable")},
		{name: "invalid coordinate", merchantID: "merchant-1", current: model.MerchantDirectoryItem{MerchantID: "merchant-1", Latitude: math.NaN(), Longitude: 120}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeDirectoryStore{current: tt.current, currentErr: tt.currentErr}
			_, err := NewMerchantDirectoryLogic(store).GetLocationContext(context.Background(), tt.merchantID)
			if err == nil || err.Error() != "该商家位置待完善" {
				t.Fatalf("error = %v, want friendly unavailable location message", err)
			}
		})
	}
}

func TestGetMerchantLocationContextKeepsCurrentWhenNearbyFails(t *testing.T) {
	store := &fakeDirectoryStore{current: visibleDirectoryMerchant("merchant-1"), nearbyErr: errors.New("db unavailable")}
	resp, err := NewMerchantDirectoryLogic(store).GetLocationContext(context.Background(), "merchant-1")

	if err != nil || resp.Current.MerchantID != "merchant-1" || resp.NearbyAvailable {
		t.Fatalf("resp=%#v err=%v", resp, err)
	}
	if resp.Nearby == nil || len(resp.Nearby) != 0 || resp.RadiusMeters != 1000 {
		t.Fatalf("resp=%#v, want empty nearby array and fixed radius", resp)
	}
	if store.nearbyRadius != 1000 || store.nearbyLimit != 20 {
		t.Fatalf("nearby query = radius %d limit %d, want 1000/20", store.nearbyRadius, store.nearbyLimit)
	}
}

func TestGetMerchantLocationContextReturnsNearbyMerchants(t *testing.T) {
	nearby := visibleDirectoryMerchant("merchant-2")
	nearby.DistanceMeters = 42.5
	store := &fakeDirectoryStore{current: visibleDirectoryMerchant("merchant-1"), nearby: []model.MerchantDirectoryItem{nearby}}
	resp, err := NewMerchantDirectoryLogic(store).GetLocationContext(context.Background(), " merchant-1 ")

	if err != nil {
		t.Fatalf("GetLocationContext() error = %v", err)
	}
	if !resp.NearbyAvailable || len(resp.Nearby) != 1 || resp.Nearby[0].MerchantID != "merchant-2" || resp.Nearby[0].DistanceMeters != 42.5 {
		t.Fatalf("resp = %#v, want available nearby merchant", resp)
	}
}

func visibleDirectoryMerchant(merchantID string) model.MerchantDirectoryItem {
	return model.MerchantDirectoryItem{
		MerchantID: merchantID, Name: "测试商家", MainCategories: []string{"童装"},
		Latitude: 30.8732, Longitude: 120.2255,
	}
}

type fakeDirectoryStore struct {
	filter       model.MerchantDirectoryFilter
	items        []model.MerchantDirectoryItem
	total        int64
	listErr      error
	countErr     error
	current      model.MerchantDirectoryItem
	currentErr   error
	nearby       []model.MerchantDirectoryItem
	nearbyErr    error
	nearbyRadius int64
	nearbyLimit  int64
	listCalled   bool
	countCalled  bool
}

func (s *fakeDirectoryStore) ListMerchantDirectory(_ context.Context, filter model.MerchantDirectoryFilter) ([]model.MerchantDirectoryItem, error) {
	s.listCalled = true
	s.filter = filter
	return append([]model.MerchantDirectoryItem(nil), s.items...), s.listErr
}

func (s *fakeDirectoryStore) CountMerchantDirectory(_ context.Context, filter model.MerchantDirectoryFilter) (int64, error) {
	s.countCalled = true
	s.filter = filter
	return s.total, s.countErr
}

func (s *fakeDirectoryStore) GetVisibleMerchantLocation(_ context.Context, merchantID string) (model.MerchantDirectoryItem, error) {
	return s.current, s.currentErr
}

func (s *fakeDirectoryStore) ListNearbyVisibleMerchants(_ context.Context, origin model.MerchantDirectoryItem, radiusMeters int64, limit int64) ([]model.MerchantDirectoryItem, error) {
	s.nearbyRadius = radiusMeters
	s.nearbyLimit = limit
	return append([]model.MerchantDirectoryItem(nil), s.nearby...), s.nearbyErr
}
