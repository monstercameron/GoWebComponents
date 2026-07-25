package app

import (
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
)

// parseSeedAdminChatAnomalyUsageRows seeds failed/slow/high-cost usage fixtures for chats anomaly RPC coverage.
func parseSeedAdminChatAnomalyUsageRows(parseT *testing.T, parseStore *Store, parseAliceUserID int64, parseBobUserID int64) {
	parseT.Helper()
	parseAliceConversations, parseErr := parseStore.parseListConversations(parseAliceUserID)
	if parseErr != nil {
		parseT.Fatalf("parseListConversations alice: %v", parseErr)
	}
	parseBobConversations, parseErr := parseStore.parseListConversations(parseBobUserID)
	if parseErr != nil {
		parseT.Fatalf("parseListConversations bob: %v", parseErr)
	}
	if len(parseAliceConversations) == 0 || len(parseBobConversations) == 0 {
		parseT.Fatalf("expected seeded conversations for anomaly usage fixtures")
	}
	parseAliceConversationID := parseAliceConversations[0].ID
	parseBobConversationID := parseBobConversations[0].ID

	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:                 "evt-alice-slow-high-cost",
		UserID:                  parseAliceUserID,
		ConversationID:          parseAliceConversationID,
		ProviderID:              "openai",
		ModelID:                 modelGPT54,
		PromptTokens:            1800,
		CompletionTokens:        600,
		UsageSource:             "exact",
		ProviderRequestID:       "req-alice-slow-high-cost",
		InputCostPerMillionUSD:  1.25,
		OutputCostPerMillionUSD: 10,
		PricingCurrency:         "USD",
		InputCostUSD:            2.25,
		OutputCostUSD:           6.75,
		TotalCostUSD:            9.0,
		ClientID:                "client-alice-anomaly",
		Status:                  "completed",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent alice slow/high-cost: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:                 "evt-bob-slow-high-cost",
		UserID:                  parseBobUserID,
		ConversationID:          parseBobConversationID,
		ProviderID:              "fake",
		ModelID:                 modelGPT54Mini,
		PromptTokens:            1400,
		CompletionTokens:        300,
		UsageSource:             "estimated",
		ProviderRequestID:       "req-bob-slow-high-cost",
		InputCostPerMillionUSD:  0.25,
		OutputCostPerMillionUSD: 2,
		PricingCurrency:         "USD",
		InputCostUSD:            1.5,
		OutputCostUSD:           4.5,
		TotalCostUSD:            6.0,
		ClientID:                "client-bob-anomaly",
		Status:                  "completed",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent bob slow/high-cost: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:                 "evt-bob-failed-anomaly",
		UserID:                  parseBobUserID,
		ConversationID:          parseBobConversationID,
		ProviderID:              "fake",
		ModelID:                 modelGPT54Mini,
		PromptTokens:            120,
		CompletionTokens:        0,
		UsageSource:             "estimated",
		ProviderRequestID:       "req-bob-failed-anomaly",
		InputCostPerMillionUSD:  0.25,
		OutputCostPerMillionUSD: 2,
		PricingCurrency:         "USD",
		InputCostUSD:            0.2,
		OutputCostUSD:           0,
		TotalCostUSD:            0.2,
		ClientID:                "client-bob-failed-anomaly",
		Status:                  "failed",
		ErrorMessage:            "timeout",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent bob failed anomaly: %v", parseErr)
	}
}

