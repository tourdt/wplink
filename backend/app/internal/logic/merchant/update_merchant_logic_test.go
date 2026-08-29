package merchant

import (
	"context"
	"database/sql"
	"math"
	"testing"

	"wplink/backend/app/internal/model"
	"wplink/backend/common/errx"
)

func TestUpdateMerchantRejectsEmptyID(t *testing.T) {
	logic := NewUpdateMerchantLogic(&fakeMerchantUpdateStore{})

	_, err := logic.UpdateMerchant(context.Background(), " ", UpdateMerchantReq{})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
}

func TestUpdateMerchantPassesPatchToStore(t *testing.T) {
	store := &fakeMerchantUpdateStore{updatedAt: "2026-06-27T10:00:00+08:00"}
	verifier := &fakeMerchantSMSVerifier{}
	logic := NewUpdateMerchantLogic(store, verifier)

	resp, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		Name:           "  织里晨星童装  ",
		MainCategories: []string{"童装", "卫衣"},
		MerchantType:   " stockist ",
		Description:    "支持小单快反",
		LogoURL:        "https://example.com/logo.png",
		Images:         []string{"https://example.com/factory.jpg"},
		ContactName:    "李厂长",
		ContactPhone:   "18800000001",
		ContactWechat:  "factory-demo",
		AddressText:    "织里镇利济路88号",
		LocationAction: "set",
		MarketName:     "织里童装城",
		BuildingName:   "一期",
		FloorNo:        "1F",
		BoothNo:        "A108",
		Latitude:       float64Ptr(30.1),
		Longitude:      float64Ptr(120.2),
		LocationName:   "织里童装城",
	})
	if err != nil {
		t.Fatalf("UpdateMerchant() error = %v", err)
	}

	if store.merchantID != "merchant-1" {
		t.Fatalf("merchantID = %q, want merchant-1", store.merchantID)
	}
	if store.patch.Name != "织里晨星童装" {
		t.Fatalf("name = %q, want trimmed merchant display name", store.patch.Name)
	}
	if store.patch.Description != "支持小单快反" {
		t.Fatalf("description = %q, want updated description", store.patch.Description)
	}
	if store.patch.MerchantType != "stockist" {
		t.Fatalf("merchantType = %q, want trimmed stockist", store.patch.MerchantType)
	}
	if store.patch.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("logoURL = %q, want updated logo URL", store.patch.LogoURL)
	}
	if store.patch.ContactName != "李厂长" || store.patch.ContactPhone != "18800000001" || store.patch.ContactWechat != "factory-demo" {
		t.Fatalf("contact patch = %#v, want contact name/phone/wechat", store.patch)
	}
	if store.patch.AddressText != "织里镇利济路88号" {
		t.Fatalf("addressText = %q, want updated merchant address", store.patch.AddressText)
	}
	if store.patch.Location.Status != model.MerchantLocationStatusVisible || store.patch.Location.BoothNo != "A108" {
		t.Fatalf("location = %#v, want visible structured location", store.patch.Location)
	}
	if verifier.phone != "" || verifier.code != "" {
		t.Fatalf("sms verifier = %q/%q, want no sms verification", verifier.phone, verifier.code)
	}
	if resp.UpdatedAt != "2026-06-27T10:00:00+08:00" {
		t.Fatalf("updatedAt = %q, want fixed time", resp.UpdatedAt)
	}
}

func TestUpdateMerchantRejectsMisleadingName(t *testing.T) {
	store := &fakeMerchantUpdateStore{updatedAt: "2026-06-27T10:00:00+08:00"}
	logic := NewUpdateMerchantLogic(store)

	_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		Name:           "官方认证童装",
		MainCategories: []string{"童装"},
	})

	if err == nil || err.Error() != "展示名称不能包含认证、官方等容易误导的字样" {
		t.Fatalf("error = %v, want misleading display name message", err)
	}
	if store.merchantID != "" {
		t.Fatalf("store was called for invalid merchant display name update")
	}
}

func TestUpdateMerchantRejectsInvalidContactPhone(t *testing.T) {
	store := &fakeMerchantUpdateStore{updatedAt: "2026-06-27T10:00:00+08:00"}
	logic := NewUpdateMerchantLogic(store, &fakeMerchantSMSVerifier{})

	_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		MainCategories: []string{"童装"},
		ContactPhone:   "1880000中文",
	})

	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error code = %q, want validation failed", errx.CodeOf(err))
	}
	if store.merchantID != "" {
		t.Fatalf("store was called for invalid contact phone update")
	}
}

func TestUpdateMerchantMapsMissingMerchant(t *testing.T) {
	store := &fakeMerchantUpdateStore{err: sql.ErrNoRows}
	logic := NewUpdateMerchantLogic(store)

	_, err := logic.UpdateMerchant(context.Background(), "merchant-missing", UpdateMerchantReq{
		MainCategories: []string{"童装"},
		LocationAction: "clear",
	})

	if errx.CodeOf(err) != errx.CodeResourceNotFound {
		t.Fatalf("error code = %q, want resource not found", errx.CodeOf(err))
	}
}

