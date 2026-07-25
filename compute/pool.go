// Package compute is v5's off-thread pool for CPU-bound work (plan item P3.12).
//
// It exists for the jobs that are neither rendering nor domain state: image
// decoding, parsing, compression, search indexing. Left on the render thread
// each one is a long frame; handed to the domain worker they compete with the
// commands that own application state.
//
// §11-Q4 is resolved as "reuse services.wasm; no third binary" — a third
// artifact adds a second Go runtime's memory for no isolation, since compute
// jobs are the app's own Go code either way. The pool scales by worker count.
//
// It reuses internal/services.Dispatcher for WORKER HEALTH and nothing else,
// and that split is deliberate. The dispatcher assigns units stickily, because
// a unit's worker holds state for it and moving it throws that away. A compute
// job holds no state, so stickiness buys nothing and costs balance: hashing job
// ids across workers leaves one worker with three jobs while another idles.
// The pool therefore assigns to the least-loaded ready worker.
package compute

import (
	"errors"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/internal/services"
)

// WorkerID identifies one pool worker.
type WorkerID = services.WorkerID

// WorkerHealth is a worker's pool-visible state.
type WorkerHealth = services.WorkerHealth

// The worker health states, re-exported so callers need not reach into an
// internal package to name them.
const (
	HealthReady      = services.HealthReady
	HealthDegraded   = services.HealthDegraded
	HealthRestarting = services.HealthRestarting
	HealthDead       = services.HealthDead
)

// JobID identifies one job.
type JobID string

// Job is one unit of CPU-bound work.
//
// Payload is opaque; the pool schedules bytes and never interprets them.
type Job struct {
	ID      JobID
	Kind    string
	Payload []byte
}

// Assignment is a job paired with the worker that should run it.
type Assignment struct {
	Job      Job
	WorkerID WorkerID
}

// ErrQueueFull is returned when backpressure refuses a submission.
var ErrQueueFull = errors.New("compute: the job queue is at its limit")

// ErrNoReadyWorker is returned when the pool has no worker able to take work.
var ErrNoReadyWorker = errors.New("compute: no ready worker is available")

// Options configure a pool.
type Options struct {
	// QueueLimit bounds jobs waiting for a worker. Zero uses DefaultQueueLimit.
	//
	// A bound rather than unbounded growth is the backpressure P3.12 requires:
	// an unbounded queue converts a producer that outruns the pool into a memory
	// leak that fails much later and somewhere else.
	QueueLimit int
	// MaxInFlightPerWorker caps concurrent jobs on one worker. Zero means one,
	// which is right for a single-threaded wasm worker.
	MaxInFlightPerWorker int
}

// DefaultQueueLimit is the queue bound applied when Options.QueueLimit is unset.
const DefaultQueueLimit = 256

// Pool schedules CPU-bound jobs across a set of workers.
//
// Not safe for concurrent use; it belongs to the thread that owns the pool.
type Pool struct {
	dispatcher *services.Dispatcher

	queue                []Job
	queueLimit           int
	maxInFlightPerWorker int

	// inFlight holds each running job WITH its payload, not just its worker.
	// Storing only the worker would let a restart requeue the right NUMBER of
	// jobs with their payloads stripped — a job count that looks correct and
	// work that can never run.
	inFlight map[JobID]Assignment
	// submitted guards against a job id being queued twice, which would run it
	// twice and complete once.
	submitted map[JobID]bool
}

// NewPool creates a pool over a set of workers.
func NewPool(parseWorkerIDs []WorkerID, parseOptions Options) (*Pool, error) {
	if len(parseWorkerIDs) == 0 {
		return nil, errors.New("compute: at least one worker is required")
	}
	parseQueueLimit := parseOptions.QueueLimit
	if parseQueueLimit <= 0 {
		parseQueueLimit = DefaultQueueLimit
	}
	parseMaxInFlight := parseOptions.MaxInFlightPerWorker
	if parseMaxInFlight <= 0 {
		parseMaxInFlight = 1
	}

	return &Pool{
		dispatcher:           services.NewDispatcher(parseWorkerIDs),
		queueLimit:           parseQueueLimit,
		maxInFlightPerWorker: parseMaxInFlight,
		inFlight:             make(map[JobID]Assignment),
		submitted:            make(map[JobID]bool),
	}, nil
}

