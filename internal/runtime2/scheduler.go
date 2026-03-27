package runtime2

import (
	"fmt"
	"log"
	"strings"
)

// SchedulerJobKind identifies the scheduler queue operation kind.
type SchedulerJobKind string

const (
	schedulerJobKindInvalid SchedulerJobKind = ""

	SchedulerJobKindMount  SchedulerJobKind = "mount"
	SchedulerJobKindUpdate SchedulerJobKind = "update"
)

// SchedulerWorkerHealth identifies scheduler-visible worker health states.
type SchedulerWorkerHealth uint8

const (
	schedulerWorkerHealthInvalid SchedulerWorkerHealth = iota

	SchedulerWorkerHealthReady
	SchedulerWorkerHealthDegraded
	SchedulerWorkerHealthRestarting
	SchedulerWorkerHealthDead
)

// SchedulerJob stores one scheduler-queued operation for a region and shard pair.
type SchedulerJob struct {
	GetSchedulerJobKind       SchedulerJobKind
	GetSchedulerRegionID      string
	GetSchedulerShardID       SchedulerShardID
	GetSchedulerCancelVersion uint64
}

// Scheduler routes region jobs onto shard assignments and stores queued work.
type Scheduler struct {
	getSchedulerShardModel              *SchedulerShardModel
	storeSchedulerShardIDs              []SchedulerShardID
	storeSchedulerQueue                 []SchedulerJob
	storeSchedulerQueueLimit            int
	storeSchedulerCancelByRegionID      map[string]uint64
	storeSchedulerFallbackByRegionID    map[string]bool
	storeSchedulerWorkerHealthByShardID map[SchedulerShardID]SchedulerWorkerHealth
	storeSchedulerPongByShardID         map[SchedulerShardID]uint64
	storeSchedulerMissedPongByShardID   map[SchedulerShardID]uint64
	isSchedulerDegraded                 bool
}

// BuildScheduler creates a scheduler with fixed live shard IDs and empty queue state.
func BuildScheduler(parseSchedulerShardIDs []SchedulerShardID) *Scheduler {
	return BuildSchedulerWithQueueLimit(parseSchedulerShardIDs, 0)
}

// BuildSchedulerWithQueueLimit creates a scheduler with fixed live shard IDs and optional queue backpressure.
func BuildSchedulerWithQueueLimit(parseSchedulerShardIDs []SchedulerShardID, parseSchedulerQueueLimit int) *Scheduler {
	getSchedulerShardIDs := append([]SchedulerShardID(nil), parseSchedulerShardIDs...)
	buildScheduler := &Scheduler{
		getSchedulerShardModel:              BuildSchedulerShardModel(),
		storeSchedulerShardIDs:              getSchedulerShardIDs,
		storeSchedulerQueueLimit:            parseSchedulerQueueLimit,
		storeSchedulerCancelByRegionID:      make(map[string]uint64),
		storeSchedulerFallbackByRegionID:    make(map[string]bool),
		storeSchedulerWorkerHealthByShardID: make(map[SchedulerShardID]SchedulerWorkerHealth, len(getSchedulerShardIDs)),
		storeSchedulerPongByShardID:         make(map[SchedulerShardID]uint64, len(getSchedulerShardIDs)),
		storeSchedulerMissedPongByShardID:   make(map[SchedulerShardID]uint64, len(getSchedulerShardIDs)),
	}
	for _, getSchedulerShardID := range getSchedulerShardIDs {
		buildScheduler.storeSchedulerWorkerHealthByShardID[getSchedulerShardID] = SchedulerWorkerHealthReady
		buildScheduler.storeSchedulerPongByShardID[getSchedulerShardID] = 0
		buildScheduler.storeSchedulerMissedPongByShardID[getSchedulerShardID] = 0
	}
	return buildScheduler
}

