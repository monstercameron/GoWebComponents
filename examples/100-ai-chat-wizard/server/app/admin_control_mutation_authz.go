package app

import (
	"context"
	"log/slog"
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

// parseExecuteFeatureFlagControlMutation authorizes one feature-flag control mutation and leaves persistence wiring as a stub.
func (parseS *chatServer) parseExecuteFeatureFlagControlMutation(parseCtx context.Context) error {
	if _, parseErr := parseS.parseAuthorizeAdminControlMutationScope(parseCtx, parseAdminControlMutationFeatureFlag, 0); parseErr != nil {
		return parseErr
	}
	if parseS != nil && parseS.logger != nil {
		parseS.logger.Info("rpc.admin control mutation authorized", slog.String("kind", string(parseAdminControlMutationFeatureFlag)))
	}
	return status.Error(codes.Unimplemented, "feature-flag control mutation stub")
}

// parseExecuteExperimentControlMutation authorizes one experiment control mutation and leaves persistence wiring as a stub.
func (parseS *chatServer) parseExecuteExperimentControlMutation(parseCtx context.Context) error {
	if _, parseErr := parseS.parseAuthorizeAdminControlMutationScope(parseCtx, parseAdminControlMutationExperiment, 0); parseErr != nil {
		return parseErr
	}
	if parseS != nil && parseS.logger != nil {
		parseS.logger.Info("rpc.admin control mutation authorized", slog.String("kind", string(parseAdminControlMutationExperiment)))
	}
	return status.Error(codes.Unimplemented, "experiment control mutation stub")
}

// parseExecuteIncidentControlMutation authorizes one incident control mutation for one workspace and leaves persistence wiring as a stub.
func (parseS *chatServer) parseExecuteIncidentControlMutation(parseCtx context.Context, parseWorkspaceID int64) error {
	if _, parseErr := parseS.parseAuthorizeAdminControlMutationScope(parseCtx, parseAdminControlMutationIncident, parseWorkspaceID); parseErr != nil {
		return parseErr
	}
	if parseS != nil && parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin control mutation authorized",
			slog.String("kind", string(parseAdminControlMutationIncident)),
			slog.Int64("workspace_id", parseWorkspaceID),
		)
	}
	return status.Error(codes.Unimplemented, "incident control mutation stub")
}
