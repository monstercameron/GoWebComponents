package app

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseWorkspaceAdminDetail struct {
	parseWorkspaceRow   parseWorkspaceRow
	parseMembershipRows []parseWorkspaceMembershipRow
	parseAPIKeyRows     []parseAPIKeyRow
	parseWebhookRows    []parseWebhookEndpointRow
	parseAuditLogRows   []parseAuditLogRow
}

// parseRequireWorkspaceAdminWorkspaceScope resolves admin scope and enforces one target workspace boundary.
func (parseS *chatServer) parseRequireWorkspaceAdminWorkspaceScope(parseCtx context.Context, parseWorkspaceID int64, parseSliceKey string) (parseAdminAccessScope, error) {
	if parseWorkspaceID <= 0 {
		return parseAdminAccessScope{}, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, parseSliceKey)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseErr
	}
	if parseScope.isPlatformScope {
		return parseScope, nil
	}
	if _, hasParseWorkspace := parseScope.workspaceIDs[parseWorkspaceID]; !hasParseWorkspace {
		return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "workspace admin scope required")
	}
	return parseScope, nil
}

// parseGetWorkspaceAdminDetail loads one typed workspace-admin detail snapshot scoped to one target workspace.
func (parseS *chatServer) parseGetWorkspaceAdminDetail(parseCtx context.Context, parseWorkspaceID int64, parseLimit int32) (parseWorkspaceAdminDetail, error) {
	parseScope, parseErr := parseS.parseRequireWorkspaceAdminWorkspaceScope(parseCtx, parseWorkspaceID, "dashboard.workspace.detail")
	if parseErr != nil {
		return parseWorkspaceAdminDetail{}, parseErr
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	if parseS == nil || parseS.store == nil {
		return parseWorkspaceAdminDetail{}, status.Error(codes.Unavailable, "store unavailable")
	}
	parseWorkspaceRow, isParseFound, parseErr := parseS.store.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil {
		return parseWorkspaceAdminDetail{}, status.Errorf(codes.Internal, "get workspace detail: %v", parseErr)
	}
	if !isParseFound {
		return parseWorkspaceAdminDetail{}, status.Error(codes.NotFound, "workspace not found")
	}

	parseMembershipRows, parseErr := parseS.store.parseListWorkspaceMembershipsByWorkspace(parseWorkspaceID)
	if parseErr != nil {
		return parseWorkspaceAdminDetail{}, status.Errorf(codes.Internal, "list workspace memberships: %v", parseErr)
	}
	parseMembershipRows = parseLimitWorkspaceMembershipRows(parseMembershipRows, parseLimit)

	parseAPIKeyRows, parseErr := parseS.store.parseListAPIKeys(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseWorkspaceAdminDetail{}, status.Errorf(codes.Internal, "list api keys: %v", parseErr)
	}
	parseAPIKeyRows = parseFilterAPIKeyRowsByWorkspace(parseAPIKeyRows, parseWorkspaceID, parseLimit)

	parseWebhookRows, parseErr := parseS.store.parseListWebhookEndpoints(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseWorkspaceAdminDetail{}, status.Errorf(codes.Internal, "list webhook endpoints: %v", parseErr)
	}
	parseWebhookRows = parseFilterWebhookRowsByWorkspace(parseWebhookRows, parseWorkspaceID, parseLimit)

	parseAuditLogRows, parseErr := parseS.store.parseListAuditLogs(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseWorkspaceAdminDetail{}, status.Errorf(codes.Internal, "list audit logs: %v", parseErr)
	}
	parseAuditLogRows = parseFilterAuditRowsByWorkspace(parseAuditLogRows, parseWorkspaceID, parseLimit)

	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"workspace",
		fmt.Sprintf("%d", parseWorkspaceID),
		"Workspace admin detail viewed",
		"{}",
		parseWorkspaceID,
	)
	return parseWorkspaceAdminDetail{
		parseWorkspaceRow:   parseWorkspaceRow,
		parseMembershipRows: parseMembershipRows,
		parseAPIKeyRows:     parseAPIKeyRows,
		parseWebhookRows:    parseWebhookRows,
		parseAuditLogRows:   parseAuditLogRows,
	}, nil
}