// HandleSchedulerKeepalivePong records one shard pong sequence and restores ready health for the shard.
func (parseScheduler *Scheduler) HandleSchedulerKeepalivePong(parseSchedulerShardID SchedulerShardID, parsePongSequence uint64) error {
	if parseScheduler == nil {
		return fmt.Errorf("runtime2: scheduler is nil")
	}
	if parsePongSequence == 0 {
		return fmt.Errorf("runtime2: pong sequence is required")
	}
	if hasSchedulerShardLive := hasSchedulerShardID(parseScheduler.storeSchedulerShardIDs, parseSchedulerShardID); !hasSchedulerShardLive {
		return fmt.Errorf("runtime2: shard %q is unknown", parseSchedulerShardID)
	}
	getLastPongSequence := parseScheduler.storeSchedulerPongByShardID[parseSchedulerShardID]
	if parsePongSequence < getLastPongSequence {
		return fmt.Errorf("runtime2: stale pong sequence %d for shard %q (latest=%d)", parsePongSequence, parseSchedulerShardID, getLastPongSequence)
	}
	parseScheduler.storeSchedulerPongByShardID[parseSchedulerShardID] = parsePongSequence
	parseScheduler.storeSchedulerMissedPongByShardID[parseSchedulerShardID] = 0
	parseScheduler.storeSchedulerWorkerHealthByShardID[parseSchedulerShardID] = SchedulerWorkerHealthReady
	parseScheduler.updateSchedulerDegradedState()
	return nil
}

// HandleSchedulerKeepaliveTimeout records one missed pong window and degrades or kills shard health after repeated misses.
func (parseScheduler *Scheduler) HandleSchedulerKeepaliveTimeout(parseSchedulerShardID SchedulerShardID) (SchedulerWorkerHealth, error) {
	if parseScheduler == nil {
		return schedulerWorkerHealthInvalid, fmt.Errorf("runtime2: scheduler is nil")
	}
	if hasSchedulerShardLive := hasSchedulerShardID(parseScheduler.storeSchedulerShardIDs, parseSchedulerShardID); !hasSchedulerShardLive {
		return schedulerWorkerHealthInvalid, fmt.Errorf("runtime2: shard %q is unknown", parseSchedulerShardID)
	}
	parseScheduler.storeSchedulerMissedPongByShardID[parseSchedulerShardID] = parseScheduler.storeSchedulerMissedPongByShardID[parseSchedulerShardID] + 1
	getMissedPongWindows := parseScheduler.storeSchedulerMissedPongByShardID[parseSchedulerShardID]
	if getMissedPongWindows == 1 {
		parseScheduler.storeSchedulerWorkerHealthByShardID[parseSchedulerShardID] = SchedulerWorkerHealthDegraded
		parseScheduler.updateSchedulerDegradedState()
		return SchedulerWorkerHealthDegraded, nil
	}
	parseScheduler.storeSchedulerWorkerHealthByShardID[parseSchedulerShardID] = SchedulerWorkerHealthDead
	parseScheduler.updateSchedulerDegradedState()
	return SchedulerWorkerHealthDead, nil
}

