package auth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
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

func testSMSResponse(status int, body string) *http.Response {
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

func TestConfiguredSMSVerifierObservesHTTPOutcomes(t *testing.T) {
	tests := []struct {
		name          string
		operation     func(*ConfiguredSMSVerifier) error
		wantOperation string
		response      *http.Response
		transportErr  error
		want          externalcall.Outcome
		wantStatus    int
		wantError     bool
	}{
		{
			name: "send success",
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "18800000001")
			},
			wantOperation: externalcall.OperationSMSCodeSend,
			response:      testSMSResponse(http.StatusOK, `{"ok":true}`),
			want:          externalcall.OutcomeSuccess,
			wantStatus:    http.StatusOK,
		},
		{
			name: "verify provider rejected",
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.VerifySMSCode(context.Background(), "18800000001", "000000")
			},
			wantOperation: externalcall.OperationSMSCodeVerify,
			response:      testSMSResponse(http.StatusOK, `{"valid":false}`),
			want:          externalcall.OutcomeProviderRejected,
			wantStatus:    http.StatusOK,
			wantError:     true,
		},
		{
			name: "http error",
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "18800000001")
			},
			wantOperation: externalcall.OperationSMSCodeSend,
			response:      testSMSResponse(http.StatusBadGateway, `{}`),
			want:          externalcall.OutcomeHTTPError,
			wantStatus:    http.StatusBadGateway,
			wantError:     true,
		},
		{
			name: "decode error",
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.VerifySMSCode(context.Background(), "18800000001", "123456")
			},
			wantOperation: externalcall.OperationSMSCodeVerify,
			response:      testSMSResponse(http.StatusOK, `{`),
			want:          externalcall.OutcomeDecodeError,
			wantStatus:    http.StatusOK,
			wantError:     true,
		},
		{
			name: "timeout",
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "18800000001")
			},
			wantOperation: externalcall.OperationSMSCodeSend,
			transportErr:  context.DeadlineExceeded,
			want:          externalcall.OutcomeTimeout,
			wantError:     true,
		},
		{
			name: "connection error",
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "18800000001")
			},
			wantOperation: externalcall.OperationSMSCodeSend,
			transportErr:  errors.New("connection reset"),
			want:          externalcall.OutcomeTransportError,
			wantError:     true,
		},
		{
			name: "canceled",
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "18800000001")
			},
			wantOperation: externalcall.OperationSMSCodeSend,
			transportErr:  context.Canceled,
			want:          externalcall.OutcomeCanceled,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingExternalCallObserver{}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return tt.response, tt.transportErr
			})}
			verifier := NewConfiguredSMSVerifierWithLimiter(config.SMSConfig{
				Provider:  "http",
				SendURL:   "https://sms.example.test/send",
				VerifyURL: "https://sms.example.test/verify",
			}, client, NewMemorySMSSendLimiter(), observer)

			err := tt.operation(verifier)
			if tt.wantError && err == nil {
				t.Fatal("operation error = nil, want error")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("operation error = %v, want nil", err)
			}
			if len(observer.events) != 1 {
				t.Fatalf("events = %d, want exactly 1", len(observer.events))
			}
			event := observer.events[0]
			if event.Provider != externalcall.ProviderSMS {
				t.Fatalf("provider = %q, want %q", event.Provider, externalcall.ProviderSMS)
			}
			if event.Operation != tt.wantOperation {
				t.Fatalf("operation = %q, want %q", event.Operation, tt.wantOperation)
			}
			if event.Outcome != tt.want {
				t.Fatalf("outcome = %q, want %q", event.Outcome, tt.want)
			}
			if event.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", event.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestConfiguredSMSVerifierObservesBodyReadFailureAndRollsBackReservation(t *testing.T) {
	observer := &recordingExternalCallObserver{}
	limiter := &recordingSMSSendLimiter{}
	verifier := NewConfiguredSMSVerifierWithLimiter(config.SMSConfig{
		Provider: "http",
		SendURL:  "https://sms.example.test/send",
	}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       errorReadCloser{err: errors.New("response stream interrupted")},
		}, nil
	})}, limiter, observer)

	err := verifier.SendSMSCode(context.Background(), "18800000001")

	if err == nil || errx.CodeOf(err) != errx.CodeInternalError {
		t.Fatalf("SendSMSCode() error = %v, code = %q, want internal error", err, errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "短信验证码发送失败，请稍后重试" {
		t.Fatalf("public message = %q, want send failure guidance", errx.PublicMessage(err))
	}
	if limiter.rolledBackToken != "reservation-1" {
		t.Fatalf("rolled back token = %q, want reservation-1", limiter.rolledBackToken)
	}
	if len(observer.events) != 1 {
		t.Fatalf("events = %d, want exactly 1", len(observer.events))
	}
	event := observer.events[0]
	if event.Provider != externalcall.ProviderSMS || event.Operation != externalcall.OperationSMSCodeSend || event.Outcome != externalcall.OutcomeTransportError || event.StatusCode != http.StatusAccepted {
		t.Fatalf("event = %+v, want sms/code_send transport_error/%d", event, http.StatusAccepted)
	}
}

func TestConfiguredSMSVerifierRequestCreationFailureDoesNotObserveAndRollsBack(t *testing.T) {
	observer := &recordingExternalCallObserver{}
	limiter := &recordingSMSSendLimiter{}
	requests := 0
	verifier := NewConfiguredSMSVerifierWithLimiter(config.SMSConfig{
		Provider: "http",
		SendURL:  "https://sms.example.test/send",
	}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return testSMSResponse(http.StatusOK, `{"ok":true}`), nil
	})}, limiter, observer)

	err := verifier.SendSMSCode(nil, "18800000001")

	if err == nil || errx.CodeOf(err) != errx.CodeInternalError {
		t.Fatalf("SendSMSCode() error = %v, code = %q, want internal error", err, errx.CodeOf(err))
	}
	if errx.PublicMessage(err) != "短信验证码发送失败，请稍后重试" {
		t.Fatalf("public message = %q, want send failure guidance", errx.PublicMessage(err))
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
	if len(observer.events) != 0 {
		t.Fatalf("events = %d, want 0", len(observer.events))
	}
	if limiter.rolledBackToken != "reservation-1" {
		t.Fatalf("rolled back token = %q, want reservation-1", limiter.rolledBackToken)
	}
}

