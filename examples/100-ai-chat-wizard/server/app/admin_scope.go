package app

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminAccessScope struct {
	isPlatformScope bool
	adminUserID     int64
	workspaceIDs    map[int64]struct{}
	userIDs         map[int64]struct{}
}

const (
	parseAdminSurfaceUnknown   = "unknown"
	parseAdminSurfaceBusiness  = "business"
	parseAdminSurfaceCustomers = "customers"
	parseAdminSurfaceChats     = "chats"
	parseAdminSurfaceProviders = "providers"
	parseAdminSurfaceOps       = "ops"
)

// parseRequireAdminSessionUserID requires one session-backed authenticated user id for admin-scope RPCs when auth is configured.
func (parseS *chatServer) parseRequireAdminSessionUserID(parseCtx context.Context) (int64, error) {
	if parseS != nil && parseS.authManager != nil {
		parseUser, _, parseOk := parseS.authManager.parseAuthenticatedSessionFromContext(parseCtx)
		if !parseOk || parseUser.ID <= 0 {
			if parsePeerAddress, parseHasPeerAddress := parsePeerAddressFromContext(parseCtx); parseHasPeerAddress {
				parseS.parseUnbindAuthenticatedPeer(parsePeerAddress)
			}
			return 0, status.Error(codes.Unauthenticated, "authentication required")
		}
		return parseUser.ID, nil
	}
	return parseS.parseRequireAuthenticatedUserID(parseCtx)
}

// parseRequireAdminAccessScope resolves one authenticated admin caller into platform or workspace scope.
func (parseS *chatServer) parseRequireAdminAccessScope(parseCtx context.Context) (parseAdminAccessScope, error) {
	var parseLogger *slog.Logger
	if parseS != nil && parseS.logger != nil {
		parseLogger = parseS.logger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	}
	parseUserID, parseErr := parseS.parseRequireAdminSessionUserID(parseCtx)
	if parseErr != nil {
		if parseLogger != nil {
			parseLogger.Warn(
				"rpc.admin role resolution denied",
				slog.Int64("user_id", parseUserID),
				slog.String("code", status.Code(parseErr).String()),
				slog.String("error", parseErr.Error()),
				slog.String("next_action", "authenticate and retry admin entry"),
			)
		}
		return parseAdminAccessScope{}, parseErr
	}
	parseScope, parseErr := parseS.parseResolveAdminAccessScopeForUserID(parseUserID)
	if parseErr != nil {
		if parseLogger != nil {
			parseLogger.Warn(
				"rpc.admin role resolution denied",
				slog.Int64("user_id", parseUserID),
				slog.String("code", status.Code(parseErr).String()),
				slog.String("error", parseErr.Error()),
				slog.String("next_action", "verify superuser role or active workspace admin membership"),
			)
		}
		return parseAdminAccessScope{}, parseErr
	}
	if parseS != nil && parseS.logger != nil {
		parseScopeType := "workspace"
		if parseScope.isPlatformScope {
			parseScopeType = "platform"
		}
		parseLogger.Info(
			"rpc.admin role resolution complete",
			slog.Int64("user_id", parseUserID),
			slog.String("scope", parseScopeType),
			slog.Int("workspace_count", len(parseScope.workspaceIDs)),
			slog.Int("scoped_user_count", len(parseScope.userIDs)),
		)
	}
	return parseScope, nil
}

// parseRequireAdminSliceScope resolves one admin scope and emits one structured denial log per slice on failures.
func (parseS *chatServer) parseRequireAdminSliceScope(parseCtx context.Context, parseSliceKey string) (parseAdminAccessScope, error) {
	var parseSliceLogger *slog.Logger
	if parseS != nil && parseS.logger != nil {
		parseSliceLogger = parseS.logger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	}
	parseUserID, _ := parseS.parseRequireAdminSessionUserID(parseCtx)
	parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
	if parseErr == nil {
		if !parseScope.isPlatformScope {
			parseSurfaceKey := parseResolveAdminDashboardSurfaceBySliceKey(parseSliceKey)
			if !parseCanWorkspaceAdminAccessDashboardSurface(parseSurfaceKey) {
				parseErr = status.Errorf(codes.PermissionDenied, "dashboard %s surface requires superuser role", parseSurfaceKey)
				if parseSliceLogger != nil {
					parseSliceLogger.Warn(
						"rpc.admin slice denied",
						slog.String("admin_slice", strings.TrimSpace(parseSliceKey)),
						slog.String("admin_surface", parseSurfaceKey),
						slog.Int64("user_id", parseUserID),
						slog.String("code", status.Code(parseErr).String()),
						slog.String("error", parseErr.Error()),
						slog.String("next_action", "sign in as superuser for business/providers/ops dashboard surfaces"),
					)
				}
				return parseAdminAccessScope{}, parseErr
			}
		}
		return parseScope, nil
	}
	if parseSliceLogger != nil {
		parseSliceLogger.Warn(
			"rpc.admin slice denied",
			slog.String("admin_slice", strings.TrimSpace(parseSliceKey)),
			slog.Int64("user_id", parseUserID),
			slog.String("code", status.Code(parseErr).String()),
			slog.String("error", parseErr.Error()),
			slog.String("next_action", "verify admin role scope and retry requested dashboard slice"),
		)
	}
	return parseAdminAccessScope{}, parseErr
}