// HandleSchedulerMount assigns a shard for the region and enqueues the first mount job.
func (parseScheduler *Scheduler) HandleSchedulerMount(parseRegionID string) (SchedulerJob, error) {
	if parseScheduler == nil {
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler is nil")
	}
	getSchedulerShardID, getSchedulerShardErr := parseScheduler.getSchedulerShardModel.GetSchedulerRegionShardID(parseRegionID, parseScheduler.storeSchedulerShardIDs, SchedulerAssignmentPolicyKeep)
	if getSchedulerShardErr != nil {
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler mount for region %q failed: %w", parseRegionID, getSchedulerShardErr)
	}
	switch parseScheduler.getSchedulerWorkerHealth(getSchedulerShardID) {
	case SchedulerWorkerHealthReady:
	case SchedulerWorkerHealthDegraded:
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler mount for region %q failed: assigned shard %q is degraded", parseRegionID, getSchedulerShardID)
	case SchedulerWorkerHealthDead, SchedulerWorkerHealthRestarting:
		getSchedulerShardID, getSchedulerShardErr = parseScheduler.getSchedulerRepairShardID(parseRegionID)
		if getSchedulerShardErr != nil {
			return SchedulerJob{}, fmt.Errorf("runtime2: scheduler mount for region %q failed: %w", parseRegionID, getSchedulerShardErr)
		}
	default:
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler mount for region %q failed: assigned shard %q has invalid health state", parseRegionID, getSchedulerShardID)
	}
	buildSchedulerJob := SchedulerJob{
		GetSchedulerJobKind:       SchedulerJobKindMount,
		GetSchedulerRegionID:      parseRegionID,
		GetSchedulerShardID:       getSchedulerShardID,
		GetSchedulerCancelVersion: parseScheduler.getSchedulerCancelVersion(parseRegionID),
	}
	if handleSchedulerQueueErr := parseScheduler.handleSchedulerQueueAppend(buildSchedulerJob); handleSchedulerQueueErr != nil {
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler mount for region %q failed: %w", parseRegionID, handleSchedulerQueueErr)
	}
	return buildSchedulerJob, nil
}

// HandleSchedulerUpdate enqueues an update job on the region's existing shard assignment.
func (parseScheduler *Scheduler) HandleSchedulerUpdate(parseRegionID string) (SchedulerJob, error) {
	if parseScheduler == nil {
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler is nil")
	}
	getSchedulerShardID, hasSchedulerRegionShardID := parseScheduler.getSchedulerShardModel.GetSchedulerRegionAssignedShardID(parseRegionID)
	if !hasSchedulerRegionShardID {
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler update for region %q failed: region is not mounted", parseRegionID)
	}
	if hasSchedulerShardLive := hasSchedulerShardID(parseScheduler.storeSchedulerShardIDs, getSchedulerShardID); !hasSchedulerShardLive {
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler update for region %q failed: assigned shard %q is unavailable", parseRegionID, getSchedulerShardID)
	}
	switch parseScheduler.getSchedulerWorkerHealth(getSchedulerShardID) {
	case SchedulerWorkerHealthReady:
	case SchedulerWorkerHealthDegraded:
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler update for region %q failed: assigned shard %q is degraded", parseRegionID, getSchedulerShardID)
	case SchedulerWorkerHealthDead, SchedulerWorkerHealthRestarting:
		getSchedulerShardRepair, getSchedulerShardRepairErr := parseScheduler.getSchedulerRepairShardID(parseRegionID)
		if getSchedulerShardRepairErr != nil {
			return SchedulerJob{}, fmt.Errorf("runtime2: scheduler update for region %q failed: %w", parseRegionID, getSchedulerShardRepairErr)
		}
		getSchedulerShardID = getSchedulerShardRepair
	default:
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler update for region %q failed: assigned shard %q has invalid health state", parseRegionID, getSchedulerShardID)
	}
	buildSchedulerJob := SchedulerJob{
		GetSchedulerJobKind:       SchedulerJobKindUpdate,
		GetSchedulerRegionID:      parseRegionID,
		GetSchedulerShardID:       getSchedulerShardID,
		GetSchedulerCancelVersion: parseScheduler.getSchedulerCancelVersion(parseRegionID),
	}
	if handleSchedulerQueueErr := parseScheduler.handleSchedulerQueueAppend(buildSchedulerJob); handleSchedulerQueueErr != nil {
		return SchedulerJob{}, fmt.Errorf("runtime2: scheduler update for region %q failed: %w", parseRegionID, handleSchedulerQueueErr)
	}
	return buildSchedulerJob, nil
}