func TestConfiguredSMSVerifierUsesOptionalSingleObserver(t *testing.T) {
	newVerifier := func(observers ...externalcall.Observer) *ConfiguredSMSVerifier {
		return NewConfiguredSMSVerifierWithLimiter(config.SMSConfig{
			Provider: "http",
			SendURL:  "https://sms.example.test/send",
		}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return testSMSResponse(http.StatusOK, `{"ok":true}`), nil
		})}, NewMemorySMSSendLimiter(), observers...)
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
			if err := newVerifier(tt.observers...).SendSMSCode(context.Background(), "18800000001"); err != nil {
				t.Fatalf("SendSMSCode() error = %v", err)
			}
			logText := collector.String()
			if !strings.Contains(logText, `"event":"external_call"`) || !strings.Contains(logText, `"provider":"sms"`) {
				t.Fatalf("log = %q, want default sms external_call observer", logText)
			}
		})
	}

	t.Run("multiple use first", func(t *testing.T) {
		first := &recordingExternalCallObserver{}
		second := &recordingExternalCallObserver{}
		if err := newVerifier(first, second).SendSMSCode(context.Background(), "18800000001"); err != nil {
			t.Fatalf("SendSMSCode() error = %v", err)
		}
		if len(first.events) != 1 {
			t.Fatalf("first events = %d, want 1", len(first.events))
		}
		if len(second.events) != 0 {
			t.Fatalf("second events = %d, want 0", len(second.events))
		}
	})
}

