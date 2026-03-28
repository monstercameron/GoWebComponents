package app

import (
	"strings"
)

// parseTrackAdminAuditEvent persists one admin dashboard/operator audit event when a scoped workspace is available.
func (parseS *chatServer) parseTrackAdminAuditEvent(parseScope parseAdminAccessScope, parseEventType, parseTargetType, parseTargetID, parseSummary, parsePayloadJSON string, parseWorkspaceIDHint int64) {
	if parseS == nil || parseS.store == nil {
		return
	}
	parseEventType = strings.TrimSpace(parseEventType)
	if parseScope.adminUserID <= 0 || parseEventType == "" {
		return
	}
	parseWorkspaceID, parseErr := parseS.parseResolveAdminAuditWorkspaceID(parseScope, parseWorkspaceIDHint)
	if parseErr != nil {
		if parseS.logger != nil {
			parseS.logger.Warn("rpc.admin audit workspace resolve failed", "error", parseErr.Error(), "event_type", parseEventType, "admin_user_id", parseScope.adminUserID)
		}
		return
	}
	if parseWorkspaceID <= 0 {
		return
	}
	if _, parseErr = parseS.store.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseScope.adminUserID,
		WorkspaceID: parseWorkspaceID,
		EventType:   parseEventType,
		TargetType:  strings.TrimSpace(parseTargetType),
		TargetID:    strings.TrimSpace(parseTargetID),
		Summary:     strings.TrimSpace(parseSummary),
		PayloadJSON: parseNormalizeBillingJSON(parsePayloadJSON),
	}); parseErr != nil && parseS.logger != nil {
		parseS.logger.Warn("rpc.admin audit write failed", "error", parseErr.Error(), "event_type", parseEventType, "admin_user_id", parseScope.adminUserID, "workspace_id", parseWorkspaceID)
	}
}

// parseResolveAdminAuditWorkspaceID resolves one workspace id for admin audit event storage.
func (parseS *chatServer) parseResolveAdminAuditWorkspaceID(parseScope parseAdminAccessScope, parseWorkspaceIDHint int64) (int64, error) {
	if parseWorkspaceIDHint > 0 {
		return parseWorkspaceIDHint, nil
	}
	if !parseScope.isPlatformScope {
		return parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs), nil
	}
	if parseS == nil || parseS.store == nil {
		return 0, nil
	}
	parseWorkspaceRows, parseErr := parseS.store.parseListWorkspaces(1)
	if parseErr != nil {
		return 0, parseErr
	}
	if len(parseWorkspaceRows) == 0 {
		if parseScope.adminUserID <= 0 {
			return 0, nil
		}
		if parseErr = parseS.store.parseUpsertWorkspace(parseWorkspaceWrite{
			WorkspaceKey: "platform-audit",
			Slug:         "platform-audit",
			Name:         "Platform Audit",
			PlanCode:     "free",
			Status:       "active",
			OwnerUserID:  parseScope.adminUserID,
			SettingsJSON: "{}",
		}); parseErr != nil {
			return 0, parseErr
		}
		parseWorkspaceRows, parseErr = parseS.store.parseListWorkspaces(25)
		if parseErr != nil {
			return 0, parseErr
		}
		for _, parseWorkspaceRow := range parseWorkspaceRows {
			if parseWorkspaceRow.WorkspaceKey == "platform-audit" {
				return parseWorkspaceRow.ID, nil
			}
		}
		return 0, nil
	}
	return parseWorkspaceRows[0].ID, nil
}

// parseResolveAdminScopeWorkspaceID selects one stable workspace id from one workspace-admin scope map.
func parseResolveAdminScopeWorkspaceID(parseWorkspaceIDs map[int64]struct{}) int64 {
	parseWorkspaceID := int64(0)
	for parseCandidateWorkspaceID := range parseWorkspaceIDs {
		if parseCandidateWorkspaceID <= 0 {
			continue
		}
		if parseWorkspaceID == 0 || parseCandidateWorkspaceID < parseWorkspaceID {
			parseWorkspaceID = parseCandidateWorkspaceID
		}
	}
	return parseWorkspaceID
}
