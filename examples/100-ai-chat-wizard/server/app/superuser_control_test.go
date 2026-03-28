package app

import (
	"context"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
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
	if parseErr = parseStore.parseUpsertWebhookDelivery(parseWebhookDeliveryWrite{
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
	if _, parseErr = parseStore.parseCreateNotificationOutbox(parseNotificationOutboxWrite{
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

// TestStoreSuperuserControlPlaneLifecycle verifies the superuser store lifecycle across all new models.
func TestStoreSuperuserControlPlaneLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseSeedSuperuserOperationalRows(parseT, parseStore, parseOwner)

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
