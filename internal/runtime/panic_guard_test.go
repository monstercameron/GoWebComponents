package runtime

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// capturePanicReports installs a report hook for the test and returns a
// thread-safe accessor for the captured reports.
func capturePanicReports(parseT *testing.T) func() []PanicReport {
	parseT.Helper()
	parsePrevious := CurrentUnhandledPanicLoggingOptions()
	var parseMu sync.Mutex
	var parseReports []PanicReport
	ConfigureUnhandledPanicLogging(PanicLoggingOptions{
		HideRawPanicOutput: true,
		OnReport: func(parseReport PanicReport) {
			parseMu.Lock()
			parseReports = append(parseReports, parseReport)
			parseMu.Unlock()
		},
	})
	parseT.Cleanup(func() { ConfigureUnhandledPanicLogging(parsePrevious) })
	return func() []PanicReport {
		parseMu.Lock()
		defer parseMu.Unlock()
		return append([]PanicReport(nil), parseReports...)
	}
}

// TestContainPanicReportsWithoutRethrow pins the containment contract: the
// recovered panic becomes one async-phase report and never re-panics.
func TestContainPanicReportsWithoutRethrow(parseT *testing.T) {
	parseGetReports := capturePanicReports(parseT)

	ContainPanic("test", PanicPhaseAsync, "exploding task", "boom-contained")

	parseReports := parseGetReports()
	if len(parseReports) != 1 {
		parseT.Fatalf("expected exactly one report, got %d", len(parseReports))
	}
	parseReport := parseReports[0]
	if parseReport.Phase != PanicPhaseAsync {
		parseT.Fatalf("report phase = %q", parseReport.Phase)
	}
	if parseReport.Code != "GWC-RUNTIME-PANIC-ASYNC" {
		parseT.Fatalf("report code = %q", parseReport.Code)
	}
	if parseReport.Subject != "exploding task" {
		parseT.Fatalf("report subject = %q", parseReport.Subject)
	}
	if !strings.Contains(parseReport.Summary, "boom-contained") {
		parseT.Fatalf("report summary lost panic value: %q", parseReport.Summary)
	}
	if !strings.Contains(parseReport.Consequence, "committed UI stays mounted") {
		parseT.Fatalf("async consequence should describe survival, got %q", parseReport.Consequence)
	}
}

// TestContainPanicNilIsNoop verifies a nil recovered value produces nothing.
func TestContainPanicNilIsNoop(parseT *testing.T) {
	parseGetReports := capturePanicReports(parseT)
	ContainPanic("test", PanicPhaseAsync, "noop", nil)
	if parseCount := len(parseGetReports()); parseCount != 0 {
		parseT.Fatalf("nil recovered value produced %d reports", parseCount)
	}
}

// TestContainPanicSkipsAlreadyReported verifies an inner-boundary-marked
// panic is swallowed without a duplicate report.
func TestContainPanicSkipsAlreadyReported(parseT *testing.T) {
	parseGetReports := capturePanicReports(parseT)
	ContainPanic("test", PanicPhaseAsync, "duplicate", reportedPanic{Original: "already-seen"})
	if parseCount := len(parseGetReports()); parseCount != 0 {
		parseT.Fatalf("already-reported panic produced %d duplicate reports", parseCount)
	}
}

// TestSafeGoContainsGoroutinePanic verifies a panic inside a SafeGo goroutine
// is reported and does not kill the program.
func TestSafeGoContainsGoroutinePanic(parseT *testing.T) {
	parseGetReports := capturePanicReports(parseT)

	parseDone := make(chan struct{})
	SafeGo("test", "panicking goroutine", func() {
		defer close(parseDone)
		panic("goroutine-boom")
	})

	select {
	case <-parseDone:
	case <-time.After(5 * time.Second):
		parseT.Fatal("SafeGo goroutine never finished")
	}

	parseDeadline := time.Now().Add(5 * time.Second)
	for len(parseGetReports()) == 0 && time.Now().Before(parseDeadline) {
		time.Sleep(time.Millisecond)
	}
	parseReports := parseGetReports()
	if len(parseReports) != 1 {
		parseT.Fatalf("expected one contained-goroutine report, got %d", len(parseReports))
	}
	if !strings.Contains(parseReports[0].Summary, "goroutine-boom") {
		parseT.Fatalf("report summary = %q", parseReports[0].Summary)
	}
}

// TestSafeGoNilFunc verifies a nil fn is a safe no-op.
func TestSafeGoNilFunc(parseT *testing.T) {
	SafeGo("test", "nil fn", nil)
}

// TestGuardCallbackContainsPanic verifies the wrapped callback contains
// panics instead of letting them unwind into the host bridge.
func TestGuardCallbackContainsPanic(parseT *testing.T) {
	parseGetReports := capturePanicReports(parseT)

	parseWrapped := GuardCallback("test", "host callback", func() {
		panic("callback-boom")
	})
	parseWrapped()

	parseReports := parseGetReports()
	if len(parseReports) != 1 {
		parseT.Fatalf("expected one report, got %d", len(parseReports))
	}
	if !strings.Contains(parseReports[0].Summary, "callback-boom") {
		parseT.Fatalf("report summary = %q", parseReports[0].Summary)
	}
}

// TestRecoverWorkLoopStateClearsInFlightRender verifies the post-panic reset
// abandons work-in-progress while preserving the committed tree.
func TestRecoverWorkLoopStateClearsInFlightRender(parseT *testing.T) {
	parseCommitted := &Fiber{}
	parseRt := &Runtime{
		wipRoot:                 &Fiber{},
		currentRoot:             parseCommitted,
		nextUnitOfWork:          &Fiber{},
		deletions:               []*Fiber{{}},
		pendingEffectFibers:     []*Fiber{{}},
		tracksPendingEffects:    true,
		updateScheduled:         true,
		pendingBoundaryRecovery: true,
		hydrating:               true,
	}

	parseRt.recoverWorkLoopState()

	if parseRt.wipRoot != nil || parseRt.nextUnitOfWork != nil {
		parseT.Fatal("in-flight render was not abandoned")
	}
	if len(parseRt.deletions) != 0 || parseRt.pendingEffectFibers != nil {
		parseT.Fatal("pending commit work was not cleared")
	}
	if parseRt.updateScheduled || parseRt.tracksPendingEffects || parseRt.pendingBoundaryRecovery || parseRt.hydrating {
		parseT.Fatal("scheduling flags were not reset")
	}
	if parseRt.currentRoot != parseCommitted {
		parseT.Fatal("committed tree must survive the reset")
	}

	// Nil receiver must be safe: the work loop may panic before a runtime exists.
	var parseNilRt *Runtime
	parseNilRt.recoverWorkLoopState()
}
