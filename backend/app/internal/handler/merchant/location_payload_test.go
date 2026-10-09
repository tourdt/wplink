package merchant

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func TestUpdateMerchantPayloadPreservesMissingAndZeroCoordinates(t *testing.T) {
	var payload updateMerchantPayload
	if err := json.Unmarshal([]byte(`{"locationAction":"set","latitude":0}`), &payload); err != nil {
		t.Fatalf("decode update merchant payload: %v", err)
	}
	if payload.Latitude == nil || *payload.Latitude != 0 {
		t.Fatalf("latitude = %#v, want explicit zero coordinate", payload.Latitude)
	}
	if payload.Longitude != nil {
		t.Fatalf("longitude = %#v, want missing coordinate to remain nil", payload.Longitude)
	}
}

func TestUpdateMerchantPayloadRequiresProfileAndMapChoice(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/merchants/merchant-1", strings.NewReader(`{"locationAction":"clear"}`))
	req.Header.Set("Content-Type", "application/json")

	var payload updateMerchantPayload
	if err := httpx.Parse(req, &payload); err == nil {
		t.Fatal("missing full-form fields accepted")
	}
}

func TestMerchantLocationPayloadOmitsUnavailableCoordinates(t *testing.T) {
	// 即使生成类型中意外带有数值坐标，Handler 外层 Payload 也必须以不可见位置覆盖并彻底省略坐标。
	payload, err := json.Marshal(merchantDetailPayload{
		MerchantDetailResp: types.MerchantDetailResp{Location: types.MerchantLocationInfo{Latitude: 30.8732, Longitude: 120.2255}},
		Location:           merchantLocationPayload{Status: "risk_hidden", Available: false},
	})
	if err != nil {
		t.Fatalf("marshal merchant location payload: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode merchant detail payload: %v", err)
	}
	location, ok := decoded["location"].(map[string]interface{})
	if !ok {
		t.Fatalf("payload = %s, want structured location object", payload)
	}
	if _, exists := location["latitude"]; exists {
		t.Fatalf("payload = %s, unavailable location must omit latitude", payload)
	}
	if _, exists := location["longitude"]; exists {
		t.Fatalf("payload = %s, unavailable location must omit longitude", payload)
	}
}

func TestMerchantPublicContactPayloadOmitsPrivateFields(t *testing.T) {
	payload, err := json.Marshal(merchantDetailPayload{Contact: &merchantContactPayload{Phone: "13800000000"}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	contact, ok := decoded["contact"].(map[string]interface{})
	if !ok || contact["phone"] != "13800000000" {
		t.Fatalf("contact = %#v", decoded["contact"])
	}
	for _, key := range []string{"name", "wechat", "phoneMasked", "wechatMasked"} {
		if _, exists := contact[key]; exists {
			t.Fatalf("private field %q exposed: %#v", key, contact)
		}
	}
}
