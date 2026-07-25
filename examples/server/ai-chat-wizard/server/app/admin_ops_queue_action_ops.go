package app

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminOpsQueueSummary derives one typed ops queue summary from queue/timeline/action row slices.
func parseBuildAdminOpsQueueSummary(
	parseFailedJobRows []parseBackgroundJobRow,
	parseFailedNotificationRows []parseNotificationOutboxRow,
	parseFailedWebhookRows []parseWebhookDeliveryRow,
	parseIncidentTimelineRows []parseIncidentUpdateRow,
	parseRecentActionRows []parseAuditLogRow,
) *chatpb.AdminOpsQueueSummary {
	return &chatpb.AdminOpsQueueSummary{
		FailedBackgroundJobCount:   int64(len(parseFailedJobRows)),
		FailedNotificationCount:    int64(len(parseFailedNotificationRows)),
		FailedWebhookDeliveryCount: int64(len(parseFailedWebhookRows)),
		IncidentTimelineCount:      int64(len(parseIncidentTimelineRows)),
		RecentAdminActionCount:     int64(len(parseRecentActionRows)),
	}
}

// parseFindBackgroundJobRowByID finds one background-job row by numeric id.
func parseFindBackgroundJobRowByID(parseRows []parseBackgroundJobRow, parseJobID int64) (parseBackgroundJobRow, bool) {
	for _, parseRow := range parseRows {
		if parseRow.ID == parseJobID {
			return parseRow, true
		}
	}
	return parseBackgroundJobRow{}, false
}

// parseFindNotificationOutboxRowByID finds one notification-outbox row by numeric id.
func parseFindNotificationOutboxRowByID(parseRows []parseNotificationOutboxRow, parseNotificationID int64) (parseNotificationOutboxRow, bool) {
	for _, parseRow := range parseRows {
		if parseRow.ID == parseNotificationID {
			return parseRow, true
		}
	}
	return parseNotificationOutboxRow{}, false
}

// parseFindWebhookDeliveryRowByID finds one webhook-delivery row by numeric id.
func parseFindWebhookDeliveryRowByID(parseRows []parseWebhookDeliveryRow, parseDeliveryID int64) (parseWebhookDeliveryRow, bool) {
	for _, parseRow := range parseRows {
		if parseRow.ID == parseDeliveryID {
			return parseRow, true
		}
	}
	return parseWebhookDeliveryRow{}, false
}

