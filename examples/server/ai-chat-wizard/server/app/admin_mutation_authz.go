package app

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminMutationAction string

const (
	parseAdminMutationDisableUser       parseAdminMutationAction = "user.disable"
	parseAdminMutationRestoreUser       parseAdminMutationAction = "user.restore"
	parseAdminMutationSuspendWorkspace  parseAdminMutationAction = "workspace.suspend"
	parseAdminMutationRestoreWorkspace  parseAdminMutationAction = "workspace.restore"
	parseAdminMutationToggleFeatureFlag parseAdminMutationAction = "feature_flag.toggle"
	parseAdminMutationControlExperiment parseAdminMutationAction = "experiment.control"
	parseAdminMutationControlIncident   parseAdminMutationAction = "incident.control"
)

type parseAdminMutationTarget struct {
	userID                    int64
	workspaceID               int64
	reason                    string
	isConfirmed               bool
	isRestoreUserAPIKeys      bool
	isRestoreWorkspaceAPIKeys bool
	isRestoreWebhookEndpoints bool
	isRestoreBackgroundJobs   bool
}

// parseAuthorizeAdminMutationAction enforces actor role/scope rules for one admin mutation intent.
func (parseS *chatServer) parseAuthorizeAdminMutationAction(parseCtx context.Context, parseAction parseAdminMutationAction, parseTarget parseAdminMutationTarget) (parseAdminAccessScope, error) {
	parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseErr
	}
	switch parseAction {
	case parseAdminMutationDisableUser, parseAdminMutationRestoreUser:
		if parseTarget.userID <= 0 {
			return parseAdminAccessScope{}, status.Error(codes.InvalidArgument, "target user id is required")
		}
		return parseS.parseAuthorizeAdminUserMutationScope(parseScope, parseAction, parseTarget.userID)
	case parseAdminMutationSuspendWorkspace, parseAdminMutationRestoreWorkspace:
		if parseTarget.workspaceID <= 0 {
			return parseAdminAccessScope{}, status.Error(codes.InvalidArgument, "target workspace id is required")
		}
		return parseS.parseAuthorizeAdminWorkspaceMutationScope(parseScope, parseAction, parseTarget.workspaceID)
	case parseAdminMutationToggleFeatureFlag, parseAdminMutationControlExperiment, parseAdminMutationControlIncident:
		return parseS.parseAuthorizeSuperuserMutationScope(parseScope, parseAction)
	default:
		return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "unsupported admin mutation action: %s", strings.TrimSpace(string(parseAction)))
	}
}

// parseAuthorizeAdminUserMutationScope checks whether one caller scope may mutate one target user state.
func (parseS *chatServer) parseAuthorizeAdminUserMutationScope(parseScope parseAdminAccessScope, parseAction parseAdminMutationAction, parseTargetUserID int64) (parseAdminAccessScope, error) {
	if parseScope.isPlatformScope {
		return parseScope, nil
	}
	if _, isParseInScope := parseScope.userIDs[parseTargetUserID]; !isParseInScope {
		return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "target user outside workspace-admin scope")
	}
	if parseS != nil && parseS.store != nil {
		isParseTargetSuperuser, parseErr := parseS.store.parseUserHasSURole(parseTargetUserID)
		if parseErr != nil {
			return parseAdminAccessScope{}, status.Errorf(codes.Internal, "target user role lookup failed: %v", parseErr)
		}
		if isParseTargetSuperuser {
			return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "workspace admin cannot mutate superuser account state")
		}
	}
	if parseS != nil && parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin mutation authorized",
			slog.String("action", strings.TrimSpace(string(parseAction))),
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("target_user_id", parseTargetUserID),
			slog.String("scope", "workspace"),
		)
	}
	return parseScope, nil
}

// parseAuthorizeAdminWorkspaceMutationScope checks whether one caller scope may mutate one target workspace state.
func (parseS *chatServer) parseAuthorizeAdminWorkspaceMutationScope(parseScope parseAdminAccessScope, parseAction parseAdminMutationAction, parseTargetWorkspaceID int64) (parseAdminAccessScope, error) {
	if parseScope.isPlatformScope {
		return parseScope, nil
	}
	if _, isParseInScope := parseScope.workspaceIDs[parseTargetWorkspaceID]; !isParseInScope {
		return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "target workspace outside workspace-admin scope")
	}
	if parseS != nil && parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin mutation authorized",
			slog.String("action", strings.TrimSpace(string(parseAction))),
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("target_workspace_id", parseTargetWorkspaceID),
			slog.String("scope", "workspace"),
		)
	}
	return parseScope, nil
}

// parseAuthorizeSuperuserMutationScope checks whether one caller scope may execute one platform-wide mutation.
func (parseS *chatServer) parseAuthorizeSuperuserMutationScope(parseScope parseAdminAccessScope, parseAction parseAdminMutationAction) (parseAdminAccessScope, error) {
	if !parseScope.isPlatformScope {
		return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "superuser role required for platform mutation")
	}
	if parseS != nil && parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin mutation authorized",
			slog.String("action", strings.TrimSpace(string(parseAction))),
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.String("scope", "platform"),
		)
	}
	return parseScope, nil
}
