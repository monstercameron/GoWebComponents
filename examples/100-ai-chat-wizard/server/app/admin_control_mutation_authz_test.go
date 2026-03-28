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

	if parseErr = parseServer.parseExecuteFeatureFlagControlMutation(parseSuperuserCtx); status.Code(parseErr) != codes.Unimplemented {
		parseT.Fatalf("feature-flag superuser status code=%v want=%v", status.Code(parseErr), codes.Unimplemented)
	}
	if parseErr = parseServer.parseExecuteExperimentControlMutation(parseSuperuserCtx); status.Code(parseErr) != codes.Unimplemented {
		parseT.Fatalf("experiment superuser status code=%v want=%v", status.Code(parseErr), codes.Unimplemented)
	}
	if parseErr = parseServer.parseExecuteIncidentControlMutation(parseSuperuserCtx, parseOutsideWorkspaceID); status.Code(parseErr) != codes.Unimplemented {
		parseT.Fatalf("incident superuser status code=%v want=%v", status.Code(parseErr), codes.Unimplemented)
	}

	if parseErr = parseServer.parseExecuteFeatureFlagControlMutation(parseWorkspaceAdminCtx); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("feature-flag workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if parseErr = parseServer.parseExecuteExperimentControlMutation(parseWorkspaceAdminCtx); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("experiment workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if parseErr = parseServer.parseExecuteIncidentControlMutation(parseWorkspaceAdminCtx, parseAdminWorkspaceID); status.Code(parseErr) != codes.Unimplemented {
		parseT.Fatalf("incident workspace-admin in-scope status code=%v want=%v", status.Code(parseErr), codes.Unimplemented)
	}
	if parseErr = parseServer.parseExecuteIncidentControlMutation(parseWorkspaceAdminCtx, parseOutsideWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("incident workspace-admin out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if parseErr = parseServer.parseExecuteIncidentControlMutation(parseWorkspaceAdminCtx, 0); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("incident workspace-admin missing workspace status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	if parseErr = parseServer.parseExecuteIncidentControlMutation(parseNormalUserCtx, parseAdminWorkspaceID); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("incident normal user status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
