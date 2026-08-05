package externalcall

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/logx/logtest"
)

func TestLogObserverWritesWhitelistedFieldsOnly(t *testing.T) {
	collector := logtest.NewCollector(t)
	observer := NewLogObserver()
	observer.Observe(context.Background(), Event{
		Provider:   ProviderSMS,
		Operation:  OperationSMSCodeSend,
		Outcome:    OutcomeHTTPError,
		Duration:   1500 * time.Millisecond,
		StatusCode: 502,
	})

	var payload map[string]any
	if err := json.Unmarshal(collector.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v, log = %s", err, collector.Bytes())
	}
	for key, want := range map[string]any{
		"event":       EventName,
		"provider":    ProviderSMS,
		"operation":   OperationSMSCodeSend,
		"outcome":     string(OutcomeHTTPError),
		"duration_ms": float64(1500),
		"status_code": float64(502),
	} {
		if payload[key] != want {
			t.Fatalf("payload[%q] = %#v, want %#v", key, payload[key], want)
		}
	}

	serialized := string(collector.Bytes())
	for _, forbiddenKey := range []string{"phone", "token", "authorization", "request_body", "response_body", "error_detail"} {
		if _, exists := payload[forbiddenKey]; exists {
			t.Fatalf("log contains forbidden key %q: %s", forbiddenKey, serialized)
		}
	}
	for _, forbidden := range []string{"18800000001", "sms-secret", "Bearer test-token"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("log contains forbidden value %q: %s", forbidden, serialized)
		}
	}
}

func TestLogObserverOmitsEmptyStatusCode(t *testing.T) {
	collector := logtest.NewCollector(t)
	NewLogObserver().Observe(context.Background(), Event{
		Provider:  ProviderWechat,
		Operation: OperationCodeToSession,
		Outcome:   OutcomeTimeout,
		Duration:  time.Second,
	})

	if strings.Contains(string(collector.Bytes()), "status_code") {
		t.Fatalf("log should omit status_code: %s", collector.Bytes())
	}
}

func TestLogObserverDoesNotInheritSensitiveContextFields(t *testing.T) {
	collector := logtest.NewCollector(t)
	ctx := logx.ContextWithFields(context.Background(),
		logx.Field("phone", "18800000001"),
		logx.Field("token", "sms-secret"),
		logx.Field("authorization", "Bearer test-token"),
	)

	NewLogObserver().Observe(ctx, Event{
		Provider:  ProviderSMS,
		Operation: OperationSMSCodeSend,
		Outcome:   OutcomeSuccess,
		Duration:  time.Second,
	})

	serialized := string(collector.Bytes())
	for _, forbidden := range []string{"phone", "18800000001", "token", "sms-secret", "authorization", "Bearer test-token"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("log inherits sensitive context field %q: %s", forbidden, serialized)
		}
	}
}

func TestLogObserverUsesExpectedLogLevel(t *testing.T) {
	tests := []struct {
		name    string
		outcome Outcome
		want    string
	}{
		{name: "success", outcome: OutcomeSuccess, want: "info"},
		{name: "canceled", outcome: OutcomeCanceled, want: "info"},
		{name: "provider rejected", outcome: OutcomeProviderRejected, want: "info"},
		{name: "timeout", outcome: OutcomeTimeout, want: "error"},
		{name: "transport error", outcome: OutcomeTransportError, want: "error"},
		{name: "http error", outcome: OutcomeHTTPError, want: "error"},
		{name: "decode error", outcome: OutcomeDecodeError, want: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := logtest.NewCollector(t)
			NewLogObserver().Observe(context.Background(), Event{
				Provider:  ProviderWechat,
				Operation: OperationCodeToSession,
				Outcome:   tt.outcome,
				Duration:  time.Second,
			})

			var payload map[string]any
			if err := json.Unmarshal(collector.Bytes(), &payload); err != nil {
				t.Fatalf("Unmarshal() error = %v, log = %s", err, collector.Bytes())
			}
			if got := payload["level"]; got != tt.want {
				t.Fatalf("level = %#v, want %q", got, tt.want)
			}
		})
	}
}