// HandleSchedulerDispose clears one region assignment from scheduler-managed state.
func (parseScheduler *Scheduler) HandleSchedulerDispose(parseRegionID string) bool {
	if parseScheduler == nil {
		return false
	}
	clearSchedulerRegion := parseScheduler.getSchedulerShardModel.ClearSchedulerRegionShardID(parseRegionID)
	if clearSchedulerRegion {
		delete(parseScheduler.storeSchedulerCancelByRegionID, parseRegionID)
	}
	clearSchedulerFallback, hasSchedulerFallback := parseScheduler.storeSchedulerFallbackByRegionID[parseRegionID]
	if hasSchedulerFallback && clearSchedulerFallback {
		delete(parseScheduler.storeSchedulerFallbackByRegionID, parseRegionID)
	}
	clearSchedulerQueuedWork := parseScheduler.clearSchedulerQueueByRegionID(parseRegionID)
	return clearSchedulerRegion || (hasSchedulerFallback && clearSchedulerFallback) || clearSchedulerQueuedWork
}

// HandleSchedulerFallback marks one mounted region as locally-owned fallback state.
func (parseScheduler *Scheduler) HandleSchedulerFallback(parseRegionID string) bool {
	if parseScheduler == nil {
		return false
	}
	if _, hasSchedulerRegion := parseScheduler.getSchedulerShardModel.GetSchedulerRegionAssignedShardID(parseRegionID); !hasSchedulerRegion {
		return false
	}
	parseScheduler.storeSchedulerFallbackByRegionID[parseRegionID] = true
	return true
}

// HandleSchedulerCancel invalidates queued and in-flight region work by advancing the region cancel generation.
func (parseScheduler *Scheduler) HandleSchedulerCancel(parseRegionID string) bool {
	if parseScheduler == nil {
		return false
	}
	if _, hasSchedulerRegion := parseScheduler.getSchedulerShardModel.GetSchedulerRegionAssignedShardID(parseRegionID); !hasSchedulerRegion {
		return false
	}
	parseScheduler.storeSchedulerCancelByRegionID[parseRegionID] = parseScheduler.getSchedulerCancelVersion(parseRegionID) + 1
	parseScheduler.clearSchedulerQueueByRegionID(parseRegionID)
	return true
}

// HasSchedulerJobStale reports whether a queued job is stale under the current region cancel generation.
func (parseScheduler *Scheduler) HasSchedulerJobStale(parseSchedulerJob SchedulerJob) bool {
	if parseScheduler == nil {
		return true
	}
	getSchedulerCancelVersion := parseScheduler.getSchedulerCancelVersion(parseSchedulerJob.GetSchedulerRegionID)
	return parseSchedulerJob.GetSchedulerCancelVersion < getSchedulerCancelVersion
}

// HasSchedulerCommitAllowed reports whether a worker result generation is current enough to commit.
func (parseScheduler *Scheduler) HasSchedulerCommitAllowed(parseRegionID string, parseSchedulerCancelVersion uint64) bool {
	if parseScheduler == nil {
		return false
	}
	if _, hasSchedulerRegion := parseScheduler.getSchedulerShardModel.GetSchedulerRegionAssignedShardID(parseRegionID); !hasSchedulerRegion {
		return false
	}
	if parseScheduler.storeSchedulerFallbackByRegionID[parseRegionID] {
		return false
	}
	return parseSchedulerCancelVersion >= parseScheduler.getSchedulerCancelVersion(parseRegionID)
}

// getSchedulerCancelVersion reads the region cancel generation and defaults unknown regions to zero.
func (parseScheduler *Scheduler) getSchedulerCancelVersion(parseRegionID string) uint64 {
	if parseScheduler == nil {
		return 0
	}
	getSchedulerCancelVersion, hasSchedulerCancelVersion := parseScheduler.storeSchedulerCancelByRegionID[parseRegionID]
	if !hasSchedulerCancelVersion {
		return 0
	}
	return getSchedulerCancelVersion
}

// clearSchedulerQueueByRegionID removes queued jobs for one region and reports whether any jobs were removed.
func (parseScheduler *Scheduler) clearSchedulerQueueByRegionID(parseRegionID string) bool {
	if parseScheduler == nil {
		return false
	}
	getSchedulerQueue := parseScheduler.storeSchedulerQueue[:0]
	hasSchedulerRemovedJob := false
	for _, getSchedulerJob := range parseScheduler.storeSchedulerQueue {
		if getSchedulerJob.GetSchedulerRegionID == parseRegionID {
			hasSchedulerRemovedJob = true
			continue
		}
		getSchedulerQueue = append(getSchedulerQueue, getSchedulerJob)
	}
	parseScheduler.storeSchedulerQueue = getSchedulerQueue
	return hasSchedulerRemovedJob
}

