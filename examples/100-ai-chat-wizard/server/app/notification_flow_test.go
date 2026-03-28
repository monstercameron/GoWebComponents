package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"
)

// TestDispatchNotificationOutboxPendingLifecycle verifies pending outbox rows transition to sent and failed states via dispatch callback outcomes.
func TestDispatchNotificationOutboxPendingLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, parseStore, "notify-owner@example.com")
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "notify-workspace",
		Slug:         "notify-workspace",
		Name:         "Notify Workspace",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseOwner.ID,
		SettingsJSON: `{"region":"us"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace: %v", parseErr)
	}
	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil || len(parseWorkspaces) == 0 {
		parseT.Fatalf("parseListWorkspaces: rows=%+v err=%v", parseWorkspaces, parseErr)
	}
	parseWorkspaceID := parseWorkspaces[0].ID
	parseNow := "2026-03-27T22:00:00Z"

	if _, parseErr = parseStore.parseCreateNotificationOutbox(context.Background(), parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseOwner.ID,
		NotificationKey: "ready-ok",
		ChannelKey:      "email",
		TemplateKey:     "weekly_value",
		Status:          "pending",
		Subject:         "OK",
		BodyText:        "ready",
		PayloadJSON:     `{}`,
		ScheduledAt:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox ready-ok: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateNotificationOutbox(context.Background(), parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseOwner.ID,
		NotificationKey: "ready-fail",
		ChannelKey:      "email",
		TemplateKey:     "weekly_value",
		Status:          "pending",
		Subject:         "FAIL",
		BodyText:        "ready",
		PayloadJSON:     `{}`,
		ScheduledAt:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox ready-fail: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateNotificationOutbox(context.Background(), parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseOwner.ID,
		NotificationKey: "future",
		ChannelKey:      "email",
		TemplateKey:     "weekly_value",
		Status:          "pending",
		Subject:         "FUTURE",
		BodyText:        "later",
		PayloadJSON:     `{}`,
		ScheduledAt:     "2026-03-27T23:00:00Z",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox future: %v", parseErr)
	}

	parseProcessed, parseErr := parseDispatchNotificationOutboxPending(context.Background(), parseStore, parseNow, 10, func(_ context.Context, parseRow parseNotificationOutboxRow) error {
		if parseRow.NotificationKey == "ready-fail" {
			return errors.New("smtp unavailable")
		}
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("parseDispatchNotificationOutboxPending: %v", parseErr)
	}
	if parseProcessed != 2 {
		parseT.Fatalf("expected 2 processed rows, got %d", parseProcessed)
	}

	parseRows, parseErr := parseStore.parseListNotificationOutbox(20)
	if parseErr != nil {
		parseT.Fatalf("parseListNotificationOutbox: %v", parseErr)
	}
	parseRowsByKey := make(map[string]parseNotificationOutboxRow, len(parseRows))
	for _, parseRow := range parseRows {
		parseRowsByKey[parseRow.NotificationKey] = parseRow
	}
	if parseRow, hasParseRow := parseRowsByKey["ready-ok"]; !hasParseRow || parseRow.Status != "sent" || parseRow.SentAt != parseNow {
		parseT.Fatalf("unexpected ready-ok row: found=%v row=%+v", hasParseRow, parseRow)
	}
	if parseRow, hasParseRow := parseRowsByKey["ready-fail"]; !hasParseRow || parseRow.Status != "failed" || parseRow.FailedAt != parseNow || !strings.Contains(parseRow.ErrorMessage, "smtp") {
		parseT.Fatalf("unexpected ready-fail row: found=%v row=%+v", hasParseRow, parseRow)
	}
	if parseRow, hasParseRow := parseRowsByKey["future"]; !hasParseRow || parseRow.Status != "pending" {
		parseT.Fatalf("unexpected future row: found=%v row=%+v", hasParseRow, parseRow)
	}
}

// TestDispatchNotificationOutboxPendingLogsDeliveryFailures verifies failed deliveries emit one boundary log entry.
func TestDispatchNotificationOutboxPendingLogsDeliveryFailures(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, parseStore, "notify-log-owner@example.com")
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: "notify-log-workspace",
		Slug:         "notify-log-workspace",
		Name:         "Notify Log Workspace",
		PlanCode:     "team",
		Status:       "active",
		OwnerUserID:  parseOwner.ID,
		SettingsJSON: `{"region":"us"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace: %v", parseErr)
	}
	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil || len(parseWorkspaces) == 0 {
		parseT.Fatalf("parseListWorkspaces: rows=%+v err=%v", parseWorkspaces, parseErr)
	}
	parseWorkspaceID := parseWorkspaces[0].ID
	parseNow := "2026-03-27T22:00:00Z"
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-notify-log-123",
		correlationIDMetadataKey, "corr-notify-log-456",
	))
	if _, parseErr = parseStore.parseCreateNotificationOutbox(parseCtx, parseNotificationOutboxWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseOwner.ID,
		NotificationKey: "notify-log",
		ChannelKey:      "email",
		TemplateKey:     "weekly_value",
		Status:          "pending",
		Subject:         "LOG",
		BodyText:        "ready",
		PayloadJSON:     `{}`,
		ScheduledAt:     parseNow,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateNotificationOutbox: %v", parseErr)
	}

	var parseBuffer bytes.Buffer
	parseOriginalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&parseBuffer, nil)))
	parseT.Cleanup(func() {
		slog.SetDefault(parseOriginalLogger)
	})

	parseProcessed, parseErr := parseDispatchNotificationOutboxPending(parseCtx, parseStore, parseNow, 10, func(context.Context, parseNotificationOutboxRow) error {
		return errors.New("smtp unavailable")
	})
	if parseErr != nil {
		parseT.Fatalf("parseDispatchNotificationOutboxPending: %v", parseErr)
	}
	if parseProcessed != 1 {
		parseT.Fatalf("expected 1 processed row, got %d", parseProcessed)
	}
	parseLogOutput := parseBuffer.String()
	if !strings.Contains(parseLogOutput, "notification delivery failed") || !strings.Contains(parseLogOutput, "notify-log") {
		parseT.Fatalf("expected notification failure log, got %q", parseLogOutput)
	}
}
