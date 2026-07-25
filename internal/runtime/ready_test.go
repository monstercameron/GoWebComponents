package runtime

import "testing"

// v5 P2.3 note: these registered hooks through the package-level OnFirstCommit
// (which targets the global runtime) while committing on a separately
// constructed NewRuntime. That only worked because "first commit" was a
// process-wide flag — the exact singleton leak P2.3 removes. They now register
// on the runtime they commit, which is the correct usage, and a dedicated test
// below still covers the package-level function targeting the global runtime.

func TestOnFirstCommitFiresOnceAfterFirstCommit(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")

	parseCalls := 0
	parseRt.OnFirstCommit(func() { parseCalls++ })

	// Not yet committed.
	if parseCalls != 0 {
		parseT.Fatalf("hook fired before first commit: %d", parseCalls)
	}

	parseRt.Render(CreateElement("div", map[string]any{"id": "a"}, "x"), parseContainer)
	if parseCalls != 1 {
		parseT.Fatalf("expected hook to fire once after first commit, got %d", parseCalls)
	}

	// A second commit must NOT re-fire.
	parseRt.Render(CreateElement("div", map[string]any{"id": "b"}, "y"), parseContainer)
	if parseCalls != 1 {
		parseT.Fatalf("hook re-fired on a later commit, got %d", parseCalls)
	}
}

func TestOnFirstCommitRunsImmediatelyAfterReady(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")

	parseRt.Render(CreateElement("div", map[string]any{"id": "a"}), parseContainer)

	// Registered AFTER the first commit → must run synchronously now.
	parseRan := false
	parseRt.OnFirstCommit(func() { parseRan = true })
	if !parseRan {
		parseT.Fatal("late-registered hook should run immediately once ready")
	}
}

func TestOnFirstCommitRunsAllHooksAndIgnoresNil(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")

	parseOrder := []int{}
	parseRt.OnFirstCommit(func() { parseOrder = append(parseOrder, 1) })
	parseRt.OnFirstCommit(nil) // ignored, no panic
	parseRt.OnFirstCommit(func() { parseOrder = append(parseOrder, 2) })

	parseRt.Render(CreateElement("div", map[string]any{"id": "a"}), parseContainer)

	if len(parseOrder) != 2 || parseOrder[0] != 1 || parseOrder[1] != 2 {
		parseT.Fatalf("expected hooks to run in registration order, got %v", parseOrder)
	}
}

func TestResetFirstCommitHooksForTestClearsState(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.Render(CreateElement("div", map[string]any{"id": "a"}), parseContainer)

	// After reset, a freshly registered hook should wait for the next commit
	// rather than fire immediately.
	parseRt.ResetFirstCommitHooksForTest()
	parseFired := false
	parseRt.OnFirstCommit(func() { parseFired = true })
	if parseFired {
		parseT.Fatal("hook fired immediately after reset (state not cleared)")
	}
	parseRt.Render(CreateElement("div", map[string]any{"id": "b"}), parseContainer)
	if !parseFired {
		parseT.Fatal("hook did not fire on the next commit after reset")
	}
}

// TestPackageLevelOnFirstCommitTargetsGlobalRuntime keeps the compatibility
// surface covered: ui.OnReady and the wasm "gwc:ready" bridge call the
// package-level function, and it must keep working for single-runtime apps.
func TestPackageLevelOnFirstCommitTargetsGlobalRuntime(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseAdapter := newTestDOMAdapter()
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Reset: true})
	ResetFirstCommitHooksForTest()

	parseFired := false
	OnFirstCommit(func() { parseFired = true })
	if parseFired {
		parseT.Fatal("hook fired before the global runtime committed")
	}

	GetGlobalRuntime().Render(CreateElement("div", map[string]any{"id": "g"}), parseAdapter.CreateElement("div"))

	if !parseFired {
		parseT.Error("the package-level hook must fire on the global runtime's first commit")
	}
}
