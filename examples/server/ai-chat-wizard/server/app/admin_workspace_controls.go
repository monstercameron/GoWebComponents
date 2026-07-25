package app

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminWorkspaceEntry maps one workspace row into protobuf form.
func parseBuildAdminWorkspaceEntry(parseRow parseWorkspaceRow) *chatpb.WorkspaceEntry {
	return &chatpb.WorkspaceEntry{
		Id:           parseRow.ID,
		WorkspaceKey: parseRow.WorkspaceKey,
		Slug:         parseRow.Slug,
		Name:         parseRow.Name,
		PlanCode:     parseRow.PlanCode,
		Status:       parseRow.Status,
		OwnerUserId:  parseRow.OwnerUserID,
		SettingsJson: parseRow.SettingsJSON,
		CreatedAt:    parseRow.CreatedAt,
		UpdatedAt:    parseRow.UpdatedAt,
	}
}

// parseBuildAdminWorkspaceMembershipEntry maps one workspace membership row into protobuf form.
func parseBuildAdminWorkspaceMembershipEntry(parseRow parseWorkspaceMembershipRow) *chatpb.WorkspaceMembershipEntry {
	return &chatpb.WorkspaceMembershipEntry{
		Id:              parseRow.ID,
		WorkspaceId:     parseRow.WorkspaceID,
		UserId:          parseRow.UserID,
		RoleKey:         parseRow.RoleKey,
		Status:          parseRow.Status,
		InvitedByUserId: parseRow.InvitedByUserID,
		CreatedAt:       parseRow.CreatedAt,
		UpdatedAt:       parseRow.UpdatedAt,
	}
}

// parseBuildAdminWorkspaceInvitationEntry maps one workspace invitation row into protobuf form.
func parseBuildAdminWorkspaceInvitationEntry(parseRow parseWorkspaceInvitationRow) *chatpb.WorkspaceInvitationEntry {
	return &chatpb.WorkspaceInvitationEntry{
		Id:                  parseRow.ID,
		WorkspaceId:         parseRow.WorkspaceID,
		Email:               parseRow.Email,
		RoleKey:             parseRow.RoleKey,
		InvitationTokenHash: parseRow.InvitationTokenHash,
		InvitedByUserId:     parseRow.InvitedByUserID,
		Status:              parseRow.Status,
		ExpiresAt:           parseRow.ExpiresAt,
		AcceptedAt:          parseRow.AcceptedAt,
		CreatedAt:           parseRow.CreatedAt,
		UpdatedAt:           parseRow.UpdatedAt,
	}
}

// parseBuildAdminAPIKeyEntry maps one API-key row into protobuf form.
func parseBuildAdminAPIKeyEntry(parseRow parseAPIKeyRow) *chatpb.APIKeyEntry {
	return &chatpb.APIKeyEntry{
		Id:          parseRow.ID,
		KeyId:       parseRow.KeyID,
		WorkspaceId: parseRow.WorkspaceID,
		UserId:      parseRow.UserID,
		Label:       parseRow.Label,
		KeyPrefix:   parseRow.KeyPrefix,
		ScopesJson:  parseRow.ScopesJSON,
		LastUsedAt:  parseRow.LastUsedAt,
		RevokedAt:   parseRow.RevokedAt,
		CreatedAt:   parseRow.CreatedAt,
	}
}

// parseBuildAdminWebhookEndpointEntry maps one webhook endpoint row into protobuf form.
func parseBuildAdminWebhookEndpointEntry(parseRow parseWebhookEndpointRow) *chatpb.WebhookEndpointEntry {
	return &chatpb.WebhookEndpointEntry{
		Id:             parseRow.ID,
		WorkspaceId:    parseRow.WorkspaceID,
		Label:          parseRow.Label,
		TargetUrl:      parseRow.TargetURL,
		EventsJson:     parseRow.EventsJSON,
		IsEnabled:      parseRow.IsEnabled,
		LastDeliveryAt: parseRow.LastDeliveryAt,
		FailureCount:   parseRow.FailureCount,
		CreatedAt:      parseRow.CreatedAt,
		UpdatedAt:      parseRow.UpdatedAt,
	}
}

