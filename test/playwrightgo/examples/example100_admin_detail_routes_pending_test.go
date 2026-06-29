//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	playwright "github.com/playwright-community/playwright-go"
)

// TestExample100AdminDetailRoutesRegression validates customer/chat/provider/ops drill-down reads and list-context route restoration.
func TestExample100AdminDetailRoutesRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseFixture := startExample100AdminListMechanicsServer(parseT, parseRepoRoot, "18114")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseUserID, parseConversationID, parseProviderID, parseTicketID := parseResolveExample100AdminDetailTargets(parseT, parseClient)
		parseAssertExample100AdminDetailRPCLoads(parseT, parseClient, parseUserID, parseConversationID, parseProviderID, parseTicketID, parseFixture.GetTargetWorkspaceID)
		parseAssertExample100AdminDetailRoutes(parseT, parsePage, parseBaseURL, parseUserID, parseConversationID, parseProviderID, parseTicketID)

		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-detail-routes regression flow", parseEvidence)
	})
}

// parseResolveExample100AdminDetailTargets resolves one deterministic set of user/conversation/provider/support identifiers for drill-down assertions.
func parseResolveExample100AdminDetailTargets(parseT *testing.T, parseClient chatpb.ChatServiceClient) (int64, int64, string, int64) {
	parseT.Helper()

	parseUsersCtx, parseUsersCancel := parseBuildExample100AdminMutationCallContext()
	parseUsersResp, parseErr := parseClient.ListAdminUsers(parseUsersCtx, &chatpb.ListAdminUsersRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         10,
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
		parseT.Fatalf("expected user rows for admin detail regression, got none")
	}
	parseUserID := parseUsersResp.GetUsers()[0].GetUserId()

	parseConversationsCtx, parseConversationsCancel := parseBuildExample100AdminMutationCallContext()
	parseConversationsResp, parseErr := parseClient.ListAdminConversations(parseConversationsCtx, &chatpb.ListAdminConversationsRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         10,
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
		parseT.Fatalf("expected conversation rows for admin detail regression, got none")
	}
	parseConversationID := parseConversationsResp.GetConversations()[0].GetConversationId()

	parseDashboardCtx, parseDashboardCancel := parseBuildExample100AdminMutationCallContext()
	parseDashboardResp, parseErr := parseClient.GetAdminDashboard(parseDashboardCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 30,
		TopLimit:     12,
		RecentLimit:  12,
	})
	parseDashboardCancel()
	if parseErr != nil {
		parseT.Fatalf("GetAdminDashboard: %v", parseErr)
	}
	parseProviderID := ""
	if len(parseDashboardResp.GetProviders()) > 0 {
		parseProviderID = parseDashboardResp.GetProviders()[0].GetProviderId()
	}
	if parseProviderID == "" && len(parseDashboardResp.GetRecentUsageEvents()) > 0 {
		parseProviderID = parseDashboardResp.GetRecentUsageEvents()[0].GetProviderId()
	}
	if parseProviderID == "" {
		parseProviderID = "openai"
	}

	parseSupportCtx, parseSupportCancel := parseBuildExample100AdminMutationCallContext()
	parseSupportResp, parseErr := parseClient.ListAdminSupportTickets(parseSupportCtx, &chatpb.ListAdminSupportTicketsRequest{
		Status: "open",
		ListQuery: &chatpb.AdminListQuery{
			Limit:         10,
			Offset:        0,
			Search:        "list-mechanics support",
			SortBy:        "ticket_key",
			SortDirection: "asc",
		},
	})
	parseSupportCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminSupportTickets: %v", parseErr)
	}
	if len(parseSupportResp.GetTickets()) == 0 {
		parseT.Fatalf("expected support ticket rows for admin detail regression, got none")
	}
	parseTicketID := parseSupportResp.GetTickets()[0].GetId()

	return parseUserID, parseConversationID, parseProviderID, parseTicketID
}

