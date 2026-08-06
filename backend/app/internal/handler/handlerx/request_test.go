package handlerx

import (
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadLimitedBodyClosesBodyWhenLimitIsNegative(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader("secret")}
	r := &http.Request{Body: body}

	if _, err := ReadLimitedBody(r, -1); err == nil {
		t.Fatal("ReadLimitedBody() error = nil, want invalid limit error")
	}
	if !body.closed {
		t.Fatal("request body was not closed for invalid negative limit")
	}
}

func TestReadLimitedBodyAcceptsSmallPayloadAtMaxInt64Limit(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("payload"))

	body, err := ReadLimitedBody(r, math.MaxInt64)
	if err != nil {
		t.Fatalf("ReadLimitedBody() error = %v", err)
	}
	if string(body) != "payload" {
		t.Fatalf("body = %q, want payload", body)
	}
}

func TestReadLimitedBodyRejectsPayloadBeyondLimit(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("12345"))
	if _, err := ReadLimitedBody(r, 4); err == nil {
		t.Fatal("ReadLimitedBody() error = nil, want oversized payload error")
	}
}

func TestReadLimitedBodyAcceptsPayloadAtLimit(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("1234"))
	body, err := ReadLimitedBody(r, 4)
	if err != nil {
		t.Fatalf("ReadLimitedBody() error = %v", err)
	}
	if string(body) != "1234" {
		t.Fatalf("body = %q, want 1234", body)
	}
}

func TestClientIPUsesTrustedProxyHeadersBeforeRemoteAddress(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:54321"
	r.Header.Set("X-Real-IP", "198.51.100.2")
	r.Header.Set("X-Forwarded-For", " 203.0.113.8, 10.0.0.2 ")

	if got := ClientIP(r); got != "203.0.113.8" {
		t.Fatalf("ClientIP() = %q, want first forwarded address", got)
	}
}

func TestClientIPFallsBackToRemoteHost(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.0.2.10:8080"

	if got := ClientIP(r); got != "192.0.2.10" {
		t.Fatalf("ClientIP() = %q, want remote host", got)
	}
}

func TestClientIPReturnsUnknownForMissingRequest(t *testing.T) {
	if got := ClientIP(nil); got != "unknown" {
		t.Fatalf("ClientIP(nil) = %q, want unknown", got)
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (b *trackingReadCloser) Close() error {
	b.closed = true
	return nil
}