// parseRequireAdminWorkspaceScope enforces that one workspace id belongs to one caller scope when not in platform scope.
func parseRequireAdminWorkspaceScope(parseScope parseAdminAccessScope, parseWorkspaceID int64) error {
	if parseWorkspaceID <= 0 {
		return status.Error(codes.InvalidArgument, "workspace id is required")
	}
	if parseScope.isPlatformScope {
		return nil
	}
	if _, hasParseWorkspace := parseScope.workspaceIDs[parseWorkspaceID]; !hasParseWorkspace {
		return status.Error(codes.PermissionDenied, "target workspace outside workspace-admin scope")
	}
	return nil
}

// parseRequireAdminWorkspaceActionConfirmation enforces explicit confirmation and one non-empty reason for workspace-scoped destructive actions.
func parseRequireAdminWorkspaceActionConfirmation(isParseConfirmed bool, parseReason string) (string, error) {
	if !isParseConfirmed {
		return "", status.Error(codes.InvalidArgument, "workspace mutation confirmation is required")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		return "", status.Error(codes.InvalidArgument, "workspace mutation reason is required")
	}
	return parseReason, nil
}

// parseBuildAdminWorkspaceMutationResponse builds one typed workspace mutation response.
func (parseS *chatServer) parseBuildAdminWorkspaceMutationResponse(parseWorkspaceID int64, parseFallbackStatus string) (*chatpb.AdminWorkspaceMutationResponse, error) {
	parseStatusValue := strings.TrimSpace(parseFallbackStatus)
	if parseS == nil || parseS.store == nil || parseWorkspaceID <= 0 {
		if parseStatusValue == "" {
			parseStatusValue = "active"
		}
		return &chatpb.AdminWorkspaceMutationResponse{
			WorkspaceId: parseWorkspaceID,
			Status:      parseStatusValue,
		}, nil
	}
	parseWorkspaceRow, hasParseWorkspace, parseErr := parseS.store.parseGetWorkspaceByID(parseWorkspaceID)
	if parseErr != nil {
		return nil, parseErr
	}
	if hasParseWorkspace && strings.TrimSpace(parseWorkspaceRow.Status) != "" {
		parseStatusValue = strings.TrimSpace(parseWorkspaceRow.Status)
	}
	if parseStatusValue == "" {
		parseStatusValue = "active"
	}
	return &chatpb.AdminWorkspaceMutationResponse{
		WorkspaceId: parseWorkspaceID,
		Status:      parseStatusValue,
	}, nil
}