// parseListWorkspaceAdminMembershipReview returns typed membership rows for one workspace-admin review scope.
func (parseS *chatServer) parseListWorkspaceAdminMembershipReview(parseCtx context.Context, parseWorkspaceID int64, parseLimit int32) ([]parseWorkspaceMembershipRow, error) {
	parseDetail, parseErr := parseS.parseGetWorkspaceAdminDetail(parseCtx, parseWorkspaceID, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	return parseDetail.parseMembershipRows, nil
}

// parseSuspendWorkspaceByAdmin applies one typed workspace suspension mutation scoped to one admin caller.
func (parseS *chatServer) parseSuspendWorkspaceByAdmin(parseCtx context.Context, parseWorkspaceID int64, isParseConfirmed bool, parseReason string) error {
	parseTarget := parseAdminMutationTarget{
		workspaceID: parseWorkspaceID,
		isConfirmed: isParseConfirmed,
		reason:      parseReason,
	}
	return parseS.parseExecuteAdminMutationAction(parseCtx, parseAdminMutationSuspendWorkspace, parseTarget)
}

// parseRestoreWorkspaceByAdmin applies one typed workspace restore mutation scoped to one admin caller.
func (parseS *chatServer) parseRestoreWorkspaceByAdmin(parseCtx context.Context, parseWorkspaceID int64, isParseConfirmed bool, parseReason string) error {
	parseTarget := parseAdminMutationTarget{
		workspaceID: parseWorkspaceID,
		isConfirmed: isParseConfirmed,
		reason:      parseReason,
	}
	return parseS.parseExecuteAdminMutationAction(parseCtx, parseAdminMutationRestoreWorkspace, parseTarget)
}

// parseRevokeWorkspaceAPIKeyByAdmin revokes one workspace API key when the target workspace is in scope.
func (parseS *chatServer) parseRevokeWorkspaceAPIKeyByAdmin(parseCtx context.Context, parseWorkspaceID int64, parseKeyID string, parseReason string) error {
	parseScope, parseErr := parseS.parseRequireWorkspaceAdminWorkspaceScope(parseCtx, parseWorkspaceID, "dashboard.workspace.api_keys.revoke")
	if parseErr != nil {
		return parseErr
	}
	parseKeyID = strings.TrimSpace(parseKeyID)
	if parseKeyID == "" {
		return status.Error(codes.InvalidArgument, "api key id is required")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		return status.Error(codes.InvalidArgument, "revoke reason is required")
	}
	if parseS == nil || parseS.store == nil {
		return status.Error(codes.Unavailable, "store unavailable")
	}
	parseAPIKeyRows, parseErr := parseS.store.parseListAPIKeys(parseAdminScopedScanLimit)
	if parseErr != nil {
		return status.Errorf(codes.Internal, "list api keys: %v", parseErr)
	}
	parseFoundKey := false
	for _, parseAPIKeyRow := range parseAPIKeyRows {
		if parseAPIKeyRow.KeyID != parseKeyID {
			continue
		}
		if parseAPIKeyRow.WorkspaceID != parseWorkspaceID {
			return status.Error(codes.PermissionDenied, "api key outside workspace scope")
		}
		parseFoundKey = true
		break
	}
	if !parseFoundKey {
		return status.Error(codes.NotFound, "api key not found")
	}
	if parseErr = parseS.store.parseRevokeAPIKey(parseKeyID); parseErr != nil {
		return status.Errorf(codes.Internal, "revoke api key: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.workspace.api_key.revoked",
		"api_key",
		parseKeyID,
		"Workspace API key revoked: "+parseReason,
		"{}",
		parseWorkspaceID,
	)
	return nil
}

// parsePauseWorkspaceWebhookByAdmin disables one workspace webhook endpoint when the target workspace is in scope.
func (parseS *chatServer) parsePauseWorkspaceWebhookByAdmin(parseCtx context.Context, parseWorkspaceID int64, parseEndpointID int64, parseReason string) error {
	parseScope, parseErr := parseS.parseRequireWorkspaceAdminWorkspaceScope(parseCtx, parseWorkspaceID, "dashboard.workspace.webhooks.pause")
	if parseErr != nil {
		return parseErr
	}
	if parseEndpointID <= 0 {
		return status.Error(codes.InvalidArgument, "webhook endpoint id is required")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		return status.Error(codes.InvalidArgument, "pause reason is required")
	}
	if parseS == nil || parseS.store == nil {
		return status.Error(codes.Unavailable, "store unavailable")
	}
	parseWebhookRows, parseErr := parseS.store.parseListWebhookEndpoints(parseAdminScopedScanLimit)
	if parseErr != nil {
		return status.Errorf(codes.Internal, "list webhook endpoints: %v", parseErr)
	}
	var parseEndpointRow parseWebhookEndpointRow
	isParseEndpointFound := false
	for _, parseCandidateWebhookRow := range parseWebhookRows {
		if parseCandidateWebhookRow.ID != parseEndpointID {
			continue
		}
		if parseCandidateWebhookRow.WorkspaceID != parseWorkspaceID {
			return status.Error(codes.PermissionDenied, "webhook endpoint outside workspace scope")
		}
		parseEndpointRow = parseCandidateWebhookRow
		isParseEndpointFound = true
		break
	}
	if !isParseEndpointFound {
		return status.Error(codes.NotFound, "webhook endpoint not found")
	}
	if parseErr = parseS.store.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID:    parseWorkspaceID,
		Label:          parseEndpointRow.Label,
		TargetURL:      parseEndpointRow.TargetURL,
		SecretHash:     parseEndpointRow.SecretHash,
		EventsJSON:     parseEndpointRow.EventsJSON,
		IsEnabled:      false,
		LastDeliveryAt: parseEndpointRow.LastDeliveryAt,
		FailureCount:   parseEndpointRow.FailureCount,
	}); parseErr != nil {
		return status.Errorf(codes.Internal, "pause workspace webhook endpoint: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.workspace.webhook.paused",
		"webhook_endpoint",
		fmt.Sprintf("%d", parseEndpointID),
		"Workspace webhook endpoint paused: "+parseReason,
		"{}",
		parseWorkspaceID,
	)
	return nil
}

// parseLimitWorkspaceMembershipRows truncates workspace membership rows to one RPC-safe limit.
func parseLimitWorkspaceMembershipRows(parseRows []parseWorkspaceMembershipRow, parseLimit int32) []parseWorkspaceMembershipRow {
	if parseLimit <= 0 || len(parseRows) <= int(parseLimit) {
		return parseRows
	}
	return parseRows[:parseLimit]
}

// parseFilterAPIKeyRowsByWorkspace filters API key rows to one workspace id and optional limit.
func parseFilterAPIKeyRowsByWorkspace(parseRows []parseAPIKeyRow, parseWorkspaceID int64, parseLimit int32) []parseAPIKeyRow {
	parseFilteredRows := make([]parseAPIKeyRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
		if parseLimit > 0 && len(parseFilteredRows) >= int(parseLimit) {
			break
		}
	}
	return parseFilteredRows
}

// parseFilterWebhookRowsByWorkspace filters webhook rows to one workspace id and optional limit.
func parseFilterWebhookRowsByWorkspace(parseRows []parseWebhookEndpointRow, parseWorkspaceID int64, parseLimit int32) []parseWebhookEndpointRow {
	parseFilteredRows := make([]parseWebhookEndpointRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
		if parseLimit > 0 && len(parseFilteredRows) >= int(parseLimit) {
			break
		}
	}
	return parseFilteredRows
}

// parseFilterAuditRowsByWorkspace filters audit rows to one workspace id and optional limit.
func parseFilterAuditRowsByWorkspace(parseRows []parseAuditLogRow, parseWorkspaceID int64, parseLimit int32) []parseAuditLogRow {
	parseFilteredRows := make([]parseAuditLogRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
		if parseLimit > 0 && len(parseFilteredRows) >= int(parseLimit) {
			break
		}
	}
	return parseFilteredRows
}
