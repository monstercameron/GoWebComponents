package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminControlMutationScopeBoundaries verifies superuser vs workspace-admin enforcement for feature-flag, experiment, and incident control mutations.
func TestAdminControlMutationScopeBoundaries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseSuperuser := parseMustCreateUser(parseT, parseStore, "control-superuser@example.com")
	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "control-workspace-admin@example.com")
	parseNormalUser := parseMustCreateUser(parseT, parseStore, "control-user@example.com")
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)

	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "ws-control-admin",
		Slug:         "ws-control-admin",
		Name:         "Control Admin Workspace",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseWorkspaceAdmin.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace admin: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "ws-control-outside",
		Slug:         "ws-control-outside",
		Name:         "Control Outside Workspace",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseNormalUser.ID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace outside: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseStore.parseListWorkspaces(20)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	parseAdminWorkspaceID := int64(0)
	parseOutsideWorkspaceID := int64(0)
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		if parseWorkspaceRow.WorkspaceKey == "ws-control-admin" {
			parseAdminWorkspaceID = parseWorkspaceRow.ID
		}
		if parseWorkspaceRow.WorkspaceKey == "ws-control-outside" {
			parseOutsideWorkspaceID = parseWorkspaceRow.ID
		}
	}
	if parseAdminWorkspaceID <= 0 || parseOutsideWorkspaceID <= 0 {
		parseT.Fatalf("expected both control workspace ids, rows=%+v", parseWorkspaceRows)
	}
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseAdminWorkspaceID,
		UserID:          parseWorkspaceAdmin.ID,
		RoleKey:         "admin",
		Status:          "active",
		InvitedByUserID: parseWorkspaceAdmin.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership admin: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseOutsideWorkspaceID,
		UserID:          parseNormalUser.ID,
		RoleKey:         "member",
		Status:          "active",
		InvitedByUserID: parseNormalUser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership user: %v", parseErr)
	}

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-control-superuser", parseSuperuser.ID, parseSuperuser.Email)
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-control-workspace-admin", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)
	parseNormalUserCtx := parseBindAuthUser(parseServer, "peer-control-user", parseNormalUser.ID, parseNormalUser.Email)

	parseSuperuserFeatureScope, parseErr := parseServer.parseAuthorizeAdminControlMutationScope(parseSuperuserCtx, parseAdminControlMutationFeatureFlag, 0)
	if parseErr != nil {
		parseT.Fatalf("feature-flag superuser authz: %v", parseErr)
	}
	if !parseSuperuserFeatureScope.isPlatformScope || parseSuperuserFeatureScope.adminUserID != parseSuperuser.ID {
		parseT.Fatalf("unexpected feature-flag superuser scope: %+v", parseSuperuserFeatureScope)
	}
	parseSuperuserExperimentScope, parseErr := parseServer.parseAuthorizeAdminControlMutationScope(parseSuperuserCtx, parseAdminControlMutationExperiment, 0)
	if parseErr != nil {
		parseT.Fatalf("experiment superuser authz: %v", parseErr)
	}
	if !parseSuperuserExperimentScope.isPlatformScope || parseSuperuserExperimentScope.adminUserID != parseSuperuser.ID {
		parseT.Fatalf("unexpected experiment superuser scope: %+v", parseSuperuserExperimentScope)
	}
	parseSuperuserIncidentScope, parseErr := parseServer.parseAuthorizeAdminControlMutationScope(parseSuperuserCtx, parseAdminControlMutationIncident, parseOutsideWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("incident superuser authz: %v", parseErr)
	}
	if !parseSuperuserIncidentScope.isPlatformScope || parseSuperuserIncidentScope.adminUserID != parseSuperuser.ID {
		parseT.Fatalf("unexpected incident superuser scope: %+v", parseSuperuserIncidentScope)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminControlMutationScope(parseWorkspaceAdminCtx, parseAdminControlMutationFeatureFlag, 0); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("feature-flag workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminControlMutationScope(parseWorkspaceAdminCtx, parseAdminControlMutationExperiment, 0); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("experiment workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	parseWorkspaceIncidentScope, parseErr := parseServer.parseAuthorizeAdminControlMutationScope(parseWorkspaceAdminCtx, parseAdminControlMutationIncident, parseAdminWorkspaceID)
	if parseErr != nil {
		parseT.Fatalf("incident workspace-admin in-scope authz: %v", parseErr)
	}
	if parseWorkspaceIncidentScope.isPlatformScope || parseWorkspaceIncidentScope.adminUserID != parseWorkspaceAdmin.ID {
		parseT.Fatalf("unexpected workspace-admin incident scope: %+v", parseWorkspaceIncidentScope)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminControlMutationScope(parseWorkspaceAdminCtx, parseAdminControlMutationIncident, parseOutsideWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("incident workspace-admin out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.parseAuthorizeAdminControlMutationScope(parseWorkspaceAdminCtx, parseAdminControlMutationIncident, 0); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("incident workspace-admin missing workspace status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	if _, parseErr = parseServer.parseAuthorizeAdminControlMutationScope(parseNormalUserCtx, parseAdminControlMutationIncident, parseAdminWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("incident normal user status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
