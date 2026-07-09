package model

import (
	"strings"
	"testing"
)

func TestBuildVIPOutTradeNoKeepsWechatSafeLength(t *testing.T) {
	got := buildVIPOutTradeNo("order-1234567890abcdefghijklmnopqrstuvwxyz")
	if !strings.HasPrefix(got, "VIP") {
		t.Fatalf("out trade no = %q, want VIP prefix", got)
	}
	if len(got) > 32 {
		t.Fatalf("out trade no length = %d, want <= 32", len(got))
	}
}

func TestVIPBenefitSnapshotFromJSONDefaultsToQuotaPolicy(t *testing.T) {
	snapshot := VIPBenefitSnapshotFromJSON(JSONMap{
		"publishQuota":     float64(80),
		"refreshQuota":     float64(30),
		"topVoucherCount":  float64(3),
		"topDurationHours": float64(24),
	})
	if snapshot.PublishPolicy != "quota" || snapshot.PublishQuota != 80 || snapshot.RefreshQuota != 30 || snapshot.TopVoucherCount != 3 {
		t.Fatalf("snapshot = %#v, want quota benefits", snapshot)
	}
}
