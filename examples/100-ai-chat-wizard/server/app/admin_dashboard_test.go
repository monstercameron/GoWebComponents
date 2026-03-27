package app

import (
	"context"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
)

// parseSeedAdminDashboardTestData inserts a compact multi-user dataset for dashboard tests.
func parseSeedAdminDashboardTestData(parseT *testing.T, parseStore *Store) {
	parseT.Helper()

	parseAlice := parseMustCreateUser(parseT, parseStore, "alice@example.com")
	parseBob := parseMustCreateUser(parseT, parseStore, "bob@example.com")
	parseNow := time.Now().Unix()
	if parseErr := parseStore.setUserName(parseAlice.ID, "Alice", parseNow); parseErr != nil {
		parseT.Fatalf("setUserName alice: %v", parseErr)
	}
	if parseErr := parseStore.setUserName(parseBob.ID, "Bob", parseNow); parseErr != nil {
		parseT.Fatalf("setUserName bob: %v", parseErr)
	}

	parseAliceConversationID, parseErr := parseStore.parseCreateConversation(parseAlice.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation alice: %v", parseErr)
	}
	parseBobConversationID, parseErr := parseStore.parseCreateConversation(parseBob.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation bob: %v", parseErr)
	}

	if parseErr = parseStore.parseSaveConversationMessage(parseAlice.ID, parseAliceConversationID, "user", "Alice needs help with billing.", "", 0, 0); parseErr != nil {
		parseT.Fatalf("parseSaveConversationMessage alice: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveConversationMessage(parseBob.ID, parseBobConversationID, "user", "Bob wants a refund status update.", "", 0, 0); parseErr != nil {
		parseT.Fatalf("parseSaveConversationMessage bob: %v", parseErr)
	}

	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:                 "evt-alice-openai",
		UserID:                  parseAlice.ID,
		ConversationID:          parseAliceConversationID,
		ProviderID:              "openai",
		ModelID:                 modelGPT54,
		PromptTokens:            100,
		CompletionTokens:        50,
		UsageSource:             "exact",
		ProviderRequestID:       "req-alice-openai",
		InputCostPerMillionUSD:  1.25,
		OutputCostPerMillionUSD: 10,
		PricingCurrency:         "USD",
		InputCostUSD:            0.50,
		OutputCostUSD:           2.00,
		TotalCostUSD:            2.50,
		ClientID:                "client-alice-1",
		Status:                  "completed",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent alice openai: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:                 "evt-alice-fake",
		UserID:                  parseAlice.ID,
		ConversationID:          parseAliceConversationID,
		ProviderID:              "fake",
		ModelID:                 modelGPT54Mini,
		PromptTokens:            60,
		CompletionTokens:        0,
		UsageSource:             "estimated",
		ProviderRequestID:       "req-alice-fake",
		InputCostPerMillionUSD:  0.25,
		OutputCostPerMillionUSD: 2,
		PricingCurrency:         "USD",
		InputCostUSD:            0.75,
		OutputCostUSD:           0,
		TotalCostUSD:            0.75,
		ClientID:                "client-alice-2",
		Status:                  "failed",
		ErrorMessage:            "provider timeout",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent alice fake: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:                 "evt-bob-fake",
		UserID:                  parseBob.ID,
		ConversationID:          parseBobConversationID,
		ProviderID:              "fake",
		ModelID:                 modelGPT54Mini,
		PromptTokens:            20,
		CompletionTokens:        10,
		UsageSource:             "exact",
		ProviderRequestID:       "req-bob-fake",
		InputCostPerMillionUSD:  0.25,
		OutputCostPerMillionUSD: 2,
		PricingCurrency:         "USD",
		InputCostUSD:            0.05,
		OutputCostUSD:           0.20,
		TotalCostUSD:            0.25,
		ClientID:                "client-bob-1",
		Status:                  "completed",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent bob fake: %v", parseErr)
	}
}

