//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	playwright "github.com/playwright-community/playwright-go"
)

// TestExample100DashboardHomeRegression covers dashboard-home surface routing and data loading for Business, Customers, Chats, Providers, and Ops.
func TestExample100DashboardHomeRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseFixture := startExample100AdminListMechanicsServer(parseT, parseRepoRoot, "18112")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseBillingOverrideKey := parseSeedExample100DashboardHomeData(parseT, parseClient, parseFixture.GetTargetUserID)
		parseAssertExample100DashboardHomeRPCLoads(parseT, parseClient, parseFixture.GetTargetUserID, parseBillingOverrideKey)
		parseAssertExample100DashboardHomeSurfaceRoutes(parseT, parsePage, parseBaseURL)

		assertExample100AdminGuardNoRuntimeErrors(parseT, "dashboard-home regression flow", parseEvidence)
	})
}

// parseSeedExample100DashboardHomeData inserts one billing and one incident fixture so dashboard drill-down tables have deterministic rows.
func parseSeedExample100DashboardHomeData(parseT *testing.T, parseClient chatpb.ChatServiceClient, parseUserID int64) string {
	parseT.Helper()

	parseBillingOverrideKey := fmt.Sprintf("dashboard-home-billing-%d", time.Now().UTC().UnixNano())
	parseCreateExample100BillingOverride(parseT, parseClient, parseUserID, parseBillingOverrideKey)

	parseSLOKey := fmt.Sprintf("dashboard-home-slo-%d", time.Now().UTC().UnixNano())
	parseSLOSetCtx, parseSLOSetCancel := parseBuildExample100AdminMutationCallContext()
	_, parseErr := parseClient.SetSuperuserServiceLevelObjective(parseSLOSetCtx, &chatpb.SetSuperuserServiceLevelObjectiveRequest{
		SloKey:             parseSLOKey,
		ServiceName:        "dashboard-home-regression",
		ObjectivePercent:   99.9,
		WindowDays:         30,
		ErrorBudgetMinutes: 43,
		StatusPageUrl:      "https://status.example.invalid/dashboard-home-regression",
		Confirm:            true,
		Reason:             "playwright dashboard-home regression: seed slo",
	})
	parseSLOSetCancel()
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserServiceLevelObjective: %v", parseErr)
	}
	parseCreateExample100Incident(parseT, parseClient, parseSLOKey, "dashboard-home-incident")

	return parseBillingOverrideKey
}