// handleSchedulerQueueAppend enforces queue backpressure before appending new scheduler work.
func (parseScheduler *Scheduler) handleSchedulerQueueAppend(parseSchedulerJob SchedulerJob) error {
	if parseScheduler == nil {
		return fmt.Errorf("scheduler is nil")
	}
	if parseScheduler.storeSchedulerQueueLimit > 0 && len(parseScheduler.storeSchedulerQueue) >= parseScheduler.storeSchedulerQueueLimit {
		return fmt.Errorf("scheduler queue is at limit %d", parseScheduler.storeSchedulerQueueLimit)
	}
	parseScheduler.storeSchedulerQueue = append(parseScheduler.storeSchedulerQueue, parseSchedulerJob)
	return nil
}

// HandleSchedulerReplaceWorker replaces one dead shard with a ready replacement shard and repairs affected regions.
func (parseScheduler *Scheduler) HandleSchedulerReplaceWorker(parseDeadSchedulerShardID SchedulerShardID, parseReplacementSchedulerShardID SchedulerShardID) error {
	if parseScheduler == nil {
		return fmt.Errorf("runtime2: scheduler is nil")
	}
	if strings.TrimSpace(string(parseReplacementSchedulerShardID)) == "" {
		parseScheduler.isSchedulerDegraded = true
		return fmt.Errorf("runtime2: replacement shard ID is required")
	}
	if hasDeadSchedulerShard := hasSchedulerShardID(parseScheduler.storeSchedulerShardIDs, parseDeadSchedulerShardID); !hasDeadSchedulerShard {
		parseScheduler.isSchedulerDegraded = true
		return fmt.Errorf("runtime2: dead shard %q is unknown", parseDeadSchedulerShardID)
	}
	if parseScheduler.getSchedulerWorkerHealth(parseDeadSchedulerShardID) != SchedulerWorkerHealthDead {
		parseScheduler.isSchedulerDegraded = true
		return fmt.Errorf("runtime2: shard %q is not marked dead", parseDeadSchedulerShardID)
	}
	parseScheduler.clearSchedulerShardIDFromList(parseDeadSchedulerShardID)
	delete(parseScheduler.storeSchedulerWorkerHealthByShardID, parseDeadSchedulerShardID)
	delete(parseScheduler.storeSchedulerPongByShardID, parseDeadSchedulerShardID)
	delete(parseScheduler.storeSchedulerMissedPongByShardID, parseDeadSchedulerShardID)
	if hasReplacementSchedulerShard := hasSchedulerShardID(parseScheduler.storeSchedulerShardIDs, parseReplacementSchedulerShardID); !hasReplacementSchedulerShard {
		parseScheduler.storeSchedulerShardIDs = append(parseScheduler.storeSchedulerShardIDs, parseReplacementSchedulerShardID)
	}
	parseScheduler.storeSchedulerWorkerHealthByShardID[parseReplacementSchedulerShardID] = SchedulerWorkerHealthReady
	if _, hasSchedulerPong := parseScheduler.storeSchedulerPongByShardID[parseReplacementSchedulerShardID]; !hasSchedulerPong {
		parseScheduler.storeSchedulerPongByShardID[parseReplacementSchedulerShardID] = 0
	}
	if _, hasSchedulerMissedPong := parseScheduler.storeSchedulerMissedPongByShardID[parseReplacementSchedulerShardID]; !hasSchedulerMissedPong {
		parseScheduler.storeSchedulerMissedPongByShardID[parseReplacementSchedulerShardID] = 0
	}
	for getRegionID, getSchedulerShardID := range parseScheduler.getSchedulerShardModel.GetSchedulerRegionAssignments() {
		if getSchedulerShardID != parseDeadSchedulerShardID {
			continue
		}
		getSchedulerReadyShardIDs := parseScheduler.getSchedulerReadyShardIDs()
		if len(getSchedulerReadyShardIDs) == 0 {
			parseScheduler.HandleSchedulerFallback(getRegionID)
			parseScheduler.isSchedulerDegraded = true
			return fmt.Errorf("runtime2: replacement left no ready shard for region %q", getRegionID)
		}
		if _, getSchedulerRepairErr := parseScheduler.getSchedulerShardModel.GetSchedulerRegionShardID(getRegionID, getSchedulerReadyShardIDs, SchedulerAssignmentPolicyRepair); getSchedulerRepairErr != nil {
			parseScheduler.HandleSchedulerFallback(getRegionID)
			parseScheduler.isSchedulerDegraded = true
			return fmt.Errorf("runtime2: replacement reassignment failed for region %q: %w", getRegionID, getSchedulerRepairErr)
		}
	}
	parseScheduler.updateSchedulerDegradedState()
	if parseScheduler.isSchedulerDegraded {
		log.Printf(
			"runtime2: warn scheduler worker replacement completed with remaining degraded shard health states (dead=%q replacement=%q)",
			parseDeadSchedulerShardID,
			parseReplacementSchedulerShardID,
		)
	}
	return nil
}

