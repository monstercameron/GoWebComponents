package runtime

// Workload budgets — plan item P4.2, threat T9.
//
// T9 is "silent degradation cliffs": the effect queue past 1024 falls back to a
// full-tree scan, and coalescing past 4096 stops counting. Both cliffs already
// had defined behaviour and a log line. What they lacked is the thing P4.2
// actually asks for — a SIGNAL. A warning in a log stream is not something a
// devtools panel, a test, or a CI gate can read, so in practice the cliffs stayed
// silent to everything except a human reading console output at the right moment.
//
// §11-Q12 asked what the cliffs should DO — reject, drop-oldest, degrade, or
// block. The answer, recorded here because it is a property of each cliff rather
// than a global policy:
//
//	effect queue overflow  -> DEGRADE. Correctness is preserved by scanning the
//	                          whole tree; only cost rises, from O(effects) to
//	                          O(tree). Rejecting effects would break the app,
//	                          and blocking would stall the commit.
//	update coalescing      -> COALESCE. Nothing is lost: an update is a request
//	                          to re-render, and re-render requests are idempotent,
//	                          so collapsing N of them into one produces the same
//	                          frame. "Drop" would be the wrong word and, until
//	                          this commit, was the word the counter used.
//
// Neither cliff rejects or blocks, and neither should: both carry work whose
// only property is "do this again", and the correct response to too much of
// that is to do it once.
//
// R6 — diagnostics carry an allocation budget. Everything here reads counters
// the runtime already keeps and returns a flat struct. Nothing walks the tree,
// formats a path, or allocates per event.

// BudgetBehavior names what a cliff does when it is reached.
type BudgetBehavior uint8

const (
	// BudgetDegrade preserves correctness at a higher cost.
	BudgetDegrade BudgetBehavior = iota
	// BudgetCoalesce merges excess work into work already scheduled, losing
	// nothing because the work is idempotent.
	BudgetCoalesce
	// BudgetReject refuses new work outright. Not currently used by any cliff;
	// defined so a future cliff that does reject can say so in the same
	// vocabulary rather than inventing one.
	BudgetReject
	// BudgetBlock stalls the producer until capacity frees. Also unused, and
	// deliberately so: blocking the render thread is the failure this whole
	// version exists to prevent.
	BudgetBlock
)

func (parseBehavior BudgetBehavior) String() string {
	switch parseBehavior {
	case BudgetDegrade:
		return "degrade"
	case BudgetCoalesce:
		return "coalesce"
	case BudgetReject:
		return "reject"
	case BudgetBlock:
		return "block"
	default:
		return "unknown"
	}
}

// LosesWork reports whether reaching this cliff can discard work.
//
// The question a reader of a devtools panel actually has. Degrade and coalesce
// do not; reject does. Answering it as a method keeps every consumer from
// re-deriving it and getting it wrong for the coalesce case, which reads like a
// loss and is not.
func (parseBehavior BudgetBehavior) LosesWork() bool {
	return parseBehavior == BudgetReject
}

// BudgetSignal is one cliff's current state.
type BudgetSignal struct {
	// Name identifies the cliff. Stable across versions; devtools and tests key
	// on it.
	Name string
	// Limit is the configured capacity, zero meaning unbounded.
	Limit int
	// Observed is the current occupancy, for a "how close are we" reading rather
	// than only a "did it blow" one. The distinction matters: a workload sitting
	// at 95% is a warning, and one that never trips is not a bug.
	Observed int
	// Behavior is what happens at the limit.
	Behavior BudgetBehavior
	// Triggered counts how many times this cliff has been reached in the
	// runtime's lifetime.
	Triggered int
	// Active reports whether the cliff is currently in its degraded state.
	Active bool
}

// AtCapacity reports whether the cliff is at or past its limit right now.
func (parseSignal BudgetSignal) AtCapacity() bool {
	return parseSignal.Limit > 0 && parseSignal.Observed >= parseSignal.Limit
}

// Headroom reports remaining capacity, or -1 when unbounded.
func (parseSignal BudgetSignal) Headroom() int {
	if parseSignal.Limit <= 0 {
		return -1
	}
	if parseSignal.Observed >= parseSignal.Limit {
		return 0
	}
	return parseSignal.Limit - parseSignal.Observed
}

// The stable cliff names.
const (
	// BudgetPendingEffects is the effect queue that degrades to a full-tree scan.
	BudgetPendingEffects = "pending-effects"
	// BudgetQueuedUpdates is the update queue that coalesces past its limit.
	BudgetQueuedUpdates = "queued-updates"
)

// WorkloadBudgets is every cliff's state at one moment.
//
// A fixed-size array rather than a slice or map: R6 says diagnostics carry an
// allocation budget, and a devtools panel polls this every frame. Returning it
// by value allocates nothing.
type WorkloadBudgets struct {
	PendingEffects BudgetSignal
	QueuedUpdates  BudgetSignal
}

// Degraded reports whether any cliff is currently active.
func (parseBudgets WorkloadBudgets) Degraded() bool {
	return parseBudgets.PendingEffects.Active || parseBudgets.QueuedUpdates.Active
}

// Signals returns the cliffs in a stable order, for iteration.
//
// Returns an array, not a slice, so ranging over it in a per-frame devtools poll
// does not allocate.
func (parseBudgets WorkloadBudgets) Signals() [2]BudgetSignal {
	return [2]BudgetSignal{parseBudgets.PendingEffects, parseBudgets.QueuedUpdates}
}

// Budgets reports the current workload budget state.
//
// Safe on a nil runtime, because a devtools panel polls on a timer and can
// easily outlive the runtime it was watching.
func (parseRt *Runtime) Budgets() WorkloadBudgets {
	if parseRt == nil {
		return WorkloadBudgets{
			PendingEffects: BudgetSignal{Name: BudgetPendingEffects, Behavior: BudgetDegrade},
			QueuedUpdates:  BudgetSignal{Name: BudgetQueuedUpdates, Behavior: BudgetCoalesce},
		}
	}

	parseLimits := parseRt.limits.withDefaults()
	return WorkloadBudgets{
		PendingEffects: BudgetSignal{
			Name:      BudgetPendingEffects,
			Limit:     parseLimits.MaxPendingEffectFibers,
			Observed:  len(parseRt.pendingEffectFibers),
			Behavior:  BudgetDegrade,
			Triggered: parseRt.pendingEffectOverflowCount,
			Active:    parseRt.pendingEffectOverflow,
		},
		QueuedUpdates: BudgetSignal{
			Name:      BudgetQueuedUpdates,
			Limit:     parseRt.schedulerState.maxQueuedUpdates,
			Observed:  parseRt.schedulerState.coalescedUpdates,
			Behavior:  BudgetCoalesce,
			Triggered: parseRt.schedulerState.coalescedAtLimit,
			// Coalescing is not a state the runtime stays in; it is what one
			// update did. A cliff that reports itself permanently "active" after
			// a single burst would train readers to ignore it.
			Active: parseRt.schedulerState.coalescedUpdates >= parseRt.schedulerState.maxQueuedUpdates &&
				parseRt.schedulerState.maxQueuedUpdates > 0,
		},
	}
}

// GlobalBudgets reports the global runtime's workload budgets, for callers that
// have no runtime handle — which is every devtools surface today.
func GlobalBudgets() WorkloadBudgets {
	return GetGlobalRuntime().Budgets()
}
