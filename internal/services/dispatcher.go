package services

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Work dispatch — extracted from runtime2's Scheduler (P3.3).
//
// Underneath the region vocabulary, the scheduler is a general work dispatcher:
// route units of work onto a pool of workers, keep the routing stable, track
// worker health from heartbeats, coalesce redundant queued work, invalidate work
// that has been superseded, apply backpressure, and repair the pool when a
// worker dies. P3.12's compute pool needs precisely that list.
//
// Two behavioral differences from the source, both deliberate and both tested:
//
//  1. ReplaceWorker repairs EVERY unit assigned to the dead worker. The source
//     returns on the first unit it cannot repair, leaving the rest assigned to a
//     worker it has already deleted. See TestReplaceWorkerRepairsEveryUnit.
//  2. Assignment is documented as deterministic-and-sticky rather than tied to a
//     particular hash. That is the property callers can depend on; the specific
//     worker chosen is not.

// WorkerID identifies one worker in the pool.
type WorkerID string

// WorkerHealth is a worker's dispatch-visible state.
type WorkerHealth uint8

const (
	// healthUnknown is the zero value and never a valid state.
	healthUnknown WorkerHealth = iota
	// HealthReady means the worker can accept work.
	HealthReady
	// HealthDegraded means one heartbeat window was missed. Work is not routed to
	// it, but it is not yet replaced — a degraded worker usually recovers.
	HealthDegraded
	// HealthRestarting means the worker is being brought back deliberately.
	HealthRestarting
	// HealthDead means the worker is gone and must be replaced.
	HealthDead
)

func (parseHealth WorkerHealth) String() string {
	switch parseHealth {
	case HealthReady:
		return "ready"
	case HealthDegraded:
		return "degraded"
	case HealthRestarting:
		return "restarting"
	case HealthDead:
		return "dead"
	default:
		return "unknown"
	}
}

// IsValidWorkerHealth reports whether a health value is one a caller may set.
func IsValidWorkerHealth(parseHealth WorkerHealth) bool {
	switch parseHealth {
	case HealthReady, HealthDegraded, HealthRestarting, HealthDead:
		return true
	default:
		return false
	}
}

// JobKind distinguishes the first job for a unit from subsequent ones.
//
// The distinction is load-bearing for coalescing: a start cannot be coalesced
// away because the worker has no prior state for the unit, while updates can
// because a later one supersedes an earlier one entirely.
type JobKind string

const (
	// JobStart is the first job for a unit, establishing its assignment.
	JobStart JobKind = "start"
	// JobUpdate is subsequent work on an established assignment.
	JobUpdate JobKind = "update"
)

// Job is one queued unit of work.
type Job struct {
	Kind     JobKind
	UnitID   string
	WorkerID WorkerID
	// CancelGeneration stamps the job with the unit's generation at enqueue time.
	// A job whose generation is behind the unit's current one has been superseded
	// and must not be committed, even if a worker already finished it.
	CancelGeneration uint64
}

// ErrQueueFull is returned when backpressure rejects new work.
var ErrQueueFull = errors.New("services: dispatcher queue is at its limit")

// ErrNoReadyWorker is returned when no worker can take a unit.
var ErrNoReadyWorker = errors.New("services: no ready worker is available")

type queueIndexEntry struct {
	queueIndex       int
	cancelGeneration uint64
}

// Dispatcher routes units of work across a worker pool.
//
// Not safe for concurrent use; it belongs to one coordinating thread.
type Dispatcher struct {
	workerIDs           []WorkerID
	healthByWorkerID    map[WorkerID]WorkerHealth
	heartbeatByWorkerID map[WorkerID]uint64
	missesByWorkerID    map[WorkerID]uint64

	assignmentByUnitID map[string]WorkerID
	generationByUnitID map[string]uint64
	fallbackByUnitID   map[string]bool

	queue             []Job
	updateIndexByUnit map[string]queueIndexEntry
	queueLimit        int

	degraded bool
}