// GetAdminWorkspaceDetail returns one typed workspace detail payload for authenticated admin callers.
func (parseS *chatServer) GetAdminWorkspaceDetail(parseCtx context.Context, parseReq *chatpb.GetAdminWorkspaceDetailRequest) (*chatpb.GetAdminWorkspaceDetailResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminWorkspaceDetail"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.workspace.detail")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseLookbackDays int32
	var parseLimit int32
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseLimit = parseClampAdminListLimit(parseLimit)
	if parseErr = parseRequireAdminWorkspaceScope(parseScope, parseWorkspaceID); parseErr != nil {
		return nil, parseErr
	}
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.GetAdminWorkspaceDetail: drill-down fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.Int64("workspace_id", parseWorkspaceID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"workspace",
		strconv.FormatInt(parseWorkspaceID, 10),
		"Admin workspace detail drill-down viewed",
		"{}",
		parseWorkspaceID,
	)
	parseResponse := &chatpb.GetAdminWorkspaceDetailResponse{
		Detail: &chatpb.AdminWorkspaceDetail{},
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminWorkspaceDetail: store unavailable",
			slog.String("next_action", "restore store availability before retrying workspace detail"),
		)
		return parseResponse, nil
	}
	parseWorkspaceRow, hasParseWorkspace, parseErr := parseS.store.parseGetAdminWorkspaceByWorkspaceID(parseWorkspaceID)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminWorkspaceDetail: workspace query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get admin workspace detail: %v", parseErr)
	}
	if !hasParseWorkspace {
		return nil, status.Error(codes.NotFound, "workspace not found")
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseMembershipRows, parseErr := parseS.store.parseListAdminWorkspaceMembershipsByWorkspace(parseWorkspaceID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminWorkspaceDetail: membership query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin workspace memberships: %v", parseErr)
	}
	parseInvitationRows, parseErr := parseS.store.parseListAdminWorkspaceInvitationsByWorkspace(parseWorkspaceID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminWorkspaceDetail: invitation query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin workspace invitations: %v", parseErr)
	}
	parseAPIKeyRows, parseErr := parseS.store.parseListAdminAPIKeysByWorkspace(parseWorkspaceID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminWorkspaceDetail: api key query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin workspace api keys: %v", parseErr)
	}
	parseWebhookRows, parseErr := parseS.store.parseListAdminWebhookEndpointsByWorkspace(parseWorkspaceID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminWorkspaceDetail: webhook query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin workspace webhooks: %v", parseErr)
	}
	parseAuditRows, parseErr := parseS.store.parseListAdminAuditLogsByWorkspace(parseWorkspaceID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminWorkspaceDetail: audit query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin workspace audit logs: %v", parseErr)
	}
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseFilteredAuditRows := make([]parseAuditLogRow, 0, len(parseAuditRows))
	for _, parseAuditRow := range parseAuditRows {
		if !parseHasAdminTimestampAtOrAfter(parseAuditRow.CreatedAt, parseSince) {
			continue
		}
		parseFilteredAuditRows = append(parseFilteredAuditRows, parseAuditRow)
		if len(parseFilteredAuditRows) >= int(parseLimit) {
			break
		}
	}

	parseResponse.Detail.Workspace = parseBuildAdminWorkspaceEntry(parseWorkspaceRow)
	parseResponse.Detail.Memberships = make([]*chatpb.WorkspaceMembershipEntry, 0, len(parseMembershipRows))
	for _, parseMembershipRow := range parseMembershipRows {
		parseResponse.Detail.Memberships = append(parseResponse.Detail.Memberships, parseBuildAdminWorkspaceMembershipEntry(parseMembershipRow))
	}
	parseResponse.Detail.Invitations = make([]*chatpb.WorkspaceInvitationEntry, 0, len(parseInvitationRows))
	for _, parseInvitationRow := range parseInvitationRows {
		parseResponse.Detail.Invitations = append(parseResponse.Detail.Invitations, parseBuildAdminWorkspaceInvitationEntry(parseInvitationRow))
	}
	parseResponse.Detail.ApiKeys = make([]*chatpb.APIKeyEntry, 0, len(parseAPIKeyRows))
	for _, parseAPIKeyRow := range parseAPIKeyRows {
		parseResponse.Detail.ApiKeys = append(parseResponse.Detail.ApiKeys, parseBuildAdminAPIKeyEntry(parseAPIKeyRow))
	}
	parseResponse.Detail.WebhookEndpoints = make([]*chatpb.WebhookEndpointEntry, 0, len(parseWebhookRows))
	for _, parseWebhookRow := range parseWebhookRows {
		parseResponse.Detail.WebhookEndpoints = append(parseResponse.Detail.WebhookEndpoints, parseBuildAdminWebhookEndpointEntry(parseWebhookRow))
	}
	parseResponse.Detail.RecentAuditLogs = make([]*chatpb.AuditLogEntry, 0, len(parseFilteredAuditRows))
	for _, parseAuditRow := range parseFilteredAuditRows {
		parseResponse.Detail.RecentAuditLogs = append(parseResponse.Detail.RecentAuditLogs, parseBuildAdminAuditLogEntry(parseAuditRow))
	}
	parseLogger.Info(
		"rpc.GetAdminWorkspaceDetail: complete",
		slog.Int64("workspace_id", parseWorkspaceID),
		slog.Int("memberships", len(parseResponse.Detail.Memberships)),
		slog.Int("invitations", len(parseResponse.Detail.Invitations)),
		slog.Int("api_keys", len(parseResponse.Detail.ApiKeys)),
		slog.Int("webhooks", len(parseResponse.Detail.WebhookEndpoints)),
		slog.Int("audit_logs", len(parseResponse.Detail.RecentAuditLogs)),
	)
	return parseResponse, nil
}

