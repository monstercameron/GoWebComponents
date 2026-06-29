// auth_rpc.go owns the session bootstrap and auth RPC cluster.
package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"log/slog"
)

func (parseS *chatServer) parseDisplayNameForUser(parseUserID int64, parseFallbackEmail string) string {
	if parseS.store == nil || parseUserID <= 0 {
		return parseDefaultDisplayNameFromEmail(parseFallbackEmail)
	}
	parseName, _, parseErr := parseS.store.getUserName(parseUserID)
	if parseErr != nil {
		return parseDefaultDisplayNameFromEmail(parseFallbackEmail)
	}
	parseName = strings.TrimSpace(parseName)
	if parseName == "" {
		return parseDefaultDisplayNameFromEmail(parseFallbackEmail)
	}
	return parseName
}

func (parseS *chatServer) parseAuthResponseForUser(parseUser authUser, parseToken string) *chatpb.AuthResponse {
	return &chatpb.AuthResponse{
		AuthToken:   parseToken,
		UserId:      parseUser.ID,
		Email:       parseUser.Email,
		DisplayName: parseS.parseDisplayNameForUser(parseUser.ID, parseUser.Email),
	}
}

// Signup handles the signup RPC.
func (parseS *chatServer) Signup(parseCtx context.Context, parseReq *chatpb.SignupRequest) (*chatpb.AuthResponse, error) {
	parseLogger := parseResolveAuthRPCLogger(parseS, parseCtx, "Signup")
	if parseS.authManager == nil {
		parseLogger.Error("rpc.Signup: auth unavailable", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"))
		return nil, status.Error(codes.Unavailable, "auth unavailable")
	}
	parseS.parseTrackFirstChatFunnelStep(parseCtx, 0, parseFirstChatStepAuthStarted, map[string]any{
		"method": "signup",
	})
	parseUser, parseErr := parseS.authManager.parseSignup(parseReq.GetEmail(), parseReq.GetPassword(), parseReq.GetDisplayName())
	if parseErr != nil {
		if errors.Is(parseErr, errUserAlreadyExists) {
			parseLogger.Warn("rpc.Signup: duplicate email", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"))
			return nil, status.Error(codes.AlreadyExists, "an account with that email already exists")
		}
		if parseIsStoreUnavailableError(parseErr) {
			parseLogger.Error("rpc.Signup: auth persistence unavailable", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"))
			return nil, status.Error(codes.Unavailable, "auth persistence unavailable")
		}
		parseLogger.Warn("rpc.Signup: validation failed", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"), slog.String("error", parseErr.Error()))
		return nil, status.Error(codes.InvalidArgument, parseErr.Error())
	}
	parseS.parseTrackFirstChatFunnelStep(parseCtx, parseUser.ID, parseFirstChatStepAuthStarted, map[string]any{
		"method":                "signup",
		"resolved_after_signup": true,
	})
	parseToken, parseErr := parseS.authManager.issueTokenForContext(parseCtx, parseUser, "")
	if parseErr != nil {
		if parseIsStoreUnavailableError(parseErr) {
			parseLogger.Error("rpc.Signup: auth persistence unavailable during token issue", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"))
			return nil, status.Error(codes.Unavailable, "auth persistence unavailable")
		}
		parseLogger.Error("rpc.Signup: token issue failed", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"), slog.String("error", parseErr.Error()))
		return nil, status.Error(codes.Internal, "issue auth token")
	}
	parseS.parseTrackFirstChatFunnelStep(parseCtx, parseUser.ID, parseFirstChatStepAuthCompleted, map[string]any{
		"method": "signup",
	})
	parseLogger.Info("rpc.Signup: complete", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"))
	return parseS.parseAuthResponseForUser(parseUser, parseToken), nil
}

// Login handles the login RPC.
func (parseS *chatServer) Login(parseCtx context.Context, parseReq *chatpb.LoginRequest) (*chatpb.AuthResponse, error) {
	parseLogger := parseResolveAuthRPCLogger(parseS, parseCtx, "Login")
	if parseS.authManager == nil {
		parseLogger.Error("rpc.Login: auth unavailable", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"))
		return nil, status.Error(codes.Unavailable, "auth unavailable")
	}
	parseS.parseTrackFirstChatFunnelStep(parseCtx, 0, parseFirstChatStepAuthStarted, map[string]any{
		"method": "login",
	})
	parseUser, parseErr := parseS.authManager.parseLogin(parseReq.GetEmail(), parseReq.GetPassword())
	if parseErr != nil {
		if errors.Is(parseErr, errInvalidCredentials) {
			parseLogger.Warn("rpc.Login: invalid credentials", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"))
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		if parseIsStoreUnavailableError(parseErr) {
			parseLogger.Error("rpc.Login: auth persistence unavailable", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"))
			return nil, status.Error(codes.Unavailable, "auth persistence unavailable")
		}
		parseLogger.Warn("rpc.Login: validation failed", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"), slog.String("error", parseErr.Error()))
		return nil, status.Error(codes.InvalidArgument, parseErr.Error())
	}
	parseS.parseTrackFirstChatFunnelStep(parseCtx, parseUser.ID, parseFirstChatStepAuthStarted, map[string]any{
		"method":               "login",
		"resolved_after_login": true,
	})
	parseToken, parseErr := parseS.authManager.issueTokenForContext(parseCtx, parseUser, "")
	if parseErr != nil {
		if parseIsStoreUnavailableError(parseErr) {
			parseLogger.Error("rpc.Login: auth persistence unavailable during token issue", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"))
			return nil, status.Error(codes.Unavailable, "auth persistence unavailable")
		}
		parseLogger.Error("rpc.Login: token issue failed", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"), slog.String("error", parseErr.Error()))
		return nil, status.Error(codes.Internal, "issue auth token")
	}
	parseS.parseTrackFirstChatFunnelStep(parseCtx, parseUser.ID, parseFirstChatStepAuthCompleted, map[string]any{
		"method": "login",
	})
	parseLogger.Info("rpc.Login: complete", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"))
	return parseS.parseAuthResponseForUser(parseUser, parseToken), nil
}

// Logout handles the logout RPC.
func (parseS *chatServer) Logout(parseCtx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if parseS.authManager != nil {
		if parseErr := parseS.authManager.parseRevokeSessionFromContext(parseCtx); parseErr != nil {
			return nil, status.Errorf(codes.Internal, "revoke auth session: %v", parseErr)
		}
	}
	if parsePeerAddress, parseHasPeerAddress := parsePeerAddressFromContext(parseCtx); parseHasPeerAddress {
		parseS.parseUnbindAuthenticatedPeer(parsePeerAddress)
	}
	return &emptypb.Empty{}, nil
}

// parseBuildAuthBootstrapRoleSummary resolves one user into one stable auth-bootstrap role summary.
func (parseS *chatServer) parseBuildAuthBootstrapRoleSummary(parseUserID int64) *chatpb.AuthRoleSummary {
	parseSummary := &chatpb.AuthRoleSummary{Scope: "user"}
	if parseUserID <= 0 || parseS == nil || parseS.store == nil {
		return parseSummary
	}
	parseScope, parseErr := parseS.parseResolveAdminAccessScopeForUserID(parseUserID)
	if parseErr != nil {
		if status.Code(parseErr) != codes.PermissionDenied && parseS.logger != nil {
			parseS.logger.Warn(
				"auth bootstrap role summary fallback",
				slog.Int64("user_id", parseUserID),
				slog.String("error", parseErr.Error()),
			)
		}
		return parseSummary
	}
	parseSummary.CanAccessAdmin = true
	if parseScope.isPlatformScope {
		parseSummary.Scope = "superuser"
		parseSummary.IsSuperuser = true
		return parseSummary
	}
	parseSummary.Scope = "workspace_admin"
	parseWorkspaceIDs := make([]int64, 0, len(parseScope.workspaceIDs))
	for parseWorkspaceID := range parseScope.workspaceIDs {
		parseWorkspaceIDs = append(parseWorkspaceIDs, parseWorkspaceID)
	}
	slices.Sort(parseWorkspaceIDs)
	parseSummary.WorkspaceAdminWorkspaceIds = parseWorkspaceIDs
	return parseSummary
}

// parseBuildAuthExpirySeconds converts one absolute expiry timestamp into one non-negative remaining-second value.
func parseBuildAuthExpirySeconds(parseExpiry time.Time) int64 {
	if parseExpiry.IsZero() {
		return 0
	}
	parseRemaining := time.Until(parseExpiry)
	if parseRemaining <= 0 {
		return 0
	}
	return int64(parseRemaining / time.Second)
}

// GetSession returns the current session snapshot.
func (parseS *chatServer) GetSession(parseCtx context.Context, _ *emptypb.Empty) (*chatpb.GetSessionResponse, error) {
	parseUser, parseOk := parseS.parseAuthenticatedUserFromContext(parseCtx)
	if parseOk && parseUser.ID > 0 {
		parseResponse := &chatpb.GetSessionResponse{
			Authenticated: true,
			UserId:        parseUser.ID,
			Email:         parseUser.Email,
			DisplayName:   parseS.parseDisplayNameForUser(parseUser.ID, parseUser.Email),
			SessionStatus: "authenticated",
			RoleSummary:   parseS.parseBuildAuthBootstrapRoleSummary(parseUser.ID),
		}
		if parseS.authManager != nil {
			if parseSessionUser, parseClaims, parseHasSession := parseS.authManager.parseAuthenticatedSessionFromContext(parseCtx); parseHasSession && parseSessionUser.ID == parseUser.ID {
				parseResponse.SessionId = strings.TrimSpace(parseClaims.SessionID)
				parseResponse.TokenVersion = parseClaims.TokenVersion
				if parseClaims.ExpiresAt != nil {
					parseExpiry := parseClaims.ExpiresAt.Time.UTC()
					parseResponse.ExpiresAt = parseExpiry.Format(time.RFC3339)
					parseResponse.ExpiresInSeconds = parseBuildAuthExpirySeconds(parseExpiry)
				}
			}
		}
		parseS.parseTrackFirstChatFunnelStep(parseCtx, parseUser.ID, parseFirstChatStepAppBooted, map[string]any{
			"session_status": parseResponse.GetSessionStatus(),
		})
		return parseResponse, nil
	}
	if parseS.authManager != nil {
		if _, parseTokenPresent := parseResolveAuthTokenFromContext(parseCtx); parseTokenPresent {
			return nil, status.Error(codes.Unauthenticated, "session invalid or expired; sign in again")
		}
	}
	if !parseOk || parseUser.ID <= 0 {
		return &chatpb.GetSessionResponse{
			SessionStatus: "unauthenticated",
			RoleSummary:   &chatpb.AuthRoleSummary{Scope: "user"},
		}, nil
	}
	return &chatpb.GetSessionResponse{
		SessionStatus: "unauthenticated",
		RoleSummary:   &chatpb.AuthRoleSummary{Scope: "user"},
	}, nil
}

// RefreshSession refreshes the current session token.
func (parseS *chatServer) RefreshSession(parseCtx context.Context, _ *emptypb.Empty) (*chatpb.AuthResponse, error) {
	parseLogger := parseResolveAuthRPCLogger(parseS, parseCtx, "RefreshSession")
	if parseS.authManager == nil {
		parseLogger.Error("rpc.RefreshSession: auth unavailable", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"))
		return nil, status.Error(codes.Unavailable, "auth unavailable")
	}
	parseUser, parseClaims, parseOk := parseS.authManager.parseAuthenticatedSessionFromContext(parseCtx)
	if !parseOk || parseUser.ID <= 0 {
		parseLogger.Warn("rpc.RefreshSession: authentication required", slog.String("actor.scope", "user"), slog.String("target.scope", "auth.session"))
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	parseToken, parseErr := parseS.authManager.issueTokenForContext(parseCtx, parseUser, parseClaims.SessionID)
	if parseErr != nil {
		if parseIsStoreUnavailableError(parseErr) {
			parseLogger.Error("rpc.RefreshSession: auth persistence unavailable during token issue", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"))
			return nil, status.Error(codes.Unavailable, "auth persistence unavailable")
		}
		parseLogger.Error("rpc.RefreshSession: token issue failed", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"), slog.String("error", parseErr.Error()))
		return nil, status.Error(codes.Internal, "issue auth token")
	}
	parseLogger.Info("rpc.RefreshSession: complete", slog.String("actor.scope", "user"), slog.Int64("actor.user_id", parseUser.ID), slog.String("target.scope", "auth.session"))
	return parseS.parseAuthResponseForUser(parseUser, parseToken), nil
}
