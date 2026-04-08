package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

// TestBuildTraceabilityContextFromRequestPreservesHeaderMetadata verifies HTTP entry keeps traceability headers in context.
func TestBuildTraceabilityContextFromRequestPreservesHeaderMetadata(parseT *testing.T) {
	parseReq, parseErr := http.NewRequest(http.MethodGet, "http://example.com/app", nil)
	if parseErr != nil {
		parseT.Fatalf("http.NewRequest: %v", parseErr)
	}
	parseReq.Header.Set(requestIDMetadataKey, "req-http-123")
	parseReq.Header.Set(correlationIDMetadataKey, "corr-http-456")
	parseReq.Header.Set(traceParentMetadataKey, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	parseReq.Header.Set(traceStateMetadataKey, "vendor=relay")

	parseTraceableCtx := parseBuildTraceabilityContextFromRequest(parseReq)
	parseIncomingMD, parseOk := metadata.FromIncomingContext(parseTraceableCtx)
	if !parseOk {
		parseT.Fatal("expected incoming metadata on traceable request context")
	}
	if parseIncomingMD.Get(requestIDMetadataKey)[0] != "req-http-123" {
		parseT.Fatalf("request id = %q, want req-http-123", parseIncomingMD.Get(requestIDMetadataKey)[0])
	}
	if parseIncomingMD.Get(correlationIDMetadataKey)[0] != "corr-http-456" {
		parseT.Fatalf("correlation id = %q, want corr-http-456", parseIncomingMD.Get(correlationIDMetadataKey)[0])
	}
	if parseIncomingMD.Get(traceParentMetadataKey)[0] != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		parseT.Fatalf("traceparent = %q, want test traceparent", parseIncomingMD.Get(traceParentMetadataKey)[0])
	}
	if parseIncomingMD.Get(traceStateMetadataKey)[0] != "vendor=relay" {
		parseT.Fatalf("tracestate = %q, want vendor=relay", parseIncomingMD.Get(traceStateMetadataKey)[0])
	}
}

// TestStoreBackgroundJobIncludesTraceabilityEnvelope verifies queued background jobs persist traceability metadata.
func TestStoreBackgroundJobIncludesTraceabilityEnvelope(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, parseStore, "traceability-job@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseOwner.ID, "ws-traceability-job")
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-job-123",
		correlationIDMetadataKey, "corr-job-456",
		traceParentMetadataKey, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		traceStateMetadataKey, "vendor=relay",
	))
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if parseErr := parseStoreWeeklySummaryJob(parseCtx, parseStore, "job-traceability", fmt.Sprintf(`{"workspace_id":%d,"user_id":%d}`, parseWorkspaceID, parseOwner.ID), parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreWeeklySummaryJob: %v", parseErr)
	}
	parseRows, parseErr := parseStore.parseListBackgroundJobs(10)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs: %v", parseErr)
	}
	if len(parseRows) != 1 {
		parseT.Fatalf("expected one background job row, got %d", len(parseRows))
	}
	var parsePayload map[string]any
	if parseErr = json.Unmarshal([]byte(parseRows[0].PayloadJSON), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal(payload): %v", parseErr)
	}
	parseTraceabilityPayload, isParseMap := parsePayload["traceability"].(map[string]any)
	if !isParseMap {
		parseT.Fatalf("traceability payload missing: %+v", parsePayload)
	}
	if parseTraceabilityPayload["request_id"] != "req-job-123" || parseTraceabilityPayload["correlation_id"] != "corr-job-456" {
		parseT.Fatalf("unexpected job traceability payload: %+v", parseTraceabilityPayload)
	}
}

