package app

import (
	"strconv"
	"strings"
)

const parseExternalAuthAuditEventLoginStart = "auth.external.login_start"
const parseExternalAuthAuditEventCallbackSuccess = "auth.external.callback_success"
const parseExternalAuthAuditEventCallbackFailure = "auth.external.callback_failure"
const parseExternalAuthAuditEventIdentityLinked = "auth.external.identity_linked"
const parseExternalAuthAuditEventIdentityUnlinked = "auth.external.identity_unlinked"
const parseExternalAuthAuditEventPolicyDeniedLogin = "auth.external.policy_denied_login"
const parseExternalAuthAuditEventWorkspaceSSOEnforcement = "auth.external.workspace_sso_enforcement"

type parseExternalAuthAuditWrite struct {
	ParseActorUserID int64
	ParseWorkspaceID int64
	ParseEventType   string
	ParseTargetType  string
	ParseTargetID    string
	ParseSummary     string
	ParsePayloadJSON string
}

// parseTrackExternalAuthAuditEvent writes one external-auth audit row when event data is valid.
func (parseS *chatServer) parseTrackExternalAuthAuditEvent(parseWrite parseExternalAuthAuditWrite) {
	if parseS == nil || parseS.store == nil {
		return
	}
	parseActorUserID := parseS.parseResolveExternalAuthAuditActorUserID(parseWrite.ParseActorUserID, parseWrite.ParseWorkspaceID)
	if parseActorUserID <= 0 {
		if parseS.logger != nil {
			parseS.logger.Warn("external auth audit write skipped: actor unresolved", "event_type", parseWrite.ParseEventType, "workspace_id", parseWrite.ParseWorkspaceID)
		}
		return
	}
	parseEventType := parseNormalizeExternalAuthAuditEventType(parseWrite.ParseEventType)
	if parseEventType == "" {
		return
	}
	parseSummary := strings.TrimSpace(parseWrite.ParseSummary)
	if parseSummary == "" {
		parseSummary = parseEventType
	}
	if _, parseErr := parseS.store.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseActorUserID,
		WorkspaceID: parseWrite.ParseWorkspaceID,
		EventType:   parseEventType,
		TargetType:  strings.TrimSpace(parseWrite.ParseTargetType),
		TargetID:    strings.TrimSpace(parseWrite.ParseTargetID),
		Summary:     parseSummary,
		PayloadJSON: parseWrite.ParsePayloadJSON,
	}); parseErr != nil && parseS.logger != nil {
		parseS.logger.Warn("external auth audit write failed", "error", parseErr.Error(), "event_type", parseEventType, "workspace_id", parseWrite.ParseWorkspaceID, "actor_user_id", parseWrite.ParseActorUserID)
	}
}

// parseResolveExternalAuthAuditActorUserID resolves one valid actor user id for external-auth audit writes.
func (parseS *chatServer) parseResolveExternalAuthAuditActorUserID(parseActorUserID, parseWorkspaceID int64) int64 {
	if parseActorUserID > 0 {
		return parseActorUserID
	}
	if parseS == nil || parseS.store == nil || parseWorkspaceID <= 0 {
		return 0
	}
	parseWorkspaceRow, isParseWorkspaceFound, parseErr := parseS.store.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil || !isParseWorkspaceFound || parseWorkspaceRow.OwnerUserID <= 0 {
		return 0
	}
	return parseWorkspaceRow.OwnerUserID
}

// parseListExternalAuthAuditLogs returns recent external-auth audit rows newest-first, optionally scoped to one workspace.
func (parseS *Store) parseListExternalAuthAuditLogs(parseWorkspaceID, parseLimit int64) ([]parseAuditLogRow, error) {
	if parseLimit <= 0 {
		parseLimit = 25
	}
	parseScanRows, parseErr := parseS.parseListAuditLogs(parseAdminControlScanLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseAuditRows := make([]parseAuditLogRow, 0, len(parseScanRows))
	for _, parseAuditRow := range parseScanRows {
		if parseNormalizeExternalAuthAuditEventType(parseAuditRow.EventType) == "" {
			continue
		}
		if parseWorkspaceID > 0 && parseAuditRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseAuditRows = append(parseAuditRows, parseAuditRow)
		if int64(len(parseAuditRows)) >= parseLimit {
			break
		}
	}
	return parseAuditRows, nil
}

// parseBuildExternalAuthAuditTargetIDFromUser formats one user id for audit target storage.
func parseBuildExternalAuthAuditTargetIDFromUser(parseUserID int64) string {
	if parseUserID <= 0 {
		return ""
	}
	return strconv.FormatInt(parseUserID, 10)
}

// parseTrackExternalAuthIdentityUnlinked writes one typed identity-unlinked audit row.
func (parseS *chatServer) parseTrackExternalAuthIdentityUnlinked(parseActorUserID, parseWorkspaceID, parseUserID int64, parseProviderKey string) {
	parseS.parseTrackExternalAuthAuditEvent(parseExternalAuthAuditWrite{
		ParseActorUserID: parseActorUserID,
		ParseWorkspaceID: parseWorkspaceID,
		ParseEventType:   parseExternalAuthAuditEventIdentityUnlinked,
		ParseTargetType:  "user",
		ParseTargetID:    parseBuildExternalAuthAuditTargetIDFromUser(parseUserID),
		ParseSummary:     "External identity unlinked for provider " + strings.TrimSpace(parseProviderKey),
		ParsePayloadJSON: "{}",
	})
}

// parseNormalizeExternalAuthAuditEventType normalizes one supported external-auth audit event key.
func parseNormalizeExternalAuthAuditEventType(parseEventType string) string {
	switch strings.TrimSpace(strings.ToLower(parseEventType)) {
	case parseExternalAuthAuditEventLoginStart,
		parseExternalAuthAuditEventCallbackSuccess,
		parseExternalAuthAuditEventCallbackFailure,
		parseExternalAuthAuditEventIdentityLinked,
		parseExternalAuthAuditEventIdentityUnlinked,
		parseExternalAuthAuditEventPolicyDeniedLogin,
		parseExternalAuthAuditEventWorkspaceSSOEnforcement:
		return strings.TrimSpace(strings.ToLower(parseEventType))
	default:
		return ""
	}
}
