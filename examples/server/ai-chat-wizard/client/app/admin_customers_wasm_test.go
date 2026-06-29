//go:build js && wasm

package app

import (
	"strings"
	"testing"
)

func TestParseFilterAndSortAdminUsersAppliesPersistedQueryState(parseT *testing.T) {
	parseT.Parallel()
	parseUsers := []adminUserRow{
		{UserID: 1, Email: "alpha@example.com", ConversationCount: 3, TotalCostUSD: 12, LastSeenAt: "2026-06-10T10:00:00Z"},
		{UserID: 2, Email: "beta@example.com", ConversationCount: 9, TotalCostUSD: 88, LastSeenAt: "2026-06-12T10:00:00Z"},
		{UserID: 3, Email: "blocked@example.com", ConversationCount: 1, TotalCostUSD: 5, LastSeenAt: "2026-06-11T10:00:00Z"},
	}
	parseStates := map[int64]string{3: "disabled"}
	parseFiltered := parseFilterAndSortAdminUsers(parseUsers, adminListQueryState{Filter: "active", Sort: "spend"}, parseStates)
	if len(parseFiltered) != 2 {
		parseT.Fatalf("len(filtered) = %d, want 2 active users", len(parseFiltered))
	}
	if parseFiltered[0].UserID != 2 || parseFiltered[1].UserID != 1 {
		parseT.Fatalf("filtered order = %+v, want spend-desc active users", parseFiltered)
	}
	parseBlocked := parseFilterAndSortAdminUsers(parseUsers, adminListQueryState{Search: "blocked", Filter: "disabled", Sort: "email"}, parseStates)
	if len(parseBlocked) != 1 || parseBlocked[0].UserID != 3 {
		parseT.Fatalf("blocked query = %+v, want disabled blocked user", parseBlocked)
	}
}

func TestParseAdminUserAccessStateForDetailPrefersAuditState(parseT *testing.T) {
	parseT.Parallel()
	parseDetail := adminUserDetailSnapshot{
		UserID: 9,
		AuditLogs: []adminAuditRow{
			{EventType: "admin.user.disable", Summary: "disabled for review"},
		},
	}
	if parseState := parseAdminUserAccessStateForDetail(parseDetail, map[int64]string{9: "active"}); parseState != "disabled" {
		parseT.Fatalf("state = %q, want disabled from audit log", parseState)
	}
	parseDetail.AuditLogs = []adminAuditRow{{EventType: "admin.user.restore"}}
	if parseState := parseAdminUserAccessStateForDetail(parseDetail, map[int64]string{9: "disabled"}); parseState != "active" {
		parseT.Fatalf("state = %q, want active from restore audit log", parseState)
	}
}

func TestParseAdminUserAccessImpactTextNamesBlockedState(parseT *testing.T) {
	parseT.Parallel()
	if !strings.Contains(parseAdminUserAccessImpactText("disabled"), "Blocked account") {
		parseT.Fatalf("disabled impact text did not name blocked account")
	}
	if !strings.Contains(parseAdminUserAccessImpactText("active"), "Active account") {
		parseT.Fatalf("active impact text did not name active account")
	}
}
