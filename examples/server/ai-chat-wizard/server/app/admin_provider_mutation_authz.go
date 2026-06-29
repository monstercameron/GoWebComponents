package app

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminProviderMutationAction string

const (
	parseAdminProviderMutationSetProviderEnabled parseAdminProviderMutationAction = "provider.enabled.set"
	parseAdminProviderMutationSetModelVisibility parseAdminProviderMutationAction = "provider.model_visibility.set"
	parseAdminProviderMutationSetFallbackRouting parseAdminProviderMutationAction = "provider.fallback_routing.set"
	parseAdminProviderMutationSetProviderLimit   parseAdminProviderMutationAction = "provider.limit.set"
	parseAdminProviderMutationSetCostGuardrail   parseAdminProviderMutationAction = "provider.cost_guardrail.set"
)

type parseAdminProviderBlastRadiusPreview struct {
	workspaceCount int64
	userCount      int64
}

// parseAuthorizeAdminProviderMutationScope enforces superuser-only provider mutation boundaries.
func (parseS *chatServer) parseAuthorizeAdminProviderMutationScope(parseCtx context.Context, parseAction parseAdminProviderMutationAction) (parseAdminAccessScope, error) {
	parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseErr
	}
	switch parseAction {
	case parseAdminProviderMutationSetProviderEnabled,
		parseAdminProviderMutationSetModelVisibility,
		parseAdminProviderMutationSetFallbackRouting,
		parseAdminProviderMutationSetProviderLimit,
		parseAdminProviderMutationSetCostGuardrail:
		if parseScope.isPlatformScope {
			return parseScope, nil
		}
		return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "%s requires superuser role", strings.TrimSpace(string(parseAction)))
	default:
		return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "unsupported provider mutation action: %s", strings.TrimSpace(string(parseAction)))
	}
}

// parseAuthorizeAdminProviderMutationWithBlastRadius enforces provider mutation authz and returns one previewable affected-scope count snapshot.
func (parseS *chatServer) parseAuthorizeAdminProviderMutationWithBlastRadius(parseCtx context.Context, parseAction parseAdminProviderMutationAction) (parseAdminAccessScope, parseAdminProviderBlastRadiusPreview, error) {
	parseScope, parseErr := parseS.parseAuthorizeAdminProviderMutationScope(parseCtx, parseAction)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseAdminProviderBlastRadiusPreview{}, parseErr
	}
	parseWorkspaceCount, parseUserCount := parseS.parseResolveProviderMutationBlastRadiusCounts()
	return parseScope, parseAdminProviderBlastRadiusPreview{
		workspaceCount: parseWorkspaceCount,
		userCount:      parseUserCount,
	}, nil
}

// parseResolveProviderMutationBlastRadiusCounts computes one workspace/user blast-radius count snapshot for provider control-plane mutations.
func (parseS *chatServer) parseResolveProviderMutationBlastRadiusCounts() (int64, int64) {
	if parseS == nil || parseS.store == nil {
		return 0, 0
	}
	parseWorkspaceRows, parseErr := parseS.store.parseListWorkspaces(parseAdminScopedScanLimit)
	if parseErr != nil {
		return 0, 0
	}
	parseUserIDSet := make(map[int64]struct{})
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		parseMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByWorkspace(parseWorkspaceRow.ID)
		if parseErr != nil {
			continue
		}
		for _, parseMembershipRow := range parseMembershipRows {
			if parseMembershipRow.UserID <= 0 {
				continue
			}
			parseUserIDSet[parseMembershipRow.UserID] = struct{}{}
		}
	}
	return int64(len(parseWorkspaceRows)), int64(len(parseUserIDSet))
}