// SuspendAdminWorkspace applies one typed workspace-suspend mutation.
func (parseS *chatServer) SuspendAdminWorkspace(parseCtx context.Context, parseReq *chatpb.AdminWorkspaceMutationRequest) (*chatpb.AdminWorkspaceMutationResponse, error) {
	var parseTarget parseAdminMutationTarget
	if parseReq != nil {
		parseTarget.workspaceID = parseReq.GetWorkspaceId()
		parseTarget.isConfirmed = parseReq.GetConfirm()
		parseTarget.reason = parseReq.GetReason()
	}
	if parseTarget.workspaceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	if parseErr := parseS.parseExecuteAdminMutationAction(parseCtx, parseAdminMutationSuspendWorkspace, parseTarget); parseErr != nil {
		return nil, parseErr
	}
	parseResp, parseErr := parseS.parseBuildAdminWorkspaceMutationResponse(parseTarget.workspaceID, "suspended")
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "build suspend workspace response: %v", parseErr)
	}
	return parseResp, nil
}

// RestoreAdminWorkspace applies one typed workspace-restore mutation.
func (parseS *chatServer) RestoreAdminWorkspace(parseCtx context.Context, parseReq *chatpb.AdminWorkspaceMutationRequest) (*chatpb.AdminWorkspaceMutationResponse, error) {
	var parseTarget parseAdminMutationTarget
	if parseReq != nil {
		parseTarget.workspaceID = parseReq.GetWorkspaceId()
		parseTarget.isConfirmed = parseReq.GetConfirm()
		parseTarget.reason = parseReq.GetReason()
		parseTarget.isRestoreWorkspaceAPIKeys = parseReq.GetRestoreApiKeys()
		parseTarget.isRestoreWebhookEndpoints = parseReq.GetRestoreWebhookEndpoints()
		parseTarget.isRestoreBackgroundJobs = parseReq.GetRestoreBackgroundJobs()
	}
	if parseTarget.workspaceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	if parseErr := parseS.parseExecuteAdminMutationAction(parseCtx, parseAdminMutationRestoreWorkspace, parseTarget); parseErr != nil {
		return nil, parseErr
	}
	parseResp, parseErr := parseS.parseBuildAdminWorkspaceMutationResponse(parseTarget.workspaceID, "active")
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "build restore workspace response: %v", parseErr)
	}
	return parseResp, nil
}