// Submit queues a job, applying backpressure.
func (parsePool *Pool) Submit(parseJob Job) error {
	if parsePool == nil {
		return errors.New("compute: pool is nil")
	}
	if parseJob.ID == "" {
		return errors.New("compute: job id is required")
	}
	if parsePool.submitted[parseJob.ID] {
		// Running one job twice and completing it once leaves a caller waiting
		// forever for a result that already arrived.
		return fmt.Errorf("compute: job %q is already submitted", parseJob.ID)
	}
	if len(parsePool.queue) >= parsePool.queueLimit {
		return fmt.Errorf("%w (limit %d)", ErrQueueFull, parsePool.queueLimit)
	}

	parsePool.queue = append(parsePool.queue, parseJob)
	parsePool.submitted[parseJob.ID] = true
	return nil
}

// Dispatch assigns as many queued jobs as there is capacity for.
//
// Returns the assignments a caller should actually deliver. Jobs beyond the
// available capacity stay queued in submission order; a pool that dispatched
// everything and let the workers queue internally would move the backpressure
// somewhere it cannot be observed.
func (parsePool *Pool) Dispatch() []Assignment {
	if parsePool == nil || len(parsePool.queue) == 0 {
		return nil
	}

	parseLoadByWorker := make(map[WorkerID]int)
	for _, parseAssignment := range parsePool.inFlight {
		parseLoadByWorker[parseAssignment.WorkerID]++
	}

	var parseAssignments []Assignment
	parseRemaining := parsePool.queue[:0]

	for _, parseJob := range parsePool.queue {
		parseWorkerID, hasWorker := parsePool.leastLoadedReadyWorker(parseLoadByWorker)
		if !hasWorker {
			// Everything from here stays queued, in order. Skipping ahead would
			// reorder jobs for no gain, since capacity is what is missing.
			parseRemaining = append(parseRemaining, parseJob)
			continue
		}
		parsePool.inFlight[parseJob.ID] = Assignment{Job: parseJob, WorkerID: parseWorkerID}
		parseLoadByWorker[parseWorkerID]++
		parseAssignments = append(parseAssignments, Assignment{Job: parseJob, WorkerID: parseWorkerID})
	}

	parsePool.queue = parseRemaining
	return parseAssignments
}

// leastLoadedReadyWorker picks the ready worker with the fewest in-flight jobs.
//
// Ties break on the dispatcher's canonical (sorted) worker order, so the choice
// is deterministic and a test can assert it.
func (parsePool *Pool) leastLoadedReadyWorker(parseLoadByWorker map[WorkerID]int) (WorkerID, bool) {
	var parseBest WorkerID
	parseBestLoad := -1

	for _, parseWorkerID := range parsePool.dispatcher.ReadyWorkers() {
		parseLoad := parseLoadByWorker[parseWorkerID]
		if parseLoad >= parsePool.maxInFlightPerWorker {
			continue
		}
		if parseBestLoad < 0 || parseLoad < parseBestLoad {
			parseBest, parseBestLoad = parseWorkerID, parseLoad
		}
	}
	return parseBest, parseBestLoad >= 0
}

// Complete records that a job finished, freeing its worker's capacity.
func (parsePool *Pool) Complete(parseJobID JobID) bool {
	if parsePool == nil {
		return false
	}
	if _, hasJob := parsePool.inFlight[parseJobID]; !hasJob {
		return false
	}
	delete(parsePool.inFlight, parseJobID)
	delete(parsePool.submitted, parseJobID)
	return true
}

