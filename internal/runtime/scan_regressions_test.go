package runtime

import (
	goruntime "runtime"
	"testing"
)

// Regressions from the second review pass. As before, each test names the defect
// rather than the function, and the ones that are here to record a NON-defect say
// so — ruling those out cost as much as finding the real ones.

// A suspended boundary must not park a new goroutine on every render.
//
// subscribeAsyncBoundary dedupes on fiber.asyncWait, and neither clone literal
// carried it, so the dedupe always failed: one watcher goroutine per render pass,
// each holding a reference to the boundary fiber and each firing its own
// ScheduleUpdateForFiber when the promise finally resolved.
//
// The boundary is dirtied directly rather than through the root, which matters:
// a root-only update bails out at the boundary and never re-subscribes, so the
// obvious test shape passes on the broken code.
func TestSuspendedBoundaryDoesNotLeakAGoroutinePerRender(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	parseNever := make(chan struct{})
	defer close(parseNever) // release the watcher when the test ends

	parseContent := func() *Element {
		SuspendUntil(parseNever, "waiting forever")
		return CreateElement("span", map[string]any{}, "loaded")
	}
	parseApp := func() *Element {
		return CreateElement(AsyncBoundaryNodeType, map[string]any{
			"content":  CreateElement(parseContent, map[string]any{}),
			"fallback": CreateElement("em", map[string]any{"id": "fallback"}, "loading"),
		})
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	if findNodeByID(parseContainer, "fallback") == nil {
		parseT.Fatal("setup: expected the fallback while suspended")
	}

	parseBefore := goruntime.NumGoroutine()
	const parsePasses = 12
	for parseI := 0; parseI < parsePasses; parseI++ {
		parseBoundary := findAsyncBoundaryFiberForTest(parseRt.currentRoot)
		if parseBoundary == nil {
			parseT.Fatalf("boundary vanished after pass %d", parseI)
		}
		parseRt.ScheduleUpdateForFiberWithOrigin(parseBoundary, "hook")
		runScheduledTimeouts(parseScheduler)
	}
	parseGrowth := goruntime.NumGoroutine() - parseBefore

	// One watcher for the whole suspension is correct; one per pass is the bug.
	// Compared against a threshold rather than zero because the scheduler and the
	// test harness may hold transient goroutines of their own.
	if parseGrowth >= parsePasses {
		parseT.Errorf("goroutines grew by %d across %d boundary re-renders: the suspension is re-subscribed every pass",
			parseGrowth, parsePasses)
	}
}

// The suspension must still be WATCHED after a re-render — the fix above must not
// dedupe the subscription out of existence.
func TestSuspendedBoundaryStillResolvesAfterARerender(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	parseDone := make(chan struct{})
	parseContent := func() *Element {
		SuspendUntil(parseDone, "waiting")
		return CreateElement("span", map[string]any{"id": "loaded"}, "loaded")
	}
	parseApp := func() *Element {
		return CreateElement(AsyncBoundaryNodeType, map[string]any{
			"content":  CreateElement(parseContent, map[string]any{}),
			"fallback": CreateElement("em", map[string]any{"id": "fallback"}, "loading"),
		})
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	// Re-render the boundary while still suspended, then resolve.
	parseBoundary := findAsyncBoundaryFiberForTest(parseRt.currentRoot)
	if parseBoundary == nil {
		parseT.Fatal("setup: no boundary fiber")
	}
	parseRt.ScheduleUpdateForFiberWithOrigin(parseBoundary, "hook")
	runScheduledTimeouts(parseScheduler)
	if findNodeByID(parseContainer, "fallback") == nil {
		parseT.Fatal("setup: still expected the fallback before resolving")
	}

	close(parseDone)
	// The watcher goroutine schedules the retry; give the scheduler its turns.
	for parseAttempt := 0; parseAttempt < 200; parseAttempt++ {
		runScheduledTimeouts(parseScheduler)
		if findNodeByID(parseContainer, "loaded") != nil {
			return
		}
		goruntime.Gosched()
	}
	parseT.Error("the boundary never resolved after a re-render; the suspension watcher was lost")
}

// NOT a defect, recorded so it is not chased again: an error boundary remembers
// its error across renders even though boundaryError is dropped by BOTH clone
// literals.
//
// The alternate 2-cycle looks like it must lose the record — acquireWorkInProgress
// zeroes the generation it reuses — but boundaryCapturedError falls back to
// fiber.alternate, and setBoundaryError writes both generations, so one hop is
// always enough. Driven for eight boundary re-renders: the error is held on every
// pass and the failing child renders exactly once.
func TestErrorBoundaryRemembersItsErrorAcrossRenders(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	parseChildRenders := 0
	parseBroken := func() *Element {
		parseChildRenders++
		panic("always broken")
	}
	parseApp := func() *Element {
		return CreateElement(NewErrorBoundaryType(), map[string]any{
			"fallback": CreateElement("em", map[string]any{"id": "fb"}, "failed"),
		}, CreateElement(parseBroken, map[string]any{}))
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	if findNodeByID(parseContainer, "fb") == nil {
		parseT.Fatal("setup: expected the fallback after the child panicked")
	}
	parseAfterMount := parseChildRenders

	for parseI := 0; parseI < 8; parseI++ {
		parseBoundary := findErrorBoundaryFiberForTest(parseRt.currentRoot)
		if parseBoundary == nil {
			parseT.Fatalf("boundary vanished at pass %d", parseI)
		}
		if boundaryCapturedError(parseBoundary) == nil {
			parseT.Fatalf("pass %d: the boundary forgot its captured error", parseI)
		}
		parseRt.ScheduleUpdateForFiberWithOrigin(parseBoundary, "hook")
		runScheduledTimeouts(parseScheduler)
	}

	if parseChildRenders != parseAfterMount {
		parseT.Errorf("the failing child re-rendered %d extra times; the boundary stopped holding its error",
			parseChildRenders-parseAfterMount)
	}
}

func findAsyncBoundaryFiberForTest(parseFiber *Fiber) *Fiber {
	if parseFiber == nil {
		return nil
	}
	if _, parseOk := parseFiber.typeOf.(*AsyncBoundaryElementType); parseOk {
		return parseFiber
	}
	if parseFound := findAsyncBoundaryFiberForTest(parseFiber.child); parseFound != nil {
		return parseFound
	}
	return findAsyncBoundaryFiberForTest(parseFiber.sibling)
}

func findErrorBoundaryFiberForTest(parseFiber *Fiber) *Fiber {
	if parseFiber == nil {
		return nil
	}
	if _, parseOk := parseFiber.typeOf.(*ErrorBoundaryType); parseOk {
		return parseFiber
	}
	if parseFound := findErrorBoundaryFiberForTest(parseFiber.child); parseFound != nil {
		return parseFound
	}
	return findErrorBoundaryFiberForTest(parseFiber.sibling)
}
