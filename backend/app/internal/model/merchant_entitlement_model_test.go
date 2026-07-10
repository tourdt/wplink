package model

import (
	"os"
	"strings"
	"testing"
)

func TestTopVoucherEntitlementsUseMerchantEntitlements(t *testing.T) {
	source, err := os.ReadFile("merchant_entitlement_model.go")
	if err != nil {
		t.Fatalf("ReadFile(merchant_entitlement_model.go) error = %v", err)
	}
	text := string(source)

	for _, snippet := range []string{
		"EntitlementTypeTopVoucher",
		"FROM merchant_entitlements",
		"entitlement_type = $2",
		"remaining_amount > 0",
		"top_duration_hours",
		"allowed_type_codes",
		"merchant_entitlement_usage_records",
		"top_started_at = now()",
		"top_expires_at = now() + make_interval(hours =>",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("merchant_entitlement_model.go should include top entitlement snippet %q", snippet)
		}
	}
	if strings.Contains(text, "FROM top_vouchers") || strings.Contains(text, "UPDATE top_vouchers") {
		t.Fatalf("merchant_entitlement_model.go must not read or update top_vouchers:\n%s", text)
	}
}

func TestUsageRecordsAreWrittenForPublishRefreshAndTop(t *testing.T) {
	resourceSource, err := os.ReadFile("resource_model.go")
	if err != nil {
		t.Fatalf("ReadFile(resource_model.go) error = %v", err)
	}
	entitlementSource, err := os.ReadFile("merchant_entitlement_model.go")
	if err != nil {
		t.Fatalf("ReadFile(merchant_entitlement_model.go) error = %v", err)
	}
	combined := string(resourceSource) + "\n" + string(entitlementSource)

	for _, snippet := range []string{
		"ActionTypePublishResource",
		"ActionTypeRefreshResource",
		"ActionTypeTopResource",
		"recordEntitlementUsageTx",
		"before_remaining_amount",
		"after_remaining_amount",
		"resource_id",
	} {
		if !strings.Contains(combined, snippet) {
			t.Fatalf("entitlement usage implementation should include snippet %q", snippet)
		}
	}
}