// RevokeWorkspaceAPIKey revokes one workspace API key in admin scope.
func (parseS *chatServer) RevokeWorkspaceAPIKey(parseCtx context.Context, parseReq *chatpb.RevokeWorkspaceAPIKeyRequest) (*chatpb.RevokeWorkspaceAPIKeyResponse, error) {
	var parseWorkspaceID int64
	var parseKeyID string
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseKeyID = parseReq.GetKeyId()
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	parseKeyID = strings.TrimSpace(parseKeyID)
	if parseWorkspaceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	if parseKeyID == "" {
		return nil, status.Error(codes.InvalidArgument, "key id is required")
	}
	parseReason, parseErr := parseRequireAdminWorkspaceActionConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.workspace.api_key.revoke")
	if parseErr != nil {
		return nil, parseErr
	}
	if parseErr = parseRequireAdminWorkspaceScope(parseScope, parseWorkspaceID); parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseAPIKeyRow, hasParseAPIKey, parseErr := parseS.store.parseGetAdminAPIKeyByKeyID(parseKeyID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get api key by key id: %v", parseErr)
	}
	if !hasParseAPIKey {
		return nil, status.Error(codes.NotFound, "api key not found")
	}
	if parseAPIKeyRow.WorkspaceID != parseWorkspaceID {
		return nil, status.Error(codes.PermissionDenied, "api key outside workspace scope")
	}
	if parseErr = parseS.store.parseRevokeAPIKey(parseKeyID); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "revoke workspace api key: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.workspace.api_key.revoked",
		"api_key",
		parseKeyID,
		"Workspace API key revoked",
		"{}",
		parseWorkspaceID,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.RevokeWorkspaceAPIKey: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.String("key_id", parseKeyID),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.RevokeWorkspaceAPIKeyResponse{
		WorkspaceId: parseWorkspaceID,
		KeyId:       parseKeyID,
		Status:      "revoked",
	}, nil
}

// PauseWorkspaceWebhookEndpoint disables one workspace webhook endpoint in admin scope.
func (parseS *chatServer) PauseWorkspaceWebhookEndpoint(parseCtx context.Context, parseReq *chatpb.PauseWorkspaceWebhookRequest) (*chatpb.PauseWorkspaceWebhookResponse, error) {
	var parseWorkspaceID int64
	var parseEndpointID int64
	var parseReason string
	var isParseConfirmed bool
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseEndpointID = parseReq.GetEndpointId()
		parseReason = parseReq.GetReason()
		isParseConfirmed = parseReq.GetConfirm()
	}
	if parseWorkspaceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	if parseEndpointID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "endpoint id is required")
	}
	parseReason, parseErr := parseRequireAdminWorkspaceActionConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.workspace.webhook.pause")
	if parseErr != nil {
		return nil, parseErr
	}
	if parseErr = parseRequireAdminWorkspaceScope(parseScope, parseWorkspaceID); parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseWebhookRow, hasParseWebhook, parseErr := parseS.store.parseGetAdminWebhookEndpointByID(parseEndpointID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get webhook endpoint by id: %v", parseErr)
	}
	if !hasParseWebhook {
		return nil, status.Error(codes.NotFound, "webhook endpoint not found")
	}
	if parseWebhookRow.WorkspaceID != parseWorkspaceID {
		return nil, status.Error(codes.PermissionDenied, "webhook endpoint outside workspace scope")
	}
	if !parseWebhookRow.IsEnabled {
		return &chatpb.PauseWorkspaceWebhookResponse{
			WorkspaceId: parseWorkspaceID,
			EndpointId:  parseEndpointID,
			Status:      "paused",
		}, nil
	}
	if parseErr = parseS.store.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID:    parseWebhookRow.WorkspaceID,
		Label:          parseWebhookRow.Label,
		TargetURL:      parseWebhookRow.TargetURL,
		SecretHash:     parseWebhookRow.SecretHash,
		EventsJSON:     parseWebhookRow.EventsJSON,
		IsEnabled:      false,
		LastDeliveryAt: parseWebhookRow.LastDeliveryAt,
		FailureCount:   parseWebhookRow.FailureCount,
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "pause workspace webhook endpoint: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.workspace.webhook.paused",
		"webhook_endpoint",
		strconv.FormatInt(parseEndpointID, 10),
		"Workspace webhook endpoint paused",
		"{}",
		parseWorkspaceID,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.PauseWorkspaceWebhookEndpoint: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.Int64("endpoint_id", parseEndpointID),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.PauseWorkspaceWebhookResponse{
		WorkspaceId: parseWorkspaceID,
		EndpointId:  parseEndpointID,
		Status:      "paused",
	}, nil
}
