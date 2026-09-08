package app

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseFindSiteConfigRowByConfigKey returns one site-config row by normalized config key.
func parseFindSiteConfigRowByConfigKey(parseRows []parseSiteConfigRow, parseConfigKey string) (parseSiteConfigRow, bool) {
	parseConfigKey = parseNormalizeSUKey(parseConfigKey)
	for _, parseRow := range parseRows {
		if parseNormalizeSUKey(parseRow.ConfigKey) != parseConfigKey {
			continue
		}
		return parseRow, true
	}
	return parseSiteConfigRow{}, false
}

// SetAdminSiteConfig upserts one ops site-config control row.
func (parseS *chatServer) SetAdminSiteConfig(parseCtx context.Context, parseReq *chatpb.SetAdminSiteConfigRequest) (*chatpb.SetAdminSiteConfigResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.ops")
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseConfigKey := parseNormalizeSUKey(parseReq.GetConfigKey())
	if parseConfigKey == "" {
		return nil, status.Error(codes.InvalidArgument, "config key is required")
	}
	parseConfigValue := strings.TrimSpace(parseReq.GetConfigValue())
	parseValueType := strings.TrimSpace(strings.ToLower(parseReq.GetValueType()))
	if parseValueType == "" {
		parseValueType = "string"
	}
	if parseErr = parseS.store.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey:       parseConfigKey,
		ConfigValue:     parseConfigValue,
		ValueType:       parseValueType,
		Description:     strings.TrimSpace(parseReq.GetDescription()),
		UpdatedByUserID: parseScope.adminUserID,
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set ops site config: %v", parseErr)
	}
	parseSiteConfigRows, parseErr := parseS.store.parseListSiteConfigs()
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops site config: %v", parseErr)
	}
	parseSiteConfigRow, hasParseSiteConfigRow := parseFindSiteConfigRowByConfigKey(parseSiteConfigRows, parseConfigKey)
	if !hasParseSiteConfigRow {
		return nil, status.Error(codes.NotFound, "site config not found after update")
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.ops.site_config.updated",
		"site_config",
		parseConfigKey,
		"Admin ops site config updated",
		"{}",
		0,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminSiteConfig: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.String("config_key", parseConfigKey),
			slog.String("value_type", parseValueType),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminSiteConfigResponse{
		SiteConfig: parseBuildAdminSiteConfigEntry(parseSiteConfigRow),
		Status:     "updated",
	}, nil
}

// SetAdminWebhookBehavior updates one webhook endpoint behavior control row.
func (parseS *chatServer) SetAdminWebhookBehavior(parseCtx context.Context, parseReq *chatpb.SetAdminWebhookBehaviorRequest) (*chatpb.SetAdminWebhookBehaviorResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.ops")
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseEndpointID := parseReq.GetEndpointId()
	if parseEndpointID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "endpoint id is required")
	}
	parseEndpointRow, hasParseEndpoint, parseErr := parseS.store.parseGetAdminWebhookEndpointByID(parseEndpointID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get webhook endpoint by id: %v", parseErr)
	}
	if !hasParseEndpoint {
		return nil, status.Error(codes.NotFound, "webhook endpoint not found")
	}
	parseFailureCount := parseEndpointRow.FailureCount
	if parseReq.GetResetFailureCount() {
		parseFailureCount = 0
	}
	if parseErr = parseS.store.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID:    parseEndpointRow.WorkspaceID,
		Label:          parseEndpointRow.Label,
		TargetURL:      parseEndpointRow.TargetURL,
		SecretHash:     parseEndpointRow.SecretHash,
		EventsJSON:     parseEndpointRow.EventsJSON,
		IsEnabled:      parseReq.GetIsEnabled(),
		LastDeliveryAt: parseEndpointRow.LastDeliveryAt,
		FailureCount:   parseFailureCount,
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set webhook behavior: %v", parseErr)
	}
	parseUpdatedEndpointRow, hasParseUpdatedEndpoint, parseErr := parseS.store.parseGetAdminWebhookEndpointByID(parseEndpointID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "get webhook endpoint after update: %v", parseErr)
	}
	if !hasParseUpdatedEndpoint {
		return nil, status.Error(codes.NotFound, "webhook endpoint not found after update")
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.ops.webhook_behavior.updated",
		"webhook_endpoint",
		strconv.FormatInt(parseEndpointID, 10),
		"Admin webhook behavior updated",
		"{}",
		parseUpdatedEndpointRow.WorkspaceID,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminWebhookBehavior: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("endpoint_id", parseEndpointID),
			slog.Bool("is_enabled", parseReq.GetIsEnabled()),
			slog.Bool("reset_failure_count", parseReq.GetResetFailureCount()),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminWebhookBehaviorResponse{
		WebhookEndpoint: parseBuildAdminWebhookEndpointEntry(parseUpdatedEndpointRow),
		Status:          "updated",
	}, nil
}
