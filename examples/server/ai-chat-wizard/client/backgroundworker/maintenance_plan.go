package main

import (
	"strings"
	"time"
)

const (
	backgroundWorkerCommandStartMaintenanceLoop = "start-maintenance-loop"
	backgroundWorkerCommandStopMaintenanceLoop  = "stop-maintenance-loop"
	backgroundWorkerEventMaintenanceBatch       = "maintenance-batch"
)

const (
	backgroundWorkerMaintenanceTaskLocaleRefreshCheck  = "locale_refresh_check"
	backgroundWorkerMaintenanceTaskUndeliveredLogDrain = "undelivered_log_drain"
	backgroundWorkerMaintenanceTaskUnsentMessageRetry  = "unsent_message_retry"
	backgroundWorkerMaintenanceTaskLowPriorityPersist  = "low_priority_cache_persist"
	backgroundWorkerMaintenanceTaskStaleRecordSweep    = "stale_record_sweep"
)

// backgroundWorkerMaintenanceCommand stores one loop-start command payload.
type backgroundWorkerMaintenanceCommand struct {
	IntervalMs  int64  `json:"intervalMs"`
	ScopeKey    string `json:"scopeKey"`
	Locale      string `json:"locale"`
	IsOnline    bool   `json:"isOnline"`
	IsGRPCReady bool   `json:"isGrpcReady"`
}

// backgroundWorkerMaintenanceTask stores one payload-agnostic maintenance task request.
type backgroundWorkerMaintenanceTask struct {
	TaskType  string `json:"taskType"`
	ScopeKey  string `json:"scopeKey"`
	QueueKey  string `json:"queueKey"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"createdAt"`
}

// backgroundWorkerMaintenanceBatch stores one grouped maintenance-task batch for main-thread reconciliation.
type backgroundWorkerMaintenanceBatch struct {
	Tasks     []backgroundWorkerMaintenanceTask `json:"tasks"`
	EmittedAt string                            `json:"emittedAt"`
}

// parseBuildBackgroundWorkerMaintenanceTaskBatch builds one deterministic maintenance-task batch from one command envelope.
func parseBuildBackgroundWorkerMaintenanceTaskBatch(parseCommand backgroundWorkerMaintenanceCommand, parseNow time.Time) backgroundWorkerMaintenanceBatch {
	parseCommand = parseNormalizeBackgroundWorkerMaintenanceCommand(parseCommand)
	if parseNow.IsZero() {
		parseNow = time.Now().UTC()
	}
	parseCreatedAt := parseNow.UTC().Format(time.RFC3339)
	parseReason := "offline_or_disconnected"
	if parseCommand.IsOnline && parseCommand.IsGRPCReady {
		parseReason = "connected_maintenance_window"
	}
	parseScopeKey := strings.TrimSpace(parseCommand.ScopeKey)
	parseTasks := []backgroundWorkerMaintenanceTask{
		{
			TaskType:  backgroundWorkerMaintenanceTaskLocaleRefreshCheck,
			ScopeKey:  parseScopeKey,
			QueueKey:  "",
			Reason:    parseReason,
			CreatedAt: parseCreatedAt,
		},
		{
			TaskType:  backgroundWorkerMaintenanceTaskUndeliveredLogDrain,
			ScopeKey:  parseScopeKey,
			QueueKey:  strings.TrimSpace(parseScopeKey + "::outbox.logs"),
			Reason:    parseReason,
			CreatedAt: parseCreatedAt,
		},
		{
			TaskType:  backgroundWorkerMaintenanceTaskUnsentMessageRetry,
			ScopeKey:  parseScopeKey,
			QueueKey:  strings.TrimSpace(parseScopeKey + "::outbox.unsent_messages"),
			Reason:    parseReason,
			CreatedAt: parseCreatedAt,
		},
		{
			TaskType:  backgroundWorkerMaintenanceTaskLowPriorityPersist,
			ScopeKey:  parseScopeKey,
			QueueKey:  "",
			Reason:    "idle_persist_window",
			CreatedAt: parseCreatedAt,
		},
		{
			TaskType:  backgroundWorkerMaintenanceTaskStaleRecordSweep,
			ScopeKey:  parseScopeKey,
			QueueKey:  "",
			Reason:    "ttl_scan_window",
			CreatedAt: parseCreatedAt,
		},
	}
	return backgroundWorkerMaintenanceBatch{
		Tasks:     parseTasks,
		EmittedAt: parseCreatedAt,
	}
}

// parseNormalizeBackgroundWorkerMaintenanceCommand normalizes one maintenance command into a stable loop policy.
func parseNormalizeBackgroundWorkerMaintenanceCommand(parseCommand backgroundWorkerMaintenanceCommand) backgroundWorkerMaintenanceCommand {
	parseCommand.ScopeKey = strings.TrimSpace(parseCommand.ScopeKey)
	parseCommand.Locale = strings.TrimSpace(parseCommand.Locale)
	if parseCommand.IntervalMs <= 0 {
		parseCommand.IntervalMs = 60000
	}
	return parseCommand
}
