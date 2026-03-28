package app

import "errors"

// parseDispatchNotificationOutboxPending dispatches pending notification rows and records sent or failed status.
func parseDispatchNotificationOutboxPending(parseStore *Store, parseNow string, parseLimit int64, parseDeliver func(parseNotificationOutboxRow) error) (int, error) {
	if parseStore == nil {
		return 0, errors.New("dispatch notification outbox pending: store is required")
	}
	if parseDeliver == nil {
		return 0, errors.New("dispatch notification outbox pending: deliver callback is required")
	}
	parsePendingRows, parseErr := parseStore.parseListNotificationOutboxPending(parseNow, parseLimit)
	if parseErr != nil {
		return 0, parseErr
	}

	parseProcessedCount := 0
	for _, parseRow := range parsePendingRows {
		if parseErr2 := parseStore.parseRequireUserOperational(parseRow.UserID); parseErr2 != nil {
			if parseUpdateErr := parseStore.parseUpdateNotificationOutboxStatus(parseRow.ID, "failed", "", parseNow, parseErr2.Error()); parseUpdateErr != nil {
				return parseProcessedCount, parseUpdateErr
			}
			parseProcessedCount++
			continue
		}
		if parseErr2 := parseStore.parseRequireWorkspaceOperational(parseRow.WorkspaceID); parseErr2 != nil {
			if parseUpdateErr := parseStore.parseUpdateNotificationOutboxStatus(parseRow.ID, "failed", "", parseNow, parseErr2.Error()); parseUpdateErr != nil {
				return parseProcessedCount, parseUpdateErr
			}
			parseProcessedCount++
			continue
		}
		if parseDeliverErr := parseDeliver(parseRow); parseDeliverErr != nil {
			if parseErr2 := parseStore.parseUpdateNotificationOutboxStatus(parseRow.ID, "failed", "", parseNow, parseDeliverErr.Error()); parseErr2 != nil {
				return parseProcessedCount, parseErr2
			}
			parseProcessedCount++
			continue
		}
		if parseErr2 := parseStore.parseUpdateNotificationOutboxStatus(parseRow.ID, "sent", parseNow, "", ""); parseErr2 != nil {
			return parseProcessedCount, parseErr2
		}
		parseProcessedCount++
	}
	return parseProcessedCount, nil
}
