package model

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
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

func TestQuotaPackBenefitsUse180DayExpiration(t *testing.T) {
	source, err := os.ReadFile("vip_model.go")
	if err != nil {
		t.Fatalf("ReadFile(vip_model.go) error = %v", err)
	}
	text := string(source)
	if !regexp.MustCompile(`quotaPackValidityDays\s*=\s*180`).MatchString(text) {
		t.Fatal("vip_model.go should keep quota pack validity at 180 days")
	}
	for _, snippet := range []string{
		"quotaPackExpiresAt := paidAt.AddDate(0, 0, quotaPackValidityDays)",
		"VALUES ($1, $2, 'quota_pack', $3, $3, $4, $5, 'active')",
		"VALUES ($1, $2, 'quota_pack', $3, $3, $4, $5, 'active', '[]'::jsonb, $6)",
		"EntitlementTypeTopVoucher",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("vip_model.go should include quota pack expiration snippet %q", snippet)
		}
	}
	if strings.Contains(text, "VALUES ($1, $2, 'quota_pack', $3, $3, $4, NULL, 'active')") {
		t.Fatal("quota pack entitlements must not be granted without expiration")
	}
	if strings.Contains(text, "INSERT INTO top_vouchers") {
		t.Fatal("quota pack top vouchers must be granted through merchant_entitlements, not top_vouchers")
	}
}

func TestVIPBenefitGrantEndUses30DayValidity(t *testing.T) {
	periodStart := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 6, 0)

	got := vipBenefitGrantEnd(periodStart, periodEnd)
	want := periodStart.AddDate(0, 0, 30)
	if !got.Equal(want) {
		t.Fatalf("vipBenefitGrantEnd() = %s, want 30-day validity ending %s", got, want)
	}

	shortSubscriptionEnd := periodStart.AddDate(0, 0, 12)
	got = vipBenefitGrantEnd(periodStart, shortSubscriptionEnd)
	if !got.Equal(shortSubscriptionEnd) {
		t.Fatalf("vipBenefitGrantEnd() = %s, want capped subscription end %s", got, shortSubscriptionEnd)
	}
}
