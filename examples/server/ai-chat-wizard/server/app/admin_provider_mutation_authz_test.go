package app

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminProviderMutationScopeBoundaries verifies provider mutation authz requires superuser scope and fails closed outside platform role.
func TestAdminProviderMutationScopeBoundaries(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-provider-authz-owner", parseOwner.ID, parseOwner.Email)

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-provider-authz@example.com")
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-workspace-admin-provider-authz")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-provider-authz-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)

	parseCarolAuth := parseMustCreateUser(parseT, parseStore, "carol-provider-authz@example.com")
	parseCarolCtx := parseBindAuthUser(parseServer, "peer-admin-provider-authz-carol", parseCarolAuth.ID, parseCarolAuth.Email)

	parseProviderActions := []parseAdminProviderMutationAction{
		parseAdminProviderMutationSetProviderEnabled,
		parseAdminProviderMutationSetModelVisibility,
		parseAdminProviderMutationSetFallbackRouting,
		parseAdminProviderMutationSetProviderLimit,
		parseAdminProviderMutationSetCostGuardrail,
	}
	for _, parseAction := range parseProviderActions {
		if _, parseErr := parseServer.parseAuthorizeAdminProviderMutationScope(parseSuperuserCtx, parseAction); parseErr != nil {
			parseT.Fatalf("superuser provider mutation %q expected allow: %v", parseAction, parseErr)
		}
		if _, parseErr := parseServer.parseAuthorizeAdminProviderMutationScope(parseWorkspaceAdminCtx, parseAction); status.Code(parseErr) != codes.PermissionDenied {
			parseT.Fatalf("workspace-admin provider mutation %q status code=%v want=%v", parseAction, status.Code(parseErr), codes.PermissionDenied)
		}
		if _, parseErr := parseServer.parseAuthorizeAdminProviderMutationScope(parseCarolCtx, parseAction); status.Code(parseErr) != codes.PermissionDenied {
			parseT.Fatalf("normal-user provider mutation %q status code=%v want=%v", parseAction, status.Code(parseErr), codes.PermissionDenied)
		}
	}

	if _, parseErr := parseServer.parseAuthorizeAdminProviderMutationScope(parseSuperuserCtx, parseAdminProviderMutationAction("provider.unknown")); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("unsupported provider mutation status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestAdminProviderMutationBlastRadiusPreview verifies provider mutation blast-radius previews return scoped counts only for authorized superuser callers.
func TestAdminProviderMutationBlastRadiusPreview(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-provider-blast-owner", parseOwner.ID, parseOwner.Email)

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-provider-blast@example.com")
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-workspace-admin-provider-blast")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-provider-blast-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)

	parseScope, parsePreview, parseErr := parseServer.parseAuthorizeAdminProviderMutationWithBlastRadius(parseSuperuserCtx, parseAdminProviderMutationSetFallbackRouting)
	if parseErr != nil {
		parseT.Fatalf("superuser provider mutation blast-radius expected allow: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseT.Fatalf("expected platform scope for superuser blast-radius call, got %+v", parseScope)
	}
	if parsePreview.workspaceCount <= 0 || parsePreview.userCount <= 0 {
		parseT.Fatalf("expected non-zero provider mutation blast-radius counts, got %+v", parsePreview)
	}

	if _, _, parseErr = parseServer.parseAuthorizeAdminProviderMutationWithBlastRadius(parseWorkspaceAdminCtx, parseAdminProviderMutationSetFallbackRouting); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("workspace-admin provider blast-radius status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
