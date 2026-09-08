package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetAdminReadOnlyReportAllowlistedSlices verifies allowlisted superuser report slices for read-only reporting.
func TestGetAdminReadOnlyReportAllowlistedSlices(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)

	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-readonly-superuser", parseAliceAuth.ID, parseAliceAuth.Email)

	parseSummaryResp, parseErr := parseServer.GetAdminReadOnlyReport(parseAliceCtx, &chatpb.GetAdminReadOnlyReportRequest{
		ReportKey:    "dashboard_summary",
		LookbackDays: 30,
		Limit:        5,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminReadOnlyReport summary: %v", parseErr)
	}
	if parseSummaryResp.GetScope() != "platform" || parseSummaryResp.GetSummary() == nil || parseSummaryResp.GetRowCount() != 1 {
		parseT.Fatalf("unexpected summary response: %+v", parseSummaryResp)
	}

	parseUsersResp, parseErr := parseServer.GetAdminReadOnlyReport(parseAliceCtx, &chatpb.GetAdminReadOnlyReportRequest{
		ReportKey: "recent_users",
		Limit:     5,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminReadOnlyReport users: %v", parseErr)
	}
	if parseUsersResp.GetScope() != "platform" || len(parseUsersResp.GetUsers()) == 0 || int(parseUsersResp.GetRowCount()) != len(parseUsersResp.GetUsers()) {
		parseT.Fatalf("unexpected users response: %+v", parseUsersResp)
	}

	parseUsageResp, parseErr := parseServer.GetAdminReadOnlyReport(parseAliceCtx, &chatpb.GetAdminReadOnlyReportRequest{
		ReportKey:    "recent_usage_events",
		LookbackDays: 30,
		Limit:        5,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminReadOnlyReport usage: %v", parseErr)
	}
	if parseUsageResp.GetScope() != "platform" || len(parseUsageResp.GetUsageEvents()) == 0 || int(parseUsageResp.GetRowCount()) != len(parseUsageResp.GetUsageEvents()) {
		parseT.Fatalf("unexpected usage response: %+v", parseUsageResp)
	}

	parseConversationsResp, parseErr := parseServer.GetAdminReadOnlyReport(parseAliceCtx, &chatpb.GetAdminReadOnlyReportRequest{
		ReportKey: "recent_conversations",
		Limit:     5,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminReadOnlyReport conversations: %v", parseErr)
	}
	if parseConversationsResp.GetScope() != "platform" || len(parseConversationsResp.GetConversations()) == 0 || int(parseConversationsResp.GetRowCount()) != len(parseConversationsResp.GetConversations()) {
		parseT.Fatalf("unexpected conversations response: %+v", parseConversationsResp)
	}
}

// TestGetAdminReadOnlyReportScopeAndGuards verifies workspace-admin scoping and invalid-key or non-admin guardrails.
func TestGetAdminReadOnlyReportScopeAndGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseBobAuth.ID, "ws-readonly-bob")

	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-readonly-workspace", parseBobAuth.ID, parseBobAuth.Email)

	parseScopedResp, parseErr := parseServer.GetAdminReadOnlyReport(parseBobCtx, &chatpb.GetAdminReadOnlyReportRequest{
		ReportKey: "recent_users",
		Limit:     10,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminReadOnlyReport workspace scope: %v", parseErr)
	}
	if parseScopedResp.GetScope() != "workspace" || len(parseScopedResp.GetUsers()) == 0 {
		parseT.Fatalf("unexpected workspace-scoped users response: %+v", parseScopedResp)
	}
	for _, parseUserRow := range parseScopedResp.GetUsers() {
		if parseUserRow.GetUserId() != parseBobAuth.ID {
			parseT.Fatalf("expected workspace-scoped user rows to only include bob (%d), got row=%+v", parseBobAuth.ID, parseUserRow)
		}
	}

	if _, parseErr = parseServer.GetAdminReadOnlyReport(parseBobCtx, &chatpb.GetAdminReadOnlyReportRequest{
		ReportKey: "drop table users;",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("GetAdminReadOnlyReport invalid key status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	parseNonAdmin := parseMustCreateUser(parseT, parseStore, "non-admin-readonly@example.com")
	parseNonAdminCtx := parseBindAuthUser(parseServer, "peer-admin-readonly-nonadmin", parseNonAdmin.ID, parseNonAdmin.Email)
	if _, parseErr = parseServer.GetAdminReadOnlyReport(parseNonAdminCtx, &chatpb.GetAdminReadOnlyReportRequest{
		ReportKey: "recent_users",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminReadOnlyReport non-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// BenchmarkParseNormalizeAdminReadOnlyReportKey reports micro-benchmark throughput for read-only report-key normalization.
func BenchmarkParseNormalizeAdminReadOnlyReportKey(parseB *testing.B) {
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseNormalizeAdminReadOnlyReportKey("recent_usage_events")
	}
}
