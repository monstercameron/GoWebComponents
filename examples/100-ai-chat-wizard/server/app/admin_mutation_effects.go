package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseUserAuthBlockKeyUserDisabled = "user.disabled"

// parseBuildAdminMutationScopeLabel maps one admin mutation scope into one stable diagnostic label.
func parseBuildAdminMutationScopeLabel(parseScope parseAdminAccessScope) string {
	if parseScope.isPlatformScope {
		return "platform"
	}
	return "workspace"
}

// parseBuildWorkspaceSuspendedAuthBlockKey builds one workspace-suspension auth-block key.
func parseBuildWorkspaceSuspendedAuthBlockKey(parseWorkspaceID int64) string {
	return fmt.Sprintf("workspace.suspended.%d", parseWorkspaceID)
}

// parseRequireAdminMutationConfirmation enforces explicit confirmation and non-empty reason capture for destructive admin actions.
func parseRequireAdminMutationConfirmation(parseAction parseAdminMutationAction, isParseConfirmed bool, parseReason string) (string, error) {
	switch parseAction {
	case parseAdminMutationDisableUser, parseAdminMutationRestoreUser, parseAdminMutationSuspendWorkspace, parseAdminMutationRestoreWorkspace:
	default:
		return "", status.Error(codes.InvalidArgument, "unsupported admin mutation action")
	}
	if !isParseConfirmed {
		return "", status.Error(codes.InvalidArgument, "admin mutation confirmation is required")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		return "", status.Error(codes.InvalidArgument, "admin mutation reason is required")
	}
	return parseReason, nil
}

