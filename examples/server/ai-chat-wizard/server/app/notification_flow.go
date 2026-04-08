package app

import (
	"context"
	"errors"
	"log/slog"
)

// parseDispatchNotificationOutboxPending dispatches pending notification rows and records sent or failed status.
func parseDispatchNotificationOutboxPending(parseCtx context.Context, parseStore *Store, parseNow string, parseLimit int64, parseDeliver func(context.Context, parseNotificationOutboxRow) error) (int, error) {
	if parseStore == nil {
		parseErr := errors.New("dispatch notification outbox pending: store is required")
		slog.Default().Error("notification dispatch failed", slog.String("error", parseErr.Error()))
		return 0, parseErr
	}
	if parseDeliver == nil {
		parseErr := errors.New("dispatch notification outbox pending: deliver callback is required")
		slog.Default().Error("notification dispatch failed", slog.String("error", parseErr.Error()))
		return 0, parseErr
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parsePendingRows, parseErr := parseStore.parseListNotificationOutboxPending(parseNow, parseLimit)
	if parseErr != nil {
		slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{ParseAction: "notification_dispatch"})...).Error("notification outbox list failed", slog.String("error", parseErr.Error()))
		return 0, parseErr
	}

	parseProcessedCount := 0
	for _, parseRow := range parsePendingRows {
		if parseErr2 := parseStore.parseRequireUserOperational(parseRow.UserID); parseErr2 != nil {
			slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
				ParseAction:       "notification_dispatch",
				ParseTargetID:     parseRow.NotificationKey,
				ParseTargetScope:  parseRow.ChannelKey,
				ParseWorkspaceID:  parseRow.WorkspaceID,
				ParseTargetUserID: parseRow.UserID,
			})...).Warn("notification blocked by user state", slog.String("error", parseErr2.Error()))
			if parseUpdateErr := parseStore.parseUpdateNotificationOutboxStatus(parseRow.ID, "failed", "", parseNow, parseErr2.Error()); parseUpdateErr != nil {
				slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
					ParseAction:       "notification_dispatch",
					ParseTargetID:     parseRow.NotificationKey,
					ParseTargetScope:  parseRow.ChannelKey,
					ParseWorkspaceID:  parseRow.WorkspaceID,
					ParseTargetUserID: parseRow.UserID,
				})...).Error("notification failure update failed", slog.String("error", parseUpdateErr.Error()))
				return parseProcessedCount, parseUpdateErr
			}
			parseProcessedCount++
			continue
		}
		if parseErr2 := parseStore.parseRequireWorkspaceOperational(parseRow.WorkspaceID); parseErr2 != nil {
			slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
				ParseAction:       "notification_dispatch",
				ParseTargetID:     parseRow.NotificationKey,
				ParseTargetScope:  parseRow.ChannelKey,
				ParseWorkspaceID:  parseRow.WorkspaceID,
				ParseTargetUserID: parseRow.UserID,
			})...).Warn("notification blocked by workspace state", slog.String("error", parseErr2.Error()))
			if parseUpdateErr := parseStore.parseUpdateNotificationOutboxStatus(parseRow.ID, "failed", "", parseNow, parseErr2.Error()); parseUpdateErr != nil {
				slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
					ParseAction:       "notification_dispatch",
					ParseTargetID:     parseRow.NotificationKey,
					ParseTargetScope:  parseRow.ChannelKey,
					ParseWorkspaceID:  parseRow.WorkspaceID,
					ParseTargetUserID: parseRow.UserID,
				})...).Error("notification failure update failed", slog.String("error", parseUpdateErr.Error()))
				return parseProcessedCount, parseUpdateErr
			}
			parseProcessedCount++
			continue
		}
		if parseDeliverErr := parseDeliver(parseCtx, parseRow); parseDeliverErr != nil {
			slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
				ParseAction:       "notification_dispatch",
				ParseTargetID:     parseRow.NotificationKey,
				ParseTargetScope:  parseRow.ChannelKey,
				ParseWorkspaceID:  parseRow.WorkspaceID,
				ParseTargetUserID: parseRow.UserID,
			})...).Error("notification delivery failed", slog.String("error", parseDeliverErr.Error()))
			if parseErr2 := parseStore.parseUpdateNotificationOutboxStatus(parseRow.ID, "failed", "", parseNow, parseDeliverErr.Error()); parseErr2 != nil {
				slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
					ParseAction:       "notification_dispatch",
					ParseTargetID:     parseRow.NotificationKey,
					ParseTargetScope:  parseRow.ChannelKey,
					ParseWorkspaceID:  parseRow.WorkspaceID,
					ParseTargetUserID: parseRow.UserID,
				})...).Error("notification delivery failure update failed", slog.String("error", parseErr2.Error()))
				return parseProcessedCount, parseErr2
			}
			parseProcessedCount++
			continue
		}
		if parseErr2 := parseStore.parseUpdateNotificationOutboxStatus(parseRow.ID, "sent", parseNow, "", ""); parseErr2 != nil {
			slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
				ParseAction:       "notification_dispatch",
				ParseTargetID:     parseRow.NotificationKey,
				ParseTargetScope:  parseRow.ChannelKey,
				ParseWorkspaceID:  parseRow.WorkspaceID,
				ParseTargetUserID: parseRow.UserID,
			})...).Error("notification sent update failed", slog.String("error", parseErr2.Error()))
			return parseProcessedCount, parseErr2
		}
		parseProcessedCount++
	}
	return parseProcessedCount, nil
}