const defaultQueueCapacityPerWorker = 64
const minimumQueueCapacity = 16

// NewDispatcher creates a dispatcher over a worker pool with no queue limit.
func NewDispatcher(parseWorkerIDs []WorkerID) *Dispatcher {
	return NewDispatcherWithQueueLimit(parseWorkerIDs, 0)
}

// NewDispatcherWithQueueLimit creates a dispatcher with queue backpressure.
//
// A non-positive limit means unbounded, which is the source's default and is
// appropriate only when the producer is already rate-limited by something else.
func NewDispatcherWithQueueLimit(parseWorkerIDs []WorkerID, parseQueueLimit int) *Dispatcher {
	parseCanonicalWorkerIDs := canonicalWorkerIDs(parseWorkerIDs)

	parseCapacity := len(parseCanonicalWorkerIDs) * defaultQueueCapacityPerWorker
	if parseQueueLimit > 0 {
		parseCapacity = parseQueueLimit
	}
	if parseCapacity < minimumQueueCapacity {
		parseCapacity = minimumQueueCapacity
	}

	parseDispatcher := &Dispatcher{
		workerIDs:           parseCanonicalWorkerIDs,
		healthByWorkerID:    make(map[WorkerID]WorkerHealth, len(parseCanonicalWorkerIDs)),
		heartbeatByWorkerID: make(map[WorkerID]uint64, len(parseCanonicalWorkerIDs)),
		missesByWorkerID:    make(map[WorkerID]uint64, len(parseCanonicalWorkerIDs)),
		assignmentByUnitID:  make(map[string]WorkerID),
		generationByUnitID:  make(map[string]uint64),
		fallbackByUnitID:    make(map[string]bool),
		queue:               make([]Job, 0, parseCapacity),
		updateIndexByUnit:   make(map[string]queueIndexEntry),
		queueLimit:          parseQueueLimit,
	}
	for _, parseWorkerID := range parseCanonicalWorkerIDs {
		parseDispatcher.healthByWorkerID[parseWorkerID] = HealthReady
	}
	return parseDispatcher
}

// canonicalWorkerIDs sorts and de-duplicates a worker list.
//
// Sorting is what makes assignment reproducible: the same units land on the same
// workers regardless of the order the pool was declared in, so a caller that
// rebuilds its pool does not silently reshuffle every assignment.
func canonicalWorkerIDs(parseWorkerIDs []WorkerID) []WorkerID {
	parseSeen := make(map[WorkerID]bool, len(parseWorkerIDs))
	parseCanonical := make([]WorkerID, 0, len(parseWorkerIDs))
	for _, parseWorkerID := range parseWorkerIDs {
		if strings.TrimSpace(string(parseWorkerID)) == "" || parseSeen[parseWorkerID] {
			continue
		}
		parseSeen[parseWorkerID] = true
		parseCanonical = append(parseCanonical, parseWorkerID)
	}
	slices.Sort(parseCanonical)
	return parseCanonical
}

// ---------------------------------------------------------------- assignment

// Start establishes a unit's assignment and enqueues its first job.
//
// Re-starting an already-assigned unit keeps its worker when that worker is
// still ready. Stability matters more than balance here: moving a unit discards
// whatever state its worker had built for it.
func (parseDispatcher *Dispatcher) Start(parseUnitID string) (Job, error) {
	if parseDispatcher == nil {
		return Job{}, errors.New("services: dispatcher is nil")
	}
	if strings.TrimSpace(parseUnitID) == "" {
		return Job{}, errors.New("services: dispatcher unit id is required")
	}

	parseWorkerID, parseWorkerErr := parseDispatcher.resolveWorkerForUnit(parseUnitID)
	if parseWorkerErr != nil {
		return Job{}, fmt.Errorf("services: starting unit %q: %w", parseUnitID, parseWorkerErr)
	}

	parseDispatcher.assignmentByUnitID[parseUnitID] = parseWorkerID
	parseJob := Job{
		Kind:             JobStart,
		UnitID:           parseUnitID,
		WorkerID:         parseWorkerID,
		CancelGeneration: parseDispatcher.generationByUnitID[parseUnitID],
	}
	if parseEnqueueErr := parseDispatcher.enqueue(parseJob); parseEnqueueErr != nil {
		return Job{}, fmt.Errorf("services: starting unit %q: %w", parseUnitID, parseEnqueueErr)
	}
	return parseJob, nil
}