// parseExecuteAdminMutationAction authorizes one admin mutation intent and applies runtime auth/session side effects.
func (parseS *chatServer) parseExecuteAdminMutationAction(parseCtx context.Context, parseAction parseAdminMutationAction, parseTarget parseAdminMutationTarget) error {
	if parseS != nil && parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin mutation submit",
			slog.String("action", strings.TrimSpace(string(parseAction))),
			slog.Int64("target_user_id", parseTarget.userID),
			slog.Int64("target_workspace_id", parseTarget.workspaceID),
			slog.Bool("confirmed", parseTarget.isConfirmed),
		)
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminMutationAction(parseCtx, parseAction, parseTarget)
	if parseErr != nil {
		if parseS != nil && parseS.logger != nil {
			parseS.logger.Warn(
				"rpc.admin mutation denied",
				slog.String("action", strings.TrimSpace(string(parseAction))),
				slog.Int64("target_user_id", parseTarget.userID),
				slog.Int64("target_workspace_id", parseTarget.workspaceID),
				slog.String("stage", "authorize"),
				slog.String("code", status.Code(parseErr).String()),
				slog.String("error", parseErr.Error()),
			)
		}
		return parseErr
	}
	parseMutationReason, parseErr := parseRequireAdminMutationConfirmation(parseAction, parseTarget.isConfirmed, parseTarget.reason)
	if parseErr != nil {
		if parseS != nil && parseS.logger != nil {
			parseS.logger.Warn(
				"rpc.admin mutation denied",
				slog.String("action", strings.TrimSpace(string(parseAction))),
				slog.Int64("admin_user_id", parseScope.adminUserID),
				slog.Int64("target_user_id", parseTarget.userID),
				slog.Int64("target_workspace_id", parseTarget.workspaceID),
				slog.String("scope", parseBuildAdminMutationScopeLabel(parseScope)),
				slog.String("stage", "confirmation"),
				slog.String("code", status.Code(parseErr).String()),
				slog.String("error", parseErr.Error()),
			)
		}
		return parseErr
	}
	if parseS != nil && parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin mutation confirmed",
			slog.String("action", strings.TrimSpace(string(parseAction))),
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("target_user_id", parseTarget.userID),
			slog.Int64("target_workspace_id", parseTarget.workspaceID),
			slog.String("scope", parseBuildAdminMutationScopeLabel(parseScope)),
		)
	}
	if parseS == nil || parseS.store == nil {
		return status.Error(codes.Unavailable, "store unavailable")
	}
	if parseErr = parseS.parseApplyAdminMutationActionEffects(parseAction, parseTarget, parseScope.adminUserID, parseMutationReason); parseErr != nil {
		if parseS != nil && parseS.logger != nil {
			parseS.logger.Error(
				"rpc.admin mutation rollback required",
				slog.String("action", strings.TrimSpace(string(parseAction))),
				slog.Int64("admin_user_id", parseScope.adminUserID),
				slog.Int64("target_user_id", parseTarget.userID),
				slog.Int64("target_workspace_id", parseTarget.workspaceID),
				slog.String("scope", parseBuildAdminMutationScopeLabel(parseScope)),
				slog.String("error", parseErr.Error()),
				slog.String("next_action", "inspect partial side effects and apply targeted rollback if needed"),
			)
		}
		return status.Errorf(codes.Internal, "admin mutation %s apply failed: %v", strings.TrimSpace(string(parseAction)), parseErr)
	}
	if parseErr = parseS.parseStoreAdminMutationAuditLog(parseAction, parseTarget, parseScope.adminUserID, parseMutationReason); parseErr != nil {
		if parseS != nil && parseS.logger != nil {
			parseS.logger.Error(
				"rpc.admin mutation audit write failed",
				slog.String("action", strings.TrimSpace(string(parseAction))),
				slog.Int64("admin_user_id", parseScope.adminUserID),
				slog.Int64("target_user_id", parseTarget.userID),
				slog.Int64("target_workspace_id", parseTarget.workspaceID),
				slog.String("scope", parseBuildAdminMutationScopeLabel(parseScope)),
				slog.String("error", parseErr.Error()),
				slog.String("next_action", "retry or repair audit trail before confirming mutation workflow"),
			)
		}
		return status.Errorf(codes.Internal, "admin mutation %s audit log failed: %v", strings.TrimSpace(string(parseAction)), parseErr)
	}
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin mutation success",
			slog.String("action", strings.TrimSpace(string(parseAction))),
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("target_user_id", parseTarget.userID),
			slog.Int64("target_workspace_id", parseTarget.workspaceID),
			slog.String("scope", parseBuildAdminMutationScopeLabel(parseScope)),
		)
	}
	return nil
}

// parseApplyAdminMutationActionEffects applies mutation side effects for one authorized action and target.
func (parseS *chatServer) parseApplyAdminMutationActionEffects(parseAction parseAdminMutationAction, parseTarget parseAdminMutationTarget, parseAdminUserID int64, parseReason string) error {
	switch parseAction {
	case parseAdminMutationDisableUser:
		return parseS.parseApplyUserDisableMutationEffects(parseTarget.userID, parseAdminUserID, parseReason)
	case parseAdminMutationRestoreUser:
		return parseS.parseApplyUserRestoreMutationEffects(parseTarget, parseAdminUserID, parseReason)
	case parseAdminMutationSuspendWorkspace:
		return parseS.parseApplyWorkspaceSuspendMutationEffects(parseTarget.workspaceID, parseAdminUserID, parseReason)
	case parseAdminMutationRestoreWorkspace:
		return parseS.parseApplyWorkspaceRestoreMutationEffects(parseTarget, parseAdminUserID, parseReason)
	default:
		return errors.New("unsupported admin mutation action")
	}
}

