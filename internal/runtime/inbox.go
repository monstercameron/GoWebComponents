package runtime

import "sync"

// Async inbox (v5 P2.1) — the keystone of Phase 2.
//
// Problem it solves. Three separate threats in the v5 plan share one root
// cause: async work touching render state at an arbitrary moment.
//
//   - T3 background work is promoted rather than deferred
//   - T4 async writes race the in-flight tree, producing either a torn render
//     (the fiber had not been visited yet) or a stranded update (it had)
//   - T5 no batching across tasks, so every goroutine or worker message costs a
//     full pass and commit
//
// Mechanism. Async producers no longer call Schedule* directly. They Post to a
// per-runtime queue that is drained at one defined point per frame. Everything
// posted between two drains is applied inside a single synchronous block, so
// the existing updateScheduled gate coalesces it into ONE pass — N messages in
// a frame produce one render, not N.
//
// The isolation property falls out for free: nothing outside the frame loop
// mutates hook state, so the in-flight tree is isolated by construction rather
// than by a mutex, a snapshot, or any hot-path cost. That is why this supersedes
// the "arm a follow-up pass" patch considered earlier — it removes the window
// instead of reacting to it.
//
// This also replaces EnqueueUI/ProcessUIQueue, which were the same idea left
// unfinished: nothing outside tests ever drained that channel, and its
// queue-full path ran the callback synchronously on the calling goroutine —
// precisely the hand-off it existed to prevent.

// inboxOverflowFactor bounds how far the queue may grow past the configured
// limit before an early drain is forced. Dropping work is never an option;
// draining early trades batching for boundedness.
const inboxOverflowFactor = 4

// asyncInbox holds work posted from outside the frame loop.
//
// The mutex is uncontended in wasm (one thread) and correct on native, where
// SSR and tests genuinely run goroutines in parallel.
type asyncInbox struct {
	mu        sync.Mutex
	entries   []func()
	scheduled bool
	// overflowed records that a drain was forced early because the queue grew
	// past its bound, so the condition can be surfaced instead of inferred.
	overflowed bool
	// drainCount and postCount make the batching property observable; a
	// benchmark or diagnostic can show N posts collapsing into one drain.
	drainCount int
	postCount  int
}

// PostAsync queues work to be applied at the next frame boundary.
//
// Safe to call from any goroutine. The function runs later, on the frame loop,
// and should perform the state mutation it wants rendered — calling the normal
// Schedule* paths from inside it is correct and is what produces the coalescing.
func (parseRt *Runtime) PostAsync(parseWork func()) {
	if parseRt == nil || parseWork == nil {
		return
	}

	parseRt.inbox.mu.Lock()
	parseRt.inbox.entries = append(parseRt.inbox.entries, parseWork)
	parseRt.inbox.postCount++
	parseLimit := parseRt.inboxLimit()
	isOverBound := parseLimit > 0 && len(parseRt.inbox.entries) > parseLimit*inboxOverflowFactor
	isAlreadyScheduled := parseRt.inbox.scheduled
	if !isAlreadyScheduled {
		parseRt.inbox.scheduled = true
	}
	if isOverBound {
		parseRt.inbox.overflowed = true
	}
	parseRt.inbox.mu.Unlock()

	if isOverBound {
		// Bounded, not lossy: drain now rather than let the queue grow without
		// limit. Batching degrades; correctness does not (R4 — the condition is
		// reported by DrainAsyncInbox).
		parseRt.DrainAsyncInbox()
		return
	}
	if isAlreadyScheduled {
		return
	}
	if parseRt.scheduler == nil {
		// Native and SSR: no frame loop to wait for. Running inline keeps the
		// semantics every non-browser caller already depends on.
		parseRt.DrainAsyncInbox()
		return
	}
	parseRt.scheduler.SetTimeout(parseRt.DrainAsyncInbox, 0)
}

// inboxLimit reports the configured queue bound, reusing the runtime's existing
// queued-update limit so inbox pressure and scheduler pressure are governed by
// one number rather than two.
func (parseRt *Runtime) inboxLimit() int {
	return parseRt.limits.withDefaults().MaxQueuedUpdates
}

// DrainAsyncInbox applies every queued entry in one synchronous block.
//
// Running the whole batch before yielding is the entire point: each entry marks
// its own state, and the updateScheduled gate collapses all of them into a
// single render pass.
func (parseRt *Runtime) DrainAsyncInbox() {
	if parseRt == nil {
		return
	}

	parseRt.inbox.mu.Lock()
	parseBatch := parseRt.inbox.entries
	parseRt.inbox.entries = nil
	parseRt.inbox.scheduled = false
	isOverflowed := parseRt.inbox.overflowed
	parseRt.inbox.overflowed = false
	if len(parseBatch) > 0 {
		parseRt.inbox.drainCount++
	}
	parseRt.inbox.mu.Unlock()

	if len(parseBatch) == 0 {
		return
	}

	if isOverflowed {
		ReportDiagnostic("runtime", DiagnosticWarning,
			"async inbox exceeded its bound and drained early; batching degraded but no work was dropped")
	}

	for _, parseWork := range parseBatch {
		parseRt.runInboxEntry(parseWork)
	}
}

// runInboxEntry isolates one entry so a panicking producer cannot strand the
// rest of the batch. An inbox that loses unrelated work on one bad message
// would be worse than the direct-scheduling it replaces.
func (parseRt *Runtime) runInboxEntry(parseWork func()) {
	// containAsyncPanic reports and CONTAINS. The deferred-panic finalizer used
	// elsewhere re-raises after reporting, which would take down the rest of
	// the batch along with the app.
	defer containAsyncPanic("runtime", "async inbox entry")
	parseWork()
}

// AsyncInboxDepth reports how many entries are waiting. Used by memory hygiene
// and by tests asserting the batching property.
func (parseRt *Runtime) AsyncInboxDepth() int {
	if parseRt == nil {
		return 0
	}
	parseRt.inbox.mu.Lock()
	defer parseRt.inbox.mu.Unlock()
	return len(parseRt.inbox.entries)
}

// AsyncInboxStats reports cumulative post and drain counts, so the "N posts
// become one pass" claim is observable rather than asserted.
func (parseRt *Runtime) AsyncInboxStats() (parsePosts int, parseDrains int) {
	if parseRt == nil {
		return 0, 0
	}
	parseRt.inbox.mu.Lock()
	defer parseRt.inbox.mu.Unlock()
	return parseRt.inbox.postCount, parseRt.inbox.drainCount
}
