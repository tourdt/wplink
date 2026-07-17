package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	locationlogic "wplink/backend/app/internal/logic/location"
)

func TestAPIRouterReverseGeocodesLocation(t *testing.T) {
	geocoder := &fakeLocationGeocoder{
		resp: locationlogic.ReverseGeocodeResp{Address: "织里童装城一区附近", Name: "织里童装城一区", Province: "浙江省"},
	}
	router := NewAPIRouter(&fakeCityAPIStore{}, WithLocationGeocoder(geocoder))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations/reverse-geocode?latitude=30.8732&longitude=120.2255", nil)
	router.ServeHTTP(rec, req)

	data := decodeEnvelopeData(t, rec, http.StatusOK)
	if data["address"] != "织里童装城一区附近" || data["name"] != "织里童装城一区" || data["province"] != "浙江省" {
		t.Fatalf("data = %#v, want reverse geocode result", data)
	}
	if geocoder.latitude != 30.8732 || geocoder.longitude != 120.2255 {
		t.Fatalf("coordinate = %.4f,%.4f, want request coordinate", geocoder.latitude, geocoder.longitude)
	}
}

type fakeLocationGeocoder struct {
	resp      locationlogic.ReverseGeocodeResp
	latitude  float64
	longitude float64
}

func (g *fakeLocationGeocoder) ReverseGeocode(ctx context.Context, latitude float64, longitude float64) (locationlogic.ReverseGeocodeResp, error) {
	g.latitude = latitude
	g.longitude = longitude
	return g.resp, nil
}
