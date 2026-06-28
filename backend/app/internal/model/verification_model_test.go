package model

import "testing"

func TestVerificationMessageTargetURLIncludesMerchantID(t *testing.T) {
	got := verificationMessageTargetURL("merchant-1")
	want := "/pages/verification/index?merchantId=merchant-1"
	if got != want {
		t.Fatalf("verificationMessageTargetURL() = %q, want %q", got, want)
	}
}