// parseApplyUserDisableMutationEffects marks one user disabled, blocks authentication, and revokes active sessions.
func (parseS *chatServer) parseApplyUserDisableMutationEffects(parseUserID int64, parseAdminUserID int64, parseReason string) error {
	if parseUserID <= 0 {
		return errors.New("user disable effects require target user id")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		parseReason = "admin mutation user.disable"
	}
	if parseErr := parseS.store.parseUpsertUserAccessState(parseUserAccessStateWrite{
		UserID:           parseUserID,
		Status:           "disabled",
		Reason:           parseReason,
		DisabledByUserID: parseAdminUserID,
	}); parseErr != nil {
		return parseErr
	}
	if parseErr := parseS.store.parseUpsertUserAuthBlock(
		parseUserID,
		parseUserAuthBlockKeyUserDisabled,
		"admin.user.disable",
		parseReason,
	); parseErr != nil {
		return parseErr
	}
	parseRevokedAPIKeyCount, parseErr := parseS.store.parseRevokeAPIKeysByUser(parseUserID)
	if parseErr != nil {
		return parseErr
	}
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin user disable operational effects applied",
			slog.Int64("target_user_id", parseUserID),
			slog.Int64("admin_user_id", parseAdminUserID),
			slog.Int64("revoked_api_keys", parseRevokedAPIKeyCount),
		)
	}
	return parseS.parseRevokeUserSessionsAndRotateTokenVersion(parseUserID)
}

// parseApplyUserRestoreMutationEffects marks one user active and clears the user-disable authentication block.
func (parseS *chatServer) parseApplyUserRestoreMutationEffects(parseTarget parseAdminMutationTarget, parseAdminUserID int64, parseReason string) error {
	parseUserID := parseTarget.userID
	if parseUserID <= 0 {
		return errors.New("user restore effects require target user id")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		parseReason = "admin mutation user.restore"
	}
	if parseErr := parseS.store.parseUpsertUserAccessState(parseUserAccessStateWrite{
		UserID:           parseUserID,
		Status:           "active",
		Reason:           parseReason,
		DisabledByUserID: parseAdminUserID,
		DisabledAt:       "",
	}); parseErr != nil {
		return parseErr
	}
	if parseErr := parseS.store.parseDeleteUserAuthBlock(parseUserID, parseUserAuthBlockKeyUserDisabled); parseErr != nil {
		return parseErr
	}
	if parseTarget.isRestoreUserAPIKeys {
		parseRestoredAPIKeyCount, parseErr := parseS.store.parseRestoreAPIKeysByUser(parseUserID)
		if parseErr != nil {
			return parseErr
		}
		if parseS.logger != nil {
			parseS.logger.Info(
				"rpc.admin user restore operational effects applied",
				slog.Int64("target_user_id", parseUserID),
				slog.Int64("admin_user_id", parseAdminUserID),
				slog.Int64("restored_api_keys", parseRestoredAPIKeyCount),
			)
		}
	}
	return nil
}

// parseApplyWorkspaceSuspendMutationEffects marks one workspace suspended, blocks member authentication, and revokes member sessions.
func (parseS *chatServer) parseApplyWorkspaceSuspendMutationEffects(parseWorkspaceID int64, parseAdminUserID int64, parseReason string) error {
	if parseWorkspaceID <= 0 {
		return errors.New("workspace suspend effects require target workspace id")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		parseReason = fmt.Sprintf("workspace %d suspended by admin user %d", parseWorkspaceID, parseAdminUserID)
	}
	if parseErr := parseS.store.parseSetWorkspaceStatusByID(parseWorkspaceID, "suspended"); parseErr != nil {
		return parseErr
	}
	parseAuthBlockKey := parseBuildWorkspaceSuspendedAuthBlockKey(parseWorkspaceID)
	parseUserIDs, parseErr := parseS.parseListWorkspaceActiveUserIDs(parseWorkspaceID)
	if parseErr != nil {
		return parseErr
	}
	for _, parseUserID := range parseUserIDs {
		if parseErr = parseS.store.parseUpsertUserAuthBlock(
			parseUserID,
			parseAuthBlockKey,
			"admin.workspace.suspend",
			parseReason,
		); parseErr != nil {
			return parseErr
		}
		if parseErr = parseS.parseRevokeUserSessionsAndRotateTokenVersion(parseUserID); parseErr != nil {
			return parseErr
		}
	}
	parseOperationalEffects, parseErr := parseS.store.parseApplyWorkspaceSuspendOperationalEffects(parseWorkspaceID, parseReason)
	if parseErr != nil {
		return parseErr
	}
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin workspace suspend operational effects applied",
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.Int64("admin_user_id", parseAdminUserID),
			slog.Int64("revoked_api_keys", parseOperationalEffects.RevokedAPIKeys),
			slog.Int64("disabled_webhooks", parseOperationalEffects.DisabledWebhookEndpoints),
			slog.Int64("suppressed_background_jobs", parseOperationalEffects.SuppressedBackgroundJobs),
		)
	}
	return nil
}

