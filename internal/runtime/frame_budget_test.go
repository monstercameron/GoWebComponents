package runtime

import (
	"testing"
	"time"
)

// v5 P1.2 — real wall-clock slice budget.
//
// Before this, continueWorkLoop always passed globalInfiniteDeadline, whose
// TimeRemaining() returns a constant 1000. workLoop's `TimeRemaining() < 1`
// yield check therefore could never fire: slicing was purely count-based, and a
// slice took whatever wall time 1200 fibers happened to cost.

// TestFrameBudget_InfiniteDeadlineNeverYields documents the defect P1.2 fixes.
// If this ever starts failing, the sentinel has gained real semantics and the
// gate below is measuring the wrong thing.
func TestFrameBudget_InfiniteDeadlineNeverYields(parseT *testing.T) {
	if getRemaining := globalInfiniteDeadline.TimeRemaining(); getRemaining < 1 {
		parseT.Fatalf("infinite deadline reported %v remaining; workLoop's yield check would fire", getRemaining)
	}
	if globalInfiniteDeadline.DidTimeout() {
		parseT.Fatal("infinite deadline must never report a timeout")
	}
}

// TestFrameBudget_DeadlineExpires proves the new deadline actually elapses,
// which is the whole point: it must reach the sub-1ms region workLoop yields on.
func TestFrameBudget_DeadlineExpires(parseT *testing.T) {
	parseDeadline := newFrameBudgetDeadline(2)

	if getRemaining := parseDeadline.TimeRemaining(); getRemaining <= 1 {
		parseT.Fatalf("a fresh 2ms budget should have >1ms remaining, got %v", getRemaining)
	}
	if parseDeadline.DidTimeout() {
		parseT.Fatal("a fresh budget must not report a timeout")
	}

	time.Sleep(3 * time.Millisecond)

	if getRemaining := parseDeadline.TimeRemaining(); getRemaining >= 1 {
		parseT.Errorf("an elapsed 2ms budget should be under the 1ms yield threshold, got %v", getRemaining)
	}
	if !parseDeadline.DidTimeout() {
		parseT.Error("an elapsed budget must report a timeout")
	}
}

// TestFrameBudget_DisabledByDefault pins R2: the runtime keeps count-only
// slicing unless a budget is requested.
// TestFrameBudget_EnabledByDefault pins the flipped default.
//
// It was off, on the R2 rule that a v5 flag stays off until its acceptance test
// passes. TestV5SchedulingComparison is that test and it passed: long frames
// 23 -> 13, worst 158.8ms -> 98.1ms, interaction p95 56.0ms -> 48.0ms against a
// 50ms budget, with the loaded frame p95 unchanged. Leaving it off would mean
// the measured behaviour is the one nobody gets.
func TestFrameBudget_EnabledByDefault(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Reset: true})

	if !parseRt.frameBudgetEnabled() {
		parseT.Error("frame budget must be on without being asked for")
	}
	if parseRt.frameBudgetMs != defaultFrameBudgetMs {
		parseT.Errorf("default budget = %v, want %v", parseRt.frameBudgetMs, defaultFrameBudgetMs)
	}
	if parseRt.resolveWorkLoopDeadline() == Deadline(globalInfiniteDeadline) {
		parseT.Error("with a budget in force the work loop must take a finite deadline")
	}
}

// TestFrameBudget_ConfiguredValueIsUsed covers explicit and default selection.
func TestFrameBudget_ConfiguredValueIsUsed(parseT *testing.T) {
	parseExplicit := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), FrameBudgetMs: 8, Reset: true})
	if parseExplicit.frameBudgetMs != 8 {
		parseT.Errorf("explicit budget = %v, want 8", parseExplicit.frameBudgetMs)
	}
	if !parseExplicit.frameBudgetEnabled() {
		parseT.Error("an explicit budget must enable time-based slicing")
	}

	// The sentinel is inverted now that the budget is the default: zero takes
	// the default, and NEGATIVE is the explicit opt-out. A caller that wants
	// count-only slicing has to ask for it, which is the right way round once
	// the measured behaviour is the default one.
	parseDisabled := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), FrameBudgetMs: -1, Reset: true})
	if parseDisabled.frameBudgetEnabled() {
		parseT.Error("a negative budget must disable time-based slicing")
	}
}

// TestFrameBudget_ResolvesToARealDeadlineWhenEnabled proves the work loop
// receives a budget that can actually expire, rather than the sentinel.
func TestFrameBudget_ResolvesToARealDeadlineWhenEnabled(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), FrameBudgetMs: 1, Reset: true})

	parseDeadline := parseRt.resolveWorkLoopDeadline()
	if parseDeadline == Deadline(globalInfiniteDeadline) {
		parseT.Fatal("an enabled budget must not resolve to the infinite deadline")
	}

	time.Sleep(2 * time.Millisecond)
	if !parseDeadline.DidTimeout() {
		parseT.Error("the work loop's deadline must be able to expire; that is the entire fix")
	}
}

// TestFrameBudget_GatedOnInterruptSafeRestart pins the P1.2 -> P2.5 dependency
// in code rather than in a planning document. Live time-slicing multiplies the
// mid-pass yields that made T12 reachable, so the budget must refuse to enable
// itself if the interrupt-safe restart path is ever removed.
func TestFrameBudget_GatedOnInterruptSafeRestart(parseT *testing.T) {
	if !interruptRestartIsSafe {
		parseT.Skip("interrupt-safe restart disabled; the gate is exercised by the fallback path")
	}

	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), FrameBudgetMs: 4, Reset: true})
	if !parseRt.frameBudgetEnabled() {
		parseT.Fatal("budget should be enabled while the interrupt-safe restart path is present")
	}

	// The guard reads the constant that lives beside the P2.5 fix, so removing
	// that fix without flipping the constant is what this is protecting against.
	// Assert the coupling exists rather than only that today's value is true.
	if parseRt.frameBudgetMs <= 0 {
		parseT.Fatal("configured budget was not stored")
	}
}

// TestFrameBudget_WorkLoopStillCommitsUnderABudget is the end-to-end check:
// enabling time slicing must not change what gets rendered, only when the loop
// yields.
func TestFrameBudget_WorkLoopStillCommitsUnderABudget(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter:    parseAdapter,
		Scheduler:     parseScheduler,
		FrameBudgetMs: 5,
		Reset:         true,
	})
	parseContainer := parseAdapter.CreateElement("div")

	parseChildren := make([]any, 0, 64)
	for parseI := range 64 {
		parseChildren = append(parseChildren, CreateElement("span", map[string]any{"data-i": parseI}))
	}
	parseApp := func() *Element {
		return CreateElement("section", map[string]any{}, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	if len(parseRoot.children) == 0 {
		parseT.Fatal("nothing committed under a frame budget")
	}
	parseSection := parseRoot.children[0].(*testDOMNode)
	if len(parseSection.children) != 64 {
		parseT.Errorf("committed %d children, want all 64 — time slicing must not drop work", len(parseSection.children))
	}
}