func TestConfiguredSMSVerifierDoesNotObserveWithoutHTTPRequest(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.SMSConfig
		limiter   SMSSendLimiter
		operation func(*ConfiguredSMSVerifier) error
	}{
		{
			name: "local config error",
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "18800000001")
			},
		},
		{
			name: "local validation",
			cfg:  config.SMSConfig{Provider: "http", SendURL: "https://sms.example.test/send"},
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "")
			},
		},
		{
			name: "dev provider",
			cfg:  config.SMSConfig{Provider: "dev"},
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "18800000001")
			},
		},
		{
			name:    "limiter rejected",
			cfg:     config.SMSConfig{Provider: "http", SendURL: "https://sms.example.test/send"},
			limiter: rejectingSMSSendLimiter{},
			operation: func(v *ConfiguredSMSVerifier) error {
				return v.SendSMSCode(context.Background(), "18800000001")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingExternalCallObserver{}
			requests := 0
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return testSMSResponse(http.StatusOK, `{"ok":true}`), nil
			})}
			verifier := NewConfiguredSMSVerifierWithLimiter(tt.cfg, client, tt.limiter, observer)

			_ = tt.operation(verifier)

			if requests != 0 {
				t.Fatalf("requests = %d, want 0", requests)
			}
			if len(observer.events) != 0 {
				t.Fatalf("events = %d, want 0", len(observer.events))
			}
		})
	}
}

type rejectingSMSSendLimiter struct{}

func (rejectingSMSSendLimiter) Reserve(context.Context, string, time.Time, time.Duration, int) (string, error) {
	return "", ErrSMSSendTooFrequent
}

func (rejectingSMSSendLimiter) Rollback(context.Context, string, time.Time, string) error {
	return nil
}

func TestConfiguredSMSVerifierAcceptsDevCode(t *testing.T) {
	verifier := NewConfiguredSMSVerifier(config.SMSConfig{Provider: "dev", DevCode: "123456"})

	if err := verifier.VerifySMSCode(context.Background(), "18800000001", "123456"); err != nil {
		t.Fatalf("VerifySMSCode() error = %v", err)
	}
}

func TestConfiguredSMSVerifierRejectsUnsupportedProvider(t *testing.T) {
	verifier := NewConfiguredSMSVerifier(config.SMSConfig{Provider: "aliyun", AccessKeyID: "ak", AccessKeySecret: "sk"})

	err := verifier.VerifySMSCode(context.Background(), "18800000001", "123456")
	if err == nil {
		t.Fatal("VerifySMSCode() error = nil, want unsupported provider error")
	}
}

func TestConfiguredSMSVerifierSendsCodeWithHTTPProvider(t *testing.T) {
	var body string
	verifier := NewConfiguredSMSVerifierWithHTTP(config.SMSConfig{
		Provider:        "http",
		SendURL:         "https://sms.example.test/send",
		AccessKeySecret: "sms-secret",
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://sms.example.test/send" {
			t.Fatalf("url = %s, want send url", r.URL.String())
		}
		if r.Header.Get("Authorization") != "Bearer sms-secret" {
			t.Fatalf("auth = %q, want bearer secret", r.Header.Get("Authorization"))
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		body = string(bodyBytes)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"ok":true}`)), Header: make(http.Header)}, nil
	})})

	if err := verifier.SendSMSCode(context.Background(), "18800000001"); err != nil {
		t.Fatalf("SendSMSCode() error = %v", err)
	}
	if !strings.Contains(body, `"phone":"18800000001"`) {
		t.Fatalf("body = %s, want phone payload", body)
	}
}

func TestConfiguredSMSVerifierRateLimitsRepeatedSend(t *testing.T) {
	requests := 0
	now := time.Date(2026, 6, 28, 10, 0, 0, 0, time.Local)
	verifier := NewConfiguredSMSVerifierWithHTTP(config.SMSConfig{
		Provider:        "http",
		SendURL:         "https://sms.example.test/send",
		SendMinInterval: time.Minute,
		DailySendLimit:  10,
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"ok":true}`)), Header: make(http.Header)}, nil
	})})
	verifier.now = func() time.Time { return now }

	if err := verifier.SendSMSCode(context.Background(), "18800000001"); err != nil {
		t.Fatalf("first SendSMSCode() error = %v", err)
	}
	err := verifier.SendSMSCode(context.Background(), "18800000001")
	if err == nil || errx.CodeOf(err) != errx.CodeRateLimited {
		t.Fatalf("second SendSMSCode() error = %v, want rate limited", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want only first request sent", requests)
	}
}

