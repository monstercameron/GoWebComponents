package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestWorkspaceAdminOpsDetailAndMutations verifies typed workspace-admin detail and mutation helpers across one scoped workspace.
func TestWorkspaceAdminOpsDetailAndMutations(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-ops@example.com")
	parseWorkspaceMember := parseMustCreateUser(parseT, parseStore, "workspace-member-ops@example.com")
	parseSuperuser := parseMustCreateUser(parseT, parseStore, "workspace-admin-ops-superuser@example.com")
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-admin-ops")
	if parseErr := parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseWorkspaceMember.ID,
		RoleKey:         "member",
		Status:          "active",
		InvitedByUserID: parseWorkspaceAdmin.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership: %v", parseErr)
	}

	const parseKeyID = "ws-admin-ops-key-1"
	if _, parseErr := parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       parseKeyID,
		WorkspaceID: parseWorkspaceID,
		UserID:      parseWorkspaceMember.ID,
		Label:       "Workspace Ops Key",
		KeyPrefix:   "rk_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Workspace Ops Hook",
		TargetURL:   "https://example.com/workspace-ops",
		SecretHash:  "secret",
		EventsJSON:  `["chat.completed"]`,
		IsEnabled:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint: %v", parseErr)
	}
	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil || len(parseWebhookRows) == 0 {
		parseT.Fatalf("parseListWebhookEndpoints: rows=%+v err=%v", parseWebhookRows, parseErr)
	}
	parseWebhookID := parseWebhookRows[0].ID

	parseCtx := parseBindAuthUser(parseServer, "peer-workspace-admin-ops", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)
	parseDetail, parseErr := parseServer.parseGetWorkspaceAdminDetail(parseCtx, parseWorkspaceID, 20)
	if parseErr != nil {
		parseT.Fatalf("parseGetWorkspaceAdminDetail: %v", parseErr)
	}
	if parseDetail.parseWorkspaceRow.ID != parseWorkspaceID {
		parseT.Fatalf("expected workspace detail id=%d, got %+v", parseWorkspaceID, parseDetail.parseWorkspaceRow)
	}
	if len(parseDetail.parseMembershipRows) == 0 || len(parseDetail.parseAPIKeyRows) == 0 || len(parseDetail.parseWebhookRows) == 0 {
		parseT.Fatalf("expected membership/api-key/webhook rows in detail, got memberships=%d api_keys=%d webhooks=%d", len(parseDetail.parseMembershipRows), len(parseDetail.parseAPIKeyRows), len(parseDetail.parseWebhookRows))
	}

	parseMembershipRows, parseErr := parseServer.parseListWorkspaceAdminMembershipReview(parseCtx, parseWorkspaceID, 20)
	if parseErr != nil || len(parseMembershipRows) == 0 {
		parseT.Fatalf("parseListWorkspaceAdminMembershipReview: rows=%+v err=%v", parseMembershipRows, parseErr)
	}

	if parseErr = parseServer.parseRevokeWorkspaceAPIKeyByAdmin(parseCtx, parseWorkspaceID, parseKeyID, "rotating key after incident"); parseErr != nil {
		parseT.Fatalf("parseRevokeWorkspaceAPIKeyByAdmin: %v", parseErr)
	}
	parseAPIKeyRows, parseErr := parseStore.parseListAPIKeys(10)
	if parseErr != nil || len(parseAPIKeyRows) == 0 {
		parseT.Fatalf("parseListAPIKeys: rows=%+v err=%v", parseAPIKeyRows, parseErr)
	}
	if parseAPIKeyRows[0].RevokedAt == "" {
		parseT.Fatalf("expected revoked api key row, got %+v", parseAPIKeyRows[0])
	}

	if parseErr = parseServer.parsePauseWorkspaceWebhookByAdmin(parseCtx, parseWorkspaceID, parseWebhookID, "pausing webhook during incident"); parseErr != nil {
		parseT.Fatalf("parsePauseWorkspaceWebhookByAdmin: %v", parseErr)
	}
	parseWebhookRows, parseErr = parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil || len(parseWebhookRows) == 0 {
		parseT.Fatalf("parseListWebhookEndpoints after pause: rows=%+v err=%v", parseWebhookRows, parseErr)
	}
	if parseWebhookRows[0].IsEnabled {
		parseT.Fatalf("expected paused webhook endpoint, got %+v", parseWebhookRows[0])
	}

	if parseErr = parseServer.parseSuspendWorkspaceByAdmin(parseCtx, parseWorkspaceID, true, "workspace hold"); parseErr != nil {
		parseT.Fatalf("parseSuspendWorkspaceByAdmin: %v", parseErr)
	}
	parseWorkspaceRow, isParseFound, parseErr := parseStore.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("parseGetWorkspaceByID suspended: found=%v row=%+v err=%v", isParseFound, parseWorkspaceRow, parseErr)
	}
	if parseWorkspaceRow.Status != "suspended" {
		parseT.Fatalf("expected suspended workspace, got %+v", parseWorkspaceRow)
	}

	parseResumeCtx := parseBindAuthUser(parseServer, "peer-workspace-admin-ops-resume", parseSuperuser.ID, parseSuperuser.Email)
	if parseErr = parseServer.parseRestoreWorkspaceByAdmin(parseResumeCtx, parseWorkspaceID, true, "workspace resumed"); parseErr != nil {
		parseT.Fatalf("parseRestoreWorkspaceByAdmin: %v", parseErr)
	}
	parseWorkspaceRow, isParseFound, parseErr = parseStore.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("parseGetWorkspaceByID restored: found=%v row=%+v err=%v", isParseFound, parseWorkspaceRow, parseErr)
	}
	if parseWorkspaceRow.Status != "active" {
		parseT.Fatalf("expected active workspace after restore, got %+v", parseWorkspaceRow)
	}
}