// TestGetAdminChatsAnomalies verifies typed failed/slow/high-cost/feature-usage anomaly slices for superuser chats operations.
func TestGetAdminChatsAnomalies(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-chats-anomaly")
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-chats-anomaly")
	parseSeedAdminChatsDrilldownGrowthRows(parseT, parseStore, parseAliceAuth.ID, parseAliceWorkspaceID, parseBobAuth.ID, parseBobWorkspaceID)
	parseSeedAdminChatAnomalyUsageRows(parseT, parseStore, parseAliceAuth.ID, parseBobAuth.ID)

	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-chats-anomaly-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseResp, parseErr := parseServer.GetAdminChatsAnomalies(parseAliceCtx, &chatpb.GetAdminChatsAnomaliesRequest{
		LookbackDays: 30,
		Limit:        50,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminChatsAnomalies: %v", parseErr)
	}
	if parseResp.GetSummary() == nil {
		parseT.Fatalf("expected chats anomalies summary payload")
	}
	if len(parseResp.GetFailedReplies()) == 0 || len(parseResp.GetSlowReplies()) == 0 || len(parseResp.GetHighCostThreads()) == 0 || len(parseResp.GetFeatureUsageSlices()) == 0 {
		parseT.Fatalf(
			"expected non-empty chats anomaly slices, got failed=%d slow=%d high_cost=%d feature=%d",
			len(parseResp.GetFailedReplies()),
			len(parseResp.GetSlowReplies()),
			len(parseResp.GetHighCostThreads()),
			len(parseResp.GetFeatureUsageSlices()),
		)
	}
	for _, parseConversation := range parseResp.GetHighCostThreads() {
		if parseConversation.GetTotalCostUsd() < parseAdminChatHighCostThreadThresholdUSD {
			parseT.Fatalf("expected high-cost thread threshold >= %.2f, got %+v", parseAdminChatHighCostThreadThresholdUSD, parseConversation)
		}
	}
	for _, parseFeatureSlice := range parseResp.GetFeatureUsageSlices() {
		if strings.TrimSpace(parseFeatureSlice.GetFeatureKey()) == "" {
			parseT.Fatalf("expected non-empty feature key in anomaly slice %+v", parseFeatureSlice)
		}
	}
}

// TestGetAdminChatsAnomaliesWorkspaceScope verifies workspace-admin callers receive scoped/redacted chat anomalies only.
func TestGetAdminChatsAnomaliesWorkspaceScope(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseAliceWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-chats-anomaly-scope")
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-chats-anomaly-scope")
	parseSeedAdminChatsDrilldownGrowthRows(parseT, parseStore, parseAliceAuth.ID, parseAliceWorkspaceID, parseBobAuth.ID, parseBobWorkspaceID)
	parseSeedAdminChatAnomalyUsageRows(parseT, parseStore, parseAliceAuth.ID, parseBobAuth.ID)

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-chats-anomaly-bob", parseBobAuth.ID, parseBobAuth.Email)
	parseResp, parseErr := parseServer.GetAdminChatsAnomalies(parseBobCtx, &chatpb.GetAdminChatsAnomaliesRequest{
		LookbackDays: 30,
		Limit:        50,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminChatsAnomalies scoped bob: %v", parseErr)
	}
	for _, parseFailedRow := range parseResp.GetFailedReplies() {
		if parseFailedRow.GetUserId() != parseBobAuth.ID {
			parseT.Fatalf("expected scoped failed replies for bob only, got %+v", parseFailedRow)
		}
		if strings.TrimSpace(parseFailedRow.GetProviderRequestId()) != "" {
			parseT.Fatalf("expected redacted provider request id for workspace-admin failed reply, got %+v", parseFailedRow)
		}
	}
	for _, parseSlowRow := range parseResp.GetSlowReplies() {
		if parseSlowRow.GetUserId() != parseBobAuth.ID {
			parseT.Fatalf("expected scoped slow replies for bob only, got %+v", parseSlowRow)
		}
		if strings.TrimSpace(parseSlowRow.GetProviderRequestId()) != "" {
			parseT.Fatalf("expected redacted provider request id for workspace-admin slow reply, got %+v", parseSlowRow)
		}
	}
	for _, parseHighCostRow := range parseResp.GetHighCostThreads() {
		if parseHighCostRow.GetUserId() != parseBobAuth.ID {
			parseT.Fatalf("expected scoped high-cost threads for bob only, got %+v", parseHighCostRow)
		}
		if strings.TrimSpace(parseHighCostRow.GetPreview()) != parseWorkspaceScopeRedactionText {
			parseT.Fatalf("expected redacted conversation preview for workspace-admin high-cost thread, got %+v", parseHighCostRow)
		}
	}
	for _, parseFeatureSlice := range parseResp.GetFeatureUsageSlices() {
		if parseFeatureSlice.GetWorkspaceCount() <= 0 {
			parseT.Fatalf("expected workspace-counted feature usage slice for scoped workspace admin, got %+v", parseFeatureSlice)
		}
	}
}
