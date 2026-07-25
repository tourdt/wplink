package contentaudit

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestSHA1WechatCallbackVerifierAcceptsValidSignatureAndRecordsAfterSuccess(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	verifier := NewSHA1WechatCallbackVerifier("callback-token", 5*time.Minute)
	verifier.now = func() time.Time { return now }

	timestamp := "1700000000"
	nonce := "nonce-1"
	signature := callbackSignature("callback-token", timestamp, nonce)

	if err := verifier.Verify(signature, timestamp, nonce, false); err != nil {
		t.Fatalf("Verify(check only) error = %v", err)
	}
	if err := verifier.Verify(signature, timestamp, nonce, true); err != nil {
		t.Fatalf("Verify(remember) error = %v", err)
	}
	if err := verifier.Verify(signature, timestamp, nonce, false); !errors.Is(err, ErrWechatCallbackReplay) {
		t.Fatalf("Verify(replay) error = %v, want ErrWechatCallbackReplay", err)
	}
}

func TestSHA1WechatCallbackVerifierRejectsInvalidSignatureAndExpiredTimestamp(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	verifier := NewSHA1WechatCallbackVerifier("callback-token", 5*time.Minute)
	verifier.now = func() time.Time { return now }

	if err := verifier.Verify("invalid", "1700000000", "nonce-1", false); err == nil {
		t.Fatal("Verify(invalid signature) error = nil")
	}

	expiredTimestamp := "1699999600"
	expiredSignature := callbackSignature("callback-token", expiredTimestamp, "nonce-2")
	if err := verifier.Verify(expiredSignature, expiredTimestamp, "nonce-2", false); err == nil {
		t.Fatal("Verify(expired timestamp) error = nil")
	}
}

func callbackSignature(token string, timestamp string, nonce string) string {
	values := []string{token, timestamp, nonce}
	sort.Strings(values)
	sum := sha1.Sum([]byte(strings.Join(values, "")))
	return hex.EncodeToString(sum[:])
}