// parseStoreAdminMutationAuditLog writes one operator audit row for one applied admin mutation.
func (parseS *chatServer) parseStoreAdminMutationAuditLog(parseAction parseAdminMutationAction, parseTarget parseAdminMutationTarget, parseAdminUserID int64, parseReason string) error {
	parseTargetType := "user"
	parseTargetID := fmt.Sprintf("%d", parseTarget.userID)
	parseWorkspaceIDHint := parseS.parseResolveAuditWorkspaceIDForUser(parseTarget.userID)
	if parseAction == parseAdminMutationSuspendWorkspace || parseAction == parseAdminMutationRestoreWorkspace {
		parseTargetType = "workspace"
		parseTargetID = fmt.Sprintf("%d", parseTarget.workspaceID)
		parseWorkspaceIDHint = parseTarget.workspaceID
	} else if parseWorkspaceIDHint <= 0 {
		parseWorkspaceIDHint = parseS.parseResolveAuditWorkspaceIDForUser(parseAdminUserID)
	}
	parseWorkspaceID, parseErr := parseS.parseResolveAdminAuditWorkspaceID(
		parseAdminAccessScope{
			isPlatformScope: true,
			adminUserID:     parseAdminUserID,
		},
		parseWorkspaceIDHint,
	)
	if parseErr != nil {
		return parseErr
	}
	if parseWorkspaceID <= 0 {
		return nil
	}
	_, parseErr = parseS.store.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseAdminUserID,
		WorkspaceID: parseWorkspaceID,
		EventType:   "admin.mutation.action",
		TargetType:  parseTargetType,
		TargetID:    parseTargetID,
		Summary:     fmt.Sprintf("%s: %s", strings.TrimSpace(string(parseAction)), strings.TrimSpace(parseReason)),
		PayloadJSON: "{}",
	})
	return parseErr
}

// parseResolveAuditWorkspaceIDForUser resolves one active membership workspace id for audit-log scoping.
func (parseS *chatServer) parseResolveAuditWorkspaceIDForUser(parseUserID int64) int64 {
	if parseS == nil || parseS.store == nil || parseUserID <= 0 {
		return 0
	}
	parseMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByUser(parseUserID)
	if parseErr != nil {
		return 0
	}
	for _, parseMembershipRow := range parseMembershipRows {
		if !parseIsWorkspaceMembershipActive(parseMembershipRow.Status) {
			continue
		}
		if parseMembershipRow.WorkspaceID <= 0 {
			continue
		}
		if _, isParseWorkspaceFound, parseErr := parseS.store.parseGetWorkspaceByID(parseMembershipRow.WorkspaceID); parseErr == nil && isParseWorkspaceFound {
			return parseMembershipRow.WorkspaceID
		}
	}
	return 0
}

