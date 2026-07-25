package app

import (
	"context"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseSeedSuperuserControlPlaneData inserts a compact superuser control-plane dataset for tests.
func parseSeedSuperuserControlPlaneData(parseT *testing.T, parseStore *Store) authUser {
	parseT.Helper()

	parseOwner := parseMustCreateUser(parseT, parseStore, "su-owner@example.com")
	parseMember := parseMustCreateUser(parseT, parseStore, "workspace-member@example.com")

	if parseErr := parseStore.parseUpsertSURole(parseSURoleWrite{
		RoleKey:     "su",
		Label:       "Superuser",
		Description: "Root control plane access",
		IsSystem:    true,
		IsEnabled:   true,
		Permissions: []parseSURolePermissionRow{
			{PermissionKey: "control_plane.*", PermissionValue: "allow"},
		},
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSURole: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSUUserRole(parseOwner.ID, "su", parseOwner.ID); parseErr != nil {
		parseT.Fatalf("parseUpsertSUUserRole: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey:       "brand.name",
		ConfigValue:     "RelayDesk",
		ValueType:       "string",
		Description:     "Primary marketing brand",
		UpdatedByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSiteConfig: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertFeatureFlag(parseFeatureFlagWrite{
		FlagKey:         "new-pricing-page",
		Description:     "Roll out the new pricing page",
		IsEnabled:       true,
		RolloutPercent:  25,
		AudienceJSON:    `{"plan":["pro","team"]}`,
		PayloadJSON:     `{"variant":"v2"}`,
		UpdatedByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertFeatureFlag: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "acme",
		Slug:         "acme",
		Name:         "Acme Workspace",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseOwner.ID,
		SettingsJSON: `{"region":"us"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace: %v", parseErr)
	}
	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	if len(parseWorkspaces) == 0 {
		parseT.Fatal("expected seeded workspace")
	}
	parseWorkspaceID := parseWorkspaces[0].ID

	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseMember.ID,
		RoleKey:         "member",
		Status:          "active",
		InvitedByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       "key_acme_1",
		WorkspaceID: parseWorkspaceID,
		UserID:      parseOwner.ID,
		Label:       "CLI",
		KeyPrefix:   "gwc_live",
		SecretHash:  "hash-1",
		ScopesJSON:  `["workspace.read","workspace.write"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID:  parseWorkspaceID,
		Label:        "Billing Sink",
		TargetURL:    "https://example.com/hooks/billing",
		SecretHash:   "hook-secret",
		EventsJSON:   `["invoice.paid","invoice.failed"]`,
		IsEnabled:    true,
		FailureCount: 0,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseOwner.ID,
		WorkspaceID: parseWorkspaceID,
		EventType:   "workspace.created",
		TargetType:  "workspace",
		TargetID:    "acme",
		Summary:     "Created workspace",
		PayloadJSON: `{"source":"test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-1001",
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseMember.ID,
		Status:         "open",
		Priority:       "high",
		Subject:        "Billing issue",
		Body:           "Need invoice copy",
		AssigneeUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertExperiment(parseExperimentWrite{
		ExperimentKey: "pricing-copy-v2",
		Name:          "Pricing Copy V2",
		Status:        "running",
		VariantsJSON:  `[{"id":"control"},{"id":"v2"}]`,
		AudienceJSON:  `{"country":["US"]}`,
		StartAt:       "2026-01-01T00:00:00Z",
		EndAt:         "2026-12-31T23:59:59Z",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertExperiment: %v", parseErr)
	}

	return parseOwner
}

// parseSeedSuperuserOperationalRows inserts operational-table rows used by lifecycle and RPC snapshot tests.
func parseSeedSuperuserOperationalRows(parseT *testing.T, parseStore *Store, parseOwner authUser) {
	parseT.Helper()

	parseNowTime := time.Date(2026, time.March, 27, 20, 0, 0, 0, time.UTC)
	parseNow := parseNowTime.Format(time.RFC3339)

	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil || len(parseWorkspaces) == 0 {
		parseT.Fatalf("parseListWorkspaces seed lookup: rows=%+v err=%v", parseWorkspaces, parseErr)
	}
	parseWorkspaceID := parseWorkspaces[0].ID
	parseWebhooks, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil || len(parseWebhooks) == 0 {
		parseT.Fatalf("parseListWebhookEndpoints seed lookup: rows=%+v err=%v", parseWebhooks, parseErr)
	}
	parseWebhookEndpointID := parseWebhooks[0].ID
	parseTickets, parseErr := parseStore.parseListSupportTickets(10)
	if parseErr != nil || len(parseTickets) == 0 {
		parseT.Fatalf("parseListSupportTickets seed lookup: rows=%+v err=%v", parseTickets, parseErr)
	}
	parseTicketID := parseTickets[0].ID

	if parseErr = parseStore.parseUpsertAuthSession(parseAuthSessionWrite{
		UserID:           parseOwner.ID,
		SessionID:        "session-su-owner",
		TokenVersion:     1,
		RefreshTokenHash: "",
		UserAgent:        "relaydesk-test",
		IPAddress:        "127.0.0.1",
		LastSeenAt:       parseNow,
		ExpiresAt:        "2026-12-31T23:59:59Z",
		RevokedAt:        "",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertAuthSession: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspaceInvitation(parseWorkspaceInvitationWrite{
		WorkspaceID:         parseWorkspaceID,
		Email:               "invitee@example.com",
		RoleKey:             "member",
		InvitationTokenHash: "invite-token-hash",
		InvitedByUserID:     parseOwner.ID,
		Status:              "pending",
		ExpiresAt:           "2026-12-31T23:59:59Z",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceInvitation: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookDelivery(context.Background(), parseWebhookDeliveryWrite{
		EndpointID:         parseWebhookEndpointID,
		EventType:          "invoice.paid",
		DeliveryKey:        "delivery-001",
		RequestHeadersJSON: `{"x-signature":"abc"}`,
		RequestBodyJSON:    `{"invoice":"paid"}`,
		ResponseStatus:     202,
		ResponseBody:       "accepted",
		AttemptCount:       1,
		DeliveredAt:        parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookDelivery: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWebhookDelivery(context.Background(), parseWebhookDeliveryWrite{
		EndpointID:         parseWebhookEndpointID,
		EventType:          "invoice.failed",
		DeliveryKey:        "delivery-retry-001",
		RequestHeadersJSON: `{"x-signature":"def"}`,
		RequestBodyJSON:    `{"invoice":"failed"}`,
		ResponseStatus:     500,
		ResponseBody:       "upstream unavailable",
		AttemptCount:       2,
		FailedAt:           parseNow,
		NextRetryAt:        parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookDelivery retry row: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateSupportTicketMessage(parseSupportTicketMessageWrite{
		TicketID:     parseTicketID,
		AuthorUserID: parseOwner.ID,
		MessageType:  "reply",
		Body:         "Working on it",
		IsInternal:   false,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateSupportTicketMessage: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO service_level_objectives (slo_key, service_name, objective_percent, window_days, error_budget_minutes, status_page_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"slo-api-latency",
		"api",
		99.9,
		30,
		0,
		"",
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed service level objective: %v", parseErr)
	}
	parseIncidentResult, parseErr := parseStore.db.Exec(
		`INSERT INTO incidents (incident_key, slo_key, severity, status, title, summary, started_at, resolved_at, postmortem_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"incident-001",
		"slo-api-latency",
		"major",
		"open",
		"Primary API degraded",
		"latency spike",
		parseNow,
		"",
		"",
		parseNow,
	)
	if parseErr != nil {
		parseT.Fatalf("seed incident: %v", parseErr)
	}
	parseIncidentID, parseErr := parseIncidentResult.LastInsertId()
	if parseErr != nil {
		parseT.Fatalf("seed incident id: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateIncidentUpdate(parseIncidentUpdateWrite{
		IncidentID:      parseIncidentID,
		Status:          "investigating",
		Message:         "Incident declared",
		IsPublic:        true,
		PublishedAt:     parseNow,
		CreatedByUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateIncidentUpdate: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateNotificationOutbox(context.Background(), parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseOwner.ID,
		NotificationKey: "weekly-value-ready",
		ChannelKey:      "email",
		TemplateKey:     "weekly_value_summary",
		Status:          "pending",
		Subject:         "Your weekly value summary",
		BodyText:        "Summary ready.",
		PayloadJSON:     `{"week":"2026-W13"}`,
		DedupeKey:       "weekly-value-2026-W13",
		ScheduledAt:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
		JobKey:       "weekly-summary-job",
		JobType:      "weekly-summary",
		QueueKey:     "default",
		Status:       "pending",
		AttemptCount: 0,
		MaxAttempts:  3,
		PayloadJSON:  `{"workspace_id":1}`,
		RunAfter:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertBackgroundJob: %v", parseErr)
	}
}

// parseSeedSuperuserGrowthRows inserts onboarding/workflow/analytics/churn rows for growth lifecycle coverage.
func parseSeedSuperuserGrowthRows(parseT *testing.T, parseStore *Store, parseOwner authUser) {
	parseT.Helper()

	parseNow := "2026-03-27T21:00:00Z"
	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil || len(parseWorkspaces) == 0 {
		parseT.Fatalf("parseListWorkspaces growth lookup: rows=%+v err=%v", parseWorkspaces, parseErr)
	}
	parseWorkspaceID := parseWorkspaces[0].ID

	if parseErr = parseStore.parseUpsertOnboardingTemplate(parseOnboardingTemplateWrite{
		TemplateKey:   "welcome-first-chat",
		Title:         "Welcome First Chat",
		Category:      "starter",
		PromptText:    "Help me draft a weekly status.",
		ChecklistJSON: `["open-composer","send-first-message"]`,
		IsDefault:     true,
		SortOrder:     1,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertOnboardingTemplate: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertUserActivationMilestone(parseUserActivationMilestoneWrite{
		UserID:       parseOwner.ID,
		MilestoneKey: "first-send-complete",
		Status:       "done",
		AchievedAt:   parseNow,
		MetadataJSON: `{"source":"test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserActivationMilestone: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertSavedWorkflow(parseSavedWorkflowWrite{
		WorkspaceID:  parseWorkspaceID,
		UserID:       parseOwner.ID,
		WorkflowKey:  "weekly-update",
		Name:         "Weekly Update",
		Description:  "Summarize weekly accomplishments.",
		WorkflowJSON: `{"steps":[{"id":"draft"}]}`,
		IsPublic:     true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSavedWorkflow: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertPromptLibraryItem(parsePromptLibraryItemWrite{
		WorkspaceID: parseWorkspaceID,
		UserID:      parseOwner.ID,
		ItemKey:     "status-draft",
		Title:       "Status Draft",
		Category:    "ops",
		PromptText:  "Draft a concise status update.",
		TagsJSON:    `["status","team"]`,
		IsPublic:    true,
		UseCount:    3,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertPromptLibraryItem: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWeeklyValueSummary(parseWeeklyValueSummaryWrite{
		WorkspaceID: parseWorkspaceID,
		UserID:      parseOwner.ID,
		SummaryWeek: "2026-W13",
		SummaryText: "Delivered launch milestones.",
		MetricsJSON: `{"wins":3}`,
		SentAt:      parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWeeklyValueSummary: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateProductAnalyticsEvent(parseProductAnalyticsEventWrite{
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseOwner.ID,
		SessionKey:     "session-growth",
		EventName:      "first_reply_completed",
		FunnelKey:      "visit-to-first-chat",
		StepKey:        "reply-complete",
		ExperimentKey:  "pricing-copy-v2",
		VariantKey:     "v2",
		EventPropsJSON: `{"latency_ms":420}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateProductAnalyticsEvent: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertExperimentAssignment(parseExperimentAssignmentWrite{
		ExperimentKey: "pricing-copy-v2",
		WorkspaceID:   parseWorkspaceID,
		UserID:        parseOwner.ID,
		VariantKey:    "v2",
		AssignedAt:    parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertExperimentAssignment: %v", parseErr)
	}

	parseMustAssignBillingPlan(parseT, parseStore, parseOwner.ID, "pro")
	parseCustomer, isParseCustomerFound, parseErr := parseStore.parseGetBillingCustomerByUser(parseOwner.ID)
	if parseErr != nil || !isParseCustomerFound {
		parseT.Fatalf("parseGetBillingCustomerByUser: found=%v row=%+v err=%v", isParseCustomerFound, parseCustomer, parseErr)
	}
	parseSubscriptions, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 10)
	if parseErr != nil || len(parseSubscriptions) == 0 {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: rows=%+v err=%v", parseSubscriptions, parseErr)
	}
	if _, parseErr = parseStore.parseCreateSubscriptionChurnFeedback(parseSubscriptionChurnFeedbackWrite{
		CustomerID:       parseCustomer.ID,
		SubscriptionID:   parseSubscriptions[0].ID,
		WorkspaceID:      parseWorkspaceID,
		ReasonKey:        "budget",
		Detail:           "Need to reduce spend.",
		RecoveryOfferKey: "discount-20",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateSubscriptionChurnFeedback: %v", parseErr)
	}
}

// parseSeedSuperuserSliceRows inserts pricing-control, cost-guardrail, and usage rows for superuser slice RPC coverage.
func parseSeedSuperuserSliceRows(parseT *testing.T, parseStore *Store, parseOwner authUser) {
	parseT.Helper()

	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil || len(parseWorkspaces) == 0 {
		parseT.Fatalf("parseListWorkspaces slice lookup: rows=%+v err=%v", parseWorkspaces, parseErr)
	}
	parseWorkspaceID := parseWorkspaces[0].ID

	parseConversationID, parseErr := parseStore.parseCreateConversation(parseOwner.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation slice seed: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveConversationMessage(parseOwner.ID, parseConversationID, "user", "Need superuser usage telemetry.", "", 0, 0); parseErr != nil {
		parseT.Fatalf("parseSaveConversationMessage slice seed: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:                 "evt-su-slice-usage-1",
		UserID:                  parseOwner.ID,
		ConversationID:          parseConversationID,
		ProviderID:              "fake",
		ModelID:                 modelGPT54Mini,
		PromptTokens:            30,
		CompletionTokens:        12,
		UsageSource:             "exact",
		ProviderRequestID:       "req-su-slice-usage-1",
		InputCostPerMillionUSD:  0.25,
		OutputCostPerMillionUSD: 2.00,
		PricingCurrency:         "USD",
		InputCostUSD:            0.10,
		OutputCostUSD:           0.20,
		TotalCostUSD:            0.30,
		ClientID:                "client-su-slice",
		Status:                  "completed",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent slice seed: %v", parseErr)
	}

	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO billing_plan_overages (plan_code, meter_key, included_units, soft_limit_units, hard_limit_units, overage_unit_size, overage_price_cents, billing_interval, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"team",
		"usage.tokens.monthly",
		100000,
		120000,
		150000,
		1000,
		2,
		"monthly",
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed billing_plan_overages: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO billing_quota_policies (plan_code, quota_key, soft_limit_value, hard_limit_value, reset_interval, enforcement_mode, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"team",
		"usage.requests.per_minute",
		500,
		750,
		"monthly",
		"block",
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed billing_quota_policies: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO billing_upgrade_triggers (plan_code, trigger_key, threshold_percent, upgrade_plan_code, message, cta_label, cta_url, is_enabled, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"team",
		"usage.tokens.threshold",
		85,
		"pro",
		"Approaching quota limits",
		"Upgrade plan",
		"/app/settings?panel=settings-billing",
		1,
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed billing_upgrade_triggers: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO workspace_cost_guardrails (workspace_id, guardrail_key, daily_budget_cents, monthly_budget_cents, max_cost_per_request_cents, alert_threshold_percent, action_mode, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		parseWorkspaceID,
		"default",
		2500,
		75000,
		500,
		80,
		"notify",
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed workspace_cost_guardrails: %v", parseErr)
	}
}

// TestStoreSuperuserControlPlaneLifecycle verifies the superuser store lifecycle across all new models.
func TestStoreSuperuserControlPlaneLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseSeedSuperuserOperationalRows(parseT, parseStore, parseOwner)
	parseSeedSuperuserGrowthRows(parseT, parseStore, parseOwner)

	if parseAllowed, parseErr := parseStore.parseUserHasSURole(parseOwner.ID); parseErr != nil || !parseAllowed {
		parseT.Fatalf("parseUserHasSURole: allowed=%v err=%v", parseAllowed, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSURoles(); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSURoles: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSURolePermissions(); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSURolePermissions: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSUUserRoles(); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSUUserRoles: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSiteConfigs(); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSiteConfigs: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListFeatureFlags(); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListFeatureFlags: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListWorkspaces(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListWorkspaces: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListWorkspaceMemberships(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListWorkspaceMemberships: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListAuthSessions(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListAuthSessions: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListWorkspaceInvitations(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListWorkspaceInvitations: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListAPIKeys(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListAPIKeys: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListWebhookEndpoints(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListWebhookEndpoints: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListWebhookDeliveries(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListWebhookDeliveries: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListWebhookDeliveriesPendingRetry("2026-03-27T20:00:00Z", 10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListWebhookDeliveriesPendingRetry initial: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseErr := parseStore.parseUpdateWebhookDeliveryAttempt(
		"delivery-retry-001",
		502,
		"gateway timeout",
		3,
		"2026-03-27T20:01:00Z",
		"2026-03-27T20:02:00Z",
	); parseErr != nil {
		parseT.Fatalf("parseUpdateWebhookDeliveryAttempt: %v", parseErr)
	}
	if parseErr := parseStore.parseUpdateWebhookDeliveryDelivered(
		"delivery-retry-001",
		200,
		"ok",
		4,
		"2026-03-27T20:03:00Z",
	); parseErr != nil {
		parseT.Fatalf("parseUpdateWebhookDeliveryDelivered: %v", parseErr)
	}
	if parseRows, parseErr := parseStore.parseListWebhookDeliveriesPendingRetry("2026-03-27T20:03:00Z", 10); parseErr != nil || len(parseRows) != 0 {
		parseT.Fatalf("parseListWebhookDeliveriesPendingRetry cleared: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListAuditLogs(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListAuditLogs: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSupportTickets(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSupportTickets: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSupportTicketMessages(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSupportTicketMessages: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListIncidentUpdates(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListIncidentUpdates: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListNotificationOutbox(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListNotificationOutbox: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListBackgroundJobs(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListBackgroundJobs: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListOnboardingTemplates(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListOnboardingTemplates: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListUserActivationMilestones(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListUserActivationMilestones: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSavedWorkflows(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSavedWorkflows: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListPromptLibraryItems(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListPromptLibraryItems: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListWeeklyValueSummaries(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListWeeklyValueSummaries: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListProductAnalyticsEvents(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListProductAnalyticsEvents: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListProductAnalyticsEventsByUser(parseOwner.ID, 10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListProductAnalyticsEventsByUser: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListExperimentAssignments(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListExperimentAssignments: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSubscriptionChurnFeedback(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSubscriptionChurnFeedback: rows=%+v err=%v", parseRows, parseErr)
	}
	parseCustomer, isParseCustomerFound, parseErr := parseStore.parseGetBillingCustomerByUser(parseOwner.ID)
	if parseErr != nil || !isParseCustomerFound {
		parseT.Fatalf("parseGetBillingCustomerByUser superuser growth list: found=%v row=%+v err=%v", isParseCustomerFound, parseCustomer, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListSubscriptionChurnFeedbackByCustomer(parseCustomer.ID, 10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListSubscriptionChurnFeedbackByCustomer: rows=%+v err=%v", parseRows, parseErr)
	}
	if parseRows, parseErr := parseStore.parseListExperiments(10); parseErr != nil || len(parseRows) == 0 {
		parseT.Fatalf("parseListExperiments: rows=%+v err=%v", parseRows, parseErr)
	}
}

// TestGetSuperuserControlPlaneRequiresSURole verifies the snapshot endpoint is gated by the su role.
func TestGetSuperuserControlPlaneRequiresSURole(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "plain-user@example.com")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-superuser-denied", parseUser.ID, parseUser.Email)

	_, parseErr := parseServer.GetSuperuserControlPlane(parseCtx, &chatpb.GetSuperuserControlPlaneRequest{})
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected permission denied, got %v", status.Code(parseErr))
	}
}

// TestGetSuperuserSlicesRequiresSURole verifies the slice endpoint is gated by the su role.
func TestGetSuperuserSlicesRequiresSURole(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "plain-user-slices@example.com")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-superuser-slices-denied", parseUser.ID, parseUser.Email)

	_, parseErr := parseServer.GetSuperuserSlices(parseCtx, &chatpb.GetSuperuserSlicesRequest{})
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected permission denied, got %v", status.Code(parseErr))
	}
}

// TestGetSuperuserControlPlaneReturnsSnapshot verifies the superuser snapshot endpoint returns seeded control-plane state.
func TestGetSuperuserControlPlaneReturnsSnapshot(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-superuser-allowed", parseOwner.ID, parseOwner.Email)

	parseResp, parseErr := parseServer.GetSuperuserControlPlane(context.Background(), &chatpb.GetSuperuserControlPlaneRequest{Limit: 20})
	if parseErr == nil {
		parseT.Fatal("expected unauthenticated call without bound peer context to fail")
	}

	parseResp, parseErr = parseServer.GetSuperuserControlPlane(parseCtx, &chatpb.GetSuperuserControlPlaneRequest{Limit: 20})
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserControlPlane: %v", parseErr)
	}
	if len(parseResp.GetRoles()) == 0 || len(parseResp.GetUserRoles()) == 0 || len(parseResp.GetSiteConfigs()) == 0 {
		parseT.Fatalf("unexpected superuser snapshot core rows: %+v", parseResp)
	}
	if len(parseResp.GetWorkspaces()) == 0 || len(parseResp.GetMemberships()) == 0 || len(parseResp.GetApiKeys()) == 0 {
		parseT.Fatalf("unexpected superuser snapshot workspace rows: %+v", parseResp)
	}
	if len(parseResp.GetWebhookEndpoints()) == 0 || len(parseResp.GetAuditLogs()) == 0 || len(parseResp.GetSupportTickets()) == 0 || len(parseResp.GetExperiments()) == 0 {
		parseT.Fatalf("unexpected superuser snapshot operational rows: %+v", parseResp)
	}
}

// TestGetSuperuserControlPlaneReturnsExtendedOperationalSnapshot verifies the snapshot returns newly wired operational rows.
func TestGetSuperuserControlPlaneReturnsExtendedOperationalSnapshot(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseSeedSuperuserOperationalRows(parseT, parseStore, parseOwner)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-superuser-extended", parseOwner.ID, parseOwner.Email)

	parseResp, parseErr := parseServer.GetSuperuserControlPlane(parseCtx, &chatpb.GetSuperuserControlPlaneRequest{Limit: 20})
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserControlPlane extended: %v", parseErr)
	}
	if len(parseResp.GetAuthSessions()) == 0 || len(parseResp.GetWorkspaceInvitations()) == 0 {
		parseT.Fatalf("expected auth session and invitation rows, got %+v", parseResp)
	}
	if len(parseResp.GetWebhookDeliveries()) == 0 || len(parseResp.GetSupportTicketMessages()) == 0 {
		parseT.Fatalf("expected webhook delivery and support message rows, got %+v", parseResp)
	}
	if len(parseResp.GetIncidentUpdates()) == 0 || len(parseResp.GetNotificationOutbox()) == 0 || len(parseResp.GetBackgroundJobs()) == 0 {
		parseT.Fatalf("expected incident/notification/job rows, got %+v", parseResp)
	}
}

// TestGetSuperuserSlicesReturnsSnapshot verifies the superuser slice endpoint returns typed global slices.
func TestGetSuperuserSlicesReturnsSnapshot(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseSeedSuperuserOperationalRows(parseT, parseStore, parseOwner)
	parseSeedSuperuserSliceRows(parseT, parseStore, parseOwner)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-superuser-slices-allowed", parseOwner.ID, parseOwner.Email)

	parseResp, parseErr := parseServer.GetSuperuserSlices(parseCtx, &chatpb.GetSuperuserSlicesRequest{
		LookbackDays: 30,
		Limit:        25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices: %v", parseErr)
	}
	if len(parseResp.GetUsers()) == 0 || len(parseResp.GetUsageEvents()) == 0 || len(parseResp.GetSupportTickets()) == 0 {
		parseT.Fatalf("expected user/usage/support slices, got %+v", parseResp)
	}
	if len(parseResp.GetBillingPlanOverages()) == 0 || len(parseResp.GetBillingQuotaPolicies()) == 0 || len(parseResp.GetBillingUpgradeTriggers()) == 0 {
		parseT.Fatalf("expected pricing-control slices, got %+v", parseResp)
	}
	if len(parseResp.GetIncidents()) == 0 || len(parseResp.GetExperiments()) == 0 || len(parseResp.GetWorkspaces()) == 0 || len(parseResp.GetWorkspaceCostGuardrails()) == 0 {
		parseT.Fatalf("expected incident/experiment/workspace/guardrail slices, got %+v", parseResp)
	}
	parseAuditRows, parseErr := parseStore.parseListAuditLogs(50)
	if parseErr != nil {
		parseT.Fatalf("parseListAuditLogs superuser slices: %v", parseErr)
	}
	hasParseSliceView := false
	hasParseDrilldown := false
	for _, parseAuditRow := range parseAuditRows {
		if parseAuditRow.ActorUserID != parseOwner.ID {
			continue
		}
		if parseAuditRow.EventType == "admin.dashboard.slice.view" && parseAuditRow.TargetID == "superuser" {
			hasParseSliceView = true
		}
		if parseAuditRow.EventType == "admin.dashboard.drilldown.access" && parseAuditRow.TargetID == "superuser" {
			hasParseDrilldown = true
		}
	}
	if !hasParseSliceView || !hasParseDrilldown {
		parseT.Fatalf("expected superuser slice/drilldown audit rows, got %+v", parseAuditRows)
	}
}

// TestGetSuperuserSlicesAppliesListQueries verifies workspace, support, and incident slice list queries are applied.
func TestGetSuperuserSlicesAppliesListQueries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseSeedSuperuserOperationalRows(parseT, parseStore, parseOwner)
	parseSeedSuperuserSliceRows(parseT, parseStore, parseOwner)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-superuser-slices-list-query", parseOwner.ID, parseOwner.Email)

	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "alpha-space",
		Slug:         "alpha-space",
		Name:         "Alpha Space",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseOwner.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace alpha-space: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "beta-space",
		Slug:         "beta-space",
		Name:         "Beta Space",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseOwner.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace beta-space: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "zeta-space",
		Slug:         "zeta-space",
		Name:         "Zeta Space",
		PlanCode:     "team",
		Status:       "suspended",
		OwnerUserID:  parseOwner.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace zeta-space: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseStore.parseListWorkspaces(25)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces list-query seed lookup: %v", parseErr)
	}
	parseAlphaWorkspaceID := parseFindSuperuserWorkspaceIDByKey(parseWorkspaceRows, "alpha-space")
	parseBetaWorkspaceID := parseFindSuperuserWorkspaceIDByKey(parseWorkspaceRows, "beta-space")
	if parseAlphaWorkspaceID <= 0 || parseBetaWorkspaceID <= 0 {
		parseT.Fatalf("expected seeded alpha/beta workspace ids, rows=%+v", parseWorkspaceRows)
	}

	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-space-open-a",
		WorkspaceID:    parseAlphaWorkspaceID,
		UserID:         parseOwner.ID,
		Status:         "open",
		Priority:       "high",
		Subject:        "Open issue A",
		Body:           "Support queue seed A.",
		AssigneeUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket ticket-space-open-a: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-space-open-b",
		WorkspaceID:    parseBetaWorkspaceID,
		UserID:         parseOwner.ID,
		Status:         "open",
		Priority:       "high",
		Subject:        "Open issue B",
		Body:           "Support queue seed B.",
		AssigneeUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket ticket-space-open-b: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-space-resolved",
		WorkspaceID:    parseAlphaWorkspaceID,
		UserID:         parseOwner.ID,
		Status:         "resolved",
		Priority:       "normal",
		Subject:        "Resolved issue",
		Body:           "Support queue resolved seed.",
		AssigneeUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket ticket-space-resolved: %v", parseErr)
	}

	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO incidents (incident_key, slo_key, severity, status, title, summary, started_at, resolved_at, postmortem_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"incident-space-open-a",
		"slo-api-latency",
		"major",
		"open",
		"Open incident A",
		"seed open A",
		"2026-03-28T00:00:00Z",
		"",
		"",
		"2026-03-28T00:00:00Z",
	); parseErr != nil {
		parseT.Fatalf("seed incident-space-open-a: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO incidents (incident_key, slo_key, severity, status, title, summary, started_at, resolved_at, postmortem_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"incident-space-open-b",
		"slo-api-latency",
		"major",
		"open",
		"Open incident B",
		"seed open B",
		"2026-03-28T00:01:00Z",
		"",
		"",
		"2026-03-28T00:01:00Z",
	); parseErr != nil {
		parseT.Fatalf("seed incident-space-open-b: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO incidents (incident_key, slo_key, severity, status, title, summary, started_at, resolved_at, postmortem_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"incident-space-resolved",
		"slo-api-latency",
		"minor",
		"resolved",
		"Resolved incident",
		"seed resolved",
		"2026-03-28T00:02:00Z",
		"2026-03-28T00:03:00Z",
		"",
		"2026-03-28T00:03:00Z",
	); parseErr != nil {
		parseT.Fatalf("seed incident-space-resolved: %v", parseErr)
	}

	parseResp, parseErr := parseServer.GetSuperuserSlices(parseCtx, &chatpb.GetSuperuserSlicesRequest{
		LookbackDays:    30,
		Limit:           25,
		WorkspaceStatus: "active",
		SupportStatus:   "open",
		IncidentStatus:  "open",
		WorkspaceListQuery: &chatpb.AdminListQuery{
			Search:        "-space",
			SortBy:        "slug",
			SortDirection: "asc",
			Limit:         1,
			Offset:        1,
		},
		SupportListQuery: &chatpb.AdminListQuery{
			Search:        "ticket-space-open",
			SortBy:        "ticket_key",
			SortDirection: "asc",
			Limit:         1,
			Offset:        1,
		},
		IncidentListQuery: &chatpb.AdminListQuery{
			Search:        "incident-space-open",
			SortBy:        "incident_key",
			SortDirection: "asc",
			Limit:         1,
			Offset:        1,
		},
	})
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices list query: %v", parseErr)
	}
	if len(parseResp.GetWorkspaces()) != 1 || parseResp.GetWorkspaces()[0].GetSlug() != "beta-space" {
		parseT.Fatalf("expected one paged active workspace row for beta-space, got %+v", parseResp.GetWorkspaces())
	}
	if len(parseResp.GetSupportTickets()) != 1 || parseResp.GetSupportTickets()[0].GetTicketKey() != "ticket-space-open-b" {
		parseT.Fatalf("expected one paged open support ticket row for ticket-space-open-b, got %+v", parseResp.GetSupportTickets())
	}
	if len(parseResp.GetIncidents()) != 1 || parseResp.GetIncidents()[0].GetIncidentKey() != "incident-space-open-b" {
		parseT.Fatalf("expected one paged open incident row for incident-space-open-b, got %+v", parseResp.GetIncidents())
	}
}

// TestGetSuperuserSlicesAppliesTypedListQueries verifies workspace, support, and incident list-query filtering on superuser slices.
func TestGetSuperuserSlicesAppliesTypedListQueries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseSeedSuperuserOperationalRows(parseT, parseStore, parseOwner)
	parseSeedSuperuserSliceRows(parseT, parseStore, parseOwner)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-superuser-slices-list-query", parseOwner.ID, parseOwner.Email)

	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil || len(parseWorkspaces) == 0 {
		parseT.Fatalf("parseListWorkspaces seed lookup: rows=%+v err=%v", parseWorkspaces, parseErr)
	}
	parseBaseWorkspaceID := parseWorkspaces[0].ID
	parseNow := time.Now().UTC().Format(time.RFC3339)

	if parseErr = parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "beta",
		Slug:         "beta",
		Name:         "Beta Workspace",
		PlanCode:     "pro",
		Status:       "active",
		OwnerUserID:  parseOwner.ID,
		SettingsJSON: `{"region":"us"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace beta: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "gamma",
		Slug:         "gamma",
		Name:         "Gamma Workspace",
		PlanCode:     "pro",
		Status:       "suspended",
		OwnerUserID:  parseOwner.ID,
		SettingsJSON: `{"region":"us"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace gamma: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-1002",
		WorkspaceID:    parseBaseWorkspaceID,
		UserID:         parseOwner.ID,
		Status:         "open",
		Priority:       "normal",
		Subject:        "API usage question",
		Body:           "Need throughput guidance.",
		AssigneeUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket ticket-1002: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-closed",
		WorkspaceID:    parseBaseWorkspaceID,
		UserID:         parseOwner.ID,
		Status:         "closed",
		Priority:       "low",
		Subject:        "Closed billing follow-up",
		Body:           "Resolved.",
		AssigneeUserID: parseOwner.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket ticket-closed: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO incidents (incident_key, slo_key, severity, status, title, summary, started_at, resolved_at, postmortem_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"incident-002",
		"slo-api-latency",
		"minor",
		"open",
		"API latency regression",
		"regional API slowdown",
		parseNow,
		"",
		"",
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed incident-002: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO incidents (incident_key, slo_key, severity, status, title, summary, started_at, resolved_at, postmortem_url, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"incident-099",
		"slo-api-latency",
		"minor",
		"resolved",
		"API incident resolved",
		"resolved incident",
		parseNow,
		parseNow,
		"",
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed incident-099: %v", parseErr)
	}

	parseResp, parseErr := parseServer.GetSuperuserSlices(parseCtx, &chatpb.GetSuperuserSlicesRequest{
		LookbackDays:    30,
		Limit:           50,
		WorkspaceStatus: "active",
		WorkspaceListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "workspace",
			SortBy:        "name",
			SortDirection: "asc",
		},
		SupportStatus: "open",
		SupportListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			SortBy:        "ticket_key",
			SortDirection: "asc",
		},
		IncidentStatus: "open",
		IncidentListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "api",
			SortBy:        "incident_key",
			SortDirection: "asc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices list query: %v", parseErr)
	}
	if len(parseResp.GetWorkspaces()) != 1 || parseResp.GetWorkspaces()[0].GetName() != "Beta Workspace" {
		parseT.Fatalf("unexpected workspace list-query rows: %+v", parseResp.GetWorkspaces())
	}
	if len(parseResp.GetSupportTickets()) != 1 || parseResp.GetSupportTickets()[0].GetTicketKey() != "ticket-1002" {
		parseT.Fatalf("unexpected support list-query rows: %+v", parseResp.GetSupportTickets())
	}
	if len(parseResp.GetIncidents()) != 1 || parseResp.GetIncidents()[0].GetIncidentKey() != "incident-002" {
		parseT.Fatalf("unexpected incident list-query rows: %+v", parseResp.GetIncidents())
	}
}

// parseFindSuperuserWorkspaceIDByKey resolves one workspace id by workspace key.
func parseFindSuperuserWorkspaceIDByKey(parseRows []parseWorkspaceRow, parseWorkspaceKey string) int64 {
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceKey == parseWorkspaceKey {
			return parseRow.ID
		}
	}
	return 0
}

// BenchmarkParseFilterSuperuserWorkspaceRows reports micro-benchmark throughput for superuser workspace list filtering.
func BenchmarkParseFilterSuperuserWorkspaceRows(parseB *testing.B) {
	parseRows := []parseWorkspaceRow{
		{ID: 1, WorkspaceKey: "alpha-space", Slug: "alpha-space", Name: "Alpha Space", PlanCode: "team", Status: "active", OwnerUserID: 10, SettingsJSON: "{}"},
		{ID: 2, WorkspaceKey: "beta-space", Slug: "beta-space", Name: "Beta Space", PlanCode: "team", Status: "active", OwnerUserID: 10, SettingsJSON: "{}"},
		{ID: 3, WorkspaceKey: "zeta-space", Slug: "zeta-space", Name: "Zeta Space", PlanCode: "team", Status: "suspended", OwnerUserID: 10, SettingsJSON: "{}"},
	}
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseFilterSuperuserWorkspaceRows(parseRows, "active", "space")
	}
}