// TestCreateNotificationOutboxIncludesTraceabilityEnvelope verifies notification payloads persist traceability metadata.
func TestCreateNotificationOutboxIncludesTraceabilityEnvelope(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, parseStore, "traceability-notify@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseOwner.ID, "ws-traceability-notify")
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-notify-123",
		correlationIDMetadataKey, "corr-notify-456",
	))
	if _, parseErr := parseStore.parseCreateNotificationOutbox(parseCtx, parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseOwner.ID,
		NotificationKey: "notify-traceability",
		ChannelKey:      "email",
		TemplateKey:     "welcome",
		Status:          "pending",
		Subject:         "Traceability",
		BodyText:        "Body",
		PayloadJSON:     `{"kind":"test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox: %v", parseErr)
	}
	parseRows, parseErr := parseStore.parseListNotificationOutbox(10)
	if parseErr != nil {
		parseT.Fatalf("parseListNotificationOutbox: %v", parseErr)
	}
	if len(parseRows) != 1 {
		parseT.Fatalf("expected one notification row, got %d", len(parseRows))
	}
	var parsePayload map[string]any
	if parseErr = json.Unmarshal([]byte(parseRows[0].PayloadJSON), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal(payload): %v", parseErr)
	}
	parseTraceabilityPayload, isParseMap := parsePayload["traceability"].(map[string]any)
	if !isParseMap {
		parseT.Fatalf("traceability payload missing: %+v", parsePayload)
	}
	if parseTraceabilityPayload["request_id"] != "req-notify-123" || parseTraceabilityPayload["correlation_id"] != "corr-notify-456" {
		parseT.Fatalf("unexpected notification traceability payload: %+v", parseTraceabilityPayload)
	}
}

// TestUpsertWebhookDeliveryIncludesTraceabilityEnvelope verifies webhook delivery request headers persist traceability metadata.
func TestUpsertWebhookDeliveryIncludesTraceabilityEnvelope(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, parseStore, "traceability-webhook@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseOwner.ID, "ws-traceability-webhook")
	if parseErr := parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID: parseWorkspaceID,
		Label:       "Traceability Webhook",
		TargetURL:   "https://example.com/hook",
		SecretHash:  "hash",
		EventsJSON:  `["chat.completed"]`,
		IsEnabled:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint: %v", parseErr)
	}
	parseEndpointRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints: %v", parseErr)
	}
	if len(parseEndpointRows) != 1 {
		parseT.Fatalf("expected one webhook endpoint row, got %d", len(parseEndpointRows))
	}
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-webhook-123",
		correlationIDMetadataKey, "corr-webhook-456",
	))
	if parseErr := parseStore.parseUpsertWebhookDelivery(parseCtx, parseWebhookDeliveryWrite{
		EndpointID:         parseEndpointRows[0].ID,
		EventType:          "chat.completed",
		DeliveryKey:        "delivery-traceability",
		RequestHeadersJSON: `{"x-signature":"abc"}`,
		RequestBodyJSON:    `{"event":"chat.completed"}`,
		ResponseStatus:     200,
		ResponseBody:       "ok",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookDelivery: %v", parseErr)
	}
	parseDeliveryRows, parseErr := parseStore.parseListWebhookDeliveries(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookDeliveries: %v", parseErr)
	}
	if len(parseDeliveryRows) != 1 {
		parseT.Fatalf("expected one webhook delivery row, got %d", len(parseDeliveryRows))
	}
	var parseHeaders map[string]any
	if parseErr = json.Unmarshal([]byte(parseDeliveryRows[0].RequestHeadersJSON), &parseHeaders); parseErr != nil {
		parseT.Fatalf("json.Unmarshal(headers): %v", parseErr)
	}
	parseTraceabilityPayload, isParseMap := parseHeaders["traceability"].(map[string]any)
	if !isParseMap {
		parseT.Fatalf("traceability headers missing: %+v", parseHeaders)
	}
	if parseTraceabilityPayload["request_id"] != "req-webhook-123" || parseTraceabilityPayload["correlation_id"] != "corr-webhook-456" {
		parseT.Fatalf("unexpected webhook traceability headers: %+v", parseTraceabilityPayload)
	}
	if parseHeaders["x-signature"] != "abc" {
		parseT.Fatalf("expected signature header to remain present, got %+v", parseHeaders)
	}
}

// TestHandleBackgroundJobsPropagatesTraceabilityContext verifies job handlers receive the same traceability context.
func TestHandleBackgroundJobsPropagatesTraceabilityContext(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-job-dispatch-123",
		correlationIDMetadataKey, "corr-job-dispatch-456",
	))
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if parseErr := parseStoreWeeklySummaryJob(parseCtx, parseStore, "job-dispatch-traceability", `{}`, parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreWeeklySummaryJob: %v", parseErr)
	}
	parseHandlerCalled := false
	parseProcessed, parseErr := parseHandleBackgroundJobs(parseCtx, parseStore, parseNow, 10, parseBackgroundJobHandlers{
		HandleWeeklySummary: func(parseHandlerCtx context.Context, parseRow parseBackgroundJobRow) error {
			parseHandlerCalled = true
			parseIncomingMD, parseOk := metadata.FromIncomingContext(parseHandlerCtx)
			if !parseOk {
				parseT.Fatal("expected incoming metadata in job handler context")
			}
			if parseIncomingMD.Get(requestIDMetadataKey)[0] != "req-job-dispatch-123" || parseIncomingMD.Get(correlationIDMetadataKey)[0] != "corr-job-dispatch-456" {
				parseT.Fatalf("unexpected handler traceability metadata: %+v", parseIncomingMD)
			}
			if parseRow.JobKey != "job-dispatch-traceability" {
				parseT.Fatalf("unexpected job row: %+v", parseRow)
			}
			return nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleBackgroundJobs: %v", parseErr)
	}
	if parseProcessed != 1 || !parseHandlerCalled {
		parseT.Fatalf("expected handler to run once, got processed=%d called=%v", parseProcessed, parseHandlerCalled)
	}
}

// TestDispatchNotificationOutboxPendingPropagatesTraceabilityContext verifies notification delivery callbacks receive the same traceability context.
func TestDispatchNotificationOutboxPendingPropagatesTraceabilityContext(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, parseStore, "traceability-notify-dispatch@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseOwner.ID, "ws-traceability-notify-dispatch")
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-notify-dispatch-123",
		correlationIDMetadataKey, "corr-notify-dispatch-456",
	))
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseStore.parseCreateNotificationOutbox(parseCtx, parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseOwner.ID,
		NotificationKey: "notify-dispatch-traceability",
		ChannelKey:      "email",
		TemplateKey:     "welcome",
		Status:          "pending",
		Subject:         "Traceability",
		BodyText:        "Body",
		ScheduledAt:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox: %v", parseErr)
	}
	parseDeliverCalled := false
	parseProcessed, parseErr := parseDispatchNotificationOutboxPending(parseCtx, parseStore, parseNow, 10, func(parseDeliverCtx context.Context, parseRow parseNotificationOutboxRow) error {
		parseDeliverCalled = true
		parseIncomingMD, parseOk := metadata.FromIncomingContext(parseDeliverCtx)
		if !parseOk {
			parseT.Fatal("expected incoming metadata in notification delivery context")
		}
		if parseIncomingMD.Get(requestIDMetadataKey)[0] != "req-notify-dispatch-123" || parseIncomingMD.Get(correlationIDMetadataKey)[0] != "corr-notify-dispatch-456" {
			parseT.Fatalf("unexpected delivery traceability metadata: %+v", parseIncomingMD)
		}
		if parseRow.NotificationKey != "notify-dispatch-traceability" {
			parseT.Fatalf("unexpected notification row: %+v", parseRow)
		}
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("parseDispatchNotificationOutboxPending: %v", parseErr)
	}
	if parseProcessed != 1 || !parseDeliverCalled {
		parseT.Fatalf("expected delivery callback once, got processed=%d called=%v", parseProcessed, parseDeliverCalled)
	}
}