// parseApplyWorkspaceRestoreMutationEffects marks one workspace active and clears workspace-suspension auth blocks.
func (parseS *chatServer) parseApplyWorkspaceRestoreMutationEffects(parseTarget parseAdminMutationTarget, parseAdminUserID int64, parseReason string) error {
	parseWorkspaceID := parseTarget.workspaceID
	if parseWorkspaceID <= 0 {
		return errors.New("workspace restore effects require target workspace id")
	}
	if parseErr := parseS.store.parseSetWorkspaceStatusByID(parseWorkspaceID, "active"); parseErr != nil {
		return parseErr
	}
	parseAuthBlockKey := parseBuildWorkspaceSuspendedAuthBlockKey(parseWorkspaceID)
	if parseErr := parseS.store.parseDeleteUserAuthBlocksByKey(parseAuthBlockKey); parseErr != nil {
		return parseErr
	}
	if parseTarget.isRestoreWorkspaceAPIKeys || parseTarget.isRestoreWebhookEndpoints || parseTarget.isRestoreBackgroundJobs {
		parseRestoreResult, parseErr := parseS.store.parseRestoreWorkspaceOperationalEffects(
			parseWorkspaceID,
			parseTarget.isRestoreWorkspaceAPIKeys,
			parseTarget.isRestoreWebhookEndpoints,
			parseTarget.isRestoreBackgroundJobs,
		)
		if parseErr != nil {
			return parseErr
		}
		if parseS.logger != nil {
			parseS.logger.Info(
				"rpc.admin workspace restore operational effects applied",
				slog.Int64("workspace_id", parseWorkspaceID),
				slog.Int64("admin_user_id", parseAdminUserID),
				slog.Int64("restored_api_keys", parseRestoreResult.RevokedAPIKeys),
				slog.Int64("enabled_webhooks", parseRestoreResult.DisabledWebhookEndpoints),
				slog.Int64("restored_background_jobs", parseRestoreResult.SuppressedBackgroundJobs),
			)
		}
	}
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.admin workspace restore auth blocks cleared",
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.Int64("admin_user_id", parseAdminUserID),
			slog.String("reason", strings.TrimSpace(parseReason)),
			slog.String("auth_block_key", parseAuthBlockKey),
		)
	}
	return nil
}

// parseListWorkspaceActiveUserIDs lists unique active member user ids (plus owner) for one workspace.
func (parseS *chatServer) parseListWorkspaceActiveUserIDs(parseWorkspaceID int64) ([]int64, error) {
	parseWorkspaceRow, isParseFound, parseErr := parseS.store.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil {
		return nil, parseErr
	}
	if !isParseFound {
		return nil, errStoreSuperuserScopeMissing
	}
	parseUserIDSet := map[int64]struct{}{}
	if parseWorkspaceRow.OwnerUserID > 0 {
		parseUserIDSet[parseWorkspaceRow.OwnerUserID] = struct{}{}
	}
	parseMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByWorkspace(parseWorkspaceID)
	if parseErr != nil {
		return nil, parseErr
	}
	for _, parseMembershipRow := range parseMembershipRows {
		if !parseIsWorkspaceMembershipActive(parseMembershipRow.Status) {
			continue
		}
		if parseMembershipRow.UserID <= 0 {
			continue
		}
		parseUserIDSet[parseMembershipRow.UserID] = struct{}{}
	}
	parseUserIDs := make([]int64, 0, len(parseUserIDSet))
	for parseUserID := range parseUserIDSet {
		parseUserIDs = append(parseUserIDs, parseUserID)
	}
	sort.Slice(parseUserIDs, func(parseI int, parseJ int) bool {
		return parseUserIDs[parseI] < parseUserIDs[parseJ]
	})
	return parseUserIDs, nil
}

// parseRevokeUserSessionsAndRotateTokenVersion revokes active sessions and increments token version for one user.
func (parseS *chatServer) parseRevokeUserSessionsAndRotateTokenVersion(parseUserID int64) error {
	if parseUserID <= 0 {
		return nil
	}
	if _, parseErr := parseS.store.parseIncrementAuthTokenVersion(parseUserID); parseErr != nil {
		return parseErr
	}
	return parseS.store.parseRevokeAuthSessionsByUser(parseUserID)
}
