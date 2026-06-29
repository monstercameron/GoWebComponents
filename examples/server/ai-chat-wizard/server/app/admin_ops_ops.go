package app

import (
	"context"
	"log/slog"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminSiteConfigEntry maps one site-config row into protobuf form.
func parseBuildAdminSiteConfigEntry(parseRow parseSiteConfigRow) *chatpb.SiteConfigEntry {
	return &chatpb.SiteConfigEntry{
		ConfigKey:       parseRow.ConfigKey,
		ConfigValue:     parseRow.ConfigValue,
		ValueType:       parseRow.ValueType,
		Description:     parseRow.Description,
		UpdatedByUserId: parseRow.UpdatedByUserID,
		UpdatedAt:       parseRow.UpdatedAt,
	}
}

// parseBuildAdminWebhookDeliveryEntry maps one webhook-delivery row into protobuf form.
func parseBuildAdminWebhookDeliveryEntry(parseRow parseWebhookDeliveryRow) *chatpb.WebhookDeliveryEntry {
	return &chatpb.WebhookDeliveryEntry{
		Id:                 parseRow.ID,
		EndpointId:         parseRow.EndpointID,
		EventType:          parseRow.EventType,
		DeliveryKey:        parseRow.DeliveryKey,
		RequestHeadersJson: parseRow.RequestHeadersJSON,
		RequestBodyJson:    parseRow.RequestBodyJSON,
		ResponseStatus:     parseRow.ResponseStatus,
		ResponseBody:       parseRow.ResponseBody,
		AttemptCount:       parseRow.AttemptCount,
		DeliveredAt:        parseRow.DeliveredAt,
		FailedAt:           parseRow.FailedAt,
		NextRetryAt:        parseRow.NextRetryAt,
		CreatedAt:          parseRow.CreatedAt,
		UpdatedAt:          parseRow.UpdatedAt,
	}
}

// parseBuildAdminNotificationOutboxEntry maps one notification-outbox row into protobuf form.
func parseBuildAdminNotificationOutboxEntry(parseRow parseNotificationOutboxRow) *chatpb.NotificationOutboxEntry {
	return &chatpb.NotificationOutboxEntry{
		Id:              parseRow.ID,
		WorkspaceId:     parseRow.WorkspaceID,
		UserId:          parseRow.UserID,
		NotificationKey: parseRow.NotificationKey,
		ChannelKey:      parseRow.ChannelKey,
		TemplateKey:     parseRow.TemplateKey,
		Status:          parseRow.Status,
		Subject:         parseRow.Subject,
		BodyText:        parseRow.BodyText,
		PayloadJson:     parseRow.PayloadJSON,
		DedupeKey:       parseRow.DedupeKey,
		ScheduledAt:     parseRow.ScheduledAt,
		SentAt:          parseRow.SentAt,
		FailedAt:        parseRow.FailedAt,
		ErrorMessage:    parseRow.ErrorMessage,
		CreatedAt:       parseRow.CreatedAt,
		UpdatedAt:       parseRow.UpdatedAt,
	}
}

// parseBuildAdminBackgroundJobEntry maps one background-job row into protobuf form.
func parseBuildAdminBackgroundJobEntry(parseRow parseBackgroundJobRow) *chatpb.BackgroundJobEntry {
	return &chatpb.BackgroundJobEntry{
		Id:           parseRow.ID,
		JobKey:       parseRow.JobKey,
		JobType:      parseRow.JobType,
		QueueKey:     parseRow.QueueKey,
		Status:       parseRow.Status,
		AttemptCount: parseRow.AttemptCount,
		MaxAttempts:  parseRow.MaxAttempts,
		PayloadJson:  parseRow.PayloadJSON,
		RunAfter:     parseRow.RunAfter,
		StartedAt:    parseRow.StartedAt,
		FinishedAt:   parseRow.FinishedAt,
		ErrorMessage: parseRow.ErrorMessage,
		CreatedAt:    parseRow.CreatedAt,
		UpdatedAt:    parseRow.UpdatedAt,
	}
}

// parseFilterAuditRowsByWorkspaceID filters audit-log rows by one workspace id.
func parseFilterAuditRowsByWorkspaceID(parseRows []parseAuditLogRow, parseWorkspaceID int64) []parseAuditLogRow {
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

// parseFilterNotificationOutboxRowsByWorkspaceID filters notification rows by one workspace id.
func parseFilterNotificationOutboxRowsByWorkspaceID(parseRows []parseNotificationOutboxRow, parseWorkspaceID int64) []parseNotificationOutboxRow {
	if parseWorkspaceID <= 0 {
		return parseRows
	}
	parseFilteredRows := make([]parseNotificationOutboxRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterWebhookEndpointRowsByWorkspaceID filters webhook-endpoint rows by one workspace id.
func parseFilterWebhookEndpointRowsByWorkspaceID(parseRows []parseWebhookEndpointRow, parseWorkspaceID int64) []parseWebhookEndpointRow {
	if parseWorkspaceID <= 0 {
		return parseRows
	}
	parseFilteredRows := make([]parseWebhookEndpointRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseBuildWebhookEndpointIDScopeSet returns one endpoint-id scope set for webhook-delivery filtering.
func parseBuildWebhookEndpointIDScopeSet(parseRows []parseWebhookEndpointRow) map[int64]struct{} {
	parseScope := make(map[int64]struct{}, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.ID <= 0 {
			continue
		}
		parseScope[parseRow.ID] = struct{}{}
	}
	return parseScope
}

// parseFilterWebhookDeliveryRowsByEndpointScope filters webhook-delivery rows by one endpoint-id scope set.
func parseFilterWebhookDeliveryRowsByEndpointScope(parseRows []parseWebhookDeliveryRow, parseEndpointIDs map[int64]struct{}) []parseWebhookDeliveryRow {
	if len(parseEndpointIDs) == 0 {
		return nil
	}
	parseFilteredRows := make([]parseWebhookDeliveryRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if _, hasParseEndpoint := parseEndpointIDs[parseRow.EndpointID]; !hasParseEndpoint {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseBuildAdminOpsDrilldownSummary derives one ops drill-down summary from typed ops row slices.
func parseBuildAdminOpsDrilldownSummary(
	parseSiteConfigRows []parseSiteConfigRow,
	parseFeatureFlagRows []parseFeatureFlagRow,
	parseAuditRows []parseAuditLogRow,
	parseSLORows []parseServiceLevelObjectiveRow,
	parseIncidentRows []parseIncidentRow,
	parseIncidentUpdateRows []parseIncidentUpdateRow,
	parseBackgroundJobRows []parseBackgroundJobRow,
	parseNotificationRows []parseNotificationOutboxRow,
	parseWebhookRows []parseWebhookEndpointRow,
	parseWebhookDeliveryRows []parseWebhookDeliveryRow,
) *chatpb.AdminOpsDrilldownSummary {
	parseSummary := &chatpb.AdminOpsDrilldownSummary{
		SiteConfigCount:            int64(len(parseSiteConfigRows)),
		FeatureFlagCount:           int64(len(parseFeatureFlagRows)),
		AuditLogCount:              int64(len(parseAuditRows)),
		ServiceLevelObjectiveCount: int64(len(parseSLORows)),
		IncidentCount:              int64(len(parseIncidentRows)),
		IncidentUpdateCount:        int64(len(parseIncidentUpdateRows)),
		BackgroundJobCount:         int64(len(parseBackgroundJobRows)),
		NotificationOutboxCount:    int64(len(parseNotificationRows)),
		WebhookEndpointCount:       int64(len(parseWebhookRows)),
		WebhookDeliveryCount:       int64(len(parseWebhookDeliveryRows)),
	}
	for _, parseIncidentRow := range parseIncidentRows {
		if strings.EqualFold(strings.TrimSpace(parseIncidentRow.Status), "open") {
			parseSummary.OpenIncidentCount++
		}
	}
	for _, parseBackgroundRow := range parseBackgroundJobRows {
		parseStatus := strings.TrimSpace(strings.ToLower(parseBackgroundRow.Status))
		if parseStatus == "failed" || parseStatus == "error" {
			parseSummary.FailedBackgroundJobCount++
		}
	}
	for _, parseNotificationRow := range parseNotificationRows {
		parseStatus := strings.TrimSpace(strings.ToLower(parseNotificationRow.Status))
		if parseStatus == "failed" || parseStatus == "error" || strings.TrimSpace(parseNotificationRow.FailedAt) != "" {
			parseSummary.FailedNotificationCount++
		}
	}
	for _, parseWebhookDeliveryRow := range parseWebhookDeliveryRows {
		if strings.TrimSpace(parseWebhookDeliveryRow.FailedAt) != "" && strings.TrimSpace(parseWebhookDeliveryRow.DeliveredAt) == "" {
			parseSummary.FailedWebhookDeliveryCount++
		}
	}
	return parseSummary
}

// GetAdminOpsDrilldown returns one typed ops drill-down snapshot for dashboard ops surfaces.
func (parseS *chatServer) GetAdminOpsDrilldown(parseCtx context.Context, parseReq *chatpb.GetAdminOpsDrilldownRequest) (*chatpb.GetAdminOpsDrilldownResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminOpsDrilldown"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.ops")
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
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.GetAdminOpsDrilldown: slice fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
		slog.Int64("workspace_id", parseWorkspaceID),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"ops",
		"Admin ops slice viewed",
		"{}",
		parseWorkspaceID,
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"slice",
		"ops",
		"Admin ops drill-down viewed",
		"{}",
		parseWorkspaceID,
	)

	parseResponse := &chatpb.GetAdminOpsDrilldownResponse{
		Summary: &chatpb.AdminOpsDrilldownSummary{},
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminOpsDrilldown: store unavailable",
			slog.String("next_action", "restore store availability before retrying ops drill-down"),
		)
		return parseResponse, nil
	}

	parseQueryLimit := int64(parseLimit)
	if parseWorkspaceID > 0 {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseSiteConfigRows, parseErr := parseS.store.parseListSiteConfigs()
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops site config: %v", parseErr)
	}
	parseFeatureFlagRows, parseErr := parseS.store.parseListFeatureFlags()
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops feature flags: %v", parseErr)
	}
	parseAuditRows, parseErr := parseS.store.parseListAuditLogs(parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops audit logs: %v", parseErr)
	}
	parseSLORows, parseErr := parseS.store.parseListServiceLevelObjectives(parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops service level objectives: %v", parseErr)
	}
	parseIncidentRows, parseErr := parseS.store.parseListIncidents(parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops incidents: %v", parseErr)
	}
	parseIncidentUpdateRows, parseErr := parseS.store.parseListIncidentUpdates(parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops incident updates: %v", parseErr)
	}
	parseBackgroundJobRows, parseErr := parseS.store.parseListBackgroundJobs(parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops background jobs: %v", parseErr)
	}
	parseNotificationRows, parseErr := parseS.store.parseListNotificationOutbox(parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops notification outbox: %v", parseErr)
	}
	parseWebhookRows, parseErr := parseS.store.parseListWebhookEndpoints(parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops webhook endpoints: %v", parseErr)
	}
	parseWebhookDeliveryRows, parseErr := parseS.store.parseListWebhookDeliveries(parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list ops webhook deliveries: %v", parseErr)
	}

	parseAuditRows = parseFilterAuditRowsByWorkspaceID(parseAuditRows, parseWorkspaceID)
	parseNotificationRows = parseFilterNotificationOutboxRowsByWorkspaceID(parseNotificationRows, parseWorkspaceID)
	parseWebhookRows = parseFilterWebhookEndpointRowsByWorkspaceID(parseWebhookRows, parseWorkspaceID)
	if parseWorkspaceID > 0 {
		parseWebhookDeliveryRows = parseFilterWebhookDeliveryRowsByEndpointScope(parseWebhookDeliveryRows, parseBuildWebhookEndpointIDScopeSet(parseWebhookRows))
	}

	parseResponse.Summary = parseBuildAdminOpsDrilldownSummary(
		parseSiteConfigRows,
		parseFeatureFlagRows,
		parseAuditRows,
		parseSLORows,
		parseIncidentRows,
		parseIncidentUpdateRows,
		parseBackgroundJobRows,
		parseNotificationRows,
		parseWebhookRows,
		parseWebhookDeliveryRows,
	)

	parseSiteConfigRows = parseApplyAdminSliceWindow(parseSiteConfigRows, 0, parseLimit)
	parseFeatureFlagRows = parseApplyAdminSliceWindow(parseFeatureFlagRows, 0, parseLimit)
	parseAuditRows = parseApplyAdminSliceWindow(parseAuditRows, 0, parseLimit)
	parseSLORows = parseApplyAdminSliceWindow(parseSLORows, 0, parseLimit)
	parseIncidentRows = parseApplyAdminSliceWindow(parseIncidentRows, 0, parseLimit)
	parseIncidentUpdateRows = parseApplyAdminSliceWindow(parseIncidentUpdateRows, 0, parseLimit)
	parseBackgroundJobRows = parseApplyAdminSliceWindow(parseBackgroundJobRows, 0, parseLimit)
	parseNotificationRows = parseApplyAdminSliceWindow(parseNotificationRows, 0, parseLimit)
	parseWebhookRows = parseApplyAdminSliceWindow(parseWebhookRows, 0, parseLimit)
	parseWebhookDeliveryRows = parseApplyAdminSliceWindow(parseWebhookDeliveryRows, 0, parseLimit)

	parseResponse.SiteConfigs = make([]*chatpb.SiteConfigEntry, 0, len(parseSiteConfigRows))
	for _, parseSiteConfigRow := range parseSiteConfigRows {
		parseResponse.SiteConfigs = append(parseResponse.SiteConfigs, parseBuildAdminSiteConfigEntry(parseSiteConfigRow))
	}
	parseResponse.FeatureFlags = make([]*chatpb.FeatureFlagEntry, 0, len(parseFeatureFlagRows))
	for _, parseFeatureFlagRow := range parseFeatureFlagRows {
		parseResponse.FeatureFlags = append(parseResponse.FeatureFlags, parseBuildAdminFeatureFlagEntry(parseFeatureFlagRow))
	}
	parseResponse.AuditLogs = make([]*chatpb.AuditLogEntry, 0, len(parseAuditRows))
	for _, parseAuditRow := range parseAuditRows {
		parseResponse.AuditLogs = append(parseResponse.AuditLogs, parseRedactAdminAuditLogEntryByScope(parseScope, parseBuildAdminAuditLogEntry(parseAuditRow)))
	}
	parseResponse.ServiceLevelObjectives = make([]*chatpb.ServiceLevelObjectiveEntry, 0, len(parseSLORows))
	for _, parseSLORow := range parseSLORows {
		parseResponse.ServiceLevelObjectives = append(parseResponse.ServiceLevelObjectives, parseBuildSuperuserServiceLevelObjectiveEntry(parseSLORow))
	}
	parseResponse.Incidents = make([]*chatpb.IncidentEntry, 0, len(parseIncidentRows))
	for _, parseIncidentRow := range parseIncidentRows {
		parseResponse.Incidents = append(parseResponse.Incidents, parseBuildAdminIncidentEntry(parseIncidentRow))
	}
	parseResponse.IncidentUpdates = make([]*chatpb.IncidentUpdateEntry, 0, len(parseIncidentUpdateRows))
	for _, parseIncidentUpdateRow := range parseIncidentUpdateRows {
		parseResponse.IncidentUpdates = append(parseResponse.IncidentUpdates, parseBuildAdminIncidentUpdateEntry(parseIncidentUpdateRow))
	}
	parseResponse.BackgroundJobs = make([]*chatpb.BackgroundJobEntry, 0, len(parseBackgroundJobRows))
	for _, parseBackgroundJobRow := range parseBackgroundJobRows {
		parseResponse.BackgroundJobs = append(parseResponse.BackgroundJobs, parseBuildAdminBackgroundJobEntry(parseBackgroundJobRow))
	}
	parseResponse.NotificationOutbox = make([]*chatpb.NotificationOutboxEntry, 0, len(parseNotificationRows))
	for _, parseNotificationRow := range parseNotificationRows {
		parseResponse.NotificationOutbox = append(parseResponse.NotificationOutbox, parseBuildAdminNotificationOutboxEntry(parseNotificationRow))
	}
	parseResponse.WebhookEndpoints = make([]*chatpb.WebhookEndpointEntry, 0, len(parseWebhookRows))
	for _, parseWebhookRow := range parseWebhookRows {
		parseResponse.WebhookEndpoints = append(parseResponse.WebhookEndpoints, parseBuildAdminWebhookEndpointEntry(parseWebhookRow))
	}
	parseResponse.WebhookDeliveries = make([]*chatpb.WebhookDeliveryEntry, 0, len(parseWebhookDeliveryRows))
	for _, parseWebhookDeliveryRow := range parseWebhookDeliveryRows {
		parseResponse.WebhookDeliveries = append(parseResponse.WebhookDeliveries, parseBuildAdminWebhookDeliveryEntry(parseWebhookDeliveryRow))
	}

	parseLogger.Info(
		"rpc.GetAdminOpsDrilldown: complete",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
		slog.Int("site_configs", len(parseResponse.SiteConfigs)),
		slog.Int("feature_flags", len(parseResponse.FeatureFlags)),
		slog.Int("audit_logs", len(parseResponse.AuditLogs)),
		slog.Int("slos", len(parseResponse.ServiceLevelObjectives)),
		slog.Int("incidents", len(parseResponse.Incidents)),
		slog.Int("incident_updates", len(parseResponse.IncidentUpdates)),
		slog.Int("background_jobs", len(parseResponse.BackgroundJobs)),
		slog.Int("notification_outbox", len(parseResponse.NotificationOutbox)),
		slog.Int("webhook_endpoints", len(parseResponse.WebhookEndpoints)),
		slog.Int("webhook_deliveries", len(parseResponse.WebhookDeliveries)),
	)
	return parseResponse, nil
}
