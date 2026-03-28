package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
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

	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAlice.ID, "ws-admin-dashboard-seed")
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-admin-dashboard-open",
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseAlice.ID,
		Status:         "open",
		Priority:       "high",
		Subject:        "Billing interruption",
		Body:           "Customer payment failed and needs follow-up.",
		AssigneeUserID: parseAlice.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket: %v", parseErr)
	}

	parseMustAssignBillingPlan(parseT, parseStore, parseAlice.ID, "free")
	parseCustomer, hasParseCustomer, parseErr := parseStore.parseGetBillingCustomerByUser(parseAlice.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !hasParseCustomer {
		parseT.Fatalf("expected billing customer for user %d", parseAlice.ID)
	}
	parseSubscriptions, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 1)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: %v", parseErr)
	}
	if len(parseSubscriptions) == 0 {
		parseT.Fatalf("expected one billing subscription for customer %d", parseCustomer.ID)
	}
	parseSubscriptionID := parseSubscriptions[0].ID
	parseNowRFC3339 := time.Now().UTC().Format(time.RFC3339)
	parseInvoice, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomer.ID,
		SubscriptionID:    parseSubscriptionID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-admin-dashboard-open",
		Status:            "open",
		Currency:          "usd",
		SubtotalCents:     1000,
		TotalCents:        1000,
		AmountDueCents:    1000,
		AmountPaidCents:   0,
		PeriodStart:       parseNowRFC3339,
		PeriodEnd:         parseNowRFC3339,
		DueAt:             parseNowRFC3339,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:     parseCustomer.ID,
		SubscriptionID: parseSubscriptionID,
		InvoiceID:      parseInvoice.ID,
		EventType:      "invoice.payment_failed",
		EventSource:    "provider",
		EventSummary:   "payment failed",
		ActorUserID:    parseAlice.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO billing_dunning_events (customer_id, subscription_id, invoice_id, status, attempt_count, failure_reason, next_attempt_at, resolved_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		parseCustomer.ID,
		parseSubscriptionID,
		parseInvoice.ID,
		"pending",
		1,
		"card_declined",
		parseNowRFC3339,
		"",
		parseNowRFC3339,
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("seed billing_dunning_events: %v", parseErr)
	}

	if parseErr = parseStore.parseUpsertExperiment(parseExperimentWrite{
		ExperimentKey: "exp-dashboard-active",
		Name:          "Dashboard Active Experiment",
		Status:        "active",
		VariantsJSON:  `["control","treatment"]`,
		AudienceJSON:  `{"scope":"all"}`,
		StartAt:       parseNowRFC3339,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertExperiment active: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertExperiment(parseExperimentWrite{
		ExperimentKey: "exp-dashboard-paused",
		Name:          "Dashboard Paused Experiment",
		Status:        "paused",
		VariantsJSON:  `["control","treatment"]`,
		AudienceJSON:  `{"scope":"all"}`,
		StartAt:       parseNowRFC3339,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertExperiment paused: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertExperimentAssignment(parseExperimentAssignmentWrite{
		ExperimentKey: "exp-dashboard-active",
		WorkspaceID:   parseWorkspaceID,
		UserID:        parseAlice.ID,
		VariantKey:    "treatment",
		AssignedAt:    parseNowRFC3339,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertExperimentAssignment: %v", parseErr)
	}

	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO service_level_objectives (slo_key, service_name, objective_percent, window_days, error_budget_minutes, status_page_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"slo-admin-dashboard",
		"chat-api",
		99.9,
		30,
		0,
		"",
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("seed service_level_objectives: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO incidents (incident_key, slo_key, severity, status, title, summary, started_at, resolved_at, postmortem_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"incident-admin-dashboard-open",
		"slo-admin-dashboard",
		"major",
		"open",
		"Admin dashboard seeded incident",
		"Regional provider instability",
		parseNowRFC3339,
		"",
		"",
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("seed incidents: %v", parseErr)
	}
}

// parseSeedAdminUserControlSignals seeds one workspace, one auth session, and one audit row for one user.
func parseSeedAdminUserControlSignals(parseT *testing.T, parseStore *Store, parseUserID int64, parseWorkspaceKey string) int64 {
	parseT.Helper()
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUserID, parseWorkspaceKey)
	parseNow := time.Now().UTC()
	if parseErr := parseStore.parseUpsertAuthSession(parseAuthSessionWrite{
		UserID:           parseUserID,
		SessionID:        fmt.Sprintf("sess-%s-%d", parseWorkspaceKey, parseUserID),
		TokenVersion:     1,
		RefreshTokenHash: "refresh-hash-test",
		UserAgent:        "gwc-test-user-agent",
		IPAddress:        "127.0.0.1",
		LastSeenAt:       parseNow.Format(time.RFC3339),
		ExpiresAt:        parseNow.Add(24 * time.Hour).Format(time.RFC3339),
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertAuthSession: %v", parseErr)
	}
	if _, parseErr := parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseUserID,
		WorkspaceID: parseWorkspaceID,
		EventType:   "user.profile.viewed",
		TargetType:  "user",
		TargetID:    fmt.Sprintf("%d", parseUserID),
		Summary:     "User detail viewed",
		PayloadJSON: `{"source":"admin_dashboard_test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog: %v", parseErr)
	}
	return parseWorkspaceID
}

// parseSeedAdminWorkspaceControlSignals seeds workspace detail + mutation fixtures and returns workspace, key, and webhook ids.
func parseSeedAdminWorkspaceControlSignals(parseT *testing.T, parseStore *Store, parseUserID int64, parseWorkspaceKey string) (int64, string, int64) {
	parseT.Helper()
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseUserID, parseWorkspaceKey)
	parseKeyID := fmt.Sprintf("key-%s-%d", parseWorkspaceKey, parseUserID)
	if _, parseErr := parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       parseKeyID,
		WorkspaceID: parseWorkspaceID,
		UserID:      parseUserID,
		Label:       "Workspace Control Key",
		KeyPrefix:   "gwc_ws_ctrl",
		SecretHash:  "workspace-control-secret",
		ScopesJSON:  `["workspace.read","workspace.write"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey workspace control: %v", parseErr)
	}
	parseTargetURL := fmt.Sprintf("https://example.com/hooks/%s", parseWorkspaceKey)
	if parseErr := parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Workspace Control Hook",
		TargetURL:   parseTargetURL,
		SecretHash:  "workspace-control-webhook-secret",
		EventsJSON:  `["chat.reply.completed"]`,
		IsEnabled:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint workspace control: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWorkspaceInvitation(parseWorkspaceInvitationWrite{
		WorkspaceID:         parseWorkspaceID,
		Email:               "workspace-review@example.com",
		RoleKey:             "member",
		InvitationTokenHash: fmt.Sprintf("invite-token-%s", parseWorkspaceKey),
		InvitedByUserID:     parseUserID,
		Status:              "pending",
		ExpiresAt:           "2026-12-31T23:59:59Z",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceInvitation workspace control: %v", parseErr)
	}
	if _, parseErr := parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseUserID,
		WorkspaceID: parseWorkspaceID,
		EventType:   "workspace.membership.reviewed",
		TargetType:  "workspace",
		TargetID:    fmt.Sprintf("%d", parseWorkspaceID),
		Summary:     "Workspace membership review prepared",
		PayloadJSON: `{"source":"admin_dashboard_test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog workspace control: %v", parseErr)
	}
	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(20)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints workspace control: %v", parseErr)
	}
	parseEndpointID := int64(0)
	for _, parseWebhookRow := range parseWebhookRows {
		if parseWebhookRow.WorkspaceID != parseWorkspaceID || parseWebhookRow.TargetURL != parseTargetURL {
			continue
		}
		parseEndpointID = parseWebhookRow.ID
		break
	}
	if parseEndpointID <= 0 {
		parseT.Fatalf("expected webhook endpoint id for workspace %d, rows=%+v", parseWorkspaceID, parseWebhookRows)
	}
	return parseWorkspaceID, parseKeyID, parseEndpointID
}

// TestParseCompareAdminFloat64 verifies float comparator direction behavior for ascending and descending admin list sorts.
func TestParseCompareAdminFloat64(parseT *testing.T) {
	if !parseCompareAdminFloat64(1.25, 2.50, true) {
		parseT.Fatal("expected ascending float comparator to rank smaller value first")
	}
	if parseCompareAdminFloat64(2.50, 1.25, true) {
		parseT.Fatal("expected ascending float comparator to reject larger value first")
	}
	if !parseCompareAdminFloat64(2.50, 1.25, false) {
		parseT.Fatal("expected descending float comparator to rank larger value first")
	}
	if parseCompareAdminFloat64(1.25, 2.50, false) {
		parseT.Fatal("expected descending float comparator to reject smaller value first")
	}
}

// TestAdminDashboardSurfaceRoleScopeMatrix verifies normal users are denied, workspace admins are restricted to scoped customer/chat surfaces, and superusers can access all surfaces.
func TestAdminDashboardSurfaceRoleScopeMatrix(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-matrix-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-admin-matrix")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-matrix-bob", parseBobAuth.ID, parseBobAuth.Email)

	parseCarolAuth := parseMustCreateUser(parseT, parseStore, "carol@example.com")
	parseCarolCtx := parseBindAuthUser(parseServer, "peer-admin-matrix-carol", parseCarolAuth.ID, parseCarolAuth.Email)

	parseWorkspaceAllowedSlices := []string{
		"dashboard.users",
		"dashboard.workspace.detail",
		"dashboard.billing.events",
		"dashboard.support.queue",
		"dashboard.usage",
		"dashboard.conversations",
	}
	for _, parseSliceKey := range parseWorkspaceAllowedSlices {
		if _, parseErr = parseServer.parseRequireAdminSliceScope(parseBobCtx, parseSliceKey); parseErr != nil {
			parseT.Fatalf("workspace-admin expected allowed slice %q: %v", parseSliceKey, parseErr)
		}
	}

	parseSuperuserOnlySlices := []string{
		"dashboard.home",
		"dashboard.providers.summary",
		"dashboard.ops.incidents",
		"admin-read-only-report",
	}
	for _, parseSliceKey := range parseSuperuserOnlySlices {
		if _, parseErr = parseServer.parseRequireAdminSliceScope(parseBobCtx, parseSliceKey); status.Code(parseErr) != codes.PermissionDenied {
			parseT.Fatalf("workspace-admin expected denied superuser slice %q code=%v want=%v", parseSliceKey, status.Code(parseErr), codes.PermissionDenied)
		}
		if _, parseErr = parseServer.parseRequireAdminSliceScope(parseAliceCtx, parseSliceKey); parseErr != nil {
			parseT.Fatalf("superuser expected allowed slice %q: %v", parseSliceKey, parseErr)
		}
	}

	parseAllSurfaceSlices := append(append([]string{}, parseWorkspaceAllowedSlices...), parseSuperuserOnlySlices...)
	for _, parseSliceKey := range parseAllSurfaceSlices {
		if _, parseErr = parseServer.parseRequireAdminSliceScope(parseCarolCtx, parseSliceKey); status.Code(parseErr) != codes.PermissionDenied {
			parseT.Fatalf("normal user expected denied slice %q code=%v want=%v", parseSliceKey, status.Code(parseErr), codes.PermissionDenied)
		}
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
	if parseSummary.WindowBillingEvents != 1 || parseSummary.WindowOpenInvoices != 1 || parseSummary.WindowOpenDunningEvents != 1 {
		parseT.Fatalf("unexpected billing summary counts: %+v", parseSummary)
	}
	if parseSummary.OpenIncidents != 1 || parseSummary.OpenSupportTickets != 1 {
		parseT.Fatalf("unexpected ops backlog summary counts: %+v", parseSummary)
	}
	if parseSummary.ActiveExperiments != 1 || parseSummary.UnhealthyExperiments != 1 || parseSummary.WindowExperimentAssignments != 1 {
		parseT.Fatalf("unexpected experiment health summary counts: %+v", parseSummary)
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

// TestStoreAdminUserControlQueries verifies typed store helpers for admin user search/detail slices.
func TestStoreAdminUserControlQueries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-user-control-store")

	parseSearchRows, parseErr := parseStore.parseSearchAdminUsers("bob@", 10)
	if parseErr != nil {
		parseT.Fatalf("parseSearchAdminUsers: %v", parseErr)
	}
	if len(parseSearchRows) != 1 || parseSearchRows[0].UserID != parseBobAuth.ID {
		parseT.Fatalf("unexpected search rows: %+v", parseSearchRows)
	}

	parseUserRow, hasParseUser, parseErr := parseStore.parseGetAdminUserSummaryByUserID(parseBobAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminUserSummaryByUserID: %v", parseErr)
	}
	if !hasParseUser || parseUserRow.UserID != parseBobAuth.ID {
		parseT.Fatalf("expected bob summary row, got found=%v row=%+v", hasParseUser, parseUserRow)
	}

	parseSessionRows, parseErr := parseStore.parseListAdminAuthSessionsByUser(parseBobAuth.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminAuthSessionsByUser: %v", parseErr)
	}
	if len(parseSessionRows) == 0 || parseSessionRows[0].UserID != parseBobAuth.ID {
		parseT.Fatalf("unexpected session rows: %+v", parseSessionRows)
	}

	parseSince := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	parseUsageRows, parseErr := parseStore.parseListAdminUsageEventsByUser(parseBobAuth.ID, parseSince, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminUsageEventsByUser: %v", parseErr)
	}
	if len(parseUsageRows) == 0 || parseUsageRows[0].UserID != parseBobAuth.ID {
		parseT.Fatalf("unexpected usage rows: %+v", parseUsageRows)
	}

	parseAuditRows, parseErr := parseStore.parseListAdminAuditLogsByUser(parseBobAuth.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminAuditLogsByUser: %v", parseErr)
	}
	if len(parseAuditRows) == 0 {
		parseT.Fatalf("expected audit rows for bob user %d", parseBobAuth.ID)
	}
}

// TestStoreAdminWorkspaceControlQueries verifies typed workspace-admin detail query helpers.
func TestStoreAdminWorkspaceControlQueries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseWorkspaceID, parseKeyID, parseEndpointID := parseSeedAdminWorkspaceControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-workspace-control-store")

	parseWorkspaceRow, hasParseWorkspace, parseErr := parseStore.parseGetAdminWorkspaceByWorkspaceID(parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminWorkspaceByWorkspaceID: %v", parseErr)
	}
	if !hasParseWorkspace || parseWorkspaceRow.ID != parseWorkspaceID {
		parseT.Fatalf("expected workspace row %d, got found=%v row=%+v", parseWorkspaceID, hasParseWorkspace, parseWorkspaceRow)
	}
	parseMembershipRows, parseErr := parseStore.parseListAdminWorkspaceMembershipsByWorkspace(parseWorkspaceID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminWorkspaceMembershipsByWorkspace: %v", parseErr)
	}
	if len(parseMembershipRows) == 0 {
		parseT.Fatalf("expected membership rows for workspace %d", parseWorkspaceID)
	}
	parseInvitationRows, parseErr := parseStore.parseListAdminWorkspaceInvitationsByWorkspace(parseWorkspaceID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminWorkspaceInvitationsByWorkspace: %v", parseErr)
	}
	if len(parseInvitationRows) == 0 {
		parseT.Fatalf("expected invitation rows for workspace %d", parseWorkspaceID)
	}
	parseAPIKeyRows, parseErr := parseStore.parseListAdminAPIKeysByWorkspace(parseWorkspaceID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminAPIKeysByWorkspace: %v", parseErr)
	}
	if len(parseAPIKeyRows) == 0 {
		parseT.Fatalf("expected api key rows for workspace %d", parseWorkspaceID)
	}
	parseAPIKeyRow, hasParseAPIKey, parseErr := parseStore.parseGetAdminAPIKeyByKeyID(parseKeyID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminAPIKeyByKeyID: %v", parseErr)
	}
	if !hasParseAPIKey || parseAPIKeyRow.KeyID != parseKeyID {
		parseT.Fatalf("expected api key %q, got found=%v row=%+v", parseKeyID, hasParseAPIKey, parseAPIKeyRow)
	}
	parseWebhookRows, parseErr := parseStore.parseListAdminWebhookEndpointsByWorkspace(parseWorkspaceID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminWebhookEndpointsByWorkspace: %v", parseErr)
	}
	if len(parseWebhookRows) == 0 {
		parseT.Fatalf("expected webhook rows for workspace %d", parseWorkspaceID)
	}
	parseWebhookRow, hasParseWebhook, parseErr := parseStore.parseGetAdminWebhookEndpointByID(parseEndpointID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminWebhookEndpointByID: %v", parseErr)
	}
	if !hasParseWebhook || parseWebhookRow.ID != parseEndpointID {
		parseT.Fatalf("expected webhook endpoint %d, got found=%v row=%+v", parseEndpointID, hasParseWebhook, parseWebhookRow)
	}
	parseAuditRows, parseErr := parseStore.parseListAdminAuditLogsByWorkspace(parseWorkspaceID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminAuditLogsByWorkspace: %v", parseErr)
	}
	if len(parseAuditRows) == 0 {
		parseT.Fatalf("expected audit rows for workspace %d", parseWorkspaceID)
	}
}

// TestAdminDashboardRPCs verifies superuser gating and seeded analytics responses for admin RPCs.
func TestAdminDashboardRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)

	if _, parseErr = parseServer.GetAdminDashboard(context.Background(), &chatpb.GetAdminDashboardRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetAdminDashboard unauthenticated status code = %v, want %v", status.Code(parseErr), codes.Unauthenticated)
	}
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-bob", parseBobAuth.ID, parseBobAuth.Email)
	if _, parseErr = parseServer.GetAdminDashboard(parseBobCtx, &chatpb.GetAdminDashboardRequest{}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminDashboard non-superuser status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseDashboardResp, parseErr := parseServer.GetAdminDashboard(parseAliceCtx, &chatpb.GetAdminDashboardRequest{
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
	if parseDashboardResp.GetSummary().GetWindowBillingEvents() != 1 || parseDashboardResp.GetSummary().GetWindowOpenInvoices() != 1 || parseDashboardResp.GetSummary().GetWindowOpenDunningEvents() != 1 {
		parseT.Fatalf("unexpected dashboard billing summary: %+v", parseDashboardResp.GetSummary())
	}
	if parseDashboardResp.GetSummary().GetOpenIncidents() != 1 || parseDashboardResp.GetSummary().GetOpenSupportTickets() != 1 {
		parseT.Fatalf("unexpected dashboard ops backlog summary: %+v", parseDashboardResp.GetSummary())
	}
	if parseDashboardResp.GetSummary().GetActiveExperiments() != 1 || parseDashboardResp.GetSummary().GetUnhealthyExperiments() != 1 || parseDashboardResp.GetSummary().GetWindowExperimentAssignments() != 1 {
		parseT.Fatalf("unexpected dashboard experiment health summary: %+v", parseDashboardResp.GetSummary())
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

	parseUsersResp, parseErr := parseServer.ListAdminUsers(parseAliceCtx, &chatpb.ListAdminUsersRequest{Limit: 10})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers: %v", parseErr)
	}
	if len(parseUsersResp.GetUsers()) != 2 {
		parseT.Fatalf("unexpected ListAdminUsers rows: %+v", parseUsersResp.GetUsers())
	}

	parseUsageResp, parseErr := parseServer.ListAdminUsageEvents(parseAliceCtx, &chatpb.ListAdminUsageEventsRequest{
		LookbackDays: 30,
		Limit:        10,
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsageEvents: %v", parseErr)
	}
	if len(parseUsageResp.GetEvents()) != 3 {
		parseT.Fatalf("unexpected ListAdminUsageEvents rows: %+v", parseUsageResp.GetEvents())
	}

	parseConversationsResp, parseErr := parseServer.ListAdminConversations(parseAliceCtx, &chatpb.ListAdminConversationsRequest{Limit: 10})
	if parseErr != nil {
		parseT.Fatalf("ListAdminConversations: %v", parseErr)
	}
	if len(parseConversationsResp.GetConversations()) != 2 {
		parseT.Fatalf("unexpected ListAdminConversations rows: %+v", parseConversationsResp.GetConversations())
	}
	parseAuditRows, parseErr := parseStore.parseListAuditLogs(50)
	if parseErr != nil {
		parseT.Fatalf("parseListAuditLogs: %v", parseErr)
	}
	hasParseDashboardEntry := false
	parseSliceViewCount := 0
	for _, parseAuditRow := range parseAuditRows {
		if parseAuditRow.EventType == "admin.dashboard.entry" {
			hasParseDashboardEntry = true
		}
		if parseAuditRow.EventType == "admin.dashboard.slice.view" {
			parseSliceViewCount++
		}
	}
	if !hasParseDashboardEntry || parseSliceViewCount < 3 {
		parseT.Fatalf("expected dashboard entry + slice audit rows, got rows=%+v", parseAuditRows)
	}
}

// TestListAdminUsersAppliesTypedListQuery verifies search, sort, and pagination behavior on ListAdminUsers.
func TestListAdminUsersAppliesTypedListQuery(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAdmin := parseMustCreateUser(parseT, parseStore, "list-query-admin@example.com")
	parseMustCreateUser(parseT, parseStore, "list-query-user-a@example.com")
	parseMustCreateUser(parseT, parseStore, "list-query-user-b@example.com")
	parseGrantSuperuserRole(parseT, parseStore, parseAdmin.ID)
	parseAdminCtx := parseBindAuthUser(parseServer, "peer-list-query-admin", parseAdmin.ID, parseAdmin.Email)

	parseResp, parseErr := parseServer.ListAdminUsers(parseAdminCtx, &chatpb.ListAdminUsersRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "list-query-user-",
			SortBy:        "email",
			SortDirection: "asc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers list query: %v", parseErr)
	}
	if len(parseResp.GetUsers()) != 1 {
		parseT.Fatalf("expected exactly one paged user row, got %+v", parseResp.GetUsers())
	}
	if parseResp.GetUsers()[0].GetEmail() != "list-query-user-b@example.com" {
		parseT.Fatalf("unexpected paged user row: %+v", parseResp.GetUsers()[0])
	}
}

// TestAdminUserControlRPCs verifies typed admin user search/detail/mutation RPC behavior for superuser callers.
func TestAdminUserControlRPCs(parseT *testing.T) {
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
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-user-control-rpc")

	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-user-control-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseSearchResp, parseErr := parseServer.SearchAdminUsers(parseAliceCtx, &chatpb.SearchAdminUsersRequest{
		Query: "bob@",
		Limit: 10,
	})
	if parseErr != nil {
		parseT.Fatalf("SearchAdminUsers: %v", parseErr)
	}
	if len(parseSearchResp.GetUsers()) != 1 || parseSearchResp.GetUsers()[0].GetUserId() != parseBobAuth.ID {
		parseT.Fatalf("unexpected SearchAdminUsers rows: %+v", parseSearchResp.GetUsers())
	}

	parseDetailResp, parseErr := parseServer.GetAdminUserDetail(parseAliceCtx, &chatpb.GetAdminUserDetailRequest{
		UserId:       parseBobAuth.ID,
		LookbackDays: 30,
		Limit:        10,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminUserDetail: %v", parseErr)
	}
	if parseDetailResp.GetDetail() == nil || parseDetailResp.GetDetail().GetUser().GetUserId() != parseBobAuth.ID {
		parseT.Fatalf("unexpected GetAdminUserDetail user payload: %+v", parseDetailResp.GetDetail())
	}
	if len(parseDetailResp.GetDetail().GetRecentSessions()) == 0 || len(parseDetailResp.GetDetail().GetRecentUsageEvents()) == 0 || len(parseDetailResp.GetDetail().GetRecentAuditLogs()) == 0 {
		parseT.Fatalf(
			"expected non-empty detail slices, got sessions=%d usage=%d audit=%d",
			len(parseDetailResp.GetDetail().GetRecentSessions()),
			len(parseDetailResp.GetDetail().GetRecentUsageEvents()),
			len(parseDetailResp.GetDetail().GetRecentAuditLogs()),
		)
	}

	if _, parseErr = parseServer.DisableAdminUser(parseAliceCtx, &chatpb.AdminUserMutationRequest{
		UserId:  parseBobAuth.ID,
		Confirm: false,
		Reason:  "policy enforcement",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("DisableAdminUser without confirm status code = %v, want %v", status.Code(parseErr), codes.InvalidArgument)
	}

	parseDisableResp, parseErr := parseServer.DisableAdminUser(parseAliceCtx, &chatpb.AdminUserMutationRequest{
		UserId:  parseBobAuth.ID,
		Confirm: true,
		Reason:  "policy enforcement",
	})
	if parseErr != nil {
		parseT.Fatalf("DisableAdminUser: %v", parseErr)
	}
	if parseDisableResp.GetUserId() != parseBobAuth.ID || parseDisableResp.GetStatus() != "disabled" {
		parseT.Fatalf("unexpected disable response: %+v", parseDisableResp)
	}
	isParseDisabled, parseErr := parseStore.parseIsUserAccessDisabled(parseBobAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseIsUserAccessDisabled after disable: %v", parseErr)
	}
	if !isParseDisabled {
		parseT.Fatalf("expected bob user %d to be disabled", parseBobAuth.ID)
	}

	parseRestoreResp, parseErr := parseServer.RestoreAdminUser(parseAliceCtx, &chatpb.AdminUserMutationRequest{
		UserId:  parseBobAuth.ID,
		Confirm: true,
		Reason:  "restored after review",
	})
	if parseErr != nil {
		parseT.Fatalf("RestoreAdminUser: %v", parseErr)
	}
	if parseRestoreResp.GetUserId() != parseBobAuth.ID || parseRestoreResp.GetStatus() != "active" {
		parseT.Fatalf("unexpected restore response: %+v", parseRestoreResp)
	}
	isParseDisabled, parseErr = parseStore.parseIsUserAccessDisabled(parseBobAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseIsUserAccessDisabled after restore: %v", parseErr)
	}
	if isParseDisabled {
		parseT.Fatalf("expected bob user %d to be active after restore", parseBobAuth.ID)
	}
}

// TestAdminUserListQueryRPCs verifies typed search, sort, and pagination behavior for admin user list RPCs.
func TestAdminUserListQueryRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-user-list-query-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseResp, parseErr := parseServer.ListAdminUsers(parseAliceCtx, &chatpb.ListAdminUsersRequest{
		ListQuery: &chatpb.AdminListQuery{
			Search:        "example.com",
			SortBy:        "email",
			SortDirection: "asc",
			Limit:         1,
			Offset:        1,
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers list query: %v", parseErr)
	}
	if len(parseResp.GetUsers()) != 1 || parseResp.GetUsers()[0].GetEmail() != "bob@example.com" {
		parseT.Fatalf("expected one paged/sorted user row for bob@example.com, got %+v", parseResp.GetUsers())
	}
}

// TestAdminWorkspaceControlRPCs verifies typed workspace detail and mutation RPC behavior for superuser callers.
func TestAdminWorkspaceControlRPCs(parseT *testing.T) {
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
	parseWorkspaceID, parseKeyID, parseEndpointID := parseSeedAdminWorkspaceControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-workspace-control-rpc")
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-workspace-control-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseDetailResp, parseErr := parseServer.GetAdminWorkspaceDetail(parseAliceCtx, &chatpb.GetAdminWorkspaceDetailRequest{
		WorkspaceId:  parseWorkspaceID,
		LookbackDays: 30,
		Limit:        25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminWorkspaceDetail: %v", parseErr)
	}
	if parseDetailResp.GetDetail() == nil || parseDetailResp.GetDetail().GetWorkspace() == nil || parseDetailResp.GetDetail().GetWorkspace().GetId() != parseWorkspaceID {
		parseT.Fatalf("unexpected workspace detail payload: %+v", parseDetailResp.GetDetail())
	}
	if len(parseDetailResp.GetDetail().GetMemberships()) == 0 || len(parseDetailResp.GetDetail().GetInvitations()) == 0 || len(parseDetailResp.GetDetail().GetApiKeys()) == 0 || len(parseDetailResp.GetDetail().GetWebhookEndpoints()) == 0 {
		parseT.Fatalf(
			"expected non-empty workspace detail slices, got memberships=%d invitations=%d api_keys=%d webhooks=%d",
			len(parseDetailResp.GetDetail().GetMemberships()),
			len(parseDetailResp.GetDetail().GetInvitations()),
			len(parseDetailResp.GetDetail().GetApiKeys()),
			len(parseDetailResp.GetDetail().GetWebhookEndpoints()),
		)
	}

	parseRevokeResp, parseErr := parseServer.RevokeWorkspaceAPIKey(parseAliceCtx, &chatpb.RevokeWorkspaceAPIKeyRequest{
		WorkspaceId: parseWorkspaceID,
		KeyId:       parseKeyID,
		Confirm:     true,
		Reason:      "rotating compromised key",
	})
	if parseErr != nil {
		parseT.Fatalf("RevokeWorkspaceAPIKey: %v", parseErr)
	}
	if parseRevokeResp.GetStatus() != "revoked" || parseRevokeResp.GetKeyId() != parseKeyID {
		parseT.Fatalf("unexpected revoke response: %+v", parseRevokeResp)
	}
	parseAPIKeyRow, hasParseAPIKey, parseErr := parseStore.parseGetAdminAPIKeyByKeyID(parseKeyID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminAPIKeyByKeyID after revoke: %v", parseErr)
	}
	if !hasParseAPIKey || parseAPIKeyRow.RevokedAt == "" {
		parseT.Fatalf("expected revoked api key row, got found=%v row=%+v", hasParseAPIKey, parseAPIKeyRow)
	}

	parsePauseResp, parseErr := parseServer.PauseWorkspaceWebhookEndpoint(parseAliceCtx, &chatpb.PauseWorkspaceWebhookRequest{
		WorkspaceId: parseWorkspaceID,
		EndpointId:  parseEndpointID,
		Confirm:     true,
		Reason:      "pause outbound callbacks during review",
	})
	if parseErr != nil {
		parseT.Fatalf("PauseWorkspaceWebhookEndpoint: %v", parseErr)
	}
	if parsePauseResp.GetStatus() != "paused" || parsePauseResp.GetEndpointId() != parseEndpointID {
		parseT.Fatalf("unexpected pause response: %+v", parsePauseResp)
	}
	parseWebhookRow, hasParseWebhook, parseErr := parseStore.parseGetAdminWebhookEndpointByID(parseEndpointID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminWebhookEndpointByID after pause: %v", parseErr)
	}
	if !hasParseWebhook || parseWebhookRow.IsEnabled {
		parseT.Fatalf("expected paused webhook endpoint, got found=%v row=%+v", hasParseWebhook, parseWebhookRow)
	}

	parseSuspendResp, parseErr := parseServer.SuspendAdminWorkspace(parseAliceCtx, &chatpb.AdminWorkspaceMutationRequest{
		WorkspaceId: parseWorkspaceID,
		Confirm:     true,
		Reason:      "security hold",
	})
	if parseErr != nil {
		parseT.Fatalf("SuspendAdminWorkspace: %v", parseErr)
	}
	if parseSuspendResp.GetWorkspaceId() != parseWorkspaceID || parseSuspendResp.GetStatus() != "suspended" {
		parseT.Fatalf("unexpected suspend response: %+v", parseSuspendResp)
	}

	parseRestoreResp, parseErr := parseServer.RestoreAdminWorkspace(parseAliceCtx, &chatpb.AdminWorkspaceMutationRequest{
		WorkspaceId:             parseWorkspaceID,
		Confirm:                 true,
		Reason:                  "restored after validation",
		RestoreApiKeys:          true,
		RestoreWebhookEndpoints: true,
		RestoreBackgroundJobs:   true,
	})
	if parseErr != nil {
		parseT.Fatalf("RestoreAdminWorkspace: %v", parseErr)
	}
	if parseRestoreResp.GetWorkspaceId() != parseWorkspaceID || parseRestoreResp.GetStatus() != "active" {
		parseT.Fatalf("unexpected restore response: %+v", parseRestoreResp)
	}
	parseWorkspaceRow, hasParseWorkspace, parseErr := parseStore.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("parseGetWorkspaceByID after restore: %v", parseErr)
	}
	if !hasParseWorkspace || parseWorkspaceRow.Status != "active" {
		parseT.Fatalf("expected restored workspace status=active, got found=%v row=%+v", hasParseWorkspace, parseWorkspaceRow)
	}
}

// TestAdminDashboardRPCsRequireSessionAuth verifies admin RPCs fail closed without a valid session token and evict cached peer auth on logout.
func TestAdminDashboardRPCsRequireSessionAuth(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail: %v", parseErr)
	}
	parseAlice := authUser{ID: parseAliceAuth.ID, Email: parseAliceAuth.Email}
	parseGrantSuperuserRole(parseT, parseStore, parseAlice.ID)

	parsePeerAddress := "peer-admin-session-required"
	parsePeerCtx := parseNewAuthenticatedContext(parsePeerAddress)
	parseServer.parseBindAuthenticatedPeer(parsePeerAddress, parseAlice)
	if _, parseErr = parseServer.GetAdminDashboard(parsePeerCtx, &chatpb.GetAdminDashboardRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetAdminDashboard peer-only status code = %v, want %v", status.Code(parseErr), codes.Unauthenticated)
	}

	parseToken, parseErr := parseServer.authManager.issueTokenForContext(parsePeerCtx, parseAlice, "")
	if parseErr != nil {
		parseT.Fatalf("issueTokenForContext: %v", parseErr)
	}
	parseAuthCtx := metadata.NewIncomingContext(parsePeerCtx, metadata.Pairs(authMetadataKey, "Bearer "+parseToken))
	if _, parseErr = parseServer.GetAdminDashboard(parseAuthCtx, &chatpb.GetAdminDashboardRequest{}); parseErr != nil {
		parseT.Fatalf("GetAdminDashboard session-backed: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertSURole(parseSURoleWrite{
		RoleKey:     "su",
		Label:       "Superuser",
		Description: "Temporarily disabled for downgrade coverage",
		IsSystem:    true,
		IsEnabled:   false,
		Permissions: []parseSURolePermissionRow{
			{PermissionKey: "control_plane.*", PermissionValue: "allow"},
		},
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSURole downgrade: %v", parseErr)
	}
	parseMembershipRows, parseErr := parseStore.parseListWorkspaceMembershipsByUser(parseAlice.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaceMembershipsByUser downgrade: %v", parseErr)
	}
	for _, parseMembershipRow := range parseMembershipRows {
		if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
			WorkspaceID:     parseMembershipRow.WorkspaceID,
			UserID:          parseMembershipRow.UserID,
			RoleKey:         parseMembershipRow.RoleKey,
			Status:          "disabled",
			InvitedByUserID: parseMembershipRow.InvitedByUserID,
		}); parseErr != nil {
			parseT.Fatalf("parseUpsertWorkspaceMembership downgrade: %v", parseErr)
		}
	}
	if _, parseErr = parseServer.GetAdminDashboard(parseAuthCtx, &chatpb.GetAdminDashboardRequest{}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminDashboard downgraded role status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.Logout(parseAuthCtx, &emptypb.Empty{}); parseErr != nil {
		parseT.Fatalf("Logout: %v", parseErr)
	}
	if _, parseFoundPeer := parseServer.authUsers[parsePeerAddress]; parseFoundPeer {
		parseT.Fatalf("expected logout to evict cached peer auth user for %q", parsePeerAddress)
	}
	if _, parseErr = parseServer.GetAdminDashboard(parsePeerCtx, &chatpb.GetAdminDashboardRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetAdminDashboard peer-only after logout status code = %v, want %v", status.Code(parseErr), codes.Unauthenticated)
	}
	if _, parseErr = parseServer.GetAdminDashboard(parseAuthCtx, &chatpb.GetAdminDashboardRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetAdminDashboard revoked token status code = %v, want %v", status.Code(parseErr), codes.Unauthenticated)
	}
}

func TestAdminDashboardRPCsWorkspaceAdminScope(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "ws-bob-admin-scope",
		Slug:         "ws-bob-admin-scope",
		Name:         "Bob Admin Scope",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseBobAuth.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	parseWorkspaceID := int64(0)
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		if parseWorkspaceRow.WorkspaceKey != "ws-bob-admin-scope" {
			continue
		}
		parseWorkspaceID = parseWorkspaceRow.ID
		break
	}
	if parseWorkspaceID <= 0 {
		parseT.Fatalf("expected seeded workspace id, rows=%+v", parseWorkspaceRows)
	}
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseBobAuth.ID,
		RoleKey:         "admin",
		Status:          "active",
		InvitedByUserID: parseBobAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership: %v", parseErr)
	}

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-bob-workspace", parseBobAuth.ID, parseBobAuth.Email)
	if _, parseErr = parseServer.GetAdminDashboard(parseBobCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 30,
		TopLimit:     10,
		RecentLimit:  10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminDashboard workspace-admin status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}

	parseUsersResp, parseErr := parseServer.ListAdminUsers(parseBobCtx, &chatpb.ListAdminUsersRequest{Limit: 10})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers workspace-admin: %v", parseErr)
	}
	if len(parseUsersResp.GetUsers()) != 1 || parseUsersResp.GetUsers()[0].GetEmail() != "bob@example.com" {
		parseT.Fatalf("expected scoped admin users to include only bob, got %+v", parseUsersResp.GetUsers())
	}

	parseUsageResp, parseErr := parseServer.ListAdminUsageEvents(parseBobCtx, &chatpb.ListAdminUsageEventsRequest{LookbackDays: 30, Limit: 10})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsageEvents workspace-admin: %v", parseErr)
	}
	if len(parseUsageResp.GetEvents()) != 1 || parseUsageResp.GetEvents()[0].GetEmail() != "bob@example.com" {
		parseT.Fatalf("expected scoped usage events to include only bob, got %+v", parseUsageResp.GetEvents())
	}
	if parseUsageResp.GetEvents()[0].GetProviderRequestId() != "" || parseUsageResp.GetEvents()[0].GetClientId() != "" {
		parseT.Fatalf("expected workspace-scoped usage diagnostics to be redacted, got %+v", parseUsageResp.GetEvents()[0])
	}

	parseConversationResp, parseErr := parseServer.ListAdminConversations(parseBobCtx, &chatpb.ListAdminConversationsRequest{Limit: 10})
	if parseErr != nil {
		parseT.Fatalf("ListAdminConversations workspace-admin: %v", parseErr)
	}
	if len(parseConversationResp.GetConversations()) != 1 || parseConversationResp.GetConversations()[0].GetEmail() != "bob@example.com" {
		parseT.Fatalf("expected scoped conversations to include only bob, got %+v", parseConversationResp.GetConversations())
	}
	if parseConversationResp.GetConversations()[0].GetPreview() != parseWorkspaceScopeRedactionText {
		parseT.Fatalf("expected workspace-scoped conversation preview redaction, got %+v", parseConversationResp.GetConversations()[0])
	}
}

// TestAdminUserControlRPCsWorkspaceAdminScope verifies workspace-admin scope filtering and fail-closed mutation boundaries.
func TestAdminUserControlRPCsWorkspaceAdminScope(parseT *testing.T) {
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
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-user-control-scope")

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-user-control-bob", parseBobAuth.ID, parseBobAuth.Email)
	parseSearchResp, parseErr := parseServer.SearchAdminUsers(parseBobCtx, &chatpb.SearchAdminUsersRequest{
		Query: "",
		Limit: 10,
	})
	if parseErr != nil {
		parseT.Fatalf("SearchAdminUsers workspace-admin: %v", parseErr)
	}
	if len(parseSearchResp.GetUsers()) != 1 || parseSearchResp.GetUsers()[0].GetUserId() != parseBobAuth.ID {
		parseT.Fatalf("expected scoped search to include bob only, got %+v", parseSearchResp.GetUsers())
	}

	parseUserDetailResp, parseErr := parseServer.GetAdminUserDetail(parseBobCtx, &chatpb.GetAdminUserDetailRequest{
		UserId:       parseBobAuth.ID,
		LookbackDays: 30,
		Limit:        10,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminUserDetail scoped bob: %v", parseErr)
	}
	if len(parseUserDetailResp.GetDetail().GetRecentSessions()) > 0 {
		parseSessionEntry := parseUserDetailResp.GetDetail().GetRecentSessions()[0]
		if parseSessionEntry.GetSessionId() != "" || parseSessionEntry.GetIpAddress() != "" {
			parseT.Fatalf("expected workspace-scoped auth session redaction, got %+v", parseSessionEntry)
		}
	}
	if len(parseUserDetailResp.GetDetail().GetRecentUsageEvents()) > 0 {
		parseUsageEntry := parseUserDetailResp.GetDetail().GetRecentUsageEvents()[0]
		if parseUsageEntry.GetProviderRequestId() != "" || parseUsageEntry.GetClientId() != "" {
			parseT.Fatalf("expected workspace-scoped user detail usage redaction, got %+v", parseUsageEntry)
		}
	}
	if len(parseUserDetailResp.GetDetail().GetRecentAuditLogs()) > 0 && parseUserDetailResp.GetDetail().GetRecentAuditLogs()[0].GetPayloadJson() != "{}" {
		parseT.Fatalf("expected workspace-scoped user detail audit payload redaction, got %+v", parseUserDetailResp.GetDetail().GetRecentAuditLogs()[0])
	}

	if _, parseErr = parseServer.GetAdminUserDetail(parseBobCtx, &chatpb.GetAdminUserDetailRequest{
		UserId:       parseAliceAuth.ID,
		LookbackDays: 30,
		Limit:        10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminUserDetail out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.DisableAdminUser(parseBobCtx, &chatpb.AdminUserMutationRequest{
		UserId:  parseAliceAuth.ID,
		Confirm: true,
		Reason:  "workspace policy",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("DisableAdminUser out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestAdminWorkspaceControlRPCsWorkspaceAdminScope verifies workspace-admin scope boundaries for workspace detail and mutations.
func TestAdminWorkspaceControlRPCsWorkspaceAdminScope(parseT *testing.T) {
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
	parseBobWorkspaceID, parseKeyID, parseEndpointID := parseSeedAdminWorkspaceControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-workspace-control-scope")
	parseAliceWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-workspace-control-scope")

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-workspace-control-bob", parseBobAuth.ID, parseBobAuth.Email)
	if _, parseErr = parseServer.GetAdminWorkspaceDetail(parseBobCtx, &chatpb.GetAdminWorkspaceDetailRequest{
		WorkspaceId:  parseBobWorkspaceID,
		LookbackDays: 30,
		Limit:        25,
	}); parseErr != nil {
		parseT.Fatalf("GetAdminWorkspaceDetail scoped bob workspace: %v", parseErr)
	}
	if _, parseErr = parseServer.GetAdminWorkspaceDetail(parseBobCtx, &chatpb.GetAdminWorkspaceDetailRequest{
		WorkspaceId:  parseAliceWorkspaceID,
		LookbackDays: 30,
		Limit:        25,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminWorkspaceDetail out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.RevokeWorkspaceAPIKey(parseBobCtx, &chatpb.RevokeWorkspaceAPIKeyRequest{
		WorkspaceId: parseBobWorkspaceID,
		KeyId:       parseKeyID,
		Confirm:     true,
		Reason:      "rotate key",
	}); parseErr != nil {
		parseT.Fatalf("RevokeWorkspaceAPIKey scoped workspace: %v", parseErr)
	}
	if _, parseErr = parseServer.PauseWorkspaceWebhookEndpoint(parseBobCtx, &chatpb.PauseWorkspaceWebhookRequest{
		WorkspaceId: parseBobWorkspaceID,
		EndpointId:  parseEndpointID,
		Confirm:     true,
		Reason:      "pause webhook",
	}); parseErr != nil {
		parseT.Fatalf("PauseWorkspaceWebhookEndpoint scoped workspace: %v", parseErr)
	}
	if _, parseErr = parseServer.SuspendAdminWorkspace(parseBobCtx, &chatpb.AdminWorkspaceMutationRequest{
		WorkspaceId: parseAliceWorkspaceID,
		Confirm:     true,
		Reason:      "out-of-scope attempt",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SuspendAdminWorkspace out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestGetWorkspaceAdminSlices verifies workspace-admin scoped control-plane slices and billing summary fields.
func TestGetWorkspaceAdminSlices(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseBobWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseBobAuth.ID, "ws-bob-workspace-slices")
	parseNowRFC3339 := time.Now().UTC().Format(time.RFC3339)

	if _, parseErr = parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "key-bob-scoped",
		WorkspaceID: parseBobWorkspaceID,
		UserID:      parseBobAuth.ID,
		Label:       "Bob Scoped Key",
		KeyPrefix:   "gwc_bob",
		SecretHash:  "hash-bob",
		ScopesJSON:  `["workspace.read"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey bob: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID:  parseBobWorkspaceID,
		Label:        "Bob Scoped Hook",
		TargetURL:    "https://example.com/hooks/bob",
		SecretHash:   "secret-bob",
		EventsJSON:   `["chat.reply.completed"]`,
		IsEnabled:    true,
		FailureCount: 0,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint bob: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseBobAuth.ID,
		WorkspaceID: parseBobWorkspaceID,
		EventType:   "workspace.member.invited",
		TargetType:  "workspace",
		TargetID:    "ws-bob-workspace-slices",
		Summary:     "Bob invited one teammate",
		PayloadJSON: `{"source":"test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog bob: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspaceInvitation(parseWorkspaceInvitationWrite{
		WorkspaceID:         parseBobWorkspaceID,
		Email:               "bob-invitee@example.com",
		RoleKey:             "member",
		InvitationTokenHash: "invite-token-hash-bob-slices",
		InvitedByUserID:     parseBobAuth.ID,
		Status:              "pending",
		ExpiresAt:           "2026-12-31T23:59:59Z",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceInvitation bob: %v", parseErr)
	}

	parseMustAssignBillingPlan(parseT, parseStore, parseBobAuth.ID, "free")
	parseBobCustomer, hasParseBobCustomer, parseErr := parseStore.parseGetBillingCustomerByUser(parseBobAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser bob: %v", parseErr)
	}
	if !hasParseBobCustomer {
		parseT.Fatalf("expected billing customer for bob user %d", parseBobAuth.ID)
	}
	parseBobSubscriptions, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseBobCustomer.ID, 1)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer bob: %v", parseErr)
	}
	if len(parseBobSubscriptions) == 0 {
		parseT.Fatalf("expected billing subscription for bob customer %d", parseBobCustomer.ID)
	}
	parseBobInvoice, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseBobCustomer.ID,
		SubscriptionID:    parseBobSubscriptions[0].ID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-bob-workspace-slices-open",
		Status:            "open",
		Currency:          "usd",
		SubtotalCents:     1200,
		TotalCents:        1200,
		AmountDueCents:    1200,
		AmountPaidCents:   0,
		PeriodStart:       parseNowRFC3339,
		PeriodEnd:         parseNowRFC3339,
		DueAt:             parseNowRFC3339,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice bob: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:     parseBobCustomer.ID,
		SubscriptionID: parseBobSubscriptions[0].ID,
		InvoiceID:      parseBobInvoice.ID,
		EventType:      "dunning.email_sent",
		EventSource:    "system",
		EventSummary:   "Dunning email sent",
		ActorUserID:    parseBobAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent bob: %v", parseErr)
	}

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseStore.parseListWorkspaces(20)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	parseAliceWorkspaceID := int64(0)
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		if parseWorkspaceRow.WorkspaceKey != "ws-admin-dashboard-seed" {
			continue
		}
		parseAliceWorkspaceID = parseWorkspaceRow.ID
		break
	}
	if parseAliceWorkspaceID <= 0 {
		parseT.Fatalf("expected alice workspace id in seeded rows: %+v", parseWorkspaceRows)
	}
	if _, parseErr = parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "key-alice-decoy",
		WorkspaceID: parseAliceWorkspaceID,
		UserID:      parseAliceAuth.ID,
		Label:       "Alice Decoy Key",
		KeyPrefix:   "gwc_alice",
		SecretHash:  "hash-alice",
		ScopesJSON:  `["workspace.write"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey alice decoy: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID:  parseAliceWorkspaceID,
		Label:        "Alice Decoy Hook",
		TargetURL:    "https://example.com/hooks/alice",
		SecretHash:   "secret-alice",
		EventsJSON:   `["chat.reply.completed"]`,
		IsEnabled:    true,
		FailureCount: 0,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint alice decoy: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseAliceAuth.ID,
		WorkspaceID: parseAliceWorkspaceID,
		EventType:   "workspace.updated",
		TargetType:  "workspace",
		TargetID:    "ws-admin-dashboard-seed",
		Summary:     "Alice decoy audit event",
		PayloadJSON: `{"source":"test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog alice decoy: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspaceInvitation(parseWorkspaceInvitationWrite{
		WorkspaceID:         parseAliceWorkspaceID,
		Email:               "alice-invitee@example.com",
		RoleKey:             "member",
		InvitationTokenHash: "invite-token-hash-alice-decoy",
		InvitedByUserID:     parseAliceAuth.ID,
		Status:              "pending",
		ExpiresAt:           "2026-12-31T23:59:59Z",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceInvitation alice decoy: %v", parseErr)
	}

	parseBobCtx := parseBindAuthUser(parseServer, "peer-workspace-slices-bob", parseBobAuth.ID, parseBobAuth.Email)
	parseSlicesResp, parseErr := parseServer.GetWorkspaceAdminSlices(parseBobCtx, &chatpb.GetWorkspaceAdminSlicesRequest{
		LookbackDays: 30,
		Limit:        25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetWorkspaceAdminSlices: %v", parseErr)
	}
	if len(parseSlicesResp.GetMemberships()) == 0 || len(parseSlicesResp.GetApiKeys()) == 0 || len(parseSlicesResp.GetWebhookEndpoints()) == 0 || len(parseSlicesResp.GetAuditLogs()) == 0 || len(parseSlicesResp.GetInvitations()) == 0 {
		parseT.Fatalf("expected non-empty scoped slices, got memberships=%d api_keys=%d webhooks=%d audit=%d invitations=%d", len(parseSlicesResp.GetMemberships()), len(parseSlicesResp.GetApiKeys()), len(parseSlicesResp.GetWebhookEndpoints()), len(parseSlicesResp.GetAuditLogs()), len(parseSlicesResp.GetInvitations()))
	}
	for _, parseMembership := range parseSlicesResp.GetMemberships() {
		if parseMembership.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped membership workspace id %d, got %+v", parseBobWorkspaceID, parseMembership)
		}
	}
	for _, parseAPIKey := range parseSlicesResp.GetApiKeys() {
		if parseAPIKey.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped api key workspace id %d, got %+v", parseBobWorkspaceID, parseAPIKey)
		}
	}
	for _, parseWebhook := range parseSlicesResp.GetWebhookEndpoints() {
		if parseWebhook.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped webhook workspace id %d, got %+v", parseBobWorkspaceID, parseWebhook)
		}
	}
	for _, parseAudit := range parseSlicesResp.GetAuditLogs() {
		if parseAudit.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped audit workspace id %d, got %+v", parseBobWorkspaceID, parseAudit)
		}
	}
	for _, parseInvitation := range parseSlicesResp.GetInvitations() {
		if parseInvitation.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped invitation workspace id %d, got %+v", parseBobWorkspaceID, parseInvitation)
		}
	}
	if len(parseSlicesResp.GetUsageEvents()) != 1 || parseSlicesResp.GetUsageEvents()[0].GetEmail() != "bob@example.com" {
		parseT.Fatalf("expected scoped usage rows for bob only, got %+v", parseSlicesResp.GetUsageEvents())
	}
	if parseSlicesResp.GetBillingSummary() == nil {
		parseT.Fatalf("expected billing summary in workspace slices response, got %+v", parseSlicesResp)
	}
	if parseSlicesResp.GetBillingSummary().GetCustomerCount() != 1 || parseSlicesResp.GetBillingSummary().GetActiveSubscriptionCount() < 1 {
		parseT.Fatalf("expected scoped billing customer/subscription counts, got %+v", parseSlicesResp.GetBillingSummary())
	}
	if parseSlicesResp.GetBillingSummary().GetOpenInvoiceCount() < 1 || parseSlicesResp.GetBillingSummary().GetDunningEventCount() < 1 {
		parseT.Fatalf("expected scoped billing invoice+dunning counts, got %+v", parseSlicesResp.GetBillingSummary())
	}
	if parseSlicesResp.GetBillingSummary().GetRecentUsageCostUsd() <= 0 {
		parseT.Fatalf("expected positive scoped usage cost in billing summary, got %+v", parseSlicesResp.GetBillingSummary())
	}
	parseAuditRows, parseErr := parseStore.parseListAuditLogs(50)
	if parseErr != nil {
		parseT.Fatalf("parseListAuditLogs workspace slices: %v", parseErr)
	}
	hasParseSliceView := false
	hasParseDrilldown := false
	for _, parseAuditRow := range parseAuditRows {
		if parseAuditRow.ActorUserID != parseBobAuth.ID {
			continue
		}
		if parseAuditRow.EventType == "admin.dashboard.slice.view" && parseAuditRow.TargetID == "workspace-admin" {
			hasParseSliceView = true
		}
		if parseAuditRow.EventType == "admin.dashboard.drilldown.access" && parseAuditRow.TargetID == "workspace-admin" {
			hasParseDrilldown = true
		}
	}
	if !hasParseSliceView || !hasParseDrilldown {
		parseT.Fatalf("expected workspace slice/drilldown audit rows for bob admin, got %+v", parseAuditRows)
	}
}

func TestGetSessionReturnsTypedRoleSummary(parseT *testing.T) {
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

	parseBobCtx := parseBindAuthUser(parseServer, "peer-session-bob", parseBobAuth.ID, parseBobAuth.Email)
	parseBobSession, parseErr := parseServer.GetSession(parseBobCtx, nil)
	if parseErr != nil {
		parseT.Fatalf("GetSession bob baseline: %v", parseErr)
	}
	if parseBobSession.GetRoleSummary() == nil || parseBobSession.GetRoleSummary().GetScope() != "user" || parseBobSession.GetRoleSummary().GetCanAccessAdmin() {
		parseT.Fatalf("expected bob baseline role summary=user, got %+v", parseBobSession.GetRoleSummary())
	}

	if parseErr = parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "ws-bob-session-scope",
		Slug:         "ws-bob-session-scope",
		Name:         "Bob Session Scope",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseBobAuth.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	parseWorkspaceID := int64(0)
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		if parseWorkspaceRow.WorkspaceKey != "ws-bob-session-scope" {
			continue
		}
		parseWorkspaceID = parseWorkspaceRow.ID
		break
	}
	if parseWorkspaceID <= 0 {
		parseT.Fatalf("expected workspace id for session role summary test, rows=%+v", parseWorkspaceRows)
	}
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseBobAuth.ID,
		RoleKey:         "admin",
		Status:          "active",
		InvitedByUserID: parseBobAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership: %v", parseErr)
	}

	parseBobWorkspaceSession, parseErr := parseServer.GetSession(parseBobCtx, nil)
	if parseErr != nil {
		parseT.Fatalf("GetSession bob workspace-admin: %v", parseErr)
	}
	if parseBobWorkspaceSession.GetRoleSummary() == nil || parseBobWorkspaceSession.GetRoleSummary().GetScope() != "workspace_admin" || !parseBobWorkspaceSession.GetRoleSummary().GetCanAccessAdmin() {
		parseT.Fatalf("expected bob workspace-admin role summary, got %+v", parseBobWorkspaceSession.GetRoleSummary())
	}
	if len(parseBobWorkspaceSession.GetRoleSummary().GetWorkspaceAdminWorkspaceIds()) != 1 || parseBobWorkspaceSession.GetRoleSummary().GetWorkspaceAdminWorkspaceIds()[0] != parseWorkspaceID {
		parseT.Fatalf("expected bob workspace id in role summary, got %+v", parseBobWorkspaceSession.GetRoleSummary().GetWorkspaceAdminWorkspaceIds())
	}

	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-session-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseAliceSession, parseErr := parseServer.GetSession(parseAliceCtx, nil)
	if parseErr != nil {
		parseT.Fatalf("GetSession alice superuser: %v", parseErr)
	}
	if parseAliceSession.GetRoleSummary() == nil || parseAliceSession.GetRoleSummary().GetScope() != "superuser" || !parseAliceSession.GetRoleSummary().GetCanAccessAdmin() || !parseAliceSession.GetRoleSummary().GetIsSuperuser() {
		parseT.Fatalf("expected alice superuser role summary, got %+v", parseAliceSession.GetRoleSummary())
	}
}

// TestAdminAccessInvalidatesCachedStateOnLogoutDowngradeAndRevocation verifies logout, role downgrade, and session revocation immediately remove privileged admin access.
func TestAdminAccessInvalidatesCachedStateOnLogoutDowngradeAndRevocation(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")

	parseAdminUser := parseMustCreateUser(parseT, parseStore, "admin-invalidation@example.com")
	parseNow := time.Now().Unix()
	if parseErr := parseStore.setUserName(parseAdminUser.ID, "Admin Invalidation", parseNow); parseErr != nil {
		parseT.Fatalf("setUserName admin: %v", parseErr)
	}

	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "ws-admin-invalidation",
		Slug:         "ws-admin-invalidation",
		Name:         "Admin Invalidation Scope",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseAdminUser.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	parseWorkspaceID := int64(0)
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		if parseWorkspaceRow.WorkspaceKey != "ws-admin-invalidation" {
			continue
		}
		parseWorkspaceID = parseWorkspaceRow.ID
		break
	}
	if parseWorkspaceID <= 0 {
		parseT.Fatalf("expected workspace id for invalidation test, rows=%+v", parseWorkspaceRows)
	}
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseAdminUser.ID,
		RoleKey:         "admin",
		Status:          "active",
		InvitedByUserID: parseAdminUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership active admin: %v", parseErr)
	}

	parseAdminToken, parseErr := parseServer.authManager.issueToken(authUser{ID: parseAdminUser.ID, Email: parseAdminUser.Email})
	if parseErr != nil {
		parseT.Fatalf("issueToken admin: %v", parseErr)
	}
	parseAdminTokenCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseAdminToken))

	parseInitialSession, parseErr := parseServer.GetSession(parseAdminTokenCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSession initial admin: %v", parseErr)
	}
	if parseInitialSession.GetRoleSummary() == nil || parseInitialSession.GetRoleSummary().GetScope() != "workspace_admin" || !parseInitialSession.GetRoleSummary().GetCanAccessAdmin() {
		parseT.Fatalf("expected initial workspace-admin role summary, got %+v", parseInitialSession.GetRoleSummary())
	}
	if parseInitialSession.GetSessionId() == "" {
		parseT.Fatalf("expected typed session id for revocation path, got %+v", parseInitialSession)
	}
	if _, parseErr = parseServer.ListAdminUsers(parseAdminTokenCtx, &chatpb.ListAdminUsersRequest{Limit: 5}); parseErr != nil {
		parseT.Fatalf("ListAdminUsers initial admin token: %v", parseErr)
	}
	if _, parseErr = parseServer.Logout(parseAdminTokenCtx, &emptypb.Empty{}); parseErr != nil {
		parseT.Fatalf("Logout admin token: %v", parseErr)
	}
	if _, parseErr = parseServer.GetSession(parseAdminTokenCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected logout to invalidate typed session, got %v", status.Code(parseErr))
	}
	if _, parseErr = parseServer.ListAdminUsers(parseAdminTokenCtx, &chatpb.ListAdminUsersRequest{Limit: 5}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected logout to invalidate privileged dashboard access, got %v", status.Code(parseErr))
	}

	parseReplacementToken, parseErr := parseServer.authManager.issueToken(authUser{ID: parseAdminUser.ID, Email: parseAdminUser.Email})
	if parseErr != nil {
		parseT.Fatalf("issueToken replacement admin: %v", parseErr)
	}
	parseReplacementTokenCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseReplacementToken))
	parseReplacementSession, parseErr := parseServer.GetSession(parseReplacementTokenCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSession replacement admin: %v", parseErr)
	}
	if parseReplacementSession.GetRoleSummary() == nil || parseReplacementSession.GetRoleSummary().GetScope() != "workspace_admin" || !parseReplacementSession.GetRoleSummary().GetCanAccessAdmin() {
		parseT.Fatalf("expected replacement session to preserve workspace-admin scope before downgrade, got %+v", parseReplacementSession.GetRoleSummary())
	}
	if parseReplacementSession.GetSessionId() == "" {
		parseT.Fatalf("expected replacement typed session id for revocation path, got %+v", parseReplacementSession)
	}
	parseAdminSessionID := parseReplacementSession.GetSessionId()

	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseAdminUser.ID,
		RoleKey:         "member",
		Status:          "active",
		InvitedByUserID: parseAdminUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership downgrade: %v", parseErr)
	}
	parseDowngradedSession, parseErr := parseServer.GetSession(parseReplacementTokenCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSession downgraded: %v", parseErr)
	}
	if parseDowngradedSession.GetRoleSummary() == nil || parseDowngradedSession.GetRoleSummary().GetScope() != "user" || parseDowngradedSession.GetRoleSummary().GetCanAccessAdmin() {
		parseT.Fatalf("expected downgraded role summary to lose admin scope, got %+v", parseDowngradedSession.GetRoleSummary())
	}
	if _, parseErr = parseServer.ListAdminUsers(parseReplacementTokenCtx, &chatpb.ListAdminUsersRequest{Limit: 5}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected downgraded admin list to be denied, got %v", status.Code(parseErr))
	}

	if parseErr = parseStore.parseRevokeAuthSession(parseAdminSessionID); parseErr != nil {
		parseT.Fatalf("parseRevokeAuthSession: %v", parseErr)
	}
	if _, parseErr = parseServer.GetSession(parseReplacementTokenCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected revoked session GetSession to return unauthenticated, got %v", status.Code(parseErr))
	}
	if _, parseErr = parseServer.ListAdminUsers(parseReplacementTokenCtx, &chatpb.ListAdminUsersRequest{Limit: 5}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected revoked session admin list to return unauthenticated, got %v", status.Code(parseErr))
	}
}