// Update enqueues work on an established assignment, coalescing when possible.
//
// Coalescing is the reason this returns the EXISTING job when one is already
// queued for the same unit and generation: a burst of updates collapses to one
// dispatch carrying the newest state, rather than a queue of superseded work the
// worker would grind through in order.
func (parseDispatcher *Dispatcher) Update(parseUnitID string) (Job, error) {
	if parseDispatcher == nil {
		return Job{}, errors.New("services: dispatcher is nil")
	}
	parseAssignedWorkerID, hasAssignment := parseDispatcher.assignmentByUnitID[parseUnitID]
	if !hasAssignment {
		return Job{}, fmt.Errorf("services: unit %q is not started", parseUnitID)
	}

	parseGeneration := parseDispatcher.generationByUnitID[parseUnitID]
	parseQueuedJob, parseQueueIndex, hasQueuedJob := parseDispatcher.queuedUpdate(parseUnitID, parseGeneration)
	if hasQueuedJob && parseQueuedJob.WorkerID == parseAssignedWorkerID {
		return parseQueuedJob, nil
	}

	parseWorkerID, parseWorkerErr := parseDispatcher.resolveWorkerForUnit(parseUnitID)
	if parseWorkerErr != nil {
		return Job{}, fmt.Errorf("services: updating unit %q: %w", parseUnitID, parseWorkerErr)
	}
	parseDispatcher.assignmentByUnitID[parseUnitID] = parseWorkerID

	parseJob := Job{
		Kind:             JobUpdate,
		UnitID:           parseUnitID,
		WorkerID:         parseWorkerID,
		CancelGeneration: parseGeneration,
	}
	if hasQueuedJob {
		// Replace in place: the queued job was for a worker that is no longer the
		// unit's assignment, and dispatching both would apply the same update
		// twice on two workers.
		parseDispatcher.queue[parseQueueIndex] = parseJob
		return parseJob, nil
	}
	if parseEnqueueErr := parseDispatcher.enqueue(parseJob); parseEnqueueErr != nil {
		return Job{}, fmt.Errorf("services: updating unit %q: %w", parseUnitID, parseEnqueueErr)
	}
	return parseJob, nil
}

// resolveWorkerForUnit picks the worker for a unit, repairing an unusable one.
func (parseDispatcher *Dispatcher) resolveWorkerForUnit(parseUnitID string) (WorkerID, error) {
	parseAssignedWorkerID, hasAssignment := parseDispatcher.assignmentByUnitID[parseUnitID]
	if hasAssignment {
		switch parseDispatcher.healthByWorkerID[parseAssignedWorkerID] {
		case HealthReady:
			return parseAssignedWorkerID, nil
		case HealthDegraded:
			// A degraded worker is expected to recover, and reassigning would throw
			// away the state it holds for this unit. Refusing the work is the
			// cheaper failure: the caller retries.
			return "", fmt.Errorf("assigned worker %q is degraded", parseAssignedWorkerID)
		}
		// Dead, restarting, or unknown: the assignment cannot be honored, so fall
		// through to repair.
	}
	return parseDispatcher.assignReadyWorker(parseUnitID)
}

