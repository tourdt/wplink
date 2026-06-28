package auth

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"wplink/backend/app/internal/config"
	"wplink/backend/common/errx"
)

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
