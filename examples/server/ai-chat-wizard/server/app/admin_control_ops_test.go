package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestStoreAdminControlMutationFuncs verifies typed store control mutations for feature flags, experiments, and incidents.
func TestStoreAdminControlMutationFuncs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertFeatureFlag(parseFeatureFlagWrite{
		FlagKey:         "flag.admin.control.seeded",
		Description:     "Seeded admin control flag",
		IsEnabled:       false,
		RolloutPercent:  0,
		AudienceJSON:    "{}",
		PayloadJSON:     "{}",
		UpdatedByUserID: parseAliceAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertFeatureFlag seed: %v", parseErr)
	}
	parseFlagRow, parseErr := parseStore.parseStoreAdminFeatureFlagToggle(
		"flag.admin.control.seeded",
		true,
		50,
		`{"workspace":"all"}`,
		`{"mode":"canary"}`,
		"enable canary path",
		parseAliceAuth.ID,
	)
	if parseErr != nil {
		parseT.Fatalf("parseStoreAdminFeatureFlagToggle: %v", parseErr)
	}
	if !parseFlagRow.IsEnabled || parseFlagRow.RolloutPercent != 50 {
		parseT.Fatalf("unexpected feature flag row after toggle: %+v", parseFlagRow)
	}

	parseExperimentRow, parseErr := parseStore.parseStoreAdminExperimentRollback("exp-dashboard-active", "rollback due elevated failure rate")
	if parseErr != nil {
		parseT.Fatalf("parseStoreAdminExperimentRollback: %v", parseErr)
	}
	if parseExperimentRow.Status != "rolled_back" || parseExperimentRow.EndAt == "" {
		parseT.Fatalf("unexpected experiment row after rollback: %+v", parseExperimentRow)
	}

	parseIncidentRows, parseErr := parseStore.parseListIncidents(10)
	if parseErr != nil || len(parseIncidentRows) == 0 {
		parseT.Fatalf("parseListIncidents seed: rows=%+v err=%v", parseIncidentRows, parseErr)
	}
	parseIncidentRow, parseIncidentUpdateRow, parseErr := parseStore.parseStoreAdminIncidentStatusUpdate(
		parseIncidentRows[0].ID,
		"resolved",
		"Incident stabilized after mitigation.",
		true,
		"manual resolution",
		parseAliceAuth.ID,
	)
	if parseErr != nil {
		parseT.Fatalf("parseStoreAdminIncidentStatusUpdate: %v", parseErr)
	}
	if parseIncidentRow.Status != "resolved" {
		parseT.Fatalf("expected resolved incident status, got %+v", parseIncidentRow)
	}
	if parseIncidentUpdateRow.IncidentID != parseIncidentRow.ID || parseIncidentUpdateRow.Status != "resolved" {
		parseT.Fatalf("unexpected incident update row after status update: %+v", parseIncidentUpdateRow)
	}
}

