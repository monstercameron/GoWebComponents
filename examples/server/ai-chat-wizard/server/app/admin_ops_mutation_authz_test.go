package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminOpsMutationScopeBoundaries verifies ops settings mutation authz requires superuser scope and fails closed otherwise.
func TestAdminOpsMutationScopeBoundaries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-ops-authz-owner", parseOwner.ID, parseOwner.Email)

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-ops-authz@example.com")
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-workspace-admin-ops-authz")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-ops-authz-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)

	parseCarolAuth := parseMustCreateUser(parseT, parseStore, "carol-ops-authz@example.com")
	parseCarolCtx := parseBindAuthUser(parseServer, "peer-admin-ops-authz-carol", parseCarolAuth.ID, parseCarolAuth.Email)

	parseOpsActions := []parseAdminOpsMutationAction{
		parseAdminOpsMutationSetSiteConfig,
		parseAdminOpsMutationSetFeatureFlag,
		parseAdminOpsMutationSetRetentionPolicy,
		parseAdminOpsMutationSetWebhookBehavior,
		parseAdminOpsMutationSetIntegrationSetting,
	}
	for _, parseAction := range parseOpsActions {
		if _, parseErr := parseServer.parseAuthorizeAdminOpsMutationScope(parseSuperuserCtx, parseAction); parseErr != nil {
			parseT.Fatalf("superuser ops mutation %q expected allow: %v", parseAction, parseErr)
		}
		if _, parseErr := parseServer.parseAuthorizeAdminOpsMutationScope(parseWorkspaceAdminCtx, parseAction); status.Code(parseErr) != codes.PermissionDenied {
			parseT.Fatalf("workspace-admin ops mutation %q status code=%v want=%v", parseAction, status.Code(parseErr), codes.PermissionDenied)
		}
		if _, parseErr := parseServer.parseAuthorizeAdminOpsMutationScope(parseCarolCtx, parseAction); status.Code(parseErr) != codes.PermissionDenied {
			parseT.Fatalf("normal-user ops mutation %q status code=%v want=%v", parseAction, status.Code(parseErr), codes.PermissionDenied)
		}
	}

	if _, parseErr := parseServer.parseAuthorizeAdminOpsMutationScope(parseSuperuserCtx, parseAdminOpsMutationAction("ops.unknown")); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unsupported ops mutation status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}
