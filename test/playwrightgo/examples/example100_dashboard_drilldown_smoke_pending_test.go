//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	playwright "github.com/mxschmitt/playwright-go"
)

// TestExample100DashboardDrilldownSmoke verifies dashboard trend/KPI payloads have matching drill-down table/detail RPC paths.
func TestExample100DashboardDrilldownSmoke(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, _ := startExample100AdminListMechanicsServer(parseT, parseRepoRoot, "18115")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseDashboardCtx, parseDashboardCancel := parseBuildExample100AdminMutationCallContext()
		parseDashboardResp, parseErr := parseClient.GetAdminDashboard(parseDashboardCtx, &chatpb.GetAdminDashboardRequest{
			LookbackDays: 30,
			TopLimit:     10,
			RecentLimit:  10,
		})
		parseDashboardCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminDashboard: %v", parseErr)
		}
		parseAssertExample100DashboardDrilldownRPCs(parseT, parseClient, parseDashboardResp)
		parseAssertExample100DashboardDrilldownRoutes(parseT, parsePage, parseBaseURL)

		assertExample100AdminGuardNoRuntimeErrors(parseT, "dashboard-drilldown smoke flow", parseEvidence)
	})
}

// parseAssertExample100DashboardDrilldownRPCs validates one drill-down RPC path for each dashboard trend/KPI family.
func parseAssertExample100DashboardDrilldownRPCs(parseT *testing.T, parseClient chatpb.ChatServiceClient, parseDashboardResp *chatpb.GetAdminDashboardResponse) {
	parseT.Helper()
	if parseDashboardResp == nil {
		parseT.Fatalf("dashboard drilldown smoke requires non-nil dashboard response")
	}
	if len(parseDashboardResp.GetDailyUsage()) == 0 {
		parseT.Fatalf("expected daily usage trend rows in dashboard response, got none")
	}
	if len(parseDashboardResp.GetProviders()) == 0 {
		parseT.Fatalf("expected provider trend rows in dashboard response, got none")
	}
	if len(parseDashboardResp.GetModels()) == 0 {
		parseT.Fatalf("expected model trend rows in dashboard response, got none")
	}
	if len(parseDashboardResp.GetTopUsers()) == 0 {
		parseT.Fatalf("expected top-user KPI rows in dashboard response, got none")
	}
	if len(parseDashboardResp.GetRecentConversations()) == 0 {
		parseT.Fatalf("expected recent conversation rows in dashboard response, got none")
	}

	parseUsageCtx, parseUsageCancel := parseBuildExample100AdminMutationCallContext()
	parseUsageResp, parseErr := parseClient.ListAdminUsageEvents(parseUsageCtx, &chatpb.ListAdminUsageEventsRequest{
		LookbackDays: 30,
		Limit:        25,
		ListQuery: &chatpb.AdminListQuery{
			Limit:         25,
			Offset:        0,
			SortBy:        "created_at",
			SortDirection: "desc",
		},
	})
	parseUsageCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsageEvents global drilldown: %v", parseErr)
	}
	if len(parseUsageResp.GetEvents()) == 0 {
		parseT.Fatalf("expected usage-event table rows for global drilldown, got none")
	}

	parseProviderID := parseDashboardResp.GetProviders()[0].GetProviderId()
	parseProviderUsageCtx, parseProviderUsageCancel := parseBuildExample100AdminMutationCallContext()
	parseProviderUsageResp, parseErr := parseClient.ListAdminUsageEvents(parseProviderUsageCtx, &chatpb.ListAdminUsageEventsRequest{
		LookbackDays: 30,
		Limit:        25,
		ProviderId:   parseProviderID,
		ListQuery: &chatpb.AdminListQuery{
			Limit:         25,
			Offset:        0,
			SortBy:        "created_at",
			SortDirection: "desc",
		},
	})
	parseProviderUsageCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsageEvents provider drilldown: %v", parseErr)
	}
	if len(parseProviderUsageResp.GetEvents()) == 0 {
		parseT.Fatalf("expected usage-event table rows for provider drilldown provider=%q, got none", parseProviderID)
	}

	parseModelProviderID := parseDashboardResp.GetModels()[0].GetProviderId()
	parseModelUsageCtx, parseModelUsageCancel := parseBuildExample100AdminMutationCallContext()
	parseModelUsageResp, parseErr := parseClient.ListAdminUsageEvents(parseModelUsageCtx, &chatpb.ListAdminUsageEventsRequest{
		LookbackDays: 30,
		Limit:        25,
		ProviderId:   parseModelProviderID,
		ListQuery: &chatpb.AdminListQuery{
			Limit:         25,
			Offset:        0,
			SortBy:        "created_at",
			SortDirection: "desc",
		},
	})
	parseModelUsageCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsageEvents model-provider drilldown: %v", parseErr)
	}
	if len(parseModelUsageResp.GetEvents()) == 0 {
		parseT.Fatalf("expected usage-event table rows for model-provider drilldown provider=%q, got none", parseModelProviderID)
	}

	parseUserID := parseDashboardResp.GetTopUsers()[0].GetUserId()
	parseUserDetailCtx, parseUserDetailCancel := parseBuildExample100AdminMutationCallContext()
	parseUserDetailResp, parseErr := parseClient.GetAdminUserDetail(parseUserDetailCtx, &chatpb.GetAdminUserDetailRequest{
		UserId:       parseUserID,
		LookbackDays: 30,
		Limit:        20,
	})
	parseUserDetailCancel()
	if parseErr != nil {
		parseT.Fatalf("GetAdminUserDetail top-user drilldown: %v", parseErr)
	}
	if parseUserDetailResp.GetDetail() == nil || parseUserDetailResp.GetDetail().GetUser() == nil || parseUserDetailResp.GetDetail().GetUser().GetUserId() != parseUserID {
		parseT.Fatalf("GetAdminUserDetail top-user drilldown returned invalid payload: %+v", parseUserDetailResp)
	}

	parseConversationID := parseDashboardResp.GetRecentConversations()[0].GetConversationId()
	parseConversationDrilldownCtx, parseConversationDrilldownCancel := parseBuildExample100AdminMutationCallContext()
	parseConversationDrilldownResp, parseErr := parseClient.ListAdminConversations(parseConversationDrilldownCtx, &chatpb.ListAdminConversationsRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         20,
			Offset:        0,
			Search:        fmt.Sprintf("%d", parseConversationID),
			SortBy:        "last_activity_at",
			SortDirection: "desc",
		},
	})
	parseConversationDrilldownCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminConversations conversation drilldown: %v", parseErr)
	}
	if len(parseConversationDrilldownResp.GetConversations()) == 0 {
		parseT.Fatalf("expected conversation table rows for conversation drilldown id=%d, got none", parseConversationID)
	}
}

// parseAssertExample100DashboardDrilldownRoutes verifies dashboard route entry remains stable for drill-down surfaces.
func parseAssertExample100DashboardDrilldownRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) {
	parseT.Helper()
	parseRoutes := []string{
		"/app/dashboard?tab=business&lookback_days=30",
		"/app/dashboard/usage?tab=providers&lookback_days=30&sort_by=provider_id&sort_direction=asc&limit=10&offset=0",
		"/app/dashboard/usage?tab=chats&lookback_days=30&sort_by=last_activity_at&sort_direction=desc&limit=10&offset=0",
		"/app/admin/users?tab=customers&lookback_days=30&search=admin&sort_by=email&sort_direction=asc&limit=10&offset=0",
	}
	for _, parseRoute := range parseRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto dashboard drilldown route %s: %v", parseRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait dashboard drilldown route %s: %v", parseRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on dashboard drilldown route %s: %v", parseRoute, parseErr)
		}
	}
}