// TestAdminControlOpsRPCs verifies control-plane RPC mutations and incident blast-radius reporting for superusers.
func TestAdminControlOpsRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-control-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseFeatureFlagResp, parseErr := parseServer.SetAdminFeatureFlag(parseAliceCtx, &chatpb.SetAdminFeatureFlagRequest{
		FlagKey:        "flag.admin.control.rpc",
		IsEnabled:      true,
		RolloutPercent: 100,
		AudienceJson:   `{"workspace":"all"}`,
		PayloadJson:    `{"variant":"enabled"}`,
		Confirm:        true,
		Reason:         "rollout to all workspaces",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminFeatureFlag: %v", parseErr)
	}
	if parseFeatureFlagResp.GetFeatureFlag().GetFlagKey() != "flag.admin.control.rpc" || !parseFeatureFlagResp.GetFeatureFlag().GetIsEnabled() {
		parseT.Fatalf("unexpected feature flag response: %+v", parseFeatureFlagResp.GetFeatureFlag())
	}

	parseExperimentResp, parseErr := parseServer.RollbackAdminExperiment(parseAliceCtx, &chatpb.RollbackAdminExperimentRequest{
		ExperimentKey: "exp-dashboard-active",
		Confirm:       true,
		Reason:        "rollback from operator rpc",
	})
	if parseErr != nil {
		parseT.Fatalf("RollbackAdminExperiment: %v", parseErr)
	}
	if parseExperimentResp.GetExperiment().GetStatus() != "rolled_back" {
		parseT.Fatalf("unexpected experiment rollback response: %+v", parseExperimentResp.GetExperiment())
	}

	parseIncidentRows, parseErr := parseStore.parseListIncidents(10)
	if parseErr != nil || len(parseIncidentRows) == 0 {
		parseT.Fatalf("parseListIncidents seed: rows=%+v err=%v", parseIncidentRows, parseErr)
	}
	parseSupportTicketRows, parseErr := parseStore.parseListSupportTickets(10)
	if parseErr != nil || len(parseSupportTicketRows) == 0 {
		parseT.Fatalf("parseListSupportTickets seed: rows=%+v err=%v", parseSupportTicketRows, parseErr)
	}
	parseWorkspaceID := parseSupportTicketRows[0].WorkspaceID
	parseIncidentResp, parseErr := parseServer.UpdateAdminIncident(parseAliceCtx, &chatpb.UpdateAdminIncidentRequest{
		IncidentId:  parseIncidentRows[0].ID,
		WorkspaceId: parseWorkspaceID,
		Status:      "mitigated",
		Message:     "Mitigation deployed and monitoring active.",
		IsPublic:    true,
		Confirm:     true,
		Reason:      "apply mitigation update",
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateAdminIncident: %v", parseErr)
	}
	if parseIncidentResp.GetIncident().GetStatus() != "mitigated" || parseIncidentResp.GetIncidentUpdate().GetStatus() != "mitigated" {
		parseT.Fatalf("unexpected incident update response: incident=%+v update=%+v", parseIncidentResp.GetIncident(), parseIncidentResp.GetIncidentUpdate())
	}

	parseBlastRadiusResp, parseErr := parseServer.GetAdminIncidentBlastRadius(parseAliceCtx, &chatpb.GetAdminIncidentBlastRadiusRequest{
		WorkspaceId:  parseWorkspaceID,
		LookbackDays: 30,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminIncidentBlastRadius: %v", parseErr)
	}
	if parseBlastRadiusResp.GetBlastRadius().GetWorkspaceId() != parseWorkspaceID {
		parseT.Fatalf("unexpected blast radius workspace id=%d want=%d", parseBlastRadiusResp.GetBlastRadius().GetWorkspaceId(), parseWorkspaceID)
	}
	if parseBlastRadiusResp.GetBlastRadius().GetOpenSupportTicketCount() == 0 {
		parseT.Fatalf("expected open support ticket count in blast radius, got %+v", parseBlastRadiusResp.GetBlastRadius())
	}
}

// TestAdminControlOpsWorkspaceScope verifies control-plane mutation scope boundaries for workspace-admin callers.
func TestAdminControlOpsWorkspaceScope(parseT *testing.T) {
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
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-admin-control-scope")
	parseSupportTicketRows, parseErr := parseStore.parseListSupportTickets(25)
	if parseErr != nil || len(parseSupportTicketRows) == 0 {
		parseT.Fatalf("parseListSupportTickets seed: rows=%+v err=%v", parseSupportTicketRows, parseErr)
	}
	parseAliceWorkspaceID := parseSupportTicketRows[0].WorkspaceID
	parseIncidentRows, parseErr := parseStore.parseListIncidents(10)
	if parseErr != nil || len(parseIncidentRows) == 0 {
		parseT.Fatalf("parseListIncidents seed: rows=%+v err=%v", parseIncidentRows, parseErr)
	}
	parseIncidentID := parseIncidentRows[0].ID

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-control-bob", parseBobAuth.ID, parseBobAuth.Email)
	if _, parseErr = parseServer.SetAdminFeatureFlag(parseBobCtx, &chatpb.SetAdminFeatureFlagRequest{
		FlagKey:   "flag.admin.workspace.denied",
		Confirm:   true,
		Reason:    "scope test",
		IsEnabled: true,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminFeatureFlag workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.RollbackAdminExperiment(parseBobCtx, &chatpb.RollbackAdminExperimentRequest{
		ExperimentKey: "exp-dashboard-active",
		Confirm:       true,
		Reason:        "scope test",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("RollbackAdminExperiment workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.UpdateAdminIncident(parseBobCtx, &chatpb.UpdateAdminIncidentRequest{
		IncidentId:  parseIncidentID,
		WorkspaceId: parseAliceWorkspaceID,
		Status:      "investigating",
		Message:     "scope test",
		IsPublic:    false,
		Confirm:     true,
		Reason:      "scope test",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("UpdateAdminIncident out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.GetAdminIncidentBlastRadius(parseBobCtx, &chatpb.GetAdminIncidentBlastRadiusRequest{
		WorkspaceId:  parseAliceWorkspaceID,
		LookbackDays: 30,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminIncidentBlastRadius out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.GetAdminIncidentBlastRadius(parseBobCtx, &chatpb.GetAdminIncidentBlastRadiusRequest{
		WorkspaceId:  parseBobWorkspaceID,
		LookbackDays: 30,
	}); parseErr != nil {
		parseT.Fatalf("GetAdminIncidentBlastRadius in-scope workspace-admin: %v", parseErr)
	}
	_ = parseAliceAuth
}

// BenchmarkParseHasAdminIncidentResolvedStatus reports micro-benchmark throughput for incident-status resolution checks.
func BenchmarkParseHasAdminIncidentResolvedStatus(parseB *testing.B) {
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseHasAdminIncidentResolvedStatus("resolved")
	}
}