// TestStoreAdminDashboardQueries verifies the dashboard SQL/store rollups on seeded data.
func TestStoreAdminDashboardQueries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)

	parseSince := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	parseSummary, parseErr := parseStore.parseGetAdminDashboardSummary(parseSince)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminDashboardSummary: %v", parseErr)
	}
	if parseSummary.TotalUsers != 2 || parseSummary.TotalConversations != 2 || parseSummary.TotalMessages != 2 {
		parseT.Fatalf("unexpected summary totals: %+v", parseSummary)
	}
	if parseSummary.WindowUsageEvents != 3 || parseSummary.WindowCompletedEvents != 2 || parseSummary.WindowFailedEvents != 1 {
		parseT.Fatalf("unexpected usage summary counts: %+v", parseSummary)
	}
	if parseSummary.WindowActiveUsers != 2 || parseSummary.WindowActiveConversations != 2 || parseSummary.WindowActiveClients != 3 {
		parseT.Fatalf("unexpected active summary counts: %+v", parseSummary)
	}
	if parseSummary.WindowTotalCostUSD != 3.5 || parseSummary.WindowPromptTokens != 180 || parseSummary.WindowCompletionTokens != 60 {
		parseT.Fatalf("unexpected cost/token summary: %+v", parseSummary)
	}

	parseDailyRows, parseErr := parseStore.parseListAdminDashboardDailyUsage(parseSince)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminDashboardDailyUsage: %v", parseErr)
	}
	if len(parseDailyRows) != 1 || parseDailyRows[0].UsageEventCount != 3 || parseDailyRows[0].TotalCostUSD != 3.5 {
		parseT.Fatalf("unexpected daily rows: %+v", parseDailyRows)
	}

	parseProviderRows, parseErr := parseStore.parseListAdminDashboardProviderUsage(parseSince, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminDashboardProviderUsage: %v", parseErr)
	}
	if len(parseProviderRows) != 2 || parseProviderRows[0].ProviderID != "openai" || parseProviderRows[0].TotalCostUSD != 2.5 {
		parseT.Fatalf("unexpected provider rows: %+v", parseProviderRows)
	}

	parseModelRows, parseErr := parseStore.parseListAdminDashboardModelUsage(parseSince, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminDashboardModelUsage: %v", parseErr)
	}
	if len(parseModelRows) != 2 || parseModelRows[0].ModelID != modelGPT54 || parseModelRows[1].ModelID != modelGPT54Mini {
		parseT.Fatalf("unexpected model rows: %+v", parseModelRows)
	}

	parseUserRows, parseErr := parseStore.parseListAdminDashboardUserUsage(parseSince, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminDashboardUserUsage: %v", parseErr)
	}
	if len(parseUserRows) != 2 || parseUserRows[0].DisplayName != "Alice" || parseUserRows[0].TotalCostUSD != 3.25 {
		parseT.Fatalf("unexpected top user rows: %+v", parseUserRows)
	}

	parseUsageRows, parseErr := parseStore.parseListAdminUsageEvents(parseSince, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminUsageEvents: %v", parseErr)
	}
	if len(parseUsageRows) != 3 || parseUsageRows[0].EventID != "evt-bob-fake" || parseUsageRows[2].EventID != "evt-alice-openai" {
		parseT.Fatalf("unexpected usage rows: %+v", parseUsageRows)
	}

	parseUserListRows, parseErr := parseStore.parseListAdminUsers(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminUsers: %v", parseErr)
	}
	if len(parseUserListRows) != 2 || parseUserListRows[0].DisplayName != "Bob" || parseUserListRows[1].DisplayName != "Alice" {
		parseT.Fatalf("unexpected admin users: %+v", parseUserListRows)
	}

	parseConversationRows, parseErr := parseStore.parseListAdminConversations(10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminConversations: %v", parseErr)
	}
	if len(parseConversationRows) != 2 || parseConversationRows[0].DisplayName != "Bob" || parseConversationRows[1].DisplayName != "Alice" {
		parseT.Fatalf("unexpected admin conversations: %+v", parseConversationRows)
	}
}

// TestAdminDashboardRPCs verifies the ungated admin dashboard RPCs return seeded analytics.
func TestAdminDashboardRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseDashboardResp, parseErr := parseServer.GetAdminDashboard(context.Background(), &chatpb.GetAdminDashboardRequest{
		LookbackDays: 30,
		TopLimit:     5,
		RecentLimit:  5,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminDashboard: %v", parseErr)
	}
	if parseDashboardResp.GetSummary().GetTotalUsers() != 2 || parseDashboardResp.GetSummary().GetWindowUsageEvents() != 3 {
		parseT.Fatalf("unexpected dashboard summary: %+v", parseDashboardResp.GetSummary())
	}
	if len(parseDashboardResp.GetTopUsers()) != 2 || parseDashboardResp.GetTopUsers()[0].GetDisplayName() != "Alice" {
		parseT.Fatalf("unexpected dashboard top users: %+v", parseDashboardResp.GetTopUsers())
	}
	if len(parseDashboardResp.GetRecentUsageEvents()) != 3 || len(parseDashboardResp.GetRecentUsers()) != 2 || len(parseDashboardResp.GetRecentConversations()) != 2 {
		parseT.Fatalf("unexpected dashboard recents: usage=%d users=%d convs=%d", len(parseDashboardResp.GetRecentUsageEvents()), len(parseDashboardResp.GetRecentUsers()), len(parseDashboardResp.GetRecentConversations()))
	}
	if len(parseDashboardResp.GetProviderSnapshots()) != 1 || parseDashboardResp.GetProviderSnapshots()[0].GetProviderId() != "fake" {
		parseT.Fatalf("unexpected provider snapshots: %+v", parseDashboardResp.GetProviderSnapshots())
	}

	parseUsersResp, parseErr := parseServer.ListAdminUsers(context.Background(), &chatpb.ListAdminUsersRequest{Limit: 10})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers: %v", parseErr)
	}
	if len(parseUsersResp.GetUsers()) != 2 {
		parseT.Fatalf("unexpected ListAdminUsers rows: %+v", parseUsersResp.GetUsers())
	}

	parseUsageResp, parseErr := parseServer.ListAdminUsageEvents(context.Background(), &chatpb.ListAdminUsageEventsRequest{
		LookbackDays: 30,
		Limit:        10,
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsageEvents: %v", parseErr)
	}
	if len(parseUsageResp.GetEvents()) != 3 {
		parseT.Fatalf("unexpected ListAdminUsageEvents rows: %+v", parseUsageResp.GetEvents())
	}

	parseConversationsResp, parseErr := parseServer.ListAdminConversations(context.Background(), &chatpb.ListAdminConversationsRequest{Limit: 10})
	if parseErr != nil {
		parseT.Fatalf("ListAdminConversations: %v", parseErr)
	}
	if len(parseConversationsResp.GetConversations()) != 2 {
		parseT.Fatalf("unexpected ListAdminConversations rows: %+v", parseConversationsResp.GetConversations())
	}
}
