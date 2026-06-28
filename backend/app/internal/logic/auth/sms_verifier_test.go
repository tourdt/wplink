package auth

import (
	"context"
	"testing"

	"wplink/backend/app/internal/config"
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
