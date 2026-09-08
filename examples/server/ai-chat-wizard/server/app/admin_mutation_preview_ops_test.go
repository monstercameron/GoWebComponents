package app

import (
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseSeedAdminMutationPreviewRows seeds provider and ops mutation-preview fixture rows for one workspace scope.
func parseSeedAdminMutationPreviewRows(parseT *testing.T, parseStore *Store, parseUserID int64, parseWorkspaceID int64) {
	parseT.Helper()
	if parseErr := parseStore.parseUpsertWorkspaceModelRoutingPolicy(parseWorkspaceModelRoutingPolicyWrite{
		WorkspaceID:                parseWorkspaceID,
		PolicyKey:                  "default",
		DefaultModelID:             modelGPT54,
		FallbackModelID:            modelGPT54Mini,
		MaxInputCostPerMillionUSD:  2.50,
		MaxOutputCostPerMillionUSD: 10.00,
		RequiresApproval:           false,
		RulesJSON:                  `{"source":"admin-mutation-preview-test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceModelRoutingPolicy: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWorkspaceCostGuardrail(parseWorkspaceCostGuardrailWrite{
		WorkspaceID:            parseWorkspaceID,
		GuardrailKey:           "default",
		DailyBudgetCents:       15000,
		MonthlyBudgetCents:     250000,
		MaxCostPerRequestCents: 500,
		AlertThresholdPercent:  80,
		ActionMode:             "notify",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceCostGuardrail: %v", parseErr)
	}
	parseSeedAdminOpsQueueRows(parseT, parseStore, parseUserID, parseWorkspaceID)
}

// TestGetAdminMutationPreviews verifies typed business/provider/ops mutation-preview RPC payloads.
func TestGetAdminMutationPreviews(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-mutation-preview")
	parseSeedAdminMutationPreviewRows(parseT, parseStore, parseAliceAuth.ID, parseWorkspaceID)

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-mutation-preview-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseBusinessResp, parseErr := parseServer.GetAdminBusinessMutationPreview(parseSuperuserCtx, &chatpb.GetAdminBusinessMutationPreviewRequest{
		UserId:       parseAliceAuth.ID,
		LookbackDays: 30,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminBusinessMutationPreview: %v", parseErr)
	}
	parseBusinessPreview := parseBusinessResp.GetPreview()
	if parseBusinessPreview == nil {
		parseT.Fatalf("expected business mutation preview payload")
	}
	if parseBusinessPreview.GetAffectedCustomerCount() <= 0 ||
		parseBusinessPreview.GetAffectedSubscriptionCount() <= 0 ||
		parseBusinessPreview.GetOpenInvoiceCount() <= 0 ||
		parseBusinessPreview.GetOpenDunningEventCount() <= 0 ||
		parseBusinessPreview.GetRecentUsageEventCount() <= 0 {
		parseT.Fatalf("unexpected business mutation preview payload: %+v", parseBusinessPreview)
	}
	if strings.TrimSpace(parseBusinessPreview.GetLikelyDownstreamImpact()) == "" {
		parseT.Fatalf("expected business mutation impact summary, got %+v", parseBusinessPreview)
	}

	parseProviderResp, parseErr := parseServer.GetAdminProviderMutationPreview(parseSuperuserCtx, &chatpb.GetAdminProviderMutationPreviewRequest{
		WorkspaceId:  parseWorkspaceID,
		ProviderId:   "fake",
		ModelId:      modelGPT54Mini,
		LookbackDays: 30,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminProviderMutationPreview: %v", parseErr)
	}
	parseProviderPreview := parseProviderResp.GetPreview()
	if parseProviderPreview == nil {
		parseT.Fatalf("expected provider mutation preview payload")
	}
	if parseProviderPreview.GetAffectedWorkspaceCount() <= 0 ||
		parseProviderPreview.GetRoutingPolicyCount() <= 0 ||
		parseProviderPreview.GetCostGuardrailCount() <= 0 ||
		parseProviderPreview.GetRecentUsageEventCount() <= 0 {
		parseT.Fatalf("unexpected provider mutation preview payload: %+v", parseProviderPreview)
	}
	if strings.TrimSpace(parseProviderPreview.GetLikelyDownstreamImpact()) == "" {
		parseT.Fatalf("expected provider mutation impact summary, got %+v", parseProviderPreview)
	}

	parseOpsResp, parseErr := parseServer.GetAdminOpsMutationPreview(parseSuperuserCtx, &chatpb.GetAdminOpsMutationPreviewRequest{
		WorkspaceId:  parseWorkspaceID,
		LookbackDays: 30,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminOpsMutationPreview: %v", parseErr)
	}
	parseOpsPreview := parseOpsResp.GetPreview()
	if parseOpsPreview == nil {
		parseT.Fatalf("expected ops mutation preview payload")
	}
	if parseOpsPreview.GetFailedBackgroundJobCount() <= 0 ||
		parseOpsPreview.GetFailedNotificationCount() <= 0 ||
		parseOpsPreview.GetFailedWebhookDeliveryCount() <= 0 ||
		parseOpsPreview.GetOpenIncidentCount() <= 0 ||
		parseOpsPreview.GetRecentAdminActionCount() <= 0 {
		parseT.Fatalf("unexpected ops mutation preview payload: %+v", parseOpsPreview)
	}
	if strings.TrimSpace(parseOpsPreview.GetLikelyDownstreamImpact()) == "" {
		parseT.Fatalf("expected ops mutation impact summary, got %+v", parseOpsPreview)
	}
}

// TestGetAdminMutationPreviewsScopeGuards verifies workspace-admin callers are denied business/provider/ops mutation-preview RPCs.
func TestGetAdminMutationPreviewsScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-mutation-preview@example.com")
	parseWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-workspace-admin-mutation-preview")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-mutation-preview-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)

	if _, parseErr := parseServer.GetAdminBusinessMutationPreview(parseWorkspaceAdminCtx, &chatpb.GetAdminBusinessMutationPreviewRequest{
		UserId:       parseWorkspaceAdmin.ID,
		LookbackDays: 30,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminBusinessMutationPreview workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr := parseServer.GetAdminProviderMutationPreview(parseWorkspaceAdminCtx, &chatpb.GetAdminProviderMutationPreviewRequest{
		WorkspaceId:  parseWorkspaceID,
		ProviderId:   "fake",
		ModelId:      modelGPT54Mini,
		LookbackDays: 30,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminProviderMutationPreview workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr := parseServer.GetAdminOpsMutationPreview(parseWorkspaceAdminCtx, &chatpb.GetAdminOpsMutationPreviewRequest{
		WorkspaceId:  parseWorkspaceID,
		LookbackDays: 30,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminOpsMutationPreview workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
