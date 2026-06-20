package runtime

import "testing"

func TestOnFirstCommitFiresOnceAfterFirstCommit(parseT *testing.T) {
	ResetFirstCommitHooksForTest()
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")

	parseCalls := 0
	OnFirstCommit(func() { parseCalls++ })

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
	ResetFirstCommitHooksForTest()
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")

	parseRt.Render(CreateElement("div", map[string]any{"id": "a"}), parseContainer)

	// Registered AFTER the first commit → must run synchronously now.
	parseRan := false
	OnFirstCommit(func() { parseRan = true })
	if !parseRan {
		parseT.Fatal("late-registered hook should run immediately once ready")
	}
}

func TestOnFirstCommitRunsAllHooksAndIgnoresNil(parseT *testing.T) {
	ResetFirstCommitHooksForTest()
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")

	parseOrder := []int{}
	OnFirstCommit(func() { parseOrder = append(parseOrder, 1) })
	OnFirstCommit(nil) // ignored, no panic
	OnFirstCommit(func() { parseOrder = append(parseOrder, 2) })

	parseRt.Render(CreateElement("div", map[string]any{"id": "a"}), parseContainer)

	if len(parseOrder) != 2 || parseOrder[0] != 1 || parseOrder[1] != 2 {
		parseT.Fatalf("expected hooks to run in registration order, got %v", parseOrder)
	}
}

func TestResetFirstCommitHooksForTestClearsState(parseT *testing.T) {
	ResetFirstCommitHooksForTest()
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")
	parseRt.Render(CreateElement("div", map[string]any{"id": "a"}), parseContainer)

	// After reset, a freshly registered hook should wait for the next commit
	// rather than fire immediately.
	ResetFirstCommitHooksForTest()
	parseFired := false
	OnFirstCommit(func() { parseFired = true })
	if parseFired {
		parseT.Fatal("hook fired immediately after reset (state not cleared)")
	}
	parseRt2 := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseRt2.Render(CreateElement("div", map[string]any{"id": "b"}), parseAdapter.CreateElement("div"))
	if !parseFired {
		parseT.Fatal("hook did not fire on the next commit after reset")
	}
}