// assignReadyWorker picks a ready worker deterministically, or enters fallback.
//
// Determinism is the contract; the particular worker is not. Callers may depend
// on the same unit reaching the same worker across runs given the same pool, and
// on the choice being stable while the pool is stable.
func (parseDispatcher *Dispatcher) assignReadyWorker(parseUnitID string) (WorkerID, error) {
	parseReadyWorkerIDs := parseDispatcher.ReadyWorkers()
	if len(parseReadyWorkerIDs) == 0 {
		// No worker can take the unit, so it must run locally rather than wait
		// indefinitely for a pool that may never recover.
		parseDispatcher.fallbackByUnitID[parseUnitID] = true
		parseDispatcher.degraded = true
		return "", ErrNoReadyWorker
	}
	parseIndex := hashUnitID(parseUnitID) % uint64(len(parseReadyWorkerIDs))
	return parseReadyWorkerIDs[parseIndex], nil
}

// hashUnitID is FNV-1a, chosen for being stable across processes and platforms.
// Go's built-in map hash is randomized per process and would reshuffle every
// assignment on restart.
func hashUnitID(parseUnitID string) uint64 {
	const parseOffsetBasis = uint64(14695981039346656037)
	const parsePrime = uint64(1099511628211)

	parseHash := parseOffsetBasis
	for parseIndex := 0; parseIndex < len(parseUnitID); parseIndex++ {
		parseHash ^= uint64(parseUnitID[parseIndex])
		parseHash *= parsePrime
	}
	return parseHash
}

// Assignment reports a unit's current worker, if it has one.
func (parseDispatcher *Dispatcher) Assignment(parseUnitID string) (WorkerID, bool) {
	if parseDispatcher == nil {
		return "", false
	}
	parseWorkerID, hasAssignment := parseDispatcher.assignmentByUnitID[parseUnitID]
	return parseWorkerID, hasAssignment
}

// --------------------------------------------------------------- generations

// Cancel supersedes all queued and in-flight work for a unit.
//
// Advancing the generation rather than hunting down in-flight work is what makes
// this correct: a worker already running the old job cannot be recalled, so its
// result is rejected at commit time instead. Queued work is dropped eagerly
// because it is still reachable.
func (parseDispatcher *Dispatcher) Cancel(parseUnitID string) bool {
	if parseDispatcher == nil {
		return false
	}
	if _, hasAssignment := parseDispatcher.assignmentByUnitID[parseUnitID]; !hasAssignment {
		return false
	}
	parseDispatcher.generationByUnitID[parseUnitID] = parseDispatcher.generationByUnitID[parseUnitID] + 1
	parseDispatcher.dropQueuedWork(parseUnitID)
	return true
}

// IsStale reports whether a job has been superseded by a cancel.
func (parseDispatcher *Dispatcher) IsStale(parseJob Job) bool {
	if parseDispatcher == nil {
		return true
	}
	return parseJob.CancelGeneration < parseDispatcher.generationByUnitID[parseJob.UnitID]
}

// CommitAllowed reports whether a completed job's result may still be applied.
//
// Three ways to lose the right to commit, all checked here: the unit was
// disposed, it fell back to local ownership, or it was cancelled and this result
// predates the cancel.
func (parseDispatcher *Dispatcher) CommitAllowed(parseUnitID string, parseGeneration uint64) bool {
	if parseDispatcher == nil {
		return false
	}
	if _, hasAssignment := parseDispatcher.assignmentByUnitID[parseUnitID]; !hasAssignment {
		return false
	}
	if parseDispatcher.fallbackByUnitID[parseUnitID] {
		return false
	}
	return parseGeneration >= parseDispatcher.generationByUnitID[parseUnitID]
}

// Fallback marks a started unit as locally owned, which blocks its commits.
func (parseDispatcher *Dispatcher) Fallback(parseUnitID string) bool {
	if parseDispatcher == nil {
		return false
	}
	if _, hasAssignment := parseDispatcher.assignmentByUnitID[parseUnitID]; !hasAssignment {
		return false
	}
	parseDispatcher.fallbackByUnitID[parseUnitID] = true
	return true
}

