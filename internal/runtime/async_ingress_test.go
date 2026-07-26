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
