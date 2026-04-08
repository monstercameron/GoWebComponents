package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminChatDrilldownScopeRules verifies chat drill-down scope and transcript/tool-trace visibility by role.
func TestAdminChatDrilldownScopeRules(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-chat-drilldown-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-chat-drilldown")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-chat-drilldown-bob", parseBobAuth.ID, parseBobAuth.Email)

	parseScope, parseVisibility, parseErr := parseServer.parseAuthorizeAdminChatDrilldownScope(parseAliceCtx, parseAliceAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("superuser chat drill-down expected allow: %v", parseErr)
	}
	if !parseScope.isPlatformScope || parseVisibility.isTranscriptRedacted || !parseVisibility.canViewToolTracePayload {
		parseT.Fatalf("superuser chat drill-down visibility mismatch scope=%+v visibility=%+v", parseScope, parseVisibility)
	}

	parseScope, parseVisibility, parseErr = parseServer.parseAuthorizeAdminChatDrilldownScope(parseBobCtx, parseBobAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("workspace-admin in-scope chat drill-down expected allow: %v", parseErr)
	}
	if parseScope.isPlatformScope || !parseVisibility.isTranscriptRedacted || parseVisibility.canViewToolTracePayload {
		parseT.Fatalf("workspace-admin chat drill-down visibility mismatch scope=%+v visibility=%+v", parseScope, parseVisibility)
	}

	if _, _, parseErr = parseServer.parseAuthorizeAdminChatDrilldownScope(parseBobCtx, parseAliceAuth.ID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin out-of-scope chat drill-down status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, _, parseErr = parseServer.parseAuthorizeAdminChatDrilldownScope(parseBobCtx, 0); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("missing chat drill-down target user status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

