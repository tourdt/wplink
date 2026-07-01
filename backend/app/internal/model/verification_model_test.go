package model

import "testing"

func TestVerificationMessageTargetURLIncludesMerchantID(t *testing.T) {
	got := verificationMessageTargetURL("merchant-1")
	want := "/pages/verification/index?merchantId=merchant-1"
	if got != want {
		t.Fatalf("verificationMessageTargetURL() = %q, want %q", got, want)
	}
}

func TestVerificationLabelUsesPrimaryIdentityCopy(t *testing.T) {
	tests := []struct {
		verificationType string
		want             string
	}{
		{verificationType: "factory", want: "源头工厂认证"},
		{verificationType: "stall", want: "现货档口认证"},
		{verificationType: "stockist", want: "库存货源认证"},
		{verificationType: "service_provider", want: "配套服务认证"},
		{verificationType: "unknown", want: "商家认证"},
	}

	for _, tt := range tests {
		if got := verificationLabel(tt.verificationType); got != tt.want {
			t.Fatalf("verificationLabel(%q) = %q, want %q", tt.verificationType, got, tt.want)
		}
	}
}
