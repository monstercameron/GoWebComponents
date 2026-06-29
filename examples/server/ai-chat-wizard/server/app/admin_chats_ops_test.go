package app

import (
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
)

// parseSeedAdminChatsDrilldownGrowthRows seeds one compact chats-growth fixture set for two workspace scopes.
func parseSeedAdminChatsDrilldownGrowthRows(parseT *testing.T, parseStore *Store, parseAliceUserID int64, parseAliceWorkspaceID int64, parseBobUserID int64, parseBobWorkspaceID int64) {
	parseT.Helper()
	if parseErr := parseStore.parseUpsertOnboardingTemplate(parseOnboardingTemplateWrite{
		TemplateKey:   "starter-chat",
		Title:         "Starter Chat",
		Category:      "onboarding",
		PromptText:    "Help me draft a weekly update.",
		ChecklistJSON: `["pick audience","generate outline"]`,
		IsDefault:     true,
		SortOrder:     1,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertOnboardingTemplate: %v", parseErr)
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if parseErr := parseStore.parseUpsertUserActivationMilestone(parseUserActivationMilestoneWrite{
		UserID:       parseAliceUserID,
		MilestoneKey: "starter.first_reply_completed",
		Status:       "completed",
		AchievedAt:   parseNow,
		MetadataJSON: `{"source":"admin-chats-test-alice"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserActivationMilestone alice: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertUserActivationMilestone(parseUserActivationMilestoneWrite{
		UserID:       parseBobUserID,
		MilestoneKey: "starter.first_reply_completed",
		Status:       "completed",
		AchievedAt:   parseNow,
		MetadataJSON: `{"source":"admin-chats-test-bob"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserActivationMilestone bob: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSavedWorkflow(parseSavedWorkflowWrite{
		WorkspaceID:  parseAliceWorkspaceID,
		UserID:       parseAliceUserID,
		WorkflowKey:  "wf-alice-summary",
		Name:         "Alice Weekly Summary",
		Description:  "Weekly summary workflow",
		WorkflowJSON: `{"steps":["summarize","send"]}`,
		IsPublic:     false,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSavedWorkflow alice: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSavedWorkflow(parseSavedWorkflowWrite{
		WorkspaceID:  parseBobWorkspaceID,
		UserID:       parseBobUserID,
		WorkflowKey:  "wf-bob-summary",
		Name:         "Bob Weekly Summary",
		Description:  "Weekly summary workflow",
		WorkflowJSON: `{"steps":["summarize","draft"]}`,
		IsPublic:     false,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSavedWorkflow bob: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertPromptLibraryItem(parsePromptLibraryItemWrite{
		WorkspaceID: parseAliceWorkspaceID,
		UserID:      parseAliceUserID,
		ItemKey:     "prompt-alice",
		Title:       "Alice Prompt",
		Category:    "assistant",
		PromptText:  "Draft a polished recap.",
		TagsJSON:    `["summary"]`,
		IsPublic:    false,
		UseCount:    2,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertPromptLibraryItem alice: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertPromptLibraryItem(parsePromptLibraryItemWrite{
		WorkspaceID: parseBobWorkspaceID,
		UserID:      parseBobUserID,
		ItemKey:     "prompt-bob",
		Title:       "Bob Prompt",
		Category:    "assistant",
		PromptText:  "Draft a support update.",
		TagsJSON:    `["support"]`,
		IsPublic:    false,
		UseCount:    1,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertPromptLibraryItem bob: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWeeklyValueSummary(parseWeeklyValueSummaryWrite{
		WorkspaceID: parseAliceWorkspaceID,
		UserID:      parseAliceUserID,
		SummaryWeek: "2026-W12",
		SummaryText: "Alice created value summary",
		MetricsJSON: `{"threads":4}`,
		SentAt:      parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWeeklyValueSummary alice: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWeeklyValueSummary(parseWeeklyValueSummaryWrite{
		WorkspaceID: parseBobWorkspaceID,
		UserID:      parseBobUserID,
		SummaryWeek: "2026-W12",
		SummaryText: "Bob created value summary",
		MetricsJSON: `{"threads":2}`,
		SentAt:      parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWeeklyValueSummary bob: %v", parseErr)
	}
	if _, parseErr := parseStore.parseCreateProductAnalyticsEvent(parseProductAnalyticsEventWrite{
		WorkspaceID:    parseAliceWorkspaceID,
		UserID:         parseAliceUserID,
		SessionKey:     "session-admin-chats-alice",
		EventName:      "chat.reply.completed",
		FunnelKey:      "visit_to_first_chat",
		StepKey:        "first_reply_completed",
		ExperimentKey:  "exp-dashboard-active",
		VariantKey:     "treatment",
		EventPropsJSON: `{"source":"admin-chats-test-alice"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateProductAnalyticsEvent alice: %v", parseErr)
	}
	if _, parseErr := parseStore.parseCreateProductAnalyticsEvent(parseProductAnalyticsEventWrite{
		WorkspaceID:    parseBobWorkspaceID,
		UserID:         parseBobUserID,
		SessionKey:     "session-admin-chats-bob",
		EventName:      "chat.reply.completed",
		FunnelKey:      "visit_to_first_chat",
		StepKey:        "first_reply_completed",
		ExperimentKey:  "exp-dashboard-active",
		VariantKey:     "treatment",
		EventPropsJSON: `{"source":"admin-chats-test-bob"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateProductAnalyticsEvent bob: %v", parseErr)
	}
}

// TestGetAdminChatsDrilldown verifies typed chats drill-down reads summary and table slices from chats-adjacent sources.
func TestGetAdminChatsDrilldown(parseT *testing.T) {
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
	parseAliceWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-chats-drilldown")
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-chats-drilldown")
	parseSeedAdminChatsDrilldownGrowthRows(parseT, parseStore, parseAliceAuth.ID, parseAliceWorkspaceID, parseBobAuth.ID, parseBobWorkspaceID)

	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-chats-drilldown-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseResp, parseErr := parseServer.GetAdminChatsDrilldown(parseAliceCtx, &chatpb.GetAdminChatsDrilldownRequest{
		LookbackDays: 30,
		Limit:        50,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminChatsDrilldown: %v", parseErr)
	}
	if parseResp.GetSummary() == nil {
		parseT.Fatalf("expected chats summary payload")
	}
	if parseResp.GetSummary().GetConversationCount() == 0 || parseResp.GetSummary().GetMessageCount() == 0 || parseResp.GetSummary().GetUsageEventCount() == 0 {
		parseT.Fatalf("expected chats summary counts from conversation/message/usage tables, got %+v", parseResp.GetSummary())
	}
	if len(parseResp.GetConversations()) == 0 ||
		len(parseResp.GetUsageEvents()) == 0 ||
		len(parseResp.GetOnboardingTemplates()) == 0 ||
		len(parseResp.GetActivationMilestones()) == 0 ||
		len(parseResp.GetSavedWorkflows()) == 0 ||
		len(parseResp.GetPromptLibraryItems()) == 0 ||
		len(parseResp.GetWeeklyValueSummaries()) == 0 ||
		len(parseResp.GetProductAnalyticsEvents()) == 0 {
		parseT.Fatalf(
			"expected non-empty chats drill-down slices, got conversations=%d usage=%d onboarding=%d milestones=%d workflows=%d prompts=%d weekly=%d analytics=%d",
			len(parseResp.GetConversations()),
			len(parseResp.GetUsageEvents()),
			len(parseResp.GetOnboardingTemplates()),
			len(parseResp.GetActivationMilestones()),
			len(parseResp.GetSavedWorkflows()),
			len(parseResp.GetPromptLibraryItems()),
			len(parseResp.GetWeeklyValueSummaries()),
			len(parseResp.GetProductAnalyticsEvents()),
		)
	}
}

// TestGetAdminChatsDrilldownWorkspaceScope verifies workspace-admin callers only receive scoped chats rows.
func TestGetAdminChatsDrilldownWorkspaceScope(parseT *testing.T) {
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
	parseAliceWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-chats-scope")
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-chats-scope")
	parseSeedAdminChatsDrilldownGrowthRows(parseT, parseStore, parseAliceAuth.ID, parseAliceWorkspaceID, parseBobAuth.ID, parseBobWorkspaceID)

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-chats-drilldown-bob", parseBobAuth.ID, parseBobAuth.Email)
	parseResp, parseErr := parseServer.GetAdminChatsDrilldown(parseBobCtx, &chatpb.GetAdminChatsDrilldownRequest{
		LookbackDays: 30,
		Limit:        25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminChatsDrilldown scoped bob: %v", parseErr)
	}
	for _, parseConversationRow := range parseResp.GetConversations() {
		if parseConversationRow.GetUserId() != parseBobAuth.ID {
			parseT.Fatalf("expected scoped conversations for bob only, got %+v", parseConversationRow)
		}
	}
	for _, parseMilestoneRow := range parseResp.GetActivationMilestones() {
		if parseMilestoneRow.GetUserId() != parseBobAuth.ID {
			parseT.Fatalf("expected scoped activation milestones for bob only, got %+v", parseMilestoneRow)
		}
	}
	for _, parseWorkflowRow := range parseResp.GetSavedWorkflows() {
		if parseWorkflowRow.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped workflows for bob workspace %d only, got %+v", parseBobWorkspaceID, parseWorkflowRow)
		}
	}
	for _, parsePromptRow := range parseResp.GetPromptLibraryItems() {
		if parsePromptRow.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped prompt-library rows for bob workspace %d only, got %+v", parseBobWorkspaceID, parsePromptRow)
		}
	}
	for _, parseWeeklyRow := range parseResp.GetWeeklyValueSummaries() {
		if parseWeeklyRow.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped weekly-value rows for bob workspace %d only, got %+v", parseBobWorkspaceID, parseWeeklyRow)
		}
	}
	for _, parseAnalyticsRow := range parseResp.GetProductAnalyticsEvents() {
		if parseAnalyticsRow.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped analytics rows for bob workspace %d only, got %+v", parseBobWorkspaceID, parseAnalyticsRow)
		}
	}
}
