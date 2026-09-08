package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseSeedAdminOpsQueueRows seeds failed ops queue rows and returns job/notification/webhook delivery ids.
func parseSeedAdminOpsQueueRows(parseT *testing.T, parseStore *Store, parseUserID int64, parseWorkspaceID int64) (int64, int64, int64) {
	parseT.Helper()
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseJobKey := fmt.Sprintf("ops-queue-job-%d", parseUserID)
	if parseErr := parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
		JobKey:       parseJobKey,
		JobType:      "health-score-refresh",
		QueueKey:     "ops",
		Status:       "failed",
		AttemptCount: 2,
		MaxAttempts:  5,
		PayloadJSON:  `{"source":"admin-ops-queue-test"}`,
		RunAfter:     parseNow,
		StartedAt:    parseNow,
		FinishedAt:   parseNow,
		ErrorMessage: "job failed",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertBackgroundJob failed row: %v", parseErr)
	}
	parseBackgroundRows, parseErr := parseStore.parseListBackgroundJobs(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs: %v", parseErr)
	}
	parseJobID := int64(0)
	for _, parseBackgroundRow := range parseBackgroundRows {
		if parseBackgroundRow.JobKey == parseJobKey {
			parseJobID = parseBackgroundRow.ID
			break
		}
	}
	if parseJobID <= 0 {
		parseT.Fatalf("expected background job id for key %q", parseJobKey)
	}

	parseNotificationID, parseErr := parseStore.parseCreateNotificationOutbox(context.Background(), parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseUserID,
		NotificationKey: "ops.failed.notification",
		ChannelKey:      "email",
		TemplateKey:     "ops-failed",
		Status:          "failed",
		Subject:         "Failed notification",
		BodyText:        "Notification delivery failed",
		PayloadJSON:     `{"source":"admin-ops-queue-test"}`,
		DedupeKey:       fmt.Sprintf("ops-failed-notification-%d", parseUserID),
		ScheduledAt:     parseNow,
		FailedAt:        parseNow,
		ErrorMessage:    "smtp timeout",
	})
	if parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox failed row: %v", parseErr)
	}

	parseTargetURL := fmt.Sprintf("https://example.com/hooks/ops-queue-%d", parseUserID)
	if parseErr = parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Ops Queue Hook",
		TargetURL:   parseTargetURL,
		SecretHash:  "ops-queue-secret",
		EventsJSON:  `["ops.alert"]`,
		IsEnabled:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint failed row: %v", parseErr)
	}
	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints: %v", parseErr)
	}
	parseEndpointID := int64(0)
	for _, parseWebhookRow := range parseWebhookRows {
		if parseWebhookRow.TargetURL == parseTargetURL {
			parseEndpointID = parseWebhookRow.ID
			break
		}
	}
	if parseEndpointID <= 0 {
		parseT.Fatalf("expected webhook endpoint id for target %q", parseTargetURL)
	}
	parseDeliveryKey := fmt.Sprintf("ops-queue-delivery-%d", parseUserID)
	if parseErr = parseStore.parseUpsertWebhookDelivery(context.Background(), parseWebhookDeliveryWrite{
		EndpointID:         parseEndpointID,
		EventType:          "ops.alert",
		DeliveryKey:        parseDeliveryKey,
		RequestHeadersJSON: `{"content-type":"application/json"}`,
		RequestBodyJSON:    `{"source":"admin-ops-queue-test"}`,
		ResponseStatus:     500,
		ResponseBody:       "gateway timeout",
		AttemptCount:       2,
		FailedAt:           parseNow,
		NextRetryAt:        parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookDelivery failed row: %v", parseErr)
	}
	parseDeliveryRows, parseErr := parseStore.parseListWebhookDeliveries(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookDeliveries: %v", parseErr)
	}
	parseDeliveryID := int64(0)
	for _, parseDeliveryRow := range parseDeliveryRows {
		if parseDeliveryRow.DeliveryKey == parseDeliveryKey {
			parseDeliveryID = parseDeliveryRow.ID
			break
		}
	}
	if parseDeliveryID <= 0 {
		parseT.Fatalf("expected webhook delivery id for delivery key %q", parseDeliveryKey)
	}

	parseIncidentRows, parseErr := parseStore.parseListIncidents(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseT.Fatalf("parseListIncidents: %v", parseErr)
	}
	if len(parseIncidentRows) > 0 {
		if _, parseErr = parseStore.parseCreateIncidentUpdate(parseIncidentUpdateWrite{
			IncidentID:      parseIncidentRows[0].ID,
			Status:          "investigating",
			Message:         "Ops queue incident timeline entry",
			IsPublic:        false,
			PublishedAt:     "",
			CreatedByUserID: parseUserID,
		}); parseErr != nil {
			parseT.Fatalf("parseCreateIncidentUpdate: %v", parseErr)
		}
	}
	if _, parseErr = parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseUserID,
		WorkspaceID: parseWorkspaceID,
		EventType:   "admin.ops.background_job.retry",
		TargetType:  "background_job",
		TargetID:    fmt.Sprintf("%d", parseJobID),
		Summary:     "Background job retry scheduled",
		PayloadJSON: `{"source":"admin-ops-queue-test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog admin.ops seed row: %v", parseErr)
	}
	return parseJobID, parseNotificationID, parseDeliveryID
}

// TestGetAdminOpsQueueAndActions verifies typed ops queue slices and explicit retry/replay action RPCs.
func TestGetAdminOpsQueueAndActions(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-ops-queue")
	parseJobID, parseNotificationID, parseDeliveryID := parseSeedAdminOpsQueueRows(parseT, parseStore, parseAliceAuth.ID, parseWorkspaceID)

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-ops-queue-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseQueueResp, parseErr := parseServer.GetAdminOpsQueue(parseSuperuserCtx, &chatpb.GetAdminOpsQueueRequest{
		LookbackDays: 30,
		Limit:        50,
		WorkspaceId:  parseWorkspaceID,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminOpsQueue: %v", parseErr)
	}
	if parseQueueResp.GetSummary() == nil {
		parseT.Fatalf("expected ops queue summary payload")
	}
	if len(parseQueueResp.GetFailedBackgroundJobs()) == 0 ||
		len(parseQueueResp.GetFailedNotifications()) == 0 ||
		len(parseQueueResp.GetFailedWebhookDeliveries()) == 0 ||
		len(parseQueueResp.GetIncidentTimeline()) == 0 ||
		len(parseQueueResp.GetRecentAdminActions()) == 0 {
		parseT.Fatalf(
			"expected non-empty ops queue slices, got jobs=%d notifications=%d webhook_deliveries=%d incident_timeline=%d actions=%d",
			len(parseQueueResp.GetFailedBackgroundJobs()),
			len(parseQueueResp.GetFailedNotifications()),
			len(parseQueueResp.GetFailedWebhookDeliveries()),
			len(parseQueueResp.GetIncidentTimeline()),
			len(parseQueueResp.GetRecentAdminActions()),
		)
	}

	if _, parseErr = parseServer.RetryAdminBackgroundJob(parseSuperuserCtx, &chatpb.RetryAdminBackgroundJobRequest{
		JobId:   parseJobID,
		Confirm: true,
		Reason:  "retry failed background job",
	}); parseErr != nil {
		parseT.Fatalf("RetryAdminBackgroundJob: %v", parseErr)
	}
	if _, parseErr = parseServer.RetryAdminNotification(parseSuperuserCtx, &chatpb.RetryAdminNotificationRequest{
		NotificationId: parseNotificationID,
		Confirm:        true,
		Reason:         "retry failed notification",
	}); parseErr != nil {
		parseT.Fatalf("RetryAdminNotification: %v", parseErr)
	}
	parseReplayResp, parseErr := parseServer.ReplayAdminWebhookDelivery(parseSuperuserCtx, &chatpb.ReplayAdminWebhookDeliveryRequest{
		DeliveryId: parseDeliveryID,
		Confirm:    true,
		Reason:     "replay failed webhook delivery",
	})
	if parseErr != nil {
		parseT.Fatalf("ReplayAdminWebhookDelivery: %v", parseErr)
	}
	if parseReplayResp.GetDeliveryId() != parseDeliveryID || parseReplayResp.GetStatus() != "replay_scheduled" || parseReplayResp.GetNextRetryAt() == "" {
		parseT.Fatalf("unexpected webhook replay response: %+v", parseReplayResp)
	}

	parseBackgroundRows, parseErr := parseStore.parseListBackgroundJobs(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs after retry: %v", parseErr)
	}
	parseJobRow, hasParseJobRow := parseFindBackgroundJobRowByID(parseBackgroundRows, parseJobID)
	if !hasParseJobRow || parseJobRow.Status != "pending" {
		parseT.Fatalf("expected pending background job after retry action, row=%+v found=%v", parseJobRow, hasParseJobRow)
	}
	parseNotificationRows, parseErr := parseStore.parseListNotificationOutbox(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseT.Fatalf("parseListNotificationOutbox after retry: %v", parseErr)
	}
	parseNotificationRow, hasParseNotificationRow := parseFindNotificationOutboxRowByID(parseNotificationRows, parseNotificationID)
	if !hasParseNotificationRow || parseNotificationRow.Status != "pending" {
		parseT.Fatalf("expected pending notification after retry action, row=%+v found=%v", parseNotificationRow, hasParseNotificationRow)
	}
	parseDeliveryRows, parseErr := parseStore.parseListWebhookDeliveries(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookDeliveries after replay: %v", parseErr)
	}
	parseDeliveryRow, hasParseDeliveryRow := parseFindWebhookDeliveryRowByID(parseDeliveryRows, parseDeliveryID)
	if !hasParseDeliveryRow || parseDeliveryRow.NextRetryAt == "" {
		parseT.Fatalf("expected replayed webhook with next retry timestamp, row=%+v found=%v", parseDeliveryRow, hasParseDeliveryRow)
	}
}

// TestGetAdminOpsQueueAndActionsScopeGuards verifies workspace-admin callers are denied ops queue and retry/replay action RPCs.
func TestGetAdminOpsQueueAndActionsScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-ops-queue@example.com")
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-workspace-admin-ops-queue")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-ops-queue-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)

	if _, parseErr := parseServer.GetAdminOpsQueue(parseWorkspaceAdminCtx, &chatpb.GetAdminOpsQueueRequest{
		LookbackDays: 30,
		Limit:        25,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminOpsQueue workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr := parseServer.RetryAdminBackgroundJob(parseWorkspaceAdminCtx, &chatpb.RetryAdminBackgroundJobRequest{
		JobId:   1,
		Confirm: true,
		Reason:  "retry",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("RetryAdminBackgroundJob workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr := parseServer.RetryAdminNotification(parseWorkspaceAdminCtx, &chatpb.RetryAdminNotificationRequest{
		NotificationId: 1,
		Confirm:        true,
		Reason:         "retry",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("RetryAdminNotification workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr := parseServer.ReplayAdminWebhookDelivery(parseWorkspaceAdminCtx, &chatpb.ReplayAdminWebhookDeliveryRequest{
		DeliveryId: 1,
		Confirm:    true,
		Reason:     "replay",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("ReplayAdminWebhookDelivery workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