// parseAssertExample100AdminDetailRPCLoads verifies detail-drilldown RPC paths for customer/chat/provider/ops surfaces.
func parseAssertExample100AdminDetailRPCLoads(parseT *testing.T, parseClient chatpb.ChatServiceClient, parseUserID int64, parseConversationID int64, parseProviderID string, parseTicketID int64, parseWorkspaceID int64) {
	parseT.Helper()

	parseUserDetailCtx, parseUserDetailCancel := parseBuildExample100AdminMutationCallContext()
	parseUserDetailResp, parseErr := parseClient.GetAdminUserDetail(parseUserDetailCtx, &chatpb.GetAdminUserDetailRequest{
		UserId:       parseUserID,
		LookbackDays: 30,
		Limit:        20,
	})
	parseUserDetailCancel()
	if parseErr != nil {
		parseT.Fatalf("GetAdminUserDetail: %v", parseErr)
	}
	if parseUserDetailResp.GetDetail() == nil || parseUserDetailResp.GetDetail().GetUser() == nil || parseUserDetailResp.GetDetail().GetUser().GetUserId() != parseUserID {
		parseT.Fatalf("GetAdminUserDetail returned invalid detail payload: %+v", parseUserDetailResp)
	}

	parseChatDrilldownCtx, parseChatDrilldownCancel := parseBuildExample100AdminMutationCallContext()
	parseChatDrilldownResp, parseErr := parseClient.ListAdminConversations(parseChatDrilldownCtx, &chatpb.ListAdminConversationsRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         10,
			Offset:        0,
			Search:        fmt.Sprintf("%d", parseConversationID),
			SortBy:        "last_activity_at",
			SortDirection: "desc",
		},
	})
	parseChatDrilldownCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminConversations drilldown search: %v", parseErr)
	}
	if len(parseChatDrilldownResp.GetConversations()) == 0 {
		parseT.Fatalf("expected chat drilldown rows for conversation id=%d, got none", parseConversationID)
	}

	parseProviderDrilldownCtx, parseProviderDrilldownCancel := parseBuildExample100AdminMutationCallContext()
	parseProviderDrilldownResp, parseErr := parseClient.ListAdminUsageEvents(parseProviderDrilldownCtx, &chatpb.ListAdminUsageEventsRequest{
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
	parseProviderDrilldownCancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsageEvents provider drilldown: %v", parseErr)
	}
	if len(parseProviderDrilldownResp.GetEvents()) == 0 {
		parseT.Fatalf("expected provider drilldown usage rows for provider=%q, got none", parseProviderID)
	}

	parseSupportDetailCtx, parseSupportDetailCancel := parseBuildExample100AdminMutationCallContext()
	parseSupportDetailResp, parseErr := parseClient.GetAdminSupportTicketDetail(parseSupportDetailCtx, &chatpb.GetAdminSupportTicketDetailRequest{
		TicketId: parseTicketID,
		Limit:    20,
	})
	parseSupportDetailCancel()
	if parseErr != nil {
		parseT.Fatalf("GetAdminSupportTicketDetail: %v", parseErr)
	}
	if parseSupportDetailResp.GetDetail() == nil || parseSupportDetailResp.GetDetail().GetTicket() == nil || parseSupportDetailResp.GetDetail().GetTicket().GetId() != parseTicketID {
		parseT.Fatalf("GetAdminSupportTicketDetail returned invalid detail payload: %+v", parseSupportDetailResp)
	}

	parseBlastCtx, parseBlastCancel := parseBuildExample100AdminMutationCallContext()
	parseBlastResp, parseErr := parseClient.GetAdminIncidentBlastRadius(parseBlastCtx, &chatpb.GetAdminIncidentBlastRadiusRequest{
		WorkspaceId:  parseWorkspaceID,
		LookbackDays: 30,
	})
	parseBlastCancel()
	if parseErr != nil {
		parseT.Fatalf("GetAdminIncidentBlastRadius: %v", parseErr)
	}
	if parseBlastResp.GetBlastRadius() == nil || parseBlastResp.GetBlastRadius().GetWorkspaceId() != parseWorkspaceID {
		parseT.Fatalf("GetAdminIncidentBlastRadius returned invalid payload: %+v", parseBlastResp)
	}
}

// parseAssertExample100AdminDetailRoutes verifies list->detail route transitions preserve query state and return to the expected list context.
func parseAssertExample100AdminDetailRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseUserID int64, parseConversationID int64, parseProviderID string, parseTicketID int64) {
	parseT.Helper()
	parseRoutes := []string{
		"/app/admin/users?tab=customers&lookback_days=30&search=admin&sort_by=email&sort_direction=asc&limit=10&offset=0",
		fmt.Sprintf("/app/admin/users?tab=customers&lookback_days=30&search=admin&sort_by=email&sort_direction=asc&limit=10&offset=0&view=user-detail&user_id=%d", parseUserID),
		"/app/dashboard/usage?tab=chats&lookback_days=30&sort_by=last_activity_at&sort_direction=desc&limit=10&offset=0",
		fmt.Sprintf("/app/dashboard/usage?tab=chats&lookback_days=30&sort_by=last_activity_at&sort_direction=desc&limit=10&offset=0&view=chat-detail&conversation_id=%d", parseConversationID),
		fmt.Sprintf("/app/dashboard/usage?tab=providers&lookback_days=30&search=%s&sort_by=provider_id&sort_direction=asc&limit=10&offset=0", parseProviderID),
		fmt.Sprintf("/app/dashboard/usage?tab=providers&lookback_days=30&search=%s&sort_by=provider_id&sort_direction=asc&limit=10&offset=0&view=provider-detail&provider_id=%s", parseProviderID, parseProviderID),
		"/app/dashboard/usage?tab=ops&lookback_days=30&search=list-mechanics%20support&sort_by=ticket_key&sort_direction=asc&limit=10&offset=0",
		fmt.Sprintf("/app/dashboard/usage?tab=ops&lookback_days=30&search=list-mechanics%%20support&sort_by=ticket_key&sort_direction=asc&limit=10&offset=0&view=support-detail&ticket_id=%d", parseTicketID),
	}
	for _, parseRoute := range parseRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto admin detail route %s: %v", parseRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait admin detail route %s: %v", parseRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on admin detail route %s: %v", parseRoute, parseErr)
		}
	}
	parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, parseRoutes)
}
