package app

import (
	"context"
	"log/slog"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminProviderHealthTrendEntry maps one provider health-trend row into protobuf form.
func parseBuildAdminProviderHealthTrendEntry(parseRow parseProviderUsageDailyRollupRow) *chatpb.AdminProviderHealthTrend {
	return &chatpb.AdminProviderHealthTrend{
		RollupDay:           parseRow.RollupDay,
		ProviderId:          parseRow.ProviderID,
		UsageEventCount:     parseRow.UsageEventCount,
		TotalCostUsd:        parseRow.TotalCostUSD,
		PromptTokens:        parseRow.PromptTokens,
		CompletionTokens:    parseRow.CompletionTokens,
		CompletedEventCount: parseRow.CompletedEventCount,
		FailedEventCount:    parseRow.FailedEventCount,
		UpdatedAt:           parseRow.UpdatedAt,
	}
}

// parseBuildAdminProviderFallbackEventEntry maps one provider fallback/control audit row into protobuf form.
func parseBuildAdminProviderFallbackEventEntry(parseScope parseAdminAccessScope, parseRow parseAuditLogRow) *chatpb.AdminProviderFallbackEvent {
	parseAuditEntry := parseRedactAdminAuditLogEntryByScope(parseScope, parseBuildAdminAuditLogEntry(parseRow))
	return &chatpb.AdminProviderFallbackEvent{
		AuditLogId:  parseAuditEntry.GetId(),
		EventType:   parseAuditEntry.GetEventType(),
		WorkspaceId: parseAuditEntry.GetWorkspaceId(),
		TargetType:  parseAuditEntry.GetTargetType(),
		TargetId:    parseAuditEntry.GetTargetId(),
		Summary:     parseAuditEntry.GetSummary(),
		PayloadJson: parseAuditEntry.GetPayloadJson(),
		CreatedAt:   parseAuditEntry.GetCreatedAt(),
	}
}

// parseFilterAdminProviderFallbackRowsByWorkspaceID filters provider fallback rows by one optional workspace id.
func parseFilterAdminProviderFallbackRowsByWorkspaceID(parseRows []parseAuditLogRow, parseWorkspaceID int64) []parseAuditLogRow {
	if parseWorkspaceID <= 0 {
		return parseRows
	}
	parseFilteredRows := make([]parseAuditLogRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseBuildAdminProviderHealthTrendSummary derives one provider-trend summary from trend rows and fallback-event rows.
func parseBuildAdminProviderHealthTrendSummary(parseTrendRows []parseProviderUsageDailyRollupRow, parseFallbackRows []parseAuditLogRow) *chatpb.AdminProviderHealthTrendSummary {
	parseSummary := &chatpb.AdminProviderHealthTrendSummary{
		TrendRowCount:      int64(len(parseTrendRows)),
		FallbackEventCount: int64(len(parseFallbackRows)),
	}
	for _, parseTrendRow := range parseTrendRows {
		parseSummary.UsageEventCount += parseTrendRow.UsageEventCount
		parseSummary.FailedEventCount += parseTrendRow.FailedEventCount
		parseSummary.TotalCostUsd += parseTrendRow.TotalCostUSD
	}
	return parseSummary
}

// GetAdminProviderHealthTrends returns typed provider health trends and provider fallback/control audit events.
func (parseS *chatServer) GetAdminProviderHealthTrends(parseCtx context.Context, parseReq *chatpb.GetAdminProviderHealthTrendsRequest) (*chatpb.GetAdminProviderHealthTrendsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminProviderHealthTrends"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.providers.health_trends")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLookbackDays int32
	var parseLimit int32
	var parseWorkspaceID int64
	if parseReq != nil {
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
		parseWorkspaceID = parseReq.GetWorkspaceId()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseResponse := &chatpb.GetAdminProviderHealthTrendsResponse{
		Summary:        &chatpb.AdminProviderHealthTrendSummary{},
		Trends:         make([]*chatpb.AdminProviderHealthTrend, 0),
		FallbackEvents: make([]*chatpb.AdminProviderFallbackEvent, 0),
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"providers-health-trends",
		"Admin provider health trends viewed",
		"{}",
		parseWorkspaceID,
	)
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminProviderHealthTrends: store unavailable",
			slog.String("next_action", "restore store availability before retrying provider health trends"),
		)
		return parseResponse, nil
	}

	if parseErr = parseS.store.parseRefreshProviderUsageDailyRollups(parseSince); parseErr != nil {
		parseLogger.Warn("rpc.GetAdminProviderHealthTrends: provider rollup refresh failed", slog.String("error", parseErr.Error()))
	}
	parseTrendRows, parseErr := parseS.store.parseListAdminProviderHealthTrends(parseSince, int64(parseLimit))
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list provider health trends: %v", parseErr)
	}
	parseFallbackQueryLimit := int64(parseLimit)
	if parseWorkspaceID > 0 {
		parseFallbackQueryLimit = parseAdminScopedScanLimit
	}
	parseFallbackRows, parseErr := parseS.store.parseListAdminProviderFallbackEvents(parseSince, parseFallbackQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list provider fallback events: %v", parseErr)
	}
	parseFallbackRows = parseFilterAdminProviderFallbackRowsByWorkspaceID(parseFallbackRows, parseWorkspaceID)
	parseResponse.Summary = parseBuildAdminProviderHealthTrendSummary(parseTrendRows, parseFallbackRows)

	parseTrendRows = parseApplyAdminSliceWindow(parseTrendRows, 0, parseLimit)
	parseFallbackRows = parseApplyAdminSliceWindow(parseFallbackRows, 0, parseLimit)
	parseResponse.Trends = make([]*chatpb.AdminProviderHealthTrend, 0, len(parseTrendRows))
	for _, parseTrendRow := range parseTrendRows {
		parseResponse.Trends = append(parseResponse.Trends, parseBuildAdminProviderHealthTrendEntry(parseTrendRow))
	}
	parseResponse.FallbackEvents = make([]*chatpb.AdminProviderFallbackEvent, 0, len(parseFallbackRows))
	for _, parseFallbackRow := range parseFallbackRows {
		parseResponse.FallbackEvents = append(parseResponse.FallbackEvents, parseBuildAdminProviderFallbackEventEntry(parseScope, parseFallbackRow))
	}
	parseLogger.Info(
		"rpc.GetAdminProviderHealthTrends: complete",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
		slog.Int64("workspace_id", parseWorkspaceID),
		slog.Int("trends", len(parseResponse.Trends)),
		slog.Int("fallback_events", len(parseResponse.FallbackEvents)),
	)
	return parseResponse, nil
}