// RestartWorker replaces a worker and returns its in-flight jobs to the queue.
//
// This is P3.12's criterion: a worker restarts WITHOUT LOSING QUEUED JOBS. Two
// distinct sets have to survive it — the jobs merely waiting, which were never
// tied to the dead worker, and the jobs already handed to it, which would
// otherwise vanish with it. The second set is the one an implementation loses by
// accident.
//
// Requeued jobs go to the FRONT, ahead of jobs submitted later. They have
// already waited once, and putting them behind newer work would starve exactly
// the jobs a restart already delayed.
//
// Semantics: AT-LEAST-ONCE. A job interrupted by a worker death is dispatched
// again, because the pool cannot know how far the dead worker got. Dropping it
// instead would be worse — silent data loss rather than repeated work — so
// compute jobs must be idempotent or have a duplicate result discarded. Pinned
// by TestWorkerRestartLosesNoJobs, which asserts an interrupted job is
// dispatched exactly twice and an untouched one exactly once.
func (parsePool *Pool) RestartWorker(parseDeadWorkerID WorkerID, parseReplacementWorkerID WorkerID) (int, error) {
	if parsePool == nil {
		return 0, errors.New("compute: pool is nil")
	}
	if parseErr := parsePool.dispatcher.SetWorkerHealth(parseDeadWorkerID, HealthDead); parseErr != nil {
		return 0, fmt.Errorf("compute: marking worker %q dead: %w", parseDeadWorkerID, parseErr)
	}

	var parseRequeued []Job
	for parseJobID, parseAssignment := range parsePool.inFlight {
		if parseAssignment.WorkerID != parseDeadWorkerID {
			continue
		}
		parseRequeued = append(parseRequeued, parseAssignment.Job)
		delete(parsePool.inFlight, parseJobID)
	}

	// Deterministic order so a restart is reproducible; map iteration is not.
	sortJobsByID(parseRequeued)
	parsePool.queue = append(parseRequeued, parsePool.queue...)

	if _, parseErr := parsePool.dispatcher.ReplaceWorker(parseDeadWorkerID, parseReplacementWorkerID); parseErr != nil {
		return len(parseRequeued), fmt.Errorf("compute: replacing worker %q: %w", parseDeadWorkerID, parseErr)
	}
	return len(parseRequeued), nil
}

// sortJobsByID orders jobs by id, insertion sort being ample for a single
// worker's in-flight set.
func sortJobsByID(parseJobs []Job) {
	for parseIndex := 1; parseIndex < len(parseJobs); parseIndex++ {
		parseJob := parseJobs[parseIndex]
		parsePosition := parseIndex - 1
		for parsePosition >= 0 && parseJobs[parsePosition].ID > parseJob.ID {
			parseJobs[parsePosition+1] = parseJobs[parsePosition]
			parsePosition--
		}
		parseJobs[parsePosition+1] = parseJob
	}
}

// SetWorkerHealth updates a worker's health.
func (parsePool *Pool) SetWorkerHealth(parseWorkerID WorkerID, parseHealth WorkerHealth) error {
	if parsePool == nil {
		return errors.New("compute: pool is nil")
	}
	return parsePool.dispatcher.SetWorkerHealth(parseWorkerID, parseHealth)
}

// RecordHeartbeat records a worker heartbeat.
func (parsePool *Pool) RecordHeartbeat(parseWorkerID WorkerID, parseSequence uint64) error {
	if parsePool == nil {
		return errors.New("compute: pool is nil")
	}
	return parsePool.dispatcher.RecordHeartbeat(parseWorkerID, parseSequence)
}

// RecordHeartbeatTimeout records a missed heartbeat window.
func (parsePool *Pool) RecordHeartbeatTimeout(parseWorkerID WorkerID) (WorkerHealth, error) {
	if parsePool == nil {
		return HealthDead, errors.New("compute: pool is nil")
	}
	return parsePool.dispatcher.RecordHeartbeatTimeout(parseWorkerID)
}

// QueueDepth reports how many jobs are waiting.
func (parsePool *Pool) QueueDepth() int {
	if parsePool == nil {
		return 0
	}
	return len(parsePool.queue)
}

// InFlight reports how many jobs are running.
func (parsePool *Pool) InFlight() int {
	if parsePool == nil {
		return 0
	}
	return len(parsePool.inFlight)
}

// ReadyWorkers lists workers able to take jobs.
func (parsePool *Pool) ReadyWorkers() []WorkerID {
	if parsePool == nil {
		return nil
	}
	return parsePool.dispatcher.ReadyWorkers()
}

// Degraded reports whether any worker is not ready.
func (parsePool *Pool) Degraded() bool {
	if parsePool == nil {
		return false
	}
	return parsePool.dispatcher.Degraded()
}
