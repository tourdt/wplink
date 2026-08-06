package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"wplink/backend/app/internal/handler/handlerx"
)

func TestReadLimitedBodyRejectsPayloadBeyondLimit(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("12345"))
	if _, err := handlerx.ReadLimitedBody(req, 4); err == nil {
		t.Fatal("handlerx.ReadLimitedBody() error = nil, want oversized payload error")
	}
}

func TestReadLimitedBodyAcceptsPayloadAtLimit(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("1234"))
	body, err := handlerx.ReadLimitedBody(req, 4)
	if err != nil {
		t.Fatalf("handlerx.ReadLimitedBody() error = %v", err)
	}
	if string(body) != "1234" {
		t.Fatalf("body = %q, want 1234", body)
	}
}