func TestUpdateMerchantSetsVisibleStructuredLocation(t *testing.T) {
	store := &fakeMerchantUpdateStore{updatedAt: "2026-08-28T10:00:00+08:00"}
	logic := NewUpdateMerchantLogic(store)

	_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		Name:           "织里晨星童装",
		LocationAction: "set",
		MarketName:     " 织里童装城 ",
		BuildingName:   " 一期 ",
		FloorNo:        " 1F ",
		BoothNo:        " A108 ",
		Latitude:       float64Ptr(30.8732),
		Longitude:      float64Ptr(120.2255),
		LocationName:   " 织里童装城 ",
	})
	if err != nil {
		t.Fatalf("UpdateMerchant() error = %v", err)
	}
	if store.patch.Location.Action != "set" || store.patch.Location.Status != model.MerchantLocationStatusVisible {
		t.Fatalf("location action/status = %#v, want set/visible", store.patch.Location)
	}
	if store.patch.Location.MarketName != "织里童装城" || store.patch.Location.BuildingName != "一期" || store.patch.Location.FloorNo != "1F" || store.patch.Location.BoothNo != "A108" {
		t.Fatalf("structured location = %#v, want trimmed stall fields", store.patch.Location)
	}
	if store.patch.Location.Latitude != 30.8732 || store.patch.Location.Longitude != 120.2255 || store.patch.Location.LocationName != "织里童装城" {
		t.Fatalf("map location = %#v, want selected Tencent location", store.patch.Location)
	}
}

func TestUpdateMerchantRejectsHalfLocation(t *testing.T) {
	logic := NewUpdateMerchantLogic(&fakeMerchantUpdateStore{})

	_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		Name:           "织里晨星童装",
		LocationAction: "set",
		Latitude:       float64Ptr(30.8732),
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("error = %v, want validation failed", err)
	}
}

func TestUpdateMerchantRejectsInvalidCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
	}{
		{name: "latitude out of range", latitude: 90.0001, longitude: 120.2255},
		{name: "longitude out of range", latitude: 30.8732, longitude: 180.0001},
		{name: "latitude not finite", latitude: math.NaN(), longitude: 120.2255},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logic := NewUpdateMerchantLogic(&fakeMerchantUpdateStore{})
			_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
				Name:           "织里晨星童装",
				LocationAction: "set",
				Latitude:       float64Ptr(tt.latitude),
				Longitude:      float64Ptr(tt.longitude),
			})
			if errx.CodeOf(err) != errx.CodeValidationFailed {
				t.Fatalf("error = %v, want validation failed", err)
			}
		})
	}
}

func TestUpdateMerchantClearsLocation(t *testing.T) {
	store := &fakeMerchantUpdateStore{updatedAt: "2026-08-28T10:00:00+08:00"}
	logic := NewUpdateMerchantLogic(store)

	_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		Name:           "织里晨星童装",
		LocationAction: "clear",
		MarketName:     "应忽略的市场",
		BuildingName:   "应忽略的楼栋",
		FloorNo:        "9F",
		BoothNo:        "Z999",
		Latitude:       float64Ptr(30.8732),
		Longitude:      float64Ptr(120.2255),
		LocationName:   "应忽略的位置",
	})
	if err != nil {
		t.Fatalf("UpdateMerchant() error = %v", err)
	}
	if store.patch.Location.Action != "clear" || store.patch.Location.Status != model.MerchantLocationStatusHidden {
		t.Fatalf("location action/status = %#v, want clear/hidden", store.patch.Location)
	}
	if store.patch.Location.MarketName != "" || store.patch.Location.BuildingName != "" || store.patch.Location.FloorNo != "" || store.patch.Location.BoothNo != "" || store.patch.Location.LocationName != "" || store.patch.Location.Latitude != 0 || store.patch.Location.Longitude != 0 {
		t.Fatalf("cleared location = %#v, want all location values empty", store.patch.Location)
	}
}

func TestUpdateMerchantRejectsUnknownLocationAction(t *testing.T) {
	logic := NewUpdateMerchantLogic(&fakeMerchantUpdateStore{})

	_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		Name:           "织里晨星童装",
		LocationAction: "keep",
	})
	if errx.CodeOf(err) != errx.CodeValidationFailed || errx.PublicMessage(err) != "地图位置操作不正确，请重新选择" {
		t.Fatalf("error = %v, want safe invalid location action message", err)
	}
}

func TestUpdateMerchantMapsStoreFailureToSafeError(t *testing.T) {
	store := &fakeMerchantUpdateStore{err: databaseError("write failed")}
	logic := NewUpdateMerchantLogic(store)

	_, err := logic.UpdateMerchant(context.Background(), "merchant-1", UpdateMerchantReq{
		Name:           "织里晨星童装",
		LocationAction: "clear",
	})
	if errx.CodeOf(err) != errx.CodeInternalError || errx.PublicMessage(err) != "商家资料保存失败，请稍后重试" {
		t.Fatalf("error = %v, want safe merchant save failure", err)
	}
}

type databaseError string

func (e databaseError) Error() string { return string(e) }

func float64Ptr(value float64) *float64 { return &value }

type fakeMerchantUpdateStore struct {
	merchantID string
	patch      model.UpdateMerchantPatch
	updatedAt  string
	err        error
}

func (s *fakeMerchantUpdateStore) UpdateMerchant(ctx context.Context, merchantID string, patch model.UpdateMerchantPatch) (string, error) {
	s.merchantID = merchantID
	s.patch = patch
	if s.err != nil {
		return "", s.err
	}
	return s.updatedAt, nil
}

type fakeMerchantSMSVerifier struct {
	phone string
	code  string
	err   error
}

func (s *fakeMerchantSMSVerifier) VerifySMSCode(ctx context.Context, phone string, code string) error {
	s.phone = phone
	s.code = code
	return s.err
}
