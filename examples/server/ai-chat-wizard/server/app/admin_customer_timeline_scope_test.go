package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminCustomerTimelineScopeRules verifies customer timeline source scope, target-user boundaries, and redaction requirements.
func TestAdminCustomerTimelineScopeRules(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-customer-timeline-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-customer-timeline")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-customer-timeline-bob", parseBobAuth.ID, parseBobAuth.Email)

	parseSources := []parseAdminCustomerTimelineSource{
		parseAdminCustomerTimelineSourceChatContent,
		parseAdminCustomerTimelineSourceBilling,
		parseAdminCustomerTimelineSourceSupport,
		parseAdminCustomerTimelineSourceAuthSession,
		parseAdminCustomerTimelineSourceAuditEvent,
	}
	for _, parseSource := range parseSources {
		parseScope, isParseRedacted, parseErr := parseServer.parseAuthorizeAdminCustomerTimelineScope(parseAliceCtx, parseAliceAuth.ID, parseSource)
		if parseErr != nil {
			parseT.Fatalf("superuser timeline source %q expected allow: %v", parseSource, parseErr)
		}
		if !parseScope.isPlatformScope || isParseRedacted {
			parseT.Fatalf("superuser timeline source %q expected platform non-redacted scope, got scope=%+v redacted=%v", parseSource, parseScope, isParseRedacted)
		}

		parseScope, isParseRedacted, parseErr = parseServer.parseAuthorizeAdminCustomerTimelineScope(parseBobCtx, parseBobAuth.ID, parseSource)
		if parseErr != nil {
			parseT.Fatalf("workspace-admin in-scope timeline source %q expected allow: %v", parseSource, parseErr)
		}
		if parseScope.isPlatformScope || !isParseRedacted {
			parseT.Fatalf("workspace-admin timeline source %q expected scoped redacted access, got scope=%+v redacted=%v", parseSource, parseScope, isParseRedacted)
		}
	}

	if _, _, parseErr = parseServer.parseAuthorizeAdminCustomerTimelineScope(parseBobCtx, parseAliceAuth.ID, parseAdminCustomerTimelineSourceAuditEvent); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin out-of-scope timeline target status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, _, parseErr = parseServer.parseAuthorizeAdminCustomerTimelineScope(parseAliceCtx, 0, parseAdminCustomerTimelineSourceAuditEvent); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("missing timeline target user status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, _, parseErr = parseServer.parseAuthorizeAdminCustomerTimelineScope(parseAliceCtx, parseAliceAuth.ID, parseAdminCustomerTimelineSource("unknown")); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unsupported timeline source status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}
