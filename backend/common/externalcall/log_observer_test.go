package externalcall

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