// parseResolveAdminDashboardSurfaceBySliceKey resolves one dashboard surface key from one slice key.
func parseResolveAdminDashboardSurfaceBySliceKey(parseSliceKey string) string {
	parseSliceKey = strings.TrimSpace(strings.ToLower(parseSliceKey))
	switch {
	case parseSliceKey == "dashboard.home", strings.HasPrefix(parseSliceKey, "dashboard.business"):
		return parseAdminSurfaceBusiness
	case strings.HasPrefix(parseSliceKey, "dashboard.users"),
		strings.HasPrefix(parseSliceKey, "dashboard.user."),
		strings.HasPrefix(parseSliceKey, "dashboard.workspace"),
		strings.HasPrefix(parseSliceKey, "dashboard.support"),
		strings.HasPrefix(parseSliceKey, "dashboard.billing"),
		strings.HasPrefix(parseSliceKey, "dashboard.customers"):
		return parseAdminSurfaceCustomers
	case strings.HasPrefix(parseSliceKey, "dashboard.usage"),
		strings.HasPrefix(parseSliceKey, "dashboard.conversations"),
		strings.HasPrefix(parseSliceKey, "dashboard.chats"):
		return parseAdminSurfaceChats
	case strings.HasPrefix(parseSliceKey, "dashboard.providers"),
		strings.HasPrefix(parseSliceKey, "dashboard.provider"),
		strings.HasPrefix(parseSliceKey, "dashboard.routing"),
		strings.HasPrefix(parseSliceKey, "dashboard.cost_guardrail"):
		return parseAdminSurfaceProviders
	case parseSliceKey == "admin-read-only-report",
		strings.HasPrefix(parseSliceKey, "dashboard.ops"),
		strings.HasPrefix(parseSliceKey, "dashboard.incident"),
		strings.HasPrefix(parseSliceKey, "dashboard.feature_flag"),
		strings.HasPrefix(parseSliceKey, "dashboard.experiment"):
		return parseAdminSurfaceOps
	default:
		return parseAdminSurfaceUnknown
	}
}

// parseCanWorkspaceAdminAccessDashboardSurface reports whether one workspace-admin role can access one dashboard surface.
func parseCanWorkspaceAdminAccessDashboardSurface(parseSurfaceKey string) bool {
	switch strings.TrimSpace(strings.ToLower(parseSurfaceKey)) {
	case parseAdminSurfaceCustomers, parseAdminSurfaceChats:
		return true
	default:
		return false
	}
}

