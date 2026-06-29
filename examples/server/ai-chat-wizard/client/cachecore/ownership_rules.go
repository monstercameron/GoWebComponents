package cachecore

import "strings"

// OwnershipActor identifies whether work belongs to the main thread or background worker.
type OwnershipActor string

const (
	OwnershipActorMainThread OwnershipActor = "main_thread"
	OwnershipActorWorker     OwnershipActor = "background_worker"
)

// OwnershipTask identifies cache/outbox lifecycle responsibilities.
type OwnershipTask string

const (
	OwnershipTaskHotRead           OwnershipTask = "hot_ui_read"
	OwnershipTaskOptimisticState   OwnershipTask = "optimistic_state"
	OwnershipTaskInvalidationSweep OwnershipTask = "invalidation_sweep"
	OwnershipTaskTTLScan           OwnershipTask = "ttl_scan"
	OwnershipTaskRetryScheduling   OwnershipTask = "retry_scheduling"
	OwnershipTaskQueuedLogFlush    OwnershipTask = "queued_log_flush"
	OwnershipTaskUnsentResend      OwnershipTask = "unsent_message_resend"
)

// OwnershipDecision stores one ownership rule resolution.
type OwnershipDecision struct {
	Actor  OwnershipActor
	Reason string
}

// ResolveOwnershipDecision resolves one canonical ownership actor for one cache/outbox task.
func ResolveOwnershipDecision(parseTask OwnershipTask) OwnershipDecision {
	switch normalizeOwnershipTask(parseTask) {
	case OwnershipTaskHotRead:
		return OwnershipDecision{Actor: OwnershipActorMainThread, Reason: "hot_ui_reads_must_remain_synchronous"}
	case OwnershipTaskOptimisticState:
		return OwnershipDecision{Actor: OwnershipActorMainThread, Reason: "optimistic_state_must_track_user_input_without_worker_roundtrips"}
	default:
		return OwnershipDecision{Actor: OwnershipActorWorker, Reason: "maintenance_work_offloaded_to_background_worker"}
	}
}

// normalizeOwnershipTask resolves one canonical ownership task fallback.
func normalizeOwnershipTask(parseTask OwnershipTask) OwnershipTask {
	switch strings.TrimSpace(string(parseTask)) {
	case string(OwnershipTaskHotRead):
		return OwnershipTaskHotRead
	case string(OwnershipTaskOptimisticState):
		return OwnershipTaskOptimisticState
	case string(OwnershipTaskInvalidationSweep):
		return OwnershipTaskInvalidationSweep
	case string(OwnershipTaskTTLScan):
		return OwnershipTaskTTLScan
	case string(OwnershipTaskRetryScheduling):
		return OwnershipTaskRetryScheduling
	case string(OwnershipTaskQueuedLogFlush):
		return OwnershipTaskQueuedLogFlush
	case string(OwnershipTaskUnsentResend):
		return OwnershipTaskUnsentResend
	default:
		return OwnershipTaskInvalidationSweep
	}
}
