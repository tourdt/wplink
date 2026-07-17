package location

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
)

func TestTencentMapGeocoderReverseGeocodeUsesRecommendedAddressAndPOIName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "map-key" {
			t.Fatalf("key = %q, want configured key", r.URL.Query().Get("key"))
		}
		if r.URL.Query().Get("location") != "30.873200,120.225500" {
			t.Fatalf("location = %q, want formatted coordinate", r.URL.Query().Get("location"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":0,
			"message":"query ok",
			"result":{
				"address":"浙江省湖州市吴兴区织里镇",
				"formatted_addresses":{"recommend":"织里童装城一区附近","rough":"织里镇附近"},
				"address_component":{"province":"浙江省"},
				"pois":[{"title":"织里童装城一区","address":"浙江省湖州市吴兴区织里镇童装城"}]
			}
		}`))
	}))
	defer server.Close()

	geocoder := NewTencentMapGeocoder(config.TencentMapConfig{
		Key:            "map-key",
		RequestTimeout: time.Second,
	}, server.Client()).WithBaseURL(server.URL)

	resp, err := geocoder.ReverseGeocode(context.Background(), 30.8732, 120.2255)
	if err != nil {
		t.Fatalf("ReverseGeocode() error = %v", err)
	}
	if resp.Address != "织里童装城一区附近" || resp.Name != "织里童装城一区" || resp.Province != "浙江省" {
		t.Fatalf("resp = %#v, want recommended address and first poi name", resp)
	}
}

func TestTencentMapGeocoderMapsDailyQuotaExhaustedToRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":121,"message":"此key每日调用量已达到上限"}`))
	}))
	defer server.Close()

	geocoder := NewTencentMapGeocoder(config.TencentMapConfig{
		Key:            "map-key",
		RequestTimeout: time.Second,
	}, server.Client()).WithBaseURL(server.URL)

	_, err := geocoder.ReverseGeocode(context.Background(), 30.8732, 120.2255)
	if err == nil || errx.CodeOf(err) != errx.CodeRateLimited {
		t.Fatalf("ReverseGeocode() error = %v, code = %s, want rate limited", err, errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "地址解析今日额度已用完，请手动填写详细地址" {
		t.Fatalf("public message = %q, want quota exhausted guidance", errx.PublicMessage(err))
	}
}

func TestLocationLogicRejectsInvalidCoordinate(t *testing.T) {
	_, err := NewLogic(fakeReverseGeocoder{}).ReverseGeocode(context.Background(), ReverseGeocodeReq{
		Latitude:  "120",
		Longitude: "120.2255",
	})
	if err == nil || !strings.Contains(err.Error(), "纬度不正确") {
		t.Fatalf("ReverseGeocode() error = %v, want latitude validation error", err)
	}
}

type fakeReverseGeocoder struct{}

func (fakeReverseGeocoder) ReverseGeocode(ctx context.Context, latitude float64, longitude float64) (ReverseGeocodeResp, error) {
	return ReverseGeocodeResp{Address: "织里童装城一区附近", Name: "织里童装城一区", Province: "浙江省"}, nil
}