// TestWorkspaceAdminOpsDenyOutOfScope verifies workspace-admin helpers fail closed for out-of-scope workspaces.
func TestWorkspaceAdminOpsDenyOutOfScope(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-outscope@example.com")
	parseScopedWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-admin-ops-scope")
	parseOutOfScopeOwner := parseMustCreateUser(parseT, parseStore, "workspace-outscope-owner@example.com")
	parseOutOfScopeWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseOutOfScopeOwner.ID, "ws-admin-ops-outscope")

	const parseOutOfScopeKeyID = "ws-admin-ops-outscope-key"
	if _, parseErr := parseStore.parseCreateAPIKey(parseAPIKeyWrite{
		KeyID:       parseOutOfScopeKeyID,
		WorkspaceID: parseOutOfScopeWorkspaceID,
		UserID:      parseOutOfScopeOwner.ID,
		Label:       "Out Scope Key",
		KeyPrefix:   "rk_",
		SecretHash:  "hash",
		ScopesJSON:  `["chat:send"]`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAPIKey out-of-scope: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseOutOfScopeWorkspaceID,
		Label:       "Out Scope Hook",
		TargetURL:   "https://example.com/outscope",
		SecretHash:  "secret",
		EventsJSON:  `["chat.completed"]`,
		IsEnabled:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint out-of-scope: %v", parseErr)
	}
	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil || len(parseWebhookRows) == 0 {
		parseT.Fatalf("parseListWebhookEndpoints out-of-scope: rows=%+v err=%v", parseWebhookRows, parseErr)
	}
	parseOutOfScopeWebhookID := parseWebhookRows[0].ID

	parseCtx := parseBindAuthUser(parseServer, "peer-workspace-admin-outscope", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)
	if _, parseErr = parseServer.parseGetWorkspaceAdminDetail(parseCtx, parseOutOfScopeWorkspaceID, 10); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("parseGetWorkspaceAdminDetail out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if parseErr = parseServer.parseRevokeWorkspaceAPIKeyByAdmin(parseCtx, parseOutOfScopeWorkspaceID, parseOutOfScopeKeyID, "out of scope test"); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("parseRevokeWorkspaceAPIKeyByAdmin out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if parseErr = parseServer.parsePauseWorkspaceWebhookByAdmin(parseCtx, parseOutOfScopeWorkspaceID, parseOutOfScopeWebhookID, "out of scope test"); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("parsePauseWorkspaceWebhookByAdmin out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if parseErr = parseServer.parseSuspendWorkspaceByAdmin(parseCtx, parseOutOfScopeWorkspaceID, true, "out of scope test"); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("parseSuspendWorkspaceByAdmin out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.parseGetWorkspaceAdminDetail(parseCtx, parseScopedWorkspaceID, 10); parseErr != nil {
		parseT.Fatalf("parseGetWorkspaceAdminDetail scoped workspace: %v", parseErr)
	}
}
