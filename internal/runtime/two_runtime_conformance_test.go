package runtime

import "testing"

// v5 P2.3 — behavioral two-runtime conformance.
//
// The acceptance criterion is deliberately NOT "-race clean". Every global this
// item touches is already mutex- or atomic-guarded, so the race detector could
// never find any of them: these are semantic singleton leaks, not data races.
// A passing -race run proves nothing about them, and on top of that -race is
// unsupported on windows/arm64, so that gate has to run in CI on a supported
// platform regardless.
//
// What these assert instead is that each runtime observes only its own effects.

// TestTwoRuntimes_EachFiresItsOwnReadyHooks was a real bug before P2.3: the
// first-commit signal was three package globals, so the second runtime's
// initial render found firstCommitDone already true and never fired.
func TestTwoRuntimes_EachFiresItsOwnReadyHooks(parseT *testing.T) {
	parseFirstAdapter := newTestDOMAdapter()
	parseSecondAdapter := newTestDOMAdapter()
	parseFirst := NewRuntime(Config{DOMAdapter: parseFirstAdapter})
	parseSecond := NewRuntime(Config{DOMAdapter: parseSecondAdapter})

	isFirstReady, isSecondReady := false, false
	parseFirst.OnFirstCommit(func() { isFirstReady = true })
	parseSecond.OnFirstCommit(func() { isSecondReady = true })

	if parseErr := parseFirst.RenderInto(parseFirstAdapter.CreateElement("div"),
		CreateElement("div", map[string]any{}, "first")); parseErr != nil {
		parseT.Fatalf("first render: %v", parseErr)
	}

	if !isFirstReady {
		parseT.Fatal("the first runtime did not fire its ready hook")
	}
	if isSecondReady {
		parseT.Fatal("the second runtime's hook fired on another runtime's commit")
	}

	if parseErr := parseSecond.RenderInto(parseSecondAdapter.CreateElement("div"),
		CreateElement("div", map[string]any{}, "second")); parseErr != nil {
		parseT.Fatalf("second render: %v", parseErr)
	}

	if !isSecondReady {
		parseT.Error("the second runtime never fired its own ready hook; a process-wide 'first commit' flag is the bug this pins")
	}
}

// TestTwoRuntimes_ReadyStateIsIndependent covers the HasCommitted view.
func TestTwoRuntimes_ReadyStateIsIndependent(parseT *testing.T) {
	parseFirstAdapter := newTestDOMAdapter()
	parseFirst := NewRuntime(Config{DOMAdapter: parseFirstAdapter})
	parseSecond := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})

	if parseFirst.HasCommitted() || parseSecond.HasCommitted() {
		parseT.Fatal("neither runtime has committed yet")
	}

	if parseErr := parseFirst.RenderInto(parseFirstAdapter.CreateElement("div"),
		CreateElement("div", map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	if !parseFirst.HasCommitted() {
		parseT.Error("the committed runtime should report having committed")
	}
	if parseSecond.HasCommitted() {
		parseT.Error("an uncommitted runtime must not report a commit from elsewhere")
	}
}

// TestTwoRuntimes_LateHookRunsImmediately keeps the documented contract: a hook
// registered after the commit runs now, per runtime.
func TestTwoRuntimes_LateHookRunsImmediately(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})

	if parseErr := parseRt.RenderInto(parseAdapter.CreateElement("div"),
		CreateElement("div", map[string]any{})); parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}

	isRan := false
	parseRt.OnFirstCommit(func() { isRan = true })
	if !isRan {
		parseT.Error("a hook registered after the first commit must run immediately")
	}
}

// TestTwoRuntimes_AtomsStayIsolated re-asserts P2.6's property as part of the
// conformance suite, since atom isolation is what P3.7 depends on.
func TestTwoRuntimes_AtomsStayIsolated(parseT *testing.T) {
	defer SetCurrentFiber(nil)

	parseFirst := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})
	parseSecond := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})

	if parseErr := parseFirst.SetAtomValue("k", "a"); parseErr != nil {
		parseT.Fatalf("set on first: %v", parseErr)
	}
	if parseErr := parseSecond.SetAtomValue("k", "b"); parseErr != nil {
		parseT.Fatalf("set on second: %v", parseErr)
	}

	if parseValue, _ := parseFirst.GetAtomValue("k"); parseValue != "a" {
		parseT.Errorf("first runtime read %#v, want %q", parseValue, "a")
	}
	if parseValue, _ := parseSecond.GetAtomValue("k"); parseValue != "b" {
		parseT.Errorf("second runtime read %#v, want %q", parseValue, "b")
	}
}

// TestTwoRuntimes_IDCountersAreIndependent: UseId must not hand two runtimes
// colliding identifiers, which would break SSR hydration matching.
func TestTwoRuntimes_IDCountersAreIndependent(parseT *testing.T) {
	defer SetCurrentFiber(nil)

	parseFirst := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})
	parseSecond := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})

	SetCurrentFiber(&Fiber{typeOf: "component", ownerRuntime: parseFirst})
	_ = GoUseId()
	_ = GoUseId()
	SetCurrentFiber(&Fiber{typeOf: "component", ownerRuntime: parseSecond})
	_ = GoUseId()

	if parseFirst.idCounter != 2 {
		parseT.Errorf("first runtime id counter = %d, want 2", parseFirst.idCounter)
	}
	if parseSecond.idCounter != 1 {
		parseT.Errorf("second runtime id counter = %d, want 1; counters must not be shared", parseSecond.idCounter)
	}
}

// TestTwoRuntimes_SchedulerStateIsIndependent: one runtime's pending pass must
// not suppress another's.
func TestTwoRuntimes_SchedulerStateIsIndependent(parseT *testing.T) {
	parseFirst := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseSecond := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseFirst.currentRoot = &Fiber{typeOf: "ROOT", props: map[string]any{"children": []any{}}}
	parseSecond.currentRoot = &Fiber{typeOf: "ROOT", props: map[string]any{"children": []any{}}}

	parseFirst.ScheduleUpdate()

	if !parseFirst.updateScheduled {
		parseT.Fatal("the first runtime should have a pass scheduled")
	}
	if parseSecond.updateScheduled {
		parseT.Error("scheduling on one runtime must not mark another as scheduled")
	}

	parseSecond.ScheduleUpdate()
	if !parseSecond.updateScheduled {
		parseT.Error("the second runtime could not schedule its own pass")
	}
}
