package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	locationlogic "wplink/backend/app/internal/logic/location"
	"wplink/backend/app/internal/svc"
	"wplink/backend/common/errx"
)

func TestLocationReverseGeocodesThroughGeneratedRoutes(t *testing.T) {
	geocoder := &fakeLocationGeocoder{
		resp: locationlogic.ReverseGeocodeResp{Address: "织里童装城一区附近", Name: "织里童装城一区", Province: "浙江省"},
	}
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{LocationGeocoder: geocoder})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations/reverse-geocode?latitude=30.8732&longitude=120.2255", nil)
	server.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["address"] != "织里童装城一区附近" || data["name"] != "织里童装城一区" || data["province"] != "浙江省" {
		t.Fatalf("data = %#v, want reverse geocode result", data)
	}
	if geocoder.latitude != 30.8732 || geocoder.longitude != 120.2255 {
		t.Fatalf("coordinate = %.4f,%.4f, want request coordinate", geocoder.latitude, geocoder.longitude)
	}
}

func TestLocationReturnsRateLimitedThroughGeneratedRoutes(t *testing.T) {
	geocoder := &fakeLocationGeocoder{
		err: errx.New(errx.CodeRateLimited, "地址解析今日额度已用完，请手动填写详细地址"),
	}
	server := newGeneratedAPIServerWithFailClosedAdminAuth(t, &svc.ServiceContext{LocationGeocoder: geocoder})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations/reverse-geocode?latitude=30.8732&longitude=120.2255", nil)
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d body = %s, want 429", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["errorCode"] != errx.CodeRateLimited || body["msg"] != "地址解析今日额度已用完，请手动填写详细地址" {
		t.Fatalf("body = %#v, want rate limited response", body)
	}
}

type fakeLocationGeocoder struct {
	resp      locationlogic.ReverseGeocodeResp
	err       error
	latitude  float64
	longitude float64
}

func (g *fakeLocationGeocoder) ReverseGeocode(ctx context.Context, latitude float64, longitude float64) (locationlogic.ReverseGeocodeResp, error) {
	g.latitude = latitude
	g.longitude = longitude
	if g.err != nil {
		return locationlogic.ReverseGeocodeResp{}, g.err
	}
	return g.resp, nil
}
