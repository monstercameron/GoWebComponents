package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminOpsActionScopeBoundaries verifies separate authz gates for incident updates, job retries, notification retries, and webhook replays.
func TestAdminOpsActionScopeBoundaries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-ops-action-owner", parseOwner.ID, parseOwner.Email)

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-ops-action@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-workspace-admin-ops-action")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-ops-action-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)

	parseOtherWorkspaceUser := parseMustCreateUser(parseT, parseStore, "workspace-other-ops-action@example.com")
	parseOtherWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseOtherWorkspaceUser.ID, "ws-other-ops-action")

	parseScope, parseErr := parseServer.parseAuthorizeAdminOpsActionScope(parseSuperuserCtx, parseAdminOpsActionIncidentUpdate, 0)
	if parseErr != nil || !parseScope.isPlatformScope {
		parseT.Fatalf("superuser incident update expected platform allow, scope=%+v err=%v", parseScope, parseErr)
	}
	if parseScope, parseErr = parseServer.parseAuthorizeAdminOpsActionScope(parseWorkspaceAdminCtx, parseAdminOpsActionIncidentUpdate, parseWorkspaceID); parseErr != nil || parseScope.isPlatformScope {
		parseT.Fatalf("workspace-admin incident update in-scope expected allow, scope=%+v err=%v", parseScope, parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminOpsActionScope(parseWorkspaceAdminCtx, parseAdminOpsActionIncidentUpdate, parseOtherWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin incident update out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminOpsActionScope(parseWorkspaceAdminCtx, parseAdminOpsActionIncidentUpdate, 0); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("workspace-admin incident update missing workspace id status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	parseRetryActions := []parseAdminOpsAction{
		parseAdminOpsActionBackgroundJobRetry,
		parseAdminOpsActionNotificationRetry,
		parseAdminOpsActionWebhookReplay,
	}
	for _, parseAction := range parseRetryActions {
		if parseScope, parseErr = parseServer.parseAuthorizeAdminOpsActionScope(parseSuperuserCtx, parseAction, 0); parseErr != nil || !parseScope.isPlatformScope {
			parseT.Fatalf("superuser ops action %q expected platform allow, scope=%+v err=%v", parseAction, parseScope, parseErr)
		}
		if _, parseErr = parseServer.parseAuthorizeAdminOpsActionScope(parseWorkspaceAdminCtx, parseAction, parseWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
			parseT.Fatalf("workspace-admin ops action %q status code=%v want=%v", parseAction, status.Code(parseErr), codes.PermissionDenied)
		}
	}

	if _, parseErr = parseServer.parseAuthorizeAdminOpsActionScope(parseSuperuserCtx, parseAdminOpsAction("ops.unknown"), 0); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unsupported ops action status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestResolveAdminOpsActionAuditEventType verifies one stable audit event key maps from each supported ops action.
func TestResolveAdminOpsActionAuditEventType(parseT *testing.T) {
	parseExpectations := map[parseAdminOpsAction]string{
		parseAdminOpsActionIncidentUpdate:     "admin.ops.incident.update",
		parseAdminOpsActionBackgroundJobRetry: "admin.ops.background_job.retry",
		parseAdminOpsActionNotificationRetry:  "admin.ops.notification.retry",
		parseAdminOpsActionWebhookReplay:      "admin.ops.webhook.replay",
		parseAdminOpsAction("ops.unknown"):   "admin.ops.unknown",
	}
	for parseAction, parseExpectedEventKey := range parseExpectations {
		if parseEventKey := parseResolveAdminOpsActionAuditEventType(parseAction); parseEventKey != parseExpectedEventKey {
			parseT.Fatalf("action %q audit event key = %q, want %q", parseAction, parseEventKey, parseExpectedEventKey)
		}
	}
}