// GetSchedulerIsDegraded reports whether scheduler state is currently degraded.
func (parseScheduler *Scheduler) GetSchedulerIsDegraded() bool {
	if parseScheduler == nil {
		return false
	}
	return parseScheduler.isSchedulerDegraded
}

// SetSchedulerWorkerHealth updates one shard health state for scheduler routing decisions.
func (parseScheduler *Scheduler) SetSchedulerWorkerHealth(parseSchedulerShardID SchedulerShardID, parseSchedulerWorkerHealth SchedulerWorkerHealth) error {
	if parseScheduler == nil {
		return fmt.Errorf("runtime2: scheduler is nil")
	}
	if hasSchedulerShardLive := hasSchedulerShardID(parseScheduler.storeSchedulerShardIDs, parseSchedulerShardID); !hasSchedulerShardLive {
		return fmt.Errorf("runtime2: shard %q is unknown", parseSchedulerShardID)
	}
	if parseSchedulerWorkerHealth != SchedulerWorkerHealthReady && parseSchedulerWorkerHealth != SchedulerWorkerHealthDegraded && parseSchedulerWorkerHealth != SchedulerWorkerHealthRestarting && parseSchedulerWorkerHealth != SchedulerWorkerHealthDead {
		return fmt.Errorf("runtime2: worker health state %d is unsupported", parseSchedulerWorkerHealth)
	}
	parseScheduler.storeSchedulerWorkerHealthByShardID[parseSchedulerShardID] = parseSchedulerWorkerHealth
	parseScheduler.updateSchedulerDegradedState()
	return nil
}

// getSchedulerWorkerHealth reads the currently known health state for one shard.
func (parseScheduler *Scheduler) getSchedulerWorkerHealth(parseSchedulerShardID SchedulerShardID) SchedulerWorkerHealth {
	if parseScheduler == nil {
		return schedulerWorkerHealthInvalid
	}
	getSchedulerWorkerHealth, hasSchedulerWorkerHealth := parseScheduler.storeSchedulerWorkerHealthByShardID[parseSchedulerShardID]
	if !hasSchedulerWorkerHealth {
		return schedulerWorkerHealthInvalid
	}
	return getSchedulerWorkerHealth
}

// updateSchedulerDegradedState refreshes degraded scheduler state from per-shard health values.
func (parseScheduler *Scheduler) updateSchedulerDegradedState() {
	if parseScheduler == nil {
		return
	}
	for _, getSchedulerWorkerHealth := range parseScheduler.storeSchedulerWorkerHealthByShardID {
		if getSchedulerWorkerHealth != SchedulerWorkerHealthReady {
			parseScheduler.isSchedulerDegraded = true
			return
		}
	}
	parseScheduler.isSchedulerDegraded = false
}

