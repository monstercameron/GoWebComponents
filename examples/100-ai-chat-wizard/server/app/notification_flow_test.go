package app

import (
	"errors"
	"strings"
	"testing"
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

	if _, parseErr = parseStore.parseCreateNotificationOutbox(parseNotificationOutboxWrite{
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
	if _, parseErr = parseStore.parseCreateNotificationOutbox(parseNotificationOutboxWrite{
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
	if _, parseErr = parseStore.parseCreateNotificationOutbox(parseNotificationOutboxWrite{
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

	parseProcessed, parseErr := parseDispatchNotificationOutboxPending(parseStore, parseNow, 10, func(parseRow parseNotificationOutboxRow) error {
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
