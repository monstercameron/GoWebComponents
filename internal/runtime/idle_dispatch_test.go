package runtime

import "testing"

// v5 P1.3 â€” idle-lane continuations run through RequestIdleCallback.
//
// RequestIdleCallback has been in the Scheduler interface, and implemented by
// jsdom and mockdom, since before v5 â€” while nothing in the runtime ever called
// it. Making it load-bearing changes the contract for any custom Scheduler that
// implemented it as a no-op, so idle dispatch always arms a SetTimeout backstop
// and takes whichever arrives first.

// noIdleScheduler implements RequestIdleCallback as a no-op â€” the exact shape
// of a custom Scheduler written while the method was dead code.
type noIdleScheduler struct {
	timeouts []func()
	idleAsks int
}

func (parseS *noIdleScheduler) RequestIdleCallback(func(Deadline)) { parseS.idleAsks++ }
func (parseS *noIdleScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	parseS.timeouts = append(parseS.timeouts, parseCallback)
}
func (parseS *noIdleScheduler) drain() {
	for len(parseS.timeouts) > 0 {
		parseNext := parseS.timeouts[0]
		parseS.timeouts = parseS.timeouts[1:]
		parseNext()
	}
}

// idleScheduler runs idle callbacks, like a browser that honors the API.
type idleScheduler struct {
	idle     []func(Deadline)
	timeouts []func()
}

func (parseS *idleScheduler) RequestIdleCallback(parseCallback func(Deadline)) {
	parseS.idle = append(parseS.idle, parseCallback)
}
func (parseS *idleScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	parseS.timeouts = append(parseS.timeouts, parseCallback)
}
func (parseS *idleScheduler) runIdle() {
	parseBatch := parseS.idle
	parseS.idle = nil
	for _, parseCallback := range parseBatch {
		parseCallback(globalInfiniteDeadline)
	}
}

// TestIdleDispatch_UsesIdleCallbackForBackgroundLanes pins the routing.
func TestIdleDispatch_UsesIdleCallbackForBackgroundLanes(parseT *testing.T) {
	parseScheduler := &idleScheduler{}
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: parseScheduler, Reset: true})

	parseRan := 0
	parseRt.dispatchLaneWork(UpdateLaneBackground, func() { parseRan++ })

	if len(parseScheduler.idle) != 1 {
		parseT.Fatalf("background lane should request an idle callback, got %d", len(parseScheduler.idle))
	}
	parseScheduler.runIdle()
	if parseRan != 1 {
		parseT.Fatalf("idle work ran %d times, want 1", parseRan)
	}

	// The backstop must not double-run the work once idle already handled it.
	for len(parseScheduler.timeouts) > 0 {
		parseNext := parseScheduler.timeouts[0]
		parseScheduler.timeouts = parseScheduler.timeouts[1:]
		parseNext()
	}
	if parseRan != 1 {
		parseT.Errorf("work ran %d times; the backstop must not re-run work idle already did", parseRan)
	}
}

// TestIdleDispatch_UsesTimeoutForUrgentLanes: input and sync must not be
// pushed into idle time.
func TestIdleDispatch_UsesTimeoutForUrgentLanes(parseT *testing.T) {
	for _, parseLane := range []UpdateLane{UpdateLaneSync, UpdateLaneInput, UpdateLaneDefault} {
		parseScheduler := &idleScheduler{}
		parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: parseScheduler, Reset: true})

		parseRt.dispatchLaneWork(parseLane, func() {})

		if len(parseScheduler.idle) != 0 {
			parseT.Errorf("lane %v must not use idle dispatch", parseLane)
		}
		if len(parseScheduler.timeouts) != 1 {
			parseT.Errorf("lane %v should schedule one timeout, got %d", parseLane, len(parseScheduler.timeouts))
		}
	}
}

// TestIdleDispatch_NoOpIdleCallbackStillMakesProgress is the contract-change
// safety net: a Scheduler that ignores RequestIdleCallback must still run its
// background work, via the backstop, rather than stalling forever.
func TestIdleDispatch_NoOpIdleCallbackStillMakesProgress(parseT *testing.T) {
	parseScheduler := &noIdleScheduler{}
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: parseScheduler, Reset: true})
	ClearDiagnostics()

	parseRan := 0
	parseRt.dispatchLaneWork(UpdateLaneBackground, func() { parseRan++ })

	if parseScheduler.idleAsks != 1 {
		parseT.Fatalf("expected one idle request, got %d", parseScheduler.idleAsks)
	}
	if parseRan != 0 {
		parseT.Fatal("work ran before any callback fired")
	}

	parseScheduler.drain()

	if parseRan != 1 {
		parseT.Fatalf("backstop did not run the work: ran=%d", parseRan)
	}

	// R4: the degradation must be visible, not silent.
	parseFoundDiagnostic := false
	for _, parseEntry := range GetDiagnostics() {
		if parseEntry.Source == "runtime" && parseEntry.Severity == DiagnosticWarning {
			parseFoundDiagnostic = true
			break
		}
	}
	if !parseFoundDiagnostic {
		parseT.Error("a no-op RequestIdleCallback must emit a diagnostic, not degrade silently")
	}
}

// TestIdleDispatch_FallbackDiagnosticIsReportedOnce keeps a no-op scheduler
// from flooding the diagnostic buffer on every idle dispatch.
func TestIdleDispatch_FallbackDiagnosticIsReportedOnce(parseT *testing.T) {
	parseScheduler := &noIdleScheduler{}
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: parseScheduler, Reset: true})
	ClearDiagnostics()

	for range 5 {
		parseRt.dispatchLaneWork(UpdateLaneBackground, func() {})
		parseScheduler.drain()
	}

	parseWarnings := 0
	for _, parseEntry := range GetDiagnostics() {
		if parseEntry.Source == "runtime" && parseEntry.Severity == DiagnosticWarning {
			parseWarnings++
		}
	}
	if parseWarnings != 1 {
		parseT.Errorf("expected the idle-fallback warning once, got %d", parseWarnings)
	}
}

// TestIdleDispatch_NoSchedulerRunsInline preserves native/SSR semantics.
func TestIdleDispatch_NoSchedulerRunsInline(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Reset: true})

	parseRan := 0
	parseRt.dispatchLaneWork(UpdateLaneBackground, func() { parseRan++ })

	if parseRan != 1 {
		parseT.Errorf("with no scheduler the continuation must run inline, ran=%d", parseRan)
	}
}