// GetAdminOpsQueue returns typed failed queue, incident timeline, recent action rows, and retry/replay-ready ids for ops workflows.
func (parseS *chatServer) GetAdminOpsQueue(parseCtx context.Context, parseReq *chatpb.GetAdminOpsQueueRequest) (*chatpb.GetAdminOpsQueueResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminOpsQueue"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.ops.queue")
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
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"ops-queue",
		"Admin ops queue viewed",
		"{}",
		parseWorkspaceID,
	)
	parseResponse := &chatpb.GetAdminOpsQueueResponse{
		Summary:                 &chatpb.AdminOpsQueueSummary{},
		FailedBackgroundJobs:    make([]*chatpb.BackgroundJobEntry, 0),
		FailedNotifications:     make([]*chatpb.NotificationOutboxEntry, 0),
		FailedWebhookDeliveries: make([]*chatpb.WebhookDeliveryEntry, 0),
		IncidentTimeline:        make([]*chatpb.IncidentUpdateEntry, 0),
		RecentAdminActions:      make([]*chatpb.AuditLogEntry, 0),
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminOpsQueue: store unavailable",
			slog.String("next_action", "restore store availability before retrying ops queue"),
		)
		return parseResponse, nil
	}
	parseQueryLimit := int64(parseLimit)
	if parseWorkspaceID > 0 {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseFailedJobRows, parseErr := parseS.store.parseListAdminFailedBackgroundJobs(parseSince, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list failed background jobs: %v", parseErr)
	}
	parseFailedNotificationRows, parseErr := parseS.store.parseListAdminFailedNotifications(parseSince, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list failed notifications: %v", parseErr)
	}
	parseFailedWebhookRows, parseErr := parseS.store.parseListAdminFailedWebhookDeliveries(parseSince, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list failed webhook deliveries: %v", parseErr)
	}
	parseIncidentTimelineRows, parseErr := parseS.store.parseListAdminIncidentTimeline(parseSince, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list incident timeline: %v", parseErr)
	}
	parseRecentActionRows, parseErr := parseS.store.parseListAdminRecentOpsActions(parseSince, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list recent ops actions: %v", parseErr)
	}
	if parseWorkspaceID > 0 {
		parseFailedNotificationRows = parseFilterNotificationOutboxRowsByWorkspaceID(parseFailedNotificationRows, parseWorkspaceID)
		parseWebhookRows, parseWebhookErr := parseS.store.parseListWebhookEndpoints(parseAdminScopedScanLimit)
		if parseWebhookErr != nil {
			return nil, status.Errorf(codes.Internal, "list webhook endpoints for scope filtering: %v", parseWebhookErr)
		}
		parseWebhookRows = parseFilterWebhookEndpointRowsByWorkspaceID(parseWebhookRows, parseWorkspaceID)
		parseFailedWebhookRows = parseFilterWebhookDeliveryRowsByEndpointScope(parseFailedWebhookRows, parseBuildWebhookEndpointIDScopeSet(parseWebhookRows))
		parseRecentActionRows = parseFilterAuditRowsByWorkspaceID(parseRecentActionRows, parseWorkspaceID)
	}
	parseResponse.Summary = parseBuildAdminOpsQueueSummary(
		parseFailedJobRows,
		parseFailedNotificationRows,
		parseFailedWebhookRows,
		parseIncidentTimelineRows,
		parseRecentActionRows,
	)

	parseFailedJobRows = parseApplyAdminSliceWindow(parseFailedJobRows, 0, parseLimit)
	parseFailedNotificationRows = parseApplyAdminSliceWindow(parseFailedNotificationRows, 0, parseLimit)
	parseFailedWebhookRows = parseApplyAdminSliceWindow(parseFailedWebhookRows, 0, parseLimit)
	parseIncidentTimelineRows = parseApplyAdminSliceWindow(parseIncidentTimelineRows, 0, parseLimit)
	parseRecentActionRows = parseApplyAdminSliceWindow(parseRecentActionRows, 0, parseLimit)

	for _, parseJobRow := range parseFailedJobRows {
		parseResponse.FailedBackgroundJobs = append(parseResponse.FailedBackgroundJobs, parseBuildAdminBackgroundJobEntry(parseJobRow))
	}
	for _, parseNotificationRow := range parseFailedNotificationRows {
		parseResponse.FailedNotifications = append(parseResponse.FailedNotifications, parseBuildAdminNotificationOutboxEntry(parseNotificationRow))
	}
	for _, parseWebhookRow := range parseFailedWebhookRows {
		parseResponse.FailedWebhookDeliveries = append(parseResponse.FailedWebhookDeliveries, parseBuildAdminWebhookDeliveryEntry(parseWebhookRow))
	}
	for _, parseIncidentRow := range parseIncidentTimelineRows {
		parseResponse.IncidentTimeline = append(parseResponse.IncidentTimeline, parseBuildAdminIncidentUpdateEntry(parseIncidentRow))
	}
	for _, parseActionRow := range parseRecentActionRows {
		parseResponse.RecentAdminActions = append(parseResponse.RecentAdminActions, parseRedactAdminAuditLogEntryByScope(parseScope, parseBuildAdminAuditLogEntry(parseActionRow)))
	}
	return parseResponse, nil
}

// RetryAdminBackgroundJob schedules one failed background job for immediate retry and emits one explicit ops-action audit row.
func (parseS *chatServer) RetryAdminBackgroundJob(parseCtx context.Context, parseReq *chatpb.RetryAdminBackgroundJobRequest) (*chatpb.RetryAdminBackgroundJobResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminOpsActionScope(parseCtx, parseAdminOpsActionBackgroundJobRetry, 0)
	if parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr = parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason()); parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseJobID := parseReq.GetJobId()
	if parseJobID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "job id is required")
	}
	parseJobRows, parseErr := parseS.store.parseListBackgroundJobs(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list background jobs: %v", parseErr)
	}
	parseJobRow, hasParseJobRow := parseFindBackgroundJobRowByID(parseJobRows, parseJobID)
	if !hasParseJobRow {
		return nil, status.Error(codes.NotFound, "background job not found")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if parseErr = parseS.store.parseUpsertBackgroundJob(parseBackgroundJobWrite{
		JobKey:       parseJobRow.JobKey,
		JobType:      parseJobRow.JobType,
		QueueKey:     parseJobRow.QueueKey,
		Status:       "pending",
		AttemptCount: parseJobRow.AttemptCount,
		MaxAttempts:  parseJobRow.MaxAttempts,
		PayloadJSON:  parseJobRow.PayloadJSON,
		RunAfter:     parseNow,
		StartedAt:    "",
		FinishedAt:   "",
		ErrorMessage: "",
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "retry background job: %v", parseErr)
	}
	parseS.parseTrackAdminOpsActionAudit(
		parseScope,
		parseAdminOpsActionBackgroundJobRetry,
		"background_job",
		strconv.FormatInt(parseJobID, 10),
		"Admin background job retry scheduled",
		0,
	)
	return &chatpb.RetryAdminBackgroundJobResponse{
		JobId:  parseJobID,
		Status: "retry_scheduled",
	}, nil
}

