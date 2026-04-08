package app

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAuthorizeAdminMutationActionScopes verifies role and scope enforcement for disable/restore and suspend/restore authorization checks.
func TestAuthorizeAdminMutationActionScopes(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseSuperuser := parseMustCreateUser(parseT, parseStore, "mut-superuser@example.com")
	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "mut-admin@example.com")
	parseWorkspaceMember := parseMustCreateUser(parseT, parseStore, "mut-member@example.com")
	parseOutOfScopeUser := parseMustCreateUser(parseT, parseStore, "mut-outsider@example.com")
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)

	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "ws-admin-mutation-scope",
		Slug:         "ws-admin-mutation-scope",
		Name:         "Mutation Scope Workspace",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseWorkspaceAdmin.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "ws-admin-mutation-outside",
		Slug:         "ws-admin-mutation-outside",
		Name:         "Outside Workspace",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseOutOfScopeUser.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace outside: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseStore.parseListWorkspaces(20)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	parseScopeWorkspaceID := int64(0)
	parseOutsideWorkspaceID := int64(0)
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		if parseWorkspaceRow.WorkspaceKey == "ws-admin-mutation-scope" {
			parseScopeWorkspaceID = parseWorkspaceRow.ID
		}
		if parseWorkspaceRow.WorkspaceKey == "ws-admin-mutation-outside" {
			parseOutsideWorkspaceID = parseWorkspaceRow.ID
		}
	}
	if parseScopeWorkspaceID <= 0 || parseOutsideWorkspaceID <= 0 {
		parseT.Fatalf("expected both workspace ids, rows=%+v", parseWorkspaceRows)
	}

	// Seed one workspace-admin scope that includes a normal member and a superuser member.
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseScopeWorkspaceID,
		UserID:          parseWorkspaceAdmin.ID,
		RoleKey:         "admin",
		Status:          "active",
		InvitedByUserID: parseWorkspaceAdmin.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership admin: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseScopeWorkspaceID,
		UserID:          parseWorkspaceMember.ID,
		RoleKey:         "member",
		Status:          "active",
		InvitedByUserID: parseWorkspaceAdmin.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership member: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseScopeWorkspaceID,
		UserID:          parseSuperuser.ID,
		RoleKey:         "member",
		Status:          "active",
		InvitedByUserID: parseWorkspaceAdmin.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership superuser: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseOutsideWorkspaceID,
		UserID:          parseOutOfScopeUser.ID,
		RoleKey:         "member",
		Status:          "active",
		InvitedByUserID: parseOutOfScopeUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership outsider: %v", parseErr)
	}

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-mutation-superuser", parseSuperuser.ID, parseSuperuser.Email)
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-mutation-workspace-admin", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)
	parseUnauthedCtx := context.Background()

	parseSuperuserScope, parseErr := parseServer.parseAuthorizeAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseOutOfScopeUser.ID},
	)
	if parseErr != nil {
		parseT.Fatalf("superuser disable user authz: %v", parseErr)
	}
	if !parseSuperuserScope.isPlatformScope {
		parseT.Fatalf("expected platform scope for superuser authz, got %+v", parseSuperuserScope)
	}

	parseWorkspaceScope, parseErr := parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationRestoreUser,
		parseAdminMutationTarget{userID: parseWorkspaceMember.ID},
	)
	if parseErr != nil {
		parseT.Fatalf("workspace admin restore in-scope user authz: %v", parseErr)
	}
	if parseWorkspaceScope.isPlatformScope {
		parseT.Fatalf("expected workspace scope for workspace-admin authz, got %+v", parseWorkspaceScope)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseOutOfScopeUser.ID},
	); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace admin out-of-scope user status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseSuperuser.ID},
	); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace admin superuser-target status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationSuspendWorkspace,
		parseAdminMutationTarget{workspaceID: parseOutsideWorkspaceID},
	); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace admin out-of-scope workspace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationRestoreWorkspace,
		parseAdminMutationTarget{workspaceID: parseScopeWorkspaceID},
	); parseErr != nil {
		parseT.Fatalf("workspace admin restore workspace authz: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationToggleFeatureFlag,
		parseAdminMutationTarget{},
	); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace admin feature-flag control status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationControlExperiment,
		parseAdminMutationTarget{},
	); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace admin experiment control status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationControlIncident,
		parseAdminMutationTarget{},
	); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace admin incident control status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationToggleFeatureFlag,
		parseAdminMutationTarget{},
	); parseErr != nil {
		parseT.Fatalf("superuser feature-flag control authz: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationControlExperiment,
		parseAdminMutationTarget{},
	); parseErr != nil {
		parseT.Fatalf("superuser experiment control authz: %v", parseErr)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseSuperuserCtx,
		parseAdminMutationControlIncident,
		parseAdminMutationTarget{},
	); parseErr != nil {
		parseT.Fatalf("superuser incident control authz: %v", parseErr)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{},
	); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("missing user target status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationSuspendWorkspace,
		parseAdminMutationTarget{},
	); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("missing workspace target status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseWorkspaceAdminCtx,
		parseAdminMutationAction("unknown.action"),
		parseAdminMutationTarget{userID: parseWorkspaceMember.ID},
	); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unsupported action status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminMutationAction(
		parseUnauthedCtx,
		parseAdminMutationDisableUser,
		parseAdminMutationTarget{userID: parseWorkspaceMember.ID},
	); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("unauthenticated mutation authz status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
}
