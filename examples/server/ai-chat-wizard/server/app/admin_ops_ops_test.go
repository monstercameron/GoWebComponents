package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetAdminOpsDrilldown verifies typed ops drill-down reads summary and table slices from ops sources.
func TestGetAdminOpsDrilldown(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseSeedSuperuserOperationalRows(parseT, parseStore, parseOwner)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	if len(parseWorkspaces) == 0 {
		parseT.Fatalf("expected seeded workspace rows")
	}
	parseWorkspaceID := parseWorkspaces[0].ID

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-ops-drilldown-owner", parseOwner.ID, parseOwner.Email)
	parseResp, parseErr := parseServer.GetAdminOpsDrilldown(parseSuperuserCtx, &chatpb.GetAdminOpsDrilldownRequest{
		LookbackDays: 30,
		Limit:        50,
		WorkspaceId:  parseWorkspaceID,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminOpsDrilldown: %v", parseErr)
	}
	if parseResp.GetSummary() == nil {
		parseT.Fatalf("expected ops summary payload")
	}
	if len(parseResp.GetSiteConfigs()) == 0 ||
		len(parseResp.GetFeatureFlags()) == 0 ||
		len(parseResp.GetAuditLogs()) == 0 ||
		len(parseResp.GetServiceLevelObjectives()) == 0 ||
		len(parseResp.GetIncidents()) == 0 ||
		len(parseResp.GetIncidentUpdates()) == 0 ||
		len(parseResp.GetBackgroundJobs()) == 0 ||
		len(parseResp.GetNotificationOutbox()) == 0 ||
		len(parseResp.GetWebhookEndpoints()) == 0 ||
		len(parseResp.GetWebhookDeliveries()) == 0 {
		parseT.Fatalf(
			"expected non-empty ops drill-down slices, got site_configs=%d feature_flags=%d audit_logs=%d slos=%d incidents=%d incident_updates=%d background_jobs=%d notifications=%d webhook_endpoints=%d webhook_deliveries=%d",
			len(parseResp.GetSiteConfigs()),
			len(parseResp.GetFeatureFlags()),
			len(parseResp.GetAuditLogs()),
			len(parseResp.GetServiceLevelObjectives()),
			len(parseResp.GetIncidents()),
			len(parseResp.GetIncidentUpdates()),
			len(parseResp.GetBackgroundJobs()),
			len(parseResp.GetNotificationOutbox()),
			len(parseResp.GetWebhookEndpoints()),
			len(parseResp.GetWebhookDeliveries()),
		)
	}
	for _, parseAuditRow := range parseResp.GetAuditLogs() {
		if parseAuditRow.GetWorkspaceId() != parseWorkspaceID {
			parseT.Fatalf("expected workspace-scoped audit rows for workspace %d, got %+v", parseWorkspaceID, parseAuditRow)
		}
	}
	for _, parseNotificationRow := range parseResp.GetNotificationOutbox() {
		if parseNotificationRow.GetWorkspaceId() != parseWorkspaceID {
			parseT.Fatalf("expected workspace-scoped notification rows for workspace %d, got %+v", parseWorkspaceID, parseNotificationRow)
		}
	}
	for _, parseWebhookRow := range parseResp.GetWebhookEndpoints() {
		if parseWebhookRow.GetWorkspaceId() != parseWorkspaceID {
			parseT.Fatalf("expected workspace-scoped webhook rows for workspace %d, got %+v", parseWorkspaceID, parseWebhookRow)
		}
	}
	if parseResp.GetSummary().GetAuditLogCount() == 0 || parseResp.GetSummary().GetWebhookDeliveryCount() == 0 {
		parseT.Fatalf("expected non-zero ops summary counts, got %+v", parseResp.GetSummary())
	}
}

// TestGetAdminOpsDrilldownScopeGuards verifies workspace-admin callers are denied ops-surface drill-down access.
func TestGetAdminOpsDrilldownScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-ops-drilldown@example.com")
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-ops-drilldown-scope")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-ops-drilldown-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)

	if _, parseErr := parseServer.GetAdminOpsDrilldown(parseWorkspaceAdminCtx, &chatpb.GetAdminOpsDrilldownRequest{
		LookbackDays: 30,
		Limit:        10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminOpsDrilldown workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
