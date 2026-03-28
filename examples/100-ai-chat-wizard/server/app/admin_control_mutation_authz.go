package app

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminControlMutationKind string

const (
	parseAdminControlMutationFeatureFlag parseAdminControlMutationKind = "feature_flag"
	parseAdminControlMutationExperiment  parseAdminControlMutationKind = "experiment"
	parseAdminControlMutationIncident    parseAdminControlMutationKind = "incident"
)

// parseAuthorizeAdminControlMutationScope enforces role/scope boundaries for feature-flag, experiment, and incident-control mutations.
func (parseS *chatServer) parseAuthorizeAdminControlMutationScope(parseCtx context.Context, parseKind parseAdminControlMutationKind, parseWorkspaceID int64) (parseAdminAccessScope, error) {
	parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseErr
	}
	if parseScope.isPlatformScope {
		return parseScope, nil
	}
	switch parseKind {
	case parseAdminControlMutationFeatureFlag, parseAdminControlMutationExperiment:
		return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "superuser role required for platform mutation")
	case parseAdminControlMutationIncident:
		if parseWorkspaceID <= 0 {
			return parseAdminAccessScope{}, status.Error(codes.InvalidArgument, "workspace id is required for incident mutation")
		}
		if _, isParseInScope := parseScope.workspaceIDs[parseWorkspaceID]; !isParseInScope {
			return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "incident workspace outside workspace-admin scope")
		}
		return parseScope, nil
	default:
		return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "unsupported admin control mutation kind: %s", strings.TrimSpace(string(parseKind)))
	}
}
