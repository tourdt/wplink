package location

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
	"wplink/backend/common/externalcall"
)

type recordingExternalCallObserver struct {
	events []externalcall.Event
}

func (o *recordingExternalCallObserver) Observe(_ context.Context, event externalcall.Event) {
	o.events = append(o.events, event)
}

type locationRoundTripFunc func(*http.Request) (*http.Response, error)

func (f locationRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func testTencentMapResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestTencentMapGeocoderObservesHTTPOutcomes(t *testing.T) {
	tests := []struct {
		name          string
		response      *http.Response
		transportErr  error
		wantOutcome   externalcall.Outcome
		wantStatus    int
		wantError     bool
		wantErrorCode string
	}{
		{
			name:        "success",
			response:    testTencentMapResponse(http.StatusOK, `{"status":0,"result":{"address":"浙江省湖州市吴兴区织里镇"}}`),
			wantOutcome: externalcall.OutcomeSuccess,
			wantStatus:  http.StatusOK,
		},
		{
			name:          "provider rejected",
			response:      testTencentMapResponse(http.StatusOK, `{"status":121,"message":"此key每日调用量已达到上限"}`),
			wantOutcome:   externalcall.OutcomeProviderRejected,
			wantStatus:    http.StatusOK,
			wantError:     true,
			wantErrorCode: errx.CodeRateLimited,
		},
		{
			name:          "http error",
			response:      testTencentMapResponse(http.StatusTooManyRequests, `{}`),
			wantOutcome:   externalcall.OutcomeHTTPError,
			wantStatus:    http.StatusTooManyRequests,
			wantError:     true,
			wantErrorCode: errx.CodeRateLimited,
		},
		{
			name:        "decode error",
			response:    testTencentMapResponse(http.StatusOK, `{`),
			wantOutcome: externalcall.OutcomeDecodeError,
			wantStatus:  http.StatusOK,
			wantError:   true,
		},
		{
			name:         "timeout",
			transportErr: context.DeadlineExceeded,
			wantOutcome:  externalcall.OutcomeTimeout,
			wantError:    true,
		},
		{
			name:        "missing required address",
			response:    testTencentMapResponse(http.StatusOK, `{"status":0,"result":{}}`),
			wantOutcome: externalcall.OutcomeDecodeError,
			wantStatus:  http.StatusOK,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingExternalCallObserver{}
			client := &http.Client{Transport: locationRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return tt.response, tt.transportErr
			})}
			geocoder := NewTencentMapGeocoder(config.TencentMapConfig{Key: "map-key"}, client, observer)

			_, err := geocoder.ReverseGeocode(context.Background(), 30.8732, 120.2255)
			if tt.wantError && err == nil {
				t.Fatal("ReverseGeocode() error = nil, want error")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("ReverseGeocode() error = %v, want nil", err)
			}
			if tt.wantErrorCode != "" && errx.CodeOf(err) != tt.wantErrorCode {
				t.Fatalf("ReverseGeocode() code = %q, want %q", errx.CodeOf(err), tt.wantErrorCode)
			}
			if len(observer.events) != 1 {
				t.Fatalf("events = %d, want exactly 1", len(observer.events))
			}
			event := observer.events[0]
			if event.Provider != externalcall.ProviderTencentMap {
				t.Fatalf("provider = %q, want %q", event.Provider, externalcall.ProviderTencentMap)
			}
			if event.Operation != externalcall.OperationReverseGeocode {
				t.Fatalf("operation = %q, want %q", event.Operation, externalcall.OperationReverseGeocode)
			}
			if event.Outcome != tt.wantOutcome {
				t.Fatalf("outcome = %q, want %q", event.Outcome, tt.wantOutcome)
			}
			if event.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", event.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestTencentMapGeocoderDoesNotObserveLocalConfigurationError(t *testing.T) {
	observer := &recordingExternalCallObserver{}
	geocoder := NewTencentMapGeocoder(config.TencentMapConfig{Key: "map-key"}, nil, observer).WithBaseURL("://invalid")

	_, _ = geocoder.ReverseGeocode(context.Background(), 30.8732, 120.2255)

	if len(observer.events) != 0 {
		t.Fatalf("events = %d, want 0", len(observer.events))
	}
}

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
