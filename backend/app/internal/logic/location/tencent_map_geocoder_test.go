package location

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
	"wplink/backend/common/externalcall"

	"github.com/zeromicro/go-zero/core/logx/logtest"
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

type errorReadCloser struct {
	err error
}

func (r errorReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (errorReadCloser) Close() error {
	return nil
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
			name:         "connection error",
			transportErr: errors.New("connection reset"),
			wantOutcome:  externalcall.OutcomeTransportError,
			wantError:    true,
		},
		{
			name:         "canceled",
			transportErr: context.Canceled,
			wantOutcome:  externalcall.OutcomeCanceled,
			wantError:    true,
		},
		{
			name: "body read failure",
			response: &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     make(http.Header),
				Body:       errorReadCloser{err: errors.New("response stream interrupted")},
			},
			wantOutcome: externalcall.OutcomeTransportError,
			wantStatus:  http.StatusPartialContent,
			wantError:   true,
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

func TestTencentMapGeocoderTransportLogsDoNotLeakAPIKeyOrURL(t *testing.T) {
	const sentinelKey = "sentinel-map-key-review"
	collector := logtest.NewCollector(t)
	client := &http.Client{Transport: locationRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})}
	geocoder := NewTencentMapGeocoder(config.TencentMapConfig{Key: sentinelKey}, client)

	_, _ = geocoder.ReverseGeocode(context.Background(), 30.8732, 120.2255)

	logText := collector.String()
	for _, forbidden := range []string{sentinelKey, "key=", "https://apis.map.qq.com/ws/geocoder/v1/?"} {
		if strings.Contains(logText, forbidden) {
			t.Fatalf("log contains forbidden value %q: %s", forbidden, logText)
		}
	}
	for _, required := range []string{`"event":"external_call"`, `"provider":"tencent_map"`, `"operation":"reverse_geocode"`, `"outcome":"timeout"`} {
		if !strings.Contains(logText, required) {
			t.Fatalf("log = %q, want safe field %q", logText, required)
		}
	}
}

func TestTencentMapGeocoderTransportFailureLogsSafeCause(t *testing.T) {
	const (
		sentinelKey = "sentinel-map-key-cause"
		secretError = "secret-transport-detail"
	)
	tests := []struct {
		name      string
		err       error
		wantCause string
	}{
		{
			name:      "canceled",
			err:       context.Canceled,
			wantCause: "canceled",
		},
		{
			name:      "timeout",
			err:       context.DeadlineExceeded,
			wantCause: "timeout",
		},
		{
			name:      "dns",
			err:       &net.DNSError{Err: "lookup failed", Name: "dns-secret.example"},
			wantCause: "dns",
		},
		{
			name:      "tls certificate",
			err:       x509.UnknownAuthorityError{Cert: &x509.Certificate{}},
			wantCause: "tls",
		},
		{
			name:      "connection refused",
			err:       &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
			wantCause: "connection_refused",
		},
		{
			name:      "connection reset",
			err:       &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
			wantCause: "connection_reset",
		},
		{
			name:      "unexpected eof",
			err:       io.ErrUnexpectedEOF,
			wantCause: "unexpected_eof",
		},
		{
			name:      "network",
			err:       &net.OpError{Op: "read", Net: "tcp", Err: errors.New("opaque network failure")},
			wantCause: "network",
		},
		{
			name:      "unknown",
			err:       errors.New("opaque failure"),
			wantCause: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := logtest.NewCollector(t)
			observer := &recordingExternalCallObserver{}
			transportErr := fmt.Errorf("%s: %w", secretError, tt.err)
			client := &http.Client{Transport: locationRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, transportErr
			})}
			geocoder := NewTencentMapGeocoder(config.TencentMapConfig{Key: sentinelKey}, client, observer)

			_, _ = geocoder.ReverseGeocode(context.Background(), 30.8732, 120.2255)

			logText := collector.String()
			if !strings.Contains(logText, "cause="+tt.wantCause) {
				t.Fatalf("log = %q, want cause=%s", logText, tt.wantCause)
			}
			for _, forbidden := range []string{sentinelKey, "key=", "https://apis.map.qq.com/", secretError, "dns-secret.example", "opaque network failure", "opaque failure"} {
				if strings.Contains(logText, forbidden) {
					t.Fatalf("log contains forbidden value %q: %s", forbidden, logText)
				}
			}
		})
	}
}