// parseResolveAdminAccessScopeForUserID resolves one user id into platform scope or one workspace-admin scoped user set.
func (parseS *chatServer) parseResolveAdminAccessScopeForUserID(parseUserID int64) (parseAdminAccessScope, error) {
	if parseUserID <= 0 {
		return parseAdminAccessScope{}, status.Error(codes.Unauthenticated, "authentication required")
	}
	if parseS.store == nil {
		return parseAdminAccessScope{}, status.Error(codes.Unavailable, "store unavailable")
	}
	isParseSuperuser, parseErr := parseS.store.parseUserHasSURole(parseUserID)
	if parseErr != nil {
		return parseAdminAccessScope{}, status.Errorf(codes.Internal, "admin role lookup failed: %v", parseErr)
	}
	if isParseSuperuser {
		return parseAdminAccessScope{
			isPlatformScope: true,
			adminUserID:     parseUserID,
		}, nil
	}

	parseMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByUser(parseUserID)
	if parseErr != nil {
		return parseAdminAccessScope{}, status.Errorf(codes.Internal, "workspace membership lookup failed: %v", parseErr)
	}
	parseWorkspaceIDs := make(map[int64]struct{})
	for _, parseMembershipRow := range parseMembershipRows {
		if !parseIsWorkspaceMembershipActive(parseMembershipRow.Status) {
			continue
		}
		if !parseIsWorkspaceAdminRole(parseMembershipRow.RoleKey) {
			continue
		}
		parseWorkspaceRow, isParseWorkspaceFound, parseErr := parseS.store.parseGetWorkspaceByID(parseMembershipRow.WorkspaceID)
		if parseErr != nil {
			return parseAdminAccessScope{}, status.Errorf(codes.Internal, "workspace lookup failed: %v", parseErr)
		}
		if !isParseWorkspaceFound {
			continue
		}
		if !parseIsWorkspaceAdminScopeStatusActive(parseWorkspaceRow.Status) {
			continue
		}
		parseWorkspaceIDs[parseMembershipRow.WorkspaceID] = struct{}{}
	}
	if len(parseWorkspaceIDs) == 0 {
		return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "admin dashboard role required")
	}

	parseScopedUserIDs := make(map[int64]struct{})
	for parseWorkspaceID := range parseWorkspaceIDs {
		parseWorkspaceMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByWorkspace(parseWorkspaceID)
		if parseErr != nil {
			return parseAdminAccessScope{}, status.Errorf(codes.Internal, "workspace membership scope lookup failed: %v", parseErr)
		}
		for _, parseWorkspaceMembershipRow := range parseWorkspaceMembershipRows {
			if !parseIsWorkspaceMembershipActive(parseWorkspaceMembershipRow.Status) {
				continue
			}
			if parseWorkspaceMembershipRow.UserID <= 0 {
				continue
			}
			parseScopedUserIDs[parseWorkspaceMembershipRow.UserID] = struct{}{}
		}
	}
	parseScopedUserIDs[parseUserID] = struct{}{}
	return parseAdminAccessScope{
		isPlatformScope: false,
		adminUserID:     parseUserID,
		workspaceIDs:    parseWorkspaceIDs,
		userIDs:         parseScopedUserIDs,
	}, nil
}

// parseIsWorkspaceAdminRole reports whether one workspace role grants admin dashboard scope.
func parseIsWorkspaceAdminRole(parseRoleKey string) bool {
	switch strings.TrimSpace(strings.ToLower(parseRoleKey)) {
	case "owner", "admin", "workspace_admin":
		return true
	default:
		return false
	}
}

// parseIsWorkspaceMembershipActive reports whether one workspace membership is currently active.
func parseIsWorkspaceMembershipActive(parseStatus string) bool {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "", "active":
		return true
	default:
		return false
	}
}

// parseIsWorkspaceAdminScopeStatusActive reports whether one workspace status allows admin-scope access.
func parseIsWorkspaceAdminScopeStatusActive(parseStatus string) bool {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "", "active":
		return true
	default:
		return false
	}
}

// parseFilterAdminUserRowsByScope filters user summary rows to one workspace-admin user scope.
func parseFilterAdminUserRowsByScope(parseRows []parseAdminUserRow, parseUserIDs map[int64]struct{}) []parseAdminUserRow {
	parseFilteredRows := make([]parseAdminUserRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if _, hasParseUser := parseUserIDs[parseRow.UserID]; !hasParseUser {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterAdminUsageRowsByScope filters usage rows to one workspace-admin user scope.
func parseFilterAdminUsageRowsByScope(parseRows []parseAdminUsageEventRow, parseUserIDs map[int64]struct{}) []parseAdminUsageEventRow {
	parseFilteredRows := make([]parseAdminUsageEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if _, hasParseUser := parseUserIDs[parseRow.UserID]; !hasParseUser {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterAdminConversationRowsByScope filters conversation rows to one workspace-admin user scope.
func parseFilterAdminConversationRowsByScope(parseRows []parseAdminConversationRow, parseUserIDs map[int64]struct{}) []parseAdminConversationRow {
	parseFilteredRows := make([]parseAdminConversationRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if _, hasParseUser := parseUserIDs[parseRow.UserID]; !hasParseUser {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseLimitAdminUserRows truncates user rows to one RPC-safe limit.
func parseLimitAdminUserRows(parseRows []parseAdminUserRow, parseLimit int32) []parseAdminUserRow {
	if parseLimit <= 0 || len(parseRows) <= int(parseLimit) {
		return parseRows
	}
	return parseRows[:parseLimit]
}

// parseLimitAdminUsageRows truncates usage rows to one RPC-safe limit.
func parseLimitAdminUsageRows(parseRows []parseAdminUsageEventRow, parseLimit int32) []parseAdminUsageEventRow {
	if parseLimit <= 0 || len(parseRows) <= int(parseLimit) {
		return parseRows
	}
	return parseRows[:parseLimit]
}

// parseLimitAdminConversationRows truncates conversation rows to one RPC-safe limit.
func parseLimitAdminConversationRows(parseRows []parseAdminConversationRow, parseLimit int32) []parseAdminConversationRow {
	if parseLimit <= 0 || len(parseRows) <= int(parseLimit) {
		return parseRows
	}
	return parseRows[:parseLimit]
}