// InFallback reports whether a unit is locally owned.
func (parseDispatcher *Dispatcher) InFallback(parseUnitID string) bool {
	if parseDispatcher == nil {
		return false
	}
	return parseDispatcher.fallbackByUnitID[parseUnitID]
}

// Dispose removes a unit entirely and reports whether anything was removed.
func (parseDispatcher *Dispatcher) Dispose(parseUnitID string) bool {
	if parseDispatcher == nil {
		return false
	}
	_, hasAssignment := parseDispatcher.assignmentByUnitID[parseUnitID]
	hasFallback := parseDispatcher.fallbackByUnitID[parseUnitID]

	delete(parseDispatcher.assignmentByUnitID, parseUnitID)
	delete(parseDispatcher.generationByUnitID, parseUnitID)
	delete(parseDispatcher.fallbackByUnitID, parseUnitID)
	hasDroppedWork := parseDispatcher.dropQueuedWork(parseUnitID)

	return hasAssignment || hasFallback || hasDroppedWork
}

// --------------------------------------------------------------------- queue

func (parseDispatcher *Dispatcher) enqueue(parseJob Job) error {
	if parseDispatcher.queueLimit > 0 && len(parseDispatcher.queue) >= parseDispatcher.queueLimit {
		return fmt.Errorf("%w (limit %d)", ErrQueueFull, parseDispatcher.queueLimit)
	}
	parseDispatcher.queue = append(parseDispatcher.queue, parseJob)
	if parseJob.Kind == JobUpdate {
		parseDispatcher.updateIndexByUnit[parseJob.UnitID] = queueIndexEntry{
			queueIndex:       len(parseDispatcher.queue) - 1,
			cancelGeneration: parseJob.CancelGeneration,
		}
	}
	return nil
}

// queuedUpdate finds a coalescable queued update, validating the index against
// the queue rather than trusting it. A stale index would otherwise coalesce a
// new update into an unrelated job.
func (parseDispatcher *Dispatcher) queuedUpdate(parseUnitID string, parseGeneration uint64) (Job, int, bool) {
	parseEntry, hasEntry := parseDispatcher.updateIndexByUnit[parseUnitID]
	if !hasEntry || parseEntry.cancelGeneration != parseGeneration {
		return Job{}, -1, false
	}
	if parseEntry.queueIndex < 0 || parseEntry.queueIndex >= len(parseDispatcher.queue) {
		delete(parseDispatcher.updateIndexByUnit, parseUnitID)
		return Job{}, -1, false
	}
	parseJob := parseDispatcher.queue[parseEntry.queueIndex]
	if parseJob.Kind != JobUpdate || parseJob.UnitID != parseUnitID || parseJob.CancelGeneration != parseGeneration {
		delete(parseDispatcher.updateIndexByUnit, parseUnitID)
		return Job{}, -1, false
	}
	return parseJob, parseEntry.queueIndex, true
}

func (parseDispatcher *Dispatcher) dropQueuedWork(parseUnitID string) bool {
	parseKept := parseDispatcher.queue[:0]
	hasDropped := false
	for _, parseJob := range parseDispatcher.queue {
		if parseJob.UnitID == parseUnitID {
			hasDropped = true
			continue
		}
		parseKept = append(parseKept, parseJob)
	}
	// In-place compaction leaves dropped jobs — and their payloads — reachable
	// in the slots past the new length.
	clear(parseDispatcher.queue[len(parseKept):])
	parseDispatcher.queue = parseKept
	if hasDropped {
		// Compaction moved every job after the removed ones, so every cached index
		// is now wrong. Rebuilding is O(queue) but only on a drop.
		parseDispatcher.rebuildUpdateIndex()
	}
	return hasDropped
}

