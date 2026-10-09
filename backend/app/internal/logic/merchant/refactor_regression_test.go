package merchant

import (
	"bytes"
	"context"
	"math"
	"strings"
	"testing"

	"wplink/backend/app/internal/model"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestMerchantPublicLocationRejectsInvalidStoredCoordinates(t *testing.T) {
	for _, latitude := range []float64{91, math.NaN(), math.Inf(1)} {
		location := model.MerchantLocation{Status: model.MerchantLocationStatusVisible, Latitude: &latitude, Longitude: float64Ptr(120)}
		got := visibleMerchantLocation(location, false)
		if got.Available || got.Latitude != nil || got.Longitude != nil {
			t.Fatalf("invalid stored location was exposed: %+v", got)
		}
	}
}

func TestMerchantSaveLogsSafeDatabaseDiagnostics(t *testing.T) {
	var output bytes.Buffer
	previous := logx.Reset()
	logx.SetWriter(logx.NewWriter(&output))
	t.Cleanup(func() {
		logx.Reset()
		if previous != nil {
			logx.SetWriter(previous)
		}
	})
	store := &fakeMerchantUpdateStore{err: &pq.Error{Code: "23514", Message: "secret-phone-13812345678", Detail: "secret-address-and-wechat"}}
	_, err := NewUpdateMerchantLogic(store).UpdateMerchant(context.Background(), "101", UpdateMerchantReq{Name: "童装店", BusinessProfile: testFactoryBusinessProfile(), MapVisible: boolPtr(false), LocationAction: "clear"})
	if err == nil {
		t.Fatal("expected save failure")
	}
	logs := output.String()
	for _, secret := range []string{"secret-phone", "13812345678", "secret-address-and-wechat"} {
		if strings.Contains(logs, secret) {
			t.Fatalf("log leaked %q", secret)
		}
	}
	for _, field := range []string{"101", "23514", "merchant_profile_update"} {
		if !strings.Contains(logs, field) {
			t.Fatalf("missing diagnostic %q: %s", field, logs)
		}
	}
}

func TestMerchantProfileNormalizesCategoriesWithoutLocationRequirement(t *testing.T) {
	store := &fakeMerchantUpdateStore{}
	_, err := NewUpdateMerchantLogic(store).UpdateMerchant(context.Background(), "101", UpdateMerchantReq{
		Name: "童装店", MainCategories: []string{" 童装 ", "", "童装", "卫衣"}, BusinessProfile: testFactoryBusinessProfile(), MapVisible: boolPtr(false), LocationAction: "clear",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(store.patch.MainCategories, ",") != "童装,卫衣" {
		t.Fatalf("categories = %#v", store.patch.MainCategories)
	}
}

func TestMerchantDirectoryAndReportLogsDoNotExposeDatabaseDetail(t *testing.T) {
	var output bytes.Buffer
	previous := logx.Reset()
	logx.SetWriter(logx.NewWriter(&output))
	t.Cleanup(func() {
		if writer := logx.Reset(); writer != nil {
			_ = writer.Close()
		}
		if previous != nil {
			logx.SetWriter(previous)
		}
	})
	failure := &pq.Error{Code: "08006", Message: "private-contact-13812345678"}
	_, _ = NewMerchantDirectoryLogic(&fakeDirectoryStore{listErr: failure}).ListDirectory(context.Background(), ListMerchantDirectoryReq{})
	_, _ = NewMerchantDirectoryLogic(&fakeDirectoryStore{countErr: failure}).ListDirectory(context.Background(), ListMerchantDirectoryReq{})
	_, _ = NewMerchantDirectoryLogic(&fakeDirectoryStore{currentErr: failure}).GetLocationContext(context.Background(), "101")
	_, _ = NewMerchantDirectoryLogic(&fakeDirectoryStore{current: model.MerchantDirectoryItem{MerchantID: "101", Latitude: 30, Longitude: 120}, nearbyErr: failure}).GetLocationContext(context.Background(), "101")
	_, _ = NewMerchantLocationReportLogic(&fakeMerchantLocationReportStore{err: failure}).CreateMerchantLocationReport(context.Background(), "101", "201", CreateMerchantLocationReportReq{ReasonCode: "other"})
	if strings.Contains(output.String(), "private-contact") {
		t.Fatal("merchant log contains database detail")
	}
	for _, operation := range []string{"list_merchant_directory", "count_merchant_directory", "get_visible_merchant_location", "list_nearby_visible_merchants", "create_location_report"} {
		if !strings.Contains(output.String(), operation) {
			t.Fatalf("missing operation %s", operation)
		}
	}
}
