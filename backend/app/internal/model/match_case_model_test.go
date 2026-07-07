package model

import (
	"strings"
	"testing"
)

func TestMatchDemandOwnerMessageSQLTargetsDemandUser(t *testing.T) {
	sql := matchDemandOwnerProgressMessageSQL
	requiredSnippets := []string{
		"recipient_user_id",
		"purchase_demands pd",
		"pd.user_id",
		"'/pages/messages/index'",
		"mc.purchase_demand_id",
		"match_status_update",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("matchDemandOwnerProgressMessageSQL missing %q:\n%s", snippet, sql)
		}
	}
}

func TestMatchCreateDemandOwnerMessageSQLTargetsDemandUser(t *testing.T) {
	sql := matchCreateDemandOwnerMessageSQL
	requiredSnippets := []string{
		"recipient_user_id",
		"purchase_demands pd",
		"pd.user_id",
		"match_create",
		"采购需求已进入撮合",
		"'/pages/messages/index'",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("matchCreateDemandOwnerMessageSQL missing %q:\n%s", snippet, sql)
		}
	}
}

func TestMatchDemandOwnerMessageSQLDoesNotTargetRetiredDemandPages(t *testing.T) {
	for name, sql := range map[string]string{
		"progress": matchDemandOwnerProgressMessageSQL,
		"create":   matchCreateDemandOwnerMessageSQL,
	} {
		t.Run(name, func(t *testing.T) {
			for _, retiredPath := range []string{"/pages/my-demands/index", "/pages/demand/index", "/pages/demand-success/index"} {
				if strings.Contains(sql, retiredPath) {
					t.Fatalf("%s SQL contains retired path %q:\n%s", name, retiredPath, sql)
				}
			}
		})
	}
}

func TestMatchCreateMerchantMessageSQLTargetsParticipants(t *testing.T) {
	sql := matchCreateMerchantMessageSQL
	requiredSnippets := []string{
		"recipient_role_code",
		"match_case_participants",
		"merchant_id IS NOT NULL",
		"match_create",
		"新的撮合机会",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("matchCreateMerchantMessageSQL missing %q:\n%s", snippet, sql)
		}
	}
}

func TestDemandStatusForMatchStatus(t *testing.T) {
	cases := map[string]string{
		MatchCaseStatusOpen:      "matching",
		MatchCaseStatusContacted: "contacted",
		MatchCaseStatusSucceeded: "closed",
		MatchCaseStatusFailed:    "closed",
		MatchCaseStatusClosed:    "closed",
	}
	for status, want := range cases {
		if got := demandStatusForMatchStatus(status); got != want {
			t.Fatalf("demandStatusForMatchStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestMatchDemandStatusSyncSQLUpdatesLinkedDemand(t *testing.T) {
	sql := matchDemandStatusSyncSQL
	requiredSnippets := []string{
		"UPDATE purchase_demands pd",
		"SET status = $2",
		"FROM match_cases mc",
		"mc.purchase_demand_id = pd.id",
		"mc.id = $1::bigint",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("matchDemandStatusSyncSQL missing %q:\n%s", snippet, sql)
		}
	}
}
