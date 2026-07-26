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

// inboxOverflowFactor is how far past the configured limit the queue may grow
// before the pressure is reported. Reaching it degrades BATCHING only — the
// drain is brought forward, not moved onto the producer.
const inboxOverflowFactor = 4

// inboxHardOverflowFactor is the last-resort bound, past which the queue is
// drained on whatever goroutine is posting.
//
// The two tiers exist because the soft bound previously did this, and it is the
// wrong price to pay at the first sign of pressure. Draining on the producer
// runs application state mutations on a goroutine the runtime does not control,
// at a moment it did not choose — which is the exact hazard the inbox was built
// to remove. Trading it away under load means the guarantee is absent precisely
// when it matters most.
//
// Reaching THIS bound means a producer has queued ~16x the soft bound without
// the event loop getting a single turn. Such a producer is not yielding at all,
// so no scheduled drain can ever run, and the choice is between draining here
// and growing until the tab dies. Draining is better, and it is reported as the
// distinct condition it is rather than as ordinary backpressure.
const inboxHardOverflowFactor = inboxOverflowFactor * 16

// asyncInbox holds work posted from outside the frame loop.
//
// The mutex is uncontended in wasm (one thread) and correct on native, where
// SSR and tests genuinely run goroutines in parallel.
type asyncInbox struct {
	mu        sync.Mutex
	entries   []func()
	scheduled bool
	// overflowed records that the queue grew past its soft bound, so the
	// condition can be surfaced instead of inferred.
	overflowed bool
	// hardOverflowed records that the queue was drained on a producer goroutine
	// because nothing else could ever have drained it. Kept separate from
	// overflowed: one says batching degraded, the other says the isolation
	// guarantee was suspended, and reporting them as the same event would hide
	// the second behind the first.
	hardOverflowed bool
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
	isPastHardBound := parseLimit > 0 && len(parseRt.inbox.entries) > parseLimit*inboxHardOverflowFactor
	if isOverBound {
		parseRt.inbox.overflowed = true
	}
	if isPastHardBound {
		parseRt.inbox.hardOverflowed = true
	}
	parseRt.inbox.mu.Unlock()

	if isPastHardBound && parseRt.scheduler != nil {
		// Last resort. Nothing scheduled can have run, or the queue could not
		// have reached this size, so the choice is between draining on this
		// goroutine and growing until the tab dies. DrainAsyncInbox reports the
		// suspension.
		parseRt.DrainAsyncInbox()
		return
	}
	if isAlreadyScheduled {
		// Past the soft bound the drain is already pending; the queue keeps
		// growing until the loop takes its turn. Deliberate: bringing the drain
		// onto THIS goroutine would trade the isolation guarantee for a
		// smaller queue, and the queue is the cheaper thing to give up.
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
	isHardOverflowed := parseRt.inbox.hardOverflowed
	parseRt.inbox.overflowed = false
	parseRt.inbox.hardOverflowed = false
	if len(parseBatch) > 0 {
		parseRt.inbox.drainCount++
	}
	parseRt.inbox.mu.Unlock()

	if len(parseBatch) == 0 {
		return
	}

	if isHardOverflowed {
		ReportDiagnostic("runtime", DiagnosticWarning,
			"async inbox was drained on a producer goroutine because the event loop never got a turn; "+
				"frame isolation was suspended for this batch. A producer posting this fast without yielding "+
				"would stall the page regardless — the fix is at the call site, not here.")
	} else if isOverflowed {
		ReportDiagnostic("runtime", DiagnosticWarning,
			"async inbox exceeded its bound; batching degraded, no work was dropped, and the drain stayed on the frame loop")
	}

	// Marked as frame-loop work: entries call the same setters an event handler
	// does, and without this each one would post itself straight back into the
	// queue and be deferred another frame, forever.
	parseRt.enterFrameLoop()
	defer parseRt.exitFrameLoop()
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

// Frame-loop marking (v5 P2.1, second half).
//
// The inbox is only half a solution while state setters still apply wherever
// they are called. A gRPC callback, a worker reply, or any goroutine reaches the
// same setter a click handler does, and the setter cannot tell them apart — so
// it mutates hook state at an arbitrary moment relative to the in-flight tree.
//
// The distinction the setter needs is not "which goroutine" but "is the runtime
// already running". Goroutine identity is the wrong question in wasm, where
// callbacks arrive on goroutines the runtime never created and the render loop
// has no stable identity of its own. Whether a frame-loop region is on the stack
// is exact, costs two field reads, and needs no platform knowledge.
//
// Regions that count as inside the loop:
//
//	workLoopDepth    render and commit (commitRoot runs from the work loop)
//	frameLoopDepth   event dispatch, and the inbox drain itself
//
// Event dispatch has to be marked or every click pays an extra task hop: the
// setter would post, the drain would apply it a task later, and only then would
// the render be scheduled. The drain has to be marked for the same reason in
// reverse — work applied during a drain is already on the loop, and posting it
// again would defer it another frame, indefinitely.

// enterFrameLoop marks the start of a frame-loop region and records which
// goroutine owns it.
//
// The owner is captured on the OUTERMOST entry only, and only when async
// ingress is on, so a default build never pays for the identity lookup.
func (parseRt *Runtime) enterFrameLoop() {
	if parseRt == nil {
		return
	}
	if parseRt.frameLoopDepth == 0 && parseRt.asyncIngress {
		parseRt.frameLoopOwner = frameLoopGoroutineID()
	}
	parseRt.frameLoopDepth++
}

// exitFrameLoop marks the end of a frame-loop region.
func (parseRt *Runtime) exitFrameLoop() {
	if parseRt == nil || parseRt.frameLoopDepth == 0 {
		return
	}
	parseRt.frameLoopDepth--
	if parseRt.frameLoopDepth == 0 {
		parseRt.frameLoopOwner = 0
	}
}

// insideFrameLoop reports whether the CALLER is running on the frame loop, and
// is therefore free to mutate state directly.
//
// Depth alone answers a different question — "is a frame-loop region on some
// stack" — and gets the important case wrong. A goroutine spawned by an event
// handler runs while that handler is still on the stack, so a depth-only check
// called it on-loop and let its write reach the tree directly. That goroutine is
// exactly what the inbox is for, so the mechanism missed its own motivating
// case in silence.
//
// The identity lookup is skipped entirely when async ingress is off: nothing
// consults this in that configuration except to preserve prior behaviour, and
// paying 1.15 µs per state write for a disabled feature would be indefensible.
func (parseRt *Runtime) insideFrameLoop() bool {
	if parseRt == nil || parseRt.frameLoopDepth == 0 {
		return false
	}
	if !parseRt.asyncIngress {
		return true
	}
	// A zero owner means the identity could not be read. Treated as "not the
	// owner" so an unreadable stack routes the write through the inbox — slower,
	// never wrong — rather than admitting it on an unverified claim.
	return parseRt.frameLoopOwner != 0 && frameLoopGoroutineID() == parseRt.frameLoopOwner
}

// shouldPostAsyncStateUpdate reports whether a state write must be queued rather
// than applied where it was called.
//
// The scheduler check is not incidental. With no scheduler — native tests, SSR —
// PostAsync drains inline, so posting would still apply the write on the calling
// goroutine, but through runInboxEntry, which CONTAINS panics. Silently changing
// panic semantics for every native caller is not worth the nothing it buys where
// there is no frame loop to be isolated from.
func (parseRt *Runtime) shouldPostAsyncStateUpdate() bool {
	return parseRt != nil && parseRt.asyncIngress && parseRt.scheduler != nil &&
		!parseRt.insideFrameLoop()
}

// AsyncIngressEnabled reports whether off-loop state writes are routed through
// the inbox, so an app can check what it is running under.
func (parseRt *Runtime) AsyncIngressEnabled() bool {
	return parseRt != nil && parseRt.asyncIngress
}
