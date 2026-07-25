package app

import (
	"fmt"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseSeedAdminProviderFallbackAuditRows seeds provider fallback/control audit rows across two workspaces.
func parseSeedAdminProviderFallbackAuditRows(parseT *testing.T, parseStore *Store, parseActorUserID int64, parseWorkspaceIDA int64, parseWorkspaceIDB int64) {
	parseT.Helper()
	if _, parseErr := parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseActorUserID,
		WorkspaceID: parseWorkspaceIDA,
		EventType:   "admin.provider.fallback_routing.updated",
		TargetType:  "workspace",
		TargetID:    fmt.Sprintf("%d", parseWorkspaceIDA),
		Summary:     "Provider fallback routing updated",
		PayloadJSON: `{"policy_key":"default","provider":"openai"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog provider fallback A: %v", parseErr)
	}
	if _, parseErr := parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseActorUserID,
		WorkspaceID: parseWorkspaceIDB,
		EventType:   "admin.provider.limit.updated",
		TargetType:  "provider",
		TargetID:    "fake",
		Summary:     "Provider limit updated",
		PayloadJSON: `{"provider":"fake","requests_per_minute":100}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog provider fallback B: %v", parseErr)
	}
}

// TestGetAdminProviderHealthTrends verifies typed provider health trends and fallback-event slices.
func TestGetAdminProviderHealthTrends(parseT *testing.T) {
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
	parseAliceWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-provider-trends")
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-provider-trends")
	parseSeedAdminProviderFallbackAuditRows(parseT, parseStore, parseAliceAuth.ID, parseAliceWorkspaceID, parseBobWorkspaceID)

	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-provider-trends-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseResp, parseErr := parseServer.GetAdminProviderHealthTrends(parseAliceCtx, &chatpb.GetAdminProviderHealthTrendsRequest{
		LookbackDays: 30,
		Limit:        50,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminProviderHealthTrends: %v", parseErr)
	}
	if parseResp.GetSummary() == nil {
		parseT.Fatalf("expected provider health trend summary")
	}
	if len(parseResp.GetTrends()) == 0 || len(parseResp.GetFallbackEvents()) == 0 {
		parseT.Fatalf(
			"expected non-empty provider trend/fallback slices, got trends=%d fallback=%d",
			len(parseResp.GetTrends()),
			len(parseResp.GetFallbackEvents()),
		)
	}
	parseScopedResp, parseErr := parseServer.GetAdminProviderHealthTrends(parseAliceCtx, &chatpb.GetAdminProviderHealthTrendsRequest{
		LookbackDays: 30,
		Limit:        50,
		WorkspaceId:  parseBobWorkspaceID,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminProviderHealthTrends scoped workspace: %v", parseErr)
	}
	if len(parseScopedResp.GetFallbackEvents()) == 0 {
		parseT.Fatalf("expected scoped fallback events for workspace %d", parseBobWorkspaceID)
	}
	for _, parseFallbackRow := range parseScopedResp.GetFallbackEvents() {
		if parseFallbackRow.GetWorkspaceId() != parseBobWorkspaceID {
			parseT.Fatalf("expected scoped fallback workspace id %d, got %+v", parseBobWorkspaceID, parseFallbackRow)
		}
	}
}

// TestGetAdminProviderHealthTrendsScopeGuards verifies workspace-admin callers are denied provider health trend surfaces.
func TestGetAdminProviderHealthTrendsScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-provider-trends-scope")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-provider-trends-bob", parseBobAuth.ID, parseBobAuth.Email)

	if _, parseErr = parseServer.GetAdminProviderHealthTrends(parseBobCtx, &chatpb.GetAdminProviderHealthTrendsRequest{
		LookbackDays: 30,
		Limit:        25,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminProviderHealthTrends workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
