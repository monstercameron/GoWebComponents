package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
)

const parseFirstChatFunnelKey = "visit_to_first_chat"
const parseFirstChatFunnelExperimentKey = "first-chat-funnel"

const (
	parseFirstChatStepLandingViewed     = "landing_viewed"
	parseFirstChatStepPricingViewed     = "pricing_viewed"
	parseFirstChatStepCTAClicked        = "cta_clicked"
	parseFirstChatStepAuthStarted       = "auth_started"
	parseFirstChatStepAuthCompleted     = "auth_completed"
	parseFirstChatStepAppBooted         = "app_booted"
	parseFirstChatStepFirstThreadCreate = "first_thread_created"
	parseFirstChatStepFirstSendStarted  = "first_send_started"
	parseFirstChatStepFirstReplyDone    = "first_reply_completed"
	parseFirstChatStepThreadReopened    = "thread_reopened"
)

// parseCollectFirstChatFunnelStepsForPath maps one shell route path to typed first-chat funnel steps.
func parseCollectFirstChatFunnelStepsForPath(parseRequestPath string) []string {
	parseCleanedPath := filepath.ToSlash(filepath.Clean("/" + strings.TrimLeft(strings.TrimSpace(parseRequestPath), "/")))
	switch parseCleanedPath {
	case "/", "/home":
		return []string{parseFirstChatStepLandingViewed}
	case "/pricing":
		return []string{parseFirstChatStepPricingViewed}
	case "/login", "/signup":
		return []string{parseFirstChatStepCTAClicked, parseFirstChatStepAuthStarted}
	default:
		return nil
	}
}

// parseNormalizeFirstChatFunnelStepKey validates and normalizes one first-chat funnel step key.
func parseNormalizeFirstChatFunnelStepKey(parseRawStepKey string) string {
	parseRawStepKey = strings.TrimSpace(strings.ToLower(parseRawStepKey))
	switch parseRawStepKey {
	case parseFirstChatStepLandingViewed,
		parseFirstChatStepPricingViewed,
		parseFirstChatStepCTAClicked,
		parseFirstChatStepAuthStarted,
		parseFirstChatStepAuthCompleted,
		parseFirstChatStepAppBooted,
		parseFirstChatStepFirstThreadCreate,
		parseFirstChatStepFirstSendStarted,
		parseFirstChatStepFirstReplyDone,
		parseFirstChatStepThreadReopened:
		return parseRawStepKey
	default:
		return ""
	}
}

// parseBuildFirstChatFunnelEventName resolves one typed analytics event name for one first-chat funnel step.
func parseBuildFirstChatFunnelEventName(parseStepKey string) string {
	parseStepKey = parseNormalizeFirstChatFunnelStepKey(parseStepKey)
	if parseStepKey == "" {
		return ""
	}
	return "funnel." + parseFirstChatFunnelKey + "." + parseStepKey
}

// parseBuildFirstChatFunnelEventPropsJSON encodes one first-chat funnel event props map as canonical JSON.
func parseBuildFirstChatFunnelEventPropsJSON(parseEventProps map[string]any) string {
	if len(parseEventProps) == 0 {
		return "{}"
	}
	parsePayload, parseErr := json.Marshal(parseEventProps)
	if parseErr != nil {
		return "{}"
	}
	return string(parsePayload)
}

// parseResolveFirstChatFunnelSessionKeyFromContext resolves one best-effort funnel session key from one gRPC context.
func parseResolveFirstChatFunnelSessionKeyFromContext(parseCtx context.Context, parseAuthManager *authManager) string {
	if parseAuthManager != nil {
		if _, parseClaims, parseHasSession := parseAuthManager.parseAuthenticatedSessionFromContext(parseCtx); parseHasSession {
			if strings.TrimSpace(parseClaims.SessionID) != "" {
				return strings.TrimSpace(parseClaims.SessionID)
			}
		}
	}
	return strings.TrimSpace(parseResolveClientIdentityFromContext(parseCtx))
}

// parseResolveFirstChatFunnelSessionKeyFromRequest resolves one best-effort funnel session key from one HTTP request.
func parseResolveFirstChatFunnelSessionKeyFromRequest(parseR *http.Request, parseAuthManager *authManager) string {
	if parseR == nil {
		return ""
	}
	if parseAuthManager != nil {
		parseCookie, parseErr := parseR.Cookie(authCookieName)
		if parseErr == nil {
			_, parseClaims, parseErr2 := parseAuthManager.parseTokenWithMetadata(parseCookie.Value, parseResolveAuthMetadataFromRequest(parseR))
			if parseErr2 == nil && strings.TrimSpace(parseClaims.SessionID) != "" {
				return strings.TrimSpace(parseClaims.SessionID)
			}
		}
	}
	return parseNormalizeClientIdentity(parseR.Header.Get(clientMetadataKey))
}

// parseTrackFirstChatFunnelStep records one typed first-chat funnel step event from one gRPC call path.
func (parseS *chatServer) parseTrackFirstChatFunnelStep(parseCtx context.Context, parseUserID int64, parseStepKey string, parseEventProps map[string]any) {
	parseSessionKey := parseResolveFirstChatFunnelSessionKeyFromContext(parseCtx, parseS.authManager)
	parseS.parseTrackFirstChatFunnelStepWithSession(parseUserID, parseSessionKey, parseStepKey, parseEventProps)
}

// parseTrackFirstChatFunnelStepWithRequest records one typed first-chat funnel step event from one HTTP request path.
func (parseS *chatServer) parseTrackFirstChatFunnelStepWithRequest(parseR *http.Request, parseUserID int64, parseStepKey string, parseEventProps map[string]any) {
	parseSessionKey := parseResolveFirstChatFunnelSessionKeyFromRequest(parseR, parseS.authManager)
	parseS.parseTrackFirstChatFunnelStepWithSession(parseUserID, parseSessionKey, parseStepKey, parseEventProps)
}