func (parseDispatcher *Dispatcher) rebuildUpdateIndex() {
	clear(parseDispatcher.updateIndexByUnit)
	for parseIndex, parseJob := range parseDispatcher.queue {
		if parseJob.Kind != JobUpdate {
			continue
		}
		parseDispatcher.updateIndexByUnit[parseJob.UnitID] = queueIndexEntry{
			queueIndex:       parseIndex,
			cancelGeneration: parseJob.CancelGeneration,
		}
	}
}

// QueueDepth reports how many jobs are queued.
func (parseDispatcher *Dispatcher) QueueDepth() int {
	if parseDispatcher == nil {
		return 0
	}
	return len(parseDispatcher.queue)
}

// DrainQueue removes and returns every queued job, dropping ones a cancel has
// superseded. Draining is how a caller actually dispatches: the queue is a
// staging buffer, not a permanent record.
func (parseDispatcher *Dispatcher) DrainQueue() []Job {
	if parseDispatcher == nil {
		return nil
	}
	var parseDrained []Job
	for _, parseJob := range parseDispatcher.queue {
		if parseDispatcher.IsStale(parseJob) {
			continue
		}
		parseDrained = append(parseDrained, parseJob)
	}
	clear(parseDispatcher.queue)
	parseDispatcher.queue = parseDispatcher.queue[:0]
	clear(parseDispatcher.updateIndexByUnit)
	return parseDrained
}

// -------------------------------------------------------------------- health

// RecordHeartbeat records a heartbeat and restores the worker to ready.
//
// Sequence numbers must not go backwards: an out-of-order heartbeat from before
// a failure would otherwise mark a dead worker ready again.
func (parseDispatcher *Dispatcher) RecordHeartbeat(parseWorkerID WorkerID, parseSequence uint64) error {
	if parseDispatcher == nil {
		return errors.New("services: dispatcher is nil")
	}
	if parseSequence == 0 {
		return errors.New("services: heartbeat sequence is required")
	}
	if _, hasWorker := parseDispatcher.healthByWorkerID[parseWorkerID]; !hasWorker {
		return fmt.Errorf("services: worker %q is unknown", parseWorkerID)
	}
	if parseSequence < parseDispatcher.heartbeatByWorkerID[parseWorkerID] {
		return fmt.Errorf("services: stale heartbeat %d for worker %q (latest %d)",
			parseSequence, parseWorkerID, parseDispatcher.heartbeatByWorkerID[parseWorkerID])
	}

	parseDispatcher.heartbeatByWorkerID[parseWorkerID] = parseSequence
	parseDispatcher.missesByWorkerID[parseWorkerID] = 0
	parseDispatcher.healthByWorkerID[parseWorkerID] = HealthReady
	parseDispatcher.refreshDegraded()
	return nil
}

// RecordHeartbeatTimeout records a missed heartbeat window.
//
// One miss degrades, two kills. The intermediate state exists because a single
// missed window is usually a long task or a scheduling hiccup, and replacing a
// worker over it would discard live state for no reason.
func (parseDispatcher *Dispatcher) RecordHeartbeatTimeout(parseWorkerID WorkerID) (WorkerHealth, error) {
	if parseDispatcher == nil {
		return healthUnknown, errors.New("services: dispatcher is nil")
	}
	if _, hasWorker := parseDispatcher.healthByWorkerID[parseWorkerID]; !hasWorker {
		return healthUnknown, fmt.Errorf("services: worker %q is unknown", parseWorkerID)
	}

	parseDispatcher.missesByWorkerID[parseWorkerID]++
	parseHealth := HealthDead
	if parseDispatcher.missesByWorkerID[parseWorkerID] == 1 {
		parseHealth = HealthDegraded
	}
	parseDispatcher.healthByWorkerID[parseWorkerID] = parseHealth
	parseDispatcher.refreshDegraded()
	return parseHealth, nil
}

