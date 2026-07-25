package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadLimitedBodyRejectsPayloadBeyondLimit(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("12345"))
	if _, err := readLimitedBody(req, 4); err == nil {
		t.Fatal("readLimitedBody() error = nil, want oversized payload error")
	}
}

func TestReadLimitedBodyAcceptsPayloadAtLimit(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("1234"))
	body, err := readLimitedBody(req, 4)
	if err != nil {
		t.Fatalf("readLimitedBody() error = %v", err)
	}
	if string(body) != "1234" {
		t.Fatalf("body = %q, want 1234", body)
	}
}