// parseTrackFirstChatFunnelStepWithSession records one typed first-chat funnel step event for one resolved session key.
func (parseS *chatServer) parseTrackFirstChatFunnelStepWithSession(parseUserID int64, parseSessionKey string, parseStepKey string, parseEventProps map[string]any) {
	if parseS == nil {
		return
	}
	parseNormalizedStepKey := parseNormalizeFirstChatFunnelStepKey(parseStepKey)
	if parseNormalizedStepKey == "" {
		return
	}
	parseLogger := parseS.logger
	if parseLogger == nil {
		parseLogger = slog.Default()
	}
	parseLogger.Info("funnel.first_chat step",
		slog.String("funnel_key", parseFirstChatFunnelKey),
		slog.String("step_key", parseNormalizedStepKey),
		slog.Int64("user_id", parseUserID),
		slog.String("session_key", strings.TrimSpace(parseSessionKey)),
	)
	if parseS.store == nil || parseUserID <= 0 {
		return
	}
	parseEventName := parseBuildFirstChatFunnelEventName(parseNormalizedStepKey)
	if parseEventName == "" {
		return
	}
	parseWorkspaceID := parseS.parseResolveFirstChatWorkspaceIDForUser(parseUserID)
	if parseWorkspaceID <= 0 {
		parseLogger.Warn("funnel.first_chat scope unresolved",
			slog.String("step_key", parseNormalizedStepKey),
			slog.Int64("user_id", parseUserID),
			slog.String("next_action", "create or attach an active workspace membership for this user"),
		)
		return
	}
	parseExperimentKey := parseS.parseEnsureFirstChatFunnelExperimentKey()
	if parseExperimentKey == "" {
		parseLogger.Warn("funnel.first_chat experiment unresolved",
			slog.String("step_key", parseNormalizedStepKey),
			slog.Int64("user_id", parseUserID),
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.String("next_action", "ensure first-chat funnel experiment bootstrap succeeds"),
		)
		return
	}
	if _, parseErr := parseS.store.parseCreateProductAnalyticsEvent(parseProductAnalyticsEventWrite{
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseUserID,
		SessionKey:     strings.TrimSpace(parseSessionKey),
		EventName:      parseEventName,
		FunnelKey:      parseFirstChatFunnelKey,
		StepKey:        parseNormalizedStepKey,
		ExperimentKey:  parseExperimentKey,
		EventPropsJSON: parseBuildFirstChatFunnelEventPropsJSON(parseEventProps),
	}); parseErr != nil {
		parseLogger.Warn("funnel.first_chat event write failed",
			slog.String("step_key", parseNormalizedStepKey),
			slog.String("session_key", strings.TrimSpace(parseSessionKey)),
			slog.Int64("user_id", parseUserID),
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.String("error", parseErr.Error()),
		)
	}
}

// parseResolveFirstChatWorkspaceIDForUser resolves one active workspace id for one user for first-chat funnel writes.
func (parseS *chatServer) parseResolveFirstChatWorkspaceIDForUser(parseUserID int64) int64 {
	if parseS == nil || parseS.store == nil || parseUserID <= 0 {
		return 0
	}
	parseMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByUser(parseUserID)
	if parseErr != nil {
		return 0
	}
	for _, parseMembershipRow := range parseMembershipRows {
		if parseMembershipRow.WorkspaceID <= 0 {
			continue
		}
		if parseIsWorkspaceMembershipActive(parseMembershipRow.Status) {
			return parseMembershipRow.WorkspaceID
		}
	}
	for _, parseMembershipRow := range parseMembershipRows {
		if parseMembershipRow.WorkspaceID > 0 {
			return parseMembershipRow.WorkspaceID
		}
	}
	return 0
}

// parseEnsureFirstChatFunnelExperimentKey ensures the typed first-chat funnel experiment row exists and returns its key.
func (parseS *chatServer) parseEnsureFirstChatFunnelExperimentKey() string {
	if parseS == nil || parseS.store == nil {
		return ""
	}
	if parseErr := parseS.store.parseUpsertExperiment(parseExperimentWrite{
		ExperimentKey: parseFirstChatFunnelExperimentKey,
		Name:          "Visit-To-First-Chat Funnel",
		Status:        "active",
		VariantsJSON:  `[{"key":"default","label":"Default"}]`,
		AudienceJSON:  `{}`,
		StartAt:       "",
		EndAt:         "",
	}); parseErr != nil {
		return ""
	}
	return parseFirstChatFunnelExperimentKey
}

// parseTrackFirstChatFunnelShellRequest records typed first-chat funnel events for one incoming shell request.
func (parseS *chatServer) parseTrackFirstChatFunnelShellRequest(parseR *http.Request) {
	if parseS == nil || parseR == nil {
		return
	}
	parseStepKeys := parseCollectFirstChatFunnelStepsForPath(parseR.URL.Path)
	if len(parseStepKeys) == 0 {
		return
	}
	parseUserID := int64(0)
	if parseS.authManager != nil {
		if parseUser, parseAuthenticated := parseS.authManager.parseAuthenticatedUserFromRequest(parseR); parseAuthenticated && parseUser.ID > 0 {
			parseUserID = parseUser.ID
		}
	}
	parseRoute := filepath.ToSlash(filepath.Clean("/" + strings.TrimLeft(strings.TrimSpace(parseR.URL.Path), "/")))
	for _, parseStepKey := range parseStepKeys {
		parseS.parseTrackFirstChatFunnelStepWithRequest(parseR, parseUserID, parseStepKey, map[string]any{
			"route": parseRoute,
		})
	}
}