// SetWorkerHealth sets a worker's health directly, for state a heartbeat cannot
// express — a deliberate restart, or a death observed out of band.
func (parseDispatcher *Dispatcher) SetWorkerHealth(parseWorkerID WorkerID, parseHealth WorkerHealth) error {
	if parseDispatcher == nil {
		return errors.New("services: dispatcher is nil")
	}
	if _, hasWorker := parseDispatcher.healthByWorkerID[parseWorkerID]; !hasWorker {
		return fmt.Errorf("services: worker %q is unknown", parseWorkerID)
	}
	if !IsValidWorkerHealth(parseHealth) {
		return fmt.Errorf("services: worker health %d is not a valid state", parseHealth)
	}
	parseDispatcher.healthByWorkerID[parseWorkerID] = parseHealth
	parseDispatcher.refreshDegraded()
	return nil
}

// WorkerHealthOf reports one worker's health.
func (parseDispatcher *Dispatcher) WorkerHealthOf(parseWorkerID WorkerID) WorkerHealth {
	if parseDispatcher == nil {
		return healthUnknown
	}
	return parseDispatcher.healthByWorkerID[parseWorkerID]
}

// ReadyWorkers lists workers that can take work, in canonical order.
func (parseDispatcher *Dispatcher) ReadyWorkers() []WorkerID {
	if parseDispatcher == nil {
		return nil
	}
	parseReady := make([]WorkerID, 0, len(parseDispatcher.workerIDs))
	for _, parseWorkerID := range parseDispatcher.workerIDs {
		if parseDispatcher.healthByWorkerID[parseWorkerID] == HealthReady {
			parseReady = append(parseReady, parseWorkerID)
		}
	}
	return parseReady
}

// Degraded reports whether any worker is not ready, or a repair failed.
func (parseDispatcher *Dispatcher) Degraded() bool {
	if parseDispatcher == nil {
		return false
	}
	return parseDispatcher.degraded
}

func (parseDispatcher *Dispatcher) refreshDegraded() {
	for _, parseHealth := range parseDispatcher.healthByWorkerID {
		if parseHealth != HealthReady {
			parseDispatcher.degraded = true
			return
		}
	}
	parseDispatcher.degraded = false
}

// --------------------------------------------------------------- replacement

// ReplaceOutcome reports what happened to each unit during a replacement.
type ReplaceOutcome struct {
	// Reassigned lists units moved onto a healthy worker.
	Reassigned []string
	// FellBack lists units that had no healthy worker to move to.
	FellBack []string
}