func TestTencentMapGeocoderBodyReadFailureLogsSafeCause(t *testing.T) {
	const secretError = "secret-body-read-detail"
	tests := []struct {
		name      string
		err       error
		wantCause string
	}{
		{name: "connection reset", err: syscall.ECONNRESET, wantCause: "connection_reset"},
		{name: "unexpected eof", err: io.ErrUnexpectedEOF, wantCause: "unexpected_eof"},
		{name: "unknown", err: errors.New("opaque body failure"), wantCause: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := logtest.NewCollector(t)
			observer := &recordingExternalCallObserver{}
			readErr := fmt.Errorf("%s: %w", secretError, tt.err)
			client := &http.Client{Transport: locationRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusPartialContent,
					Header:     make(http.Header),
					Body:       errorReadCloser{err: readErr},
				}, nil
			})}
			geocoder := NewTencentMapGeocoder(config.TencentMapConfig{Key: "sentinel-body-key"}, client, observer)

			_, _ = geocoder.ReverseGeocode(context.Background(), 30.8732, 120.2255)

			logText := collector.String()
			if !strings.Contains(logText, "cause="+tt.wantCause) {
				t.Fatalf("log = %q, want cause=%s", logText, tt.wantCause)
			}
			for _, forbidden := range []string{"sentinel-body-key", "key=", "https://apis.map.qq.com/", secretError, "opaque body failure"} {
				if strings.Contains(logText, forbidden) {
					t.Fatalf("log contains forbidden value %q: %s", forbidden, logText)
				}
			}
			if len(observer.events) != 1 || observer.events[0].Outcome != externalcall.OutcomeTransportError || observer.events[0].StatusCode != http.StatusPartialContent {
				t.Fatalf("events = %+v, want exactly one transport_error/%d", observer.events, http.StatusPartialContent)
			}
		})
	}
}

func TestTencentMapGeocoderRequestCreationFailureDoesNotObserve(t *testing.T) {
	observer := &recordingExternalCallObserver{}
	requests := 0
	client := &http.Client{Transport: locationRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return testTencentMapResponse(http.StatusOK, `{"status":0,"result":{"address":"织里镇"}}`), nil
	})}
	geocoder := NewTencentMapGeocoder(config.TencentMapConfig{Key: "map-key"}, client, observer)

	_, err := geocoder.ReverseGeocode(nil, 30.8732, 120.2255)

	if err == nil || errx.CodeOf(err) != errx.CodeInternalError {
		t.Fatalf("ReverseGeocode() error = %v, code = %q, want internal error", err, errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "地址解析失败，请手动填写详细地址" {
		t.Fatalf("public message = %q, want request creation failure guidance", errx.PublicMessage(err))
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
	if len(observer.events) != 0 {
		t.Fatalf("events = %d, want 0", len(observer.events))
	}
}

func TestTencentMapGeocoderUsesOptionalSingleObserver(t *testing.T) {
	newGeocoder := func(observers ...externalcall.Observer) *TencentMapGeocoder {
		client := &http.Client{Transport: locationRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return testTencentMapResponse(http.StatusOK, `{"status":0,"result":{"address":"织里镇"}}`), nil
		})}
		return NewTencentMapGeocoder(config.TencentMapConfig{Key: "map-key"}, client, observers...)
	}

	for _, tt := range []struct {
		name      string
		observers []externalcall.Observer
	}{
		{name: "omitted"},
		{name: "explicit nil", observers: []externalcall.Observer{nil}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			collector := logtest.NewCollector(t)
			if _, err := newGeocoder(tt.observers...).ReverseGeocode(context.Background(), 30.8732, 120.2255); err != nil {
				t.Fatalf("ReverseGeocode() error = %v", err)
			}
			logText := collector.String()
			if !strings.Contains(logText, `"event":"external_call"`) || !strings.Contains(logText, `"provider":"tencent_map"`) {
				t.Fatalf("log = %q, want default tencent_map external_call observer", logText)
			}
		})
	}

	t.Run("multiple use first", func(t *testing.T) {
		first := &recordingExternalCallObserver{}
		second := &recordingExternalCallObserver{}
		if _, err := newGeocoder(first, second).ReverseGeocode(context.Background(), 30.8732, 120.2255); err != nil {
			t.Fatalf("ReverseGeocode() error = %v", err)
		}
		if len(first.events) != 1 {
			t.Fatalf("first events = %d, want 1", len(first.events))
		}
		if len(second.events) != 0 {
			t.Fatalf("second events = %d, want 0", len(second.events))
		}
	})
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