func TestConfiguredSMSVerifierRateLimitsDailySendCount(t *testing.T) {
	now := time.Date(2026, 6, 28, 10, 0, 0, 0, time.Local)
	verifier := NewConfiguredSMSVerifierWithHTTP(config.SMSConfig{
		Provider:        "http",
		SendURL:         "https://sms.example.test/send",
		SendMinInterval: time.Nanosecond,
		DailySendLimit:  2,
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"ok":true}`)), Header: make(http.Header)}, nil
	})})
	verifier.now = func() time.Time { return now }

	if err := verifier.SendSMSCode(context.Background(), "18800000001"); err != nil {
		t.Fatalf("first SendSMSCode() error = %v", err)
	}
	now = now.Add(time.Second)
	if err := verifier.SendSMSCode(context.Background(), "18800000001"); err != nil {
		t.Fatalf("second SendSMSCode() error = %v", err)
	}
	now = now.Add(time.Second)
	err := verifier.SendSMSCode(context.Background(), "18800000001")
	if err == nil || errx.CodeOf(err) != errx.CodeRateLimited {
		t.Fatalf("third SendSMSCode() error = %v, want daily rate limited", err)
	}
}

func TestConfiguredSMSVerifierVerifiesCodeWithHTTPProvider(t *testing.T) {
	verifier := NewConfiguredSMSVerifierWithHTTP(config.SMSConfig{
		Provider:        "http",
		VerifyURL:       "https://sms.example.test/verify",
		AccessKeySecret: "sms-secret",
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://sms.example.test/verify" {
			t.Fatalf("url = %s, want verify url", r.URL.String())
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"valid":true}`)), Header: make(http.Header)}, nil
	})})

	if err := verifier.VerifySMSCode(context.Background(), "18800000001", "123456"); err != nil {
		t.Fatalf("VerifySMSCode() error = %v", err)
	}
}

func TestConfiguredSMSVerifierRejectsFailedHTTPSend(t *testing.T) {
	verifier := NewConfiguredSMSVerifierWithHTTP(config.SMSConfig{
		Provider: "http",
		SendURL:  "https://sms.example.test/send",
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"ok":false}`)), Header: make(http.Header)}, nil
	})})

	if err := verifier.SendSMSCode(context.Background(), "18800000001"); err == nil {
		t.Fatal("SendSMSCode() error = nil, want failed send error")
	}
}

func TestConfiguredSMSVerifierRollsBackLimitReservationWhenProviderFails(t *testing.T) {
	limiter := &recordingSMSSendLimiter{}
	verifier := NewConfiguredSMSVerifierWithLimiter(config.SMSConfig{
		Provider: "http",
		SendURL:  "https://sms.example.test/send",
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	})}, limiter)

	if err := verifier.SendSMSCode(context.Background(), "18800000001"); err == nil {
		t.Fatal("SendSMSCode() error = nil, want provider failure")
	}
	if limiter.rolledBackToken != "reservation-1" {
		t.Fatalf("rolled back token = %q, want reservation-1", limiter.rolledBackToken)
	}
}

func TestConfiguredSMSVerifierRejectsInvalidHTTPCode(t *testing.T) {
	verifier := NewConfiguredSMSVerifierWithHTTP(config.SMSConfig{
		Provider:  "http",
		VerifyURL: "https://sms.example.test/verify",
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"valid":false}`)), Header: make(http.Header)}, nil
	})})

	if err := verifier.VerifySMSCode(context.Background(), "18800000001", "123456"); err == nil {
		t.Fatal("VerifySMSCode() error = nil, want invalid code error")
	}
}

type recordingSMSSendLimiter struct {
	rolledBackToken string
}

func (l *recordingSMSSendLimiter) Reserve(_ context.Context, _ string, _ time.Time, _ time.Duration, _ int) (string, error) {
	return "reservation-1", nil
}

func (l *recordingSMSSendLimiter) Rollback(_ context.Context, _ string, _ time.Time, reservationToken string) error {
	l.rolledBackToken = reservationToken
	return nil
}