// ReplaceWorker swaps a dead worker for a replacement and repairs its units.
//
// EVERY affected unit is repaired, including after one of them fails. The source
// returns on the first failure, which leaves the remaining units assigned to a
// worker it has already deleted from the pool — they then dispatch into nothing
// and never recover, and the symptom appears far from the cause. Collecting the
// failures and continuing costs nothing and makes the outcome describable.
func (parseDispatcher *Dispatcher) ReplaceWorker(parseDeadWorkerID WorkerID, parseReplacementWorkerID WorkerID) (ReplaceOutcome, error) {
	if parseDispatcher == nil {
		return ReplaceOutcome{}, errors.New("services: dispatcher is nil")
	}
	if strings.TrimSpace(string(parseReplacementWorkerID)) == "" {
		parseDispatcher.degraded = true
		return ReplaceOutcome{}, errors.New("services: replacement worker id is required")
	}
	if _, hasDeadWorker := parseDispatcher.healthByWorkerID[parseDeadWorkerID]; !hasDeadWorker {
		parseDispatcher.degraded = true
		return ReplaceOutcome{}, fmt.Errorf("services: worker %q is unknown", parseDeadWorkerID)
	}
	if parseDispatcher.healthByWorkerID[parseDeadWorkerID] != HealthDead {
		// Replacing a live worker would strand whatever it is running. A caller
		// that means to retire a healthy worker marks it dead first.
		parseDispatcher.degraded = true
		return ReplaceOutcome{}, fmt.Errorf("services: worker %q is not marked dead", parseDeadWorkerID)
	}

	parseDispatcher.removeWorker(parseDeadWorkerID)
	parseDispatcher.addWorker(parseReplacementWorkerID)

	// Snapshot the affected units before reassigning: repairing mutates the
	// assignment map, and ranging a map while writing it is unpredictable.
	var parseAffectedUnitIDs []string
	for parseUnitID, parseWorkerID := range parseDispatcher.assignmentByUnitID {
		if parseWorkerID == parseDeadWorkerID {
			parseAffectedUnitIDs = append(parseAffectedUnitIDs, parseUnitID)
		}
	}
	sort.Strings(parseAffectedUnitIDs)

	parseOutcome := ReplaceOutcome{}
	for _, parseUnitID := range parseAffectedUnitIDs {
		parseWorkerID, parseAssignErr := parseDispatcher.assignReadyWorker(parseUnitID)
		if parseAssignErr != nil {
			delete(parseDispatcher.assignmentByUnitID, parseUnitID)
			parseDispatcher.dropQueuedWork(parseUnitID)
			parseOutcome.FellBack = append(parseOutcome.FellBack, parseUnitID)
			continue
		}
		parseDispatcher.assignmentByUnitID[parseUnitID] = parseWorkerID
		parseOutcome.Reassigned = append(parseOutcome.Reassigned, parseUnitID)
	}

	// Queued work still names the dead worker, so retarget it rather than drop
	// it — the work is not stale, only misaddressed.
	for parseIndex := range parseDispatcher.queue {
		if parseDispatcher.queue[parseIndex].WorkerID != parseDeadWorkerID {
			continue
		}
		if parseWorkerID, hasAssignment := parseDispatcher.assignmentByUnitID[parseDispatcher.queue[parseIndex].UnitID]; hasAssignment {
			parseDispatcher.queue[parseIndex].WorkerID = parseWorkerID
		}
	}

	parseDispatcher.refreshDegraded()
	if len(parseOutcome.FellBack) > 0 {
		parseDispatcher.degraded = true
		return parseOutcome, fmt.Errorf("services: replacing worker %q left %d unit(s) without a worker: %s",
			parseDeadWorkerID, len(parseOutcome.FellBack), strings.Join(parseOutcome.FellBack, ", "))
	}
	return parseOutcome, nil
}

func (parseDispatcher *Dispatcher) removeWorker(parseWorkerID WorkerID) {
	parseKept := parseDispatcher.workerIDs[:0]
	for _, parseExisting := range parseDispatcher.workerIDs {
		if parseExisting == parseWorkerID {
			continue
		}
		parseKept = append(parseKept, parseExisting)
	}
	parseDispatcher.workerIDs = parseKept
	delete(parseDispatcher.healthByWorkerID, parseWorkerID)
	delete(parseDispatcher.heartbeatByWorkerID, parseWorkerID)
	delete(parseDispatcher.missesByWorkerID, parseWorkerID)
}

// addWorker admits a worker to the pool, ready only if it is genuinely new.
//
// An ID already in the pool keeps whatever health it has. Forcing every added
// worker to ready would let a replacement that is itself already known dead be
// silently resurrected — the dispatcher would then route work to it, and the
// units would fail again for a reason nothing recorded.
func (parseDispatcher *Dispatcher) addWorker(parseWorkerID WorkerID) {
	if _, hasWorker := parseDispatcher.healthByWorkerID[parseWorkerID]; hasWorker {
		return
	}
	parseDispatcher.workerIDs = append(parseDispatcher.workerIDs, parseWorkerID)
	slices.Sort(parseDispatcher.workerIDs)
	parseDispatcher.healthByWorkerID[parseWorkerID] = HealthReady
	if _, hasHeartbeat := parseDispatcher.heartbeatByWorkerID[parseWorkerID]; !hasHeartbeat {
		parseDispatcher.heartbeatByWorkerID[parseWorkerID] = 0
	}
	parseDispatcher.missesByWorkerID[parseWorkerID] = 0
}