// RetryAdminNotification schedules one failed notification row for immediate retry and emits one explicit ops-action audit row.
func (parseS *chatServer) RetryAdminNotification(parseCtx context.Context, parseReq *chatpb.RetryAdminNotificationRequest) (*chatpb.RetryAdminNotificationResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminOpsActionScope(parseCtx, parseAdminOpsActionNotificationRetry, 0)
	if parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr = parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason()); parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseNotificationID := parseReq.GetNotificationId()
	if parseNotificationID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "notification id is required")
	}
	parseNotificationRows, parseErr := parseS.store.parseListNotificationOutbox(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list notifications: %v", parseErr)
	}
	if _, hasParseNotificationRow := parseFindNotificationOutboxRowByID(parseNotificationRows, parseNotificationID); !hasParseNotificationRow {
		return nil, status.Error(codes.NotFound, "notification not found")
	}
	if parseErr = parseS.store.parseUpdateNotificationOutboxStatus(parseNotificationID, "pending", "", "", ""); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "retry notification: %v", parseErr)
	}
	parseS.parseTrackAdminOpsActionAudit(
		parseScope,
		parseAdminOpsActionNotificationRetry,
		"notification_outbox",
		strconv.FormatInt(parseNotificationID, 10),
		"Admin notification retry scheduled",
		0,
	)
	return &chatpb.RetryAdminNotificationResponse{
		NotificationId: parseNotificationID,
		Status:         "retry_scheduled",
	}, nil
}

// ReplayAdminWebhookDelivery schedules one failed webhook delivery for immediate replay and emits one explicit ops-action audit row.
func (parseS *chatServer) ReplayAdminWebhookDelivery(parseCtx context.Context, parseReq *chatpb.ReplayAdminWebhookDeliveryRequest) (*chatpb.ReplayAdminWebhookDeliveryResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminOpsActionScope(parseCtx, parseAdminOpsActionWebhookReplay, 0)
	if parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr = parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason()); parseErr != nil {
		return nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseDeliveryID := parseReq.GetDeliveryId()
	if parseDeliveryID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "delivery id is required")
	}
	parseDeliveryRows, parseErr := parseS.store.parseListWebhookDeliveries(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list webhook deliveries: %v", parseErr)
	}
	parseDeliveryRow, hasParseDeliveryRow := parseFindWebhookDeliveryRowByID(parseDeliveryRows, parseDeliveryID)
	if !hasParseDeliveryRow {
		return nil, status.Error(codes.NotFound, "webhook delivery not found")
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseAttemptCount := parseDeliveryRow.AttemptCount
	if parseAttemptCount <= 0 {
		parseAttemptCount = 1
	}
	if parseErr = parseS.store.parseUpdateWebhookDeliveryAttempt(
		parseDeliveryRow.DeliveryKey,
		parseDeliveryRow.ResponseStatus,
		parseDeliveryRow.ResponseBody,
		parseAttemptCount,
		parseNow,
		parseNow,
	); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "replay webhook delivery: %v", parseErr)
	}
	parseWorkspaceID := int64(0)
	parseEndpointRow, hasParseEndpointRow, parseEndpointErr := parseS.store.parseGetAdminWebhookEndpointByID(parseDeliveryRow.EndpointID)
	if parseEndpointErr == nil && hasParseEndpointRow {
		parseWorkspaceID = parseEndpointRow.WorkspaceID
	}
	parseS.parseTrackAdminOpsActionAudit(
		parseScope,
		parseAdminOpsActionWebhookReplay,
		"webhook_delivery",
		strconv.FormatInt(parseDeliveryID, 10),
		"Admin webhook replay scheduled",
		parseWorkspaceID,
	)
	return &chatpb.ReplayAdminWebhookDeliveryResponse{
		DeliveryId:  parseDeliveryID,
		Status:      "replay_scheduled",
		NextRetryAt: parseNow,
	}, nil
}