// parseAssertExample100DashboardHomeRPCLoads verifies KPI and table data loads for all five dashboard surfaces.
func parseAssertExample100DashboardHomeRPCLoads(parseT *testing.T, parseClient chatpb.ChatServiceClient, parseUserID int64, parseBillingOverrideKey string) {
	parseT.Helper()

	parseDashboard7Ctx, parseDashboard7Cancel := parseBuildExample100AdminMutationCallContext()
	parseDashboard7Resp, parseErr := parseClient.GetAdminDashboard(parseDashboard7Ctx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 7,
		TopLimit:     8,
		RecentLimit:  8,
	})
	parseDashboard7Cancel()
	if parseErr != nil {
		parseT.Fatalf("GetAdminDashboard lookback_days=7: %v", parseErr)
	}
	parseDashboard30Ctx, parseDashboard30Cancel := parseBuildExample100AdminMutationCallContext()
	parseDashboard30Resp, parseErr := parseClient.GetAdminDashboard(parseDashboard30Ctx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 30,
		TopLimit:     8,
		RecentLimit:  8,
	})
	parseDashboard30Cancel()
	if parseErr != nil {
		parseT.Fatalf("GetAdminDashboard lookback_days=30: %v", parseErr)
	}
	if parseDashboard30Resp.GetSummary() == nil {
		parseT.Fatalf("GetAdminDashboard returned nil summary for lookback_days=30")
	}
	if len(parseDashboard30Resp.GetDailyUsage()) < len(parseDashboard7Resp.GetDailyUsage()) {
		parseT.Fatalf("expected daily usage rows for lookback_days=30 to be >= lookback_days=7, got day30=%d day7=%d", len(parseDashboard30Resp.GetDailyUsage()), len(parseDashboard7Resp.GetDailyUsage()))
	}

	parseUsersCtx, parseUsersCancel := parseBuildExample100AdminMutationCallContext()
	parseUsersResp, parseErr := parseClient.ListAdminUsers(parseUsersCtx, &chatpb.ListAdminUsersRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "admin",
			SortBy:        "email",
			SortDirection: "asc",
		},
	})
	parseUsersCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers: %v", parseErr)
	}
	if len(parseUsersResp.GetUsers()) == 0 {
		parseT.Fatalf("expected customer table rows from ListAdminUsers, got none")
	}

	parseConversationsCtx, parseConversationsCancel := parseBuildExample100AdminMutationCallContext()
	parseConversationsResp, parseErr := parseClient.ListAdminConversations(parseConversationsCtx, &chatpb.ListAdminConversationsRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			SortBy:        "last_activity_at",
			SortDirection: "desc",
		},
	})
	parseConversationsCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminConversations: %v", parseErr)
	}
	if len(parseConversationsResp.GetConversations()) == 0 {
		parseT.Fatalf("expected chats table rows from ListAdminConversations, got none")
	}

	parseBillingCtx, parseBillingCancel := parseBuildExample100AdminMutationCallContext()
	parseBillingResp, parseErr := parseClient.ListAdminBillingAccessOverrides(parseBillingCtx, &chatpb.ListAdminBillingAccessOverridesRequest{
		UserId:      parseUserID,
		OverrideKey: parseBillingOverrideKey,
		ListQuery: &chatpb.AdminListQuery{
			Limit:         5,
			Offset:        0,
			Search:        "dashboard-home-billing",
			SortBy:        "override_key",
			SortDirection: "asc",
		},
	})
	parseBillingCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingAccessOverrides: %v", parseErr)
	}
	if len(parseBillingResp.GetOverrides()) == 0 {
		parseT.Fatalf("expected business table rows from ListAdminBillingAccessOverrides, got none")
	}

	parseSlicesCtx, parseSlicesCancel := parseBuildExample100AdminMutationCallContext()
	parseSlicesResp, parseErr := parseClient.GetSuperuserSlices(parseSlicesCtx, &chatpb.GetSuperuserSlicesRequest{
		LookbackDays: 30,
		Limit:        200,
		WorkspaceListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "ws-list-mech",
			SortBy:        "workspace_key",
			SortDirection: "asc",
		},
		WorkspaceStatus: "active",
		SupportListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "list-mechanics support",
			SortBy:        "ticket_key",
			SortDirection: "asc",
		},
		SupportStatus: "open",
		IncidentListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "dashboard-home-incident",
			SortBy:        "incident_key",
			SortDirection: "asc",
		},
		IncidentStatus: "open",
	})
	parseSlicesCancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices: %v", parseErr)
	}
	if len(parseSlicesResp.GetWorkspaces()) == 0 {
		parseT.Fatalf("expected customer workspace rows from GetSuperuserSlices, got none")
	}
	if len(parseSlicesResp.GetSupportTickets()) == 0 {
		parseT.Fatalf("expected ops support rows from GetSuperuserSlices, got none")
	}
	if len(parseSlicesResp.GetIncidents()) == 0 {
		parseT.Fatalf("expected ops incident rows from GetSuperuserSlices, got none")
	}
}

// parseAssertExample100DashboardHomeSurfaceRoutes verifies dashboard-home route entry for all five surfaces plus time-range route-state stability.
func parseAssertExample100DashboardHomeSurfaceRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) {
	parseT.Helper()
	parseSurfaceRoutes := []string{
		"/app/dashboard?tab=business&lookback_days=7",
		"/app/dashboard?tab=business&lookback_days=30",
		"/app/admin/users?tab=customers&lookback_days=30&search=admin&sort_by=email&sort_direction=asc&limit=1&offset=0",
		"/app/dashboard/usage?tab=chats&lookback_days=30&sort_by=last_activity_at&sort_direction=desc&limit=1&offset=0",
		"/app/dashboard/usage?tab=providers&lookback_days=30&sort_by=provider_id&sort_direction=asc&limit=1&offset=0",
		"/app/dashboard/usage?tab=ops&lookback_days=30&search=dashboard-home-incident&sort_by=incident_key&sort_direction=asc&limit=1&offset=0",
	}
	for _, parseSurfaceRoute := range parseSurfaceRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseSurfaceRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto dashboard-home surface route %s: %v", parseSurfaceRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseSurfaceRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait dashboard-home surface route %s: %v", parseSurfaceRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on dashboard-home surface route %s: %v", parseSurfaceRoute, parseErr)
		}
	}

	parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, parseSurfaceRoutes)
}