// clearSchedulerShardIDFromList removes one shard from scheduler routing and reports whether it existed.
func (parseScheduler *Scheduler) clearSchedulerShardIDFromList(parseSchedulerShardID SchedulerShardID) bool {
	if parseScheduler == nil {
		return false
	}
	getSchedulerShardIDs := parseScheduler.storeSchedulerShardIDs[:0]
	hasSchedulerShardRemoved := false
	for _, getSchedulerShardID := range parseScheduler.storeSchedulerShardIDs {
		if getSchedulerShardID == parseSchedulerShardID {
			hasSchedulerShardRemoved = true
			continue
		}
		getSchedulerShardIDs = append(getSchedulerShardIDs, getSchedulerShardID)
	}
	parseScheduler.storeSchedulerShardIDs = getSchedulerShardIDs
	return hasSchedulerShardRemoved
}

// getSchedulerReadyShardIDs returns shards currently marked healthy and eligible for assignment.
func (parseScheduler *Scheduler) getSchedulerReadyShardIDs() []SchedulerShardID {
	if parseScheduler == nil {
		return nil
	}
	getSchedulerReadyShardIDs := make([]SchedulerShardID, 0, len(parseScheduler.storeSchedulerShardIDs))
	for _, getSchedulerShardID := range parseScheduler.storeSchedulerShardIDs {
		if parseScheduler.storeSchedulerWorkerHealthByShardID[getSchedulerShardID] == SchedulerWorkerHealthReady {
			getSchedulerReadyShardIDs = append(getSchedulerReadyShardIDs, getSchedulerShardID)
		}
	}
	return getSchedulerReadyShardIDs
}

// getSchedulerRepairShardID reassigns a region to a ready shard or enters fallback when no repair target exists.
func (parseScheduler *Scheduler) getSchedulerRepairShardID(parseRegionID string) (SchedulerShardID, error) {
	getSchedulerReadyShardIDs := parseScheduler.getSchedulerReadyShardIDs()
	if len(getSchedulerReadyShardIDs) == 0 {
		parseScheduler.HandleSchedulerFallback(parseRegionID)
		return "", fmt.Errorf("no ready shards available; region entered fallback")
	}
	getSchedulerShardID, getSchedulerShardErr := parseScheduler.getSchedulerShardModel.GetSchedulerRegionShardID(parseRegionID, getSchedulerReadyShardIDs, SchedulerAssignmentPolicyRepair)
	if getSchedulerShardErr != nil {
		parseScheduler.HandleSchedulerFallback(parseRegionID)
		return "", fmt.Errorf("region repair reassignment failed: %w", getSchedulerShardErr)
	}
	return getSchedulerShardID, nil
}

// GetSchedulerQueueDepth reports how many jobs are currently enqueued.
func (parseScheduler *Scheduler) GetSchedulerQueueDepth() int {
	if parseScheduler == nil {
		return 0
	}
	return len(parseScheduler.storeSchedulerQueue)
}

// HasSchedulerFallbackOwnership reports whether one region is currently marked fallback-owned in the scheduler.
func (parseScheduler *Scheduler) HasSchedulerFallbackOwnership(parseRegionID string) bool {
	if parseScheduler == nil {
		return false
	}
	return parseScheduler.storeSchedulerFallbackByRegionID[parseRegionID]
}

// ClearSchedulerFallbackOwnership clears one region's scheduler fallback ownership marker.
func (parseScheduler *Scheduler) ClearSchedulerFallbackOwnership(parseRegionID string) bool {
	if parseScheduler == nil {
		return false
	}
	if !parseScheduler.storeSchedulerFallbackByRegionID[parseRegionID] {
		return false
	}
	delete(parseScheduler.storeSchedulerFallbackByRegionID, parseRegionID)
	return true
}
