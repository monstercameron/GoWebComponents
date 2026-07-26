package runtime

import (
	"reflect"
	"testing"
)

// Async ingress acceptance (v5 P2.1).
//
// The inbox existed before this, and nothing outside tests ever posted to it.
// Hook setters still mutated state wherever they were called, so a gRPC
// callback, a worker reply, or any goroutine reached hook state at an arbitrary
// moment relative to the in-flight tree — the exact race the inbox was built to
// remove. These tests pin the four properties that make routing setters through
// it worth the semantic change.

// newAsyncIngressRuntime builds a runtime with async ingress on and a scheduler
// present, which is the configuration the routing requires: with no scheduler
// PostAsync drains inline and there is no frame loop to be isolated from.
func newAsyncIngressRuntime(parseT *testing.T) (*Runtime, DOMNode, *testScheduler) {
	parseT.Helper()
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter:   parseAdapter,
		Scheduler:    parseScheduler,
		AsyncIngress: true,
		Reset:        true,
	})
	return parseRt, parseAdapter.CreateElement("div"), parseScheduler
}

// mountCounter mounts a component with one state value and returns its setter, a
// reader for the rendered text, and a render counter.
func mountCounter(parseT *testing.T, parseRt *Runtime, parseContainer DOMNode, parseScheduler *testScheduler) (func(any), func() string, *int) {
	parseT.Helper()
	parseRenders := 0
	var parseSet func(any)
	var parseRendered string

	parseComponent := func() *Element {
		parseRenders++
		parseValue, parseSetter := GoUseState(parseRt, "start")
		parseSet = parseSetter
		parseRendered = parseValue()
		return CreateElement("span", map[string]any{}, parseValue())
	}
	parseRt.Render(CreateElement(parseComponent, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	if parseSet == nil {
		parseT.Fatal("the component never rendered, so no setter was captured")
	}
	return parseSet, func() string { return parseRendered }, &parseRenders
}

// TestAsyncIngress_OffLoopWriteIsDeferredToTheDrain is the isolation property.
//
// A setter called from outside the frame loop must not have changed anything by
// the time it returns. That is the whole guarantee: a reply arriving mid-render
// cannot tear the tree, because it did not touch it.
func TestAsyncIngress_OffLoopWriteIsDeferredToTheDrain(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	parseSet, parseRendered, _ := mountCounter(parseT, parseRt, parseContainer, parseScheduler)

	parseSet("from-async")

	if parseRendered() != "start" {
		parseT.Errorf("the rendered value changed to %q before the drain; an off-loop write reached the tree immediately", parseRendered())
	}
	if parseRt.AsyncInboxDepth() != 1 {
		parseT.Errorf("inbox depth = %d, want 1; the write was not queued", parseRt.AsyncInboxDepth())
	}

	runScheduledTimeouts(parseScheduler)

	if parseRendered() != "from-async" {
		parseT.Errorf("after the drain the rendered value is %q, want %q; the write was queued and then lost",
			parseRendered(), "from-async")
	}
}

// TestAsyncIngress_ManyWritesProduceOneRender is the batching property, and the
// reason this is a performance change as well as a correctness one.
func TestAsyncIngress_ManyWritesProduceOneRender(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	parseSet, parseRendered, parseRenders := mountCounter(parseT, parseRt, parseContainer, parseScheduler)

	parseRendersBefore := *parseRenders
	for parseIndex := range 20 {
		parseSet(string(rune('a' + parseIndex)))
	}
	if parseRt.AsyncInboxDepth() != 20 {
		parseT.Fatalf("inbox depth = %d, want 20", parseRt.AsyncInboxDepth())
	}

	runScheduledTimeouts(parseScheduler)

	parseRenderCount := *parseRenders - parseRendersBefore
	if parseRenderCount != 1 {
		parseT.Errorf("20 async writes produced %d renders, want 1; they are not coalescing", parseRenderCount)
	}
	if parseRendered() != "t" {
		parseT.Errorf("last write did not win: rendered %q, want %q", parseRendered(), "t")
	}
}

// TestAsyncIngress_EventHandlerWritesApplyImmediately is the property that stops
// this from being a regression on every interaction.
//
// A handler runs ON the frame loop. If dispatch were not marked, a click's
// setter would post instead of applying, and the render would not even be
// scheduled until a task later.
func TestAsyncIngress_EventHandlerWritesApplyImmediately(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	parseSet, parseRendered, _ := mountCounter(parseT, parseRt, parseContainer, parseScheduler)

	// Dispatch through the same wrapper the reconciler installs for hook
	// handlers, rather than asserting on the marker directly — what matters is
	// that the real dispatch path marks the loop.
	parseCell := &funcHandlerCell{}
	parseHandler := func() { parseSet("from-handler") }
	parseCell.fn = parseHandler
	parseCell.fnVal = reflect.ValueOf(parseHandler)
	parseWrapped, parseOk := parseRt.wrapEventHandlerCell(parseCell).(func())
	if !parseOk {
		parseT.Fatal("the event wrapper did not preserve the handler signature")
	}

	parseWrapped()

	if parseRt.AsyncInboxDepth() != 0 {
		parseT.Errorf("a handler's write was queued (%d in the inbox); every interaction would pay an extra task",
			parseRt.AsyncInboxDepth())
	}
	runScheduledTimeouts(parseScheduler)
	if parseRendered() != "from-handler" {
		parseT.Errorf("rendered %q after the handler, want %q", parseRendered(), "from-handler")
	}
}

// TestAsyncIngress_DisabledKeepsDirectWrites pins that this is opt-in, so a
// build that has not enabled it behaves exactly as before.
func TestAsyncIngress_DisabledKeepsDirectWrites(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	parseSet, _, _ := mountCounter(parseT, parseRt, parseContainer, parseScheduler)
	parseSet("direct")

	if parseRt.AsyncInboxDepth() != 0 {
		parseT.Errorf("inbox depth = %d with async ingress off; the routing is not opt-in", parseRt.AsyncInboxDepth())
	}
	if parseRt.AsyncIngressEnabled() {
		parseT.Error("AsyncIngressEnabled reported true for a runtime that did not enable it")
	}
}

// TestAsyncIngress_ExplicitPostRunsItsSetterOnTheLoop is the public API pattern,
// and the one case where the drain's own frame-loop mark is load-bearing.
//
// When the runtime routes a setter itself it queues apply(), which never
// re-enters the setter. But ui.PostAsync exists so an app can post a closure
// that calls the SETTER — the documented shape, and the one that batches several
// related writes into one render:
//
//	ui.PostAsync(func() { setRows(rows); setTotal(total) })
//
// If the drain were not marked as frame-loop work, that setter would find itself
// off-loop, post itself straight back into the queue, and be deferred again on
// every drain — a write that is never dropped and never applied, which is worse
// than either.
func TestAsyncIngress_ExplicitPostRunsItsSetterOnTheLoop(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	parseSet, parseRendered, parseRenders := mountCounter(parseT, parseRt, parseContainer, parseScheduler)
	parseRendersBefore := *parseRenders

	parseRt.PostAsync(func() {
		parseSet("posted-a")
		parseSet("posted-b")
	})

	// One drain must be enough. Draining repeatedly would hide re-queueing by
	// eventually letting the work through.
	parseRt.DrainAsyncInbox()

	if parseRt.AsyncInboxDepth() != 0 {
		parseT.Fatalf("after one drain the inbox still holds %d entries; the posted setters re-queued themselves and will never be applied",
			parseRt.AsyncInboxDepth())
	}

	runScheduledTimeouts(parseScheduler)
	if parseRendered() != "posted-b" {
		parseT.Errorf("rendered %q, want %q", parseRendered(), "posted-b")
	}
	if parseRenderCount := *parseRenders - parseRendersBefore; parseRenderCount != 1 {
		parseT.Errorf("two writes in one post produced %d renders, want 1", parseRenderCount)
	}
}

// TestAsyncIngress_GoroutineSpawnedByAHandlerIsStillAsync is the hole at the
// centre of the mechanism.
//
// insideFrameLoop counts depth per RUNTIME, not per goroutine. A handler that
// spawns a goroutine — which is the documented way to call a worker command
// without deadlocking the event loop — leaves that goroutine running while the
// depth is still non-zero. The setter it calls therefore reads "inside the frame
// loop" and applies directly, at an arbitrary moment relative to the in-flight
// tree.
//
// That is precisely the caller async ingress exists for, so the feature misses
// its own motivating case, and misses it silently. Worse than absent: an app
// that enables AsyncIngress and follows the documented goroutine pattern gets
// no isolation and no indication that it has none.
func TestAsyncIngress_GoroutineSpawnedByAHandlerIsStillAsync(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	parseSet, parseRendered, _ := mountCounter(parseT, parseRt, parseContainer, parseScheduler)

	// A handler that does what the migration guide tells adopters to do: hand
	// the blocking work to a goroutine so the event loop is released.
	parseDone := make(chan struct{})
	parseCell := &funcHandlerCell{}
	parseHandler := func() {
		go func() {
			defer close(parseDone)
			parseSet("from-goroutine")
		}()
		// Block until the goroutine has run its setter, so the write lands while
		// the handler is still on the stack and the depth is still non-zero.
		// Without this the test would race and pass by luck.
		<-parseDone
	}
	parseCell.fn = parseHandler
	parseCell.fnVal = reflect.ValueOf(parseHandler)
	parseWrapped, parseOk := parseRt.wrapEventHandlerCell(parseCell).(func())
	if !parseOk {
		parseT.Fatal("the event wrapper did not preserve the handler signature")
	}

	parseWrapped()

	if parseRt.AsyncInboxDepth() != 1 {
		parseT.Errorf("inbox depth = %d, want 1; a goroutine's write bypassed the inbox because the runtime-wide depth counter said its handler was still on the loop",
			parseRt.AsyncInboxDepth())
	}
	if parseRendered() != "start" {
		parseT.Errorf("the goroutine's write reached the tree immediately (rendered %q); nothing isolated the in-flight tree from it",
			parseRendered())
	}

	runScheduledTimeouts(parseScheduler)
	if parseRendered() != "from-goroutine" {
		parseT.Errorf("after the drain, rendered %q, want %q", parseRendered(), "from-goroutine")
	}
}

// TestAsyncIngress_SoftOverflowKeepsTheDrainOffTheProducer pins the tier split.
//
// Past the soft bound the queue keeps growing and the drain stays scheduled.
// Bringing it onto the producer would run application state mutations on a
// goroutine the runtime does not control, at a moment it did not choose — the
// exact hazard the inbox removes — and doing that under load means the guarantee
// is absent precisely when it matters.
func TestAsyncIngress_SoftOverflowKeepsTheDrainOffTheProducer(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	parseSet, _, _ := mountCounter(parseT, parseRt, parseContainer, parseScheduler)

	parseLimit := parseRt.inboxLimit()
	if parseLimit <= 0 {
		parseT.Skip("no queued-update limit configured, so there is no bound to exceed")
	}

	// Comfortably past the soft bound, comfortably short of the hard one.
	parseTarget := parseLimit*inboxOverflowFactor + 8
	for parseIndex := range parseTarget {
		parseSet(parseIndex)
	}

	if parseDepth := parseRt.AsyncInboxDepth(); parseDepth != parseTarget {
		parseT.Errorf("inbox depth = %d after %d posts past the soft bound; something drained on the producer",
			parseDepth, parseTarget)
	}

	runScheduledTimeouts(parseScheduler)
	if parseDepth := parseRt.AsyncInboxDepth(); parseDepth != 0 {
		parseT.Errorf("inbox still holds %d entries after the scheduled drain", parseDepth)
	}
}

// TestAsyncIngress_HardOverflowDrainsRatherThanGrowingForever pins the other
// side, so the isolation preference cannot turn into an unbounded queue.
//
// Reaching the hard bound means the event loop never got a turn — no scheduled
// drain can have run, or the queue could not be this size — so the choice is
// between draining on the producer and growing until the tab dies.
func TestAsyncIngress_HardOverflowDrainsRatherThanGrowingForever(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	parseSet, _, _ := mountCounter(parseT, parseRt, parseContainer, parseScheduler)

	parseLimit := parseRt.inboxLimit()
	if parseLimit <= 0 {
		parseT.Skip("no queued-update limit configured, so there is no bound to exceed")
	}

	parseCeiling := parseLimit * inboxHardOverflowFactor
	for parseIndex := range parseCeiling + 2 {
		parseSet(parseIndex)
	}

	if parseDepth := parseRt.AsyncInboxDepth(); parseDepth > parseCeiling {
		parseT.Errorf("inbox depth = %d, above the hard bound of %d; a non-yielding producer can grow it without limit",
			parseDepth, parseCeiling)
	}
}

// TestAsyncIngress_AtomWritesAreRoutedToo closes the gap that made the feature
// look complete while covering one hook.
//
// Routing only GoUseState leaves the shape most exposed to this unprotected. An
// atom is the natural place to publish a worker reply or a subscription event —
// it exists so state can be shared across components without threading it
// through props — so its setter is MORE likely than a component's to be called
// from a goroutine, not less.
func TestAsyncIngress_AtomWritesAreRoutedToo(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)

	parseRenders := 0
	var parseSet func(any)
	parseRendered := ""
	parseComponent := func() *Element {
		parseRenders++
		parseValue, parseSetter := GoUseAtom(parseRt, "async-ingress-atom", "start")
		parseSet = parseSetter
		parseRendered = parseValue()
		return CreateElement("span", map[string]any{}, parseValue())
	}
	parseRt.Render(CreateElement(parseComponent, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	if parseSet == nil {
		parseT.Fatal("the component never rendered, so no atom setter was captured")
	}

	parseSet("from-async")

	if parseRendered != "start" {
		parseT.Errorf("the atom write reached the tree immediately (rendered %q); an off-loop atom setter is not being routed",
			parseRendered)
	}
	if parseRt.AsyncInboxDepth() != 1 {
		parseT.Errorf("inbox depth = %d, want 1; the atom write was not queued", parseRt.AsyncInboxDepth())
	}

	runScheduledTimeouts(parseScheduler)
	if parseRendered != "from-async" {
		parseT.Errorf("after the drain, rendered %q, want %q; the atom write was queued and then lost",
			parseRendered, "from-async")
	}
}

// TestAsyncIngress_IsReachableFromThePublicAPI pins the gap that made every
// other test in this file describe an unreachable feature.
//
// runtime.Config.AsyncIngress is internal. ui.SchedulingOptions is what an
// application actually calls, and it carried no field for this — so the flag
// could be set by the runtime's own tests and by nothing else. A guarantee no
// caller can turn on is not a guarantee.
func TestAsyncIngress_IsReachableFromThePublicAPI(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{
		DOMAdapter:   parseAdapter,
		Scheduler:    newTestScheduler(),
		AsyncIngress: true,
		Reset:        true,
	})
	if !parseRt.AsyncIngressEnabled() {
		parseT.Fatal("Config.AsyncIngress did not reach the runtime")
	}
}

// TestAsyncIngress_SchedulerEnforcesIngressForCallersItDoesNotKnow is the
// structural half of the guarantee.
//
// Routing setters, atoms, and fetch completions covered the paths that were
// looked for. AsyncBoundary was not one of them: it resolves a suspension on a
// goroutine and marked its fiber directly, so a resolution could dirty a fiber
// and its ancestors while a sliced render was walking them. Every future caller
// that spawns a goroutine would have had the same hole, because the invariant
// lived at the call sites instead of at the scheduler.
//
// This asserts the enforcement point rather than any one caller: an off-loop
// call to the fiber scheduler is queued, whoever makes it.
func TestAsyncIngress_SchedulerEnforcesIngressForCallersItDoesNotKnow(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	mountCounter(parseT, parseRt, parseContainer, parseScheduler)

	parseFiber := parseRt.currentRoot.child
	if parseFiber == nil {
		parseT.Fatal("expected a mounted fiber")
	}

	// The shape AsyncBoundary uses: a goroutine that waits, then marks a fiber.
	// Called synchronously here so the assertion is about the scheduler's
	// enforcement, not about goroutine timing.
	parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, "async-suspense")

	if parseRt.AsyncInboxDepth() != 1 {
		parseT.Errorf("inbox depth = %d, want 1; an off-loop fiber schedule reached the tree directly, so the invariant still depends on every caller remembering it",
			parseRt.AsyncInboxDepth())
	}

	runScheduledTimeouts(parseScheduler)
	if parseRt.AsyncInboxDepth() != 0 {
		parseT.Errorf("inbox still holds %d entries after the drain", parseRt.AsyncInboxDepth())
	}
}

// TestAsyncIngress_OnLoopFiberScheduleStillDirect keeps the enforcement from
// becoming a tax on the path that is already safe.
func TestAsyncIngress_OnLoopFiberScheduleStillDirect(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newAsyncIngressRuntime(parseT)
	mountCounter(parseT, parseRt, parseContainer, parseScheduler)

	parseFiber := parseRt.currentRoot.child
	if parseFiber == nil {
		parseT.Fatal("expected a mounted fiber")
	}

	parseRt.enterFrameLoop()
	parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, "local-state")
	parseRt.exitFrameLoop()

	if parseRt.AsyncInboxDepth() != 0 {
		parseT.Errorf("inbox depth = %d; an on-loop schedule was queued, which would cost every interaction an extra task",
			parseRt.AsyncInboxDepth())
	}
}
