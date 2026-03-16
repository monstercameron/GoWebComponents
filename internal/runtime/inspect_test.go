package runtime

import (
	"testing"
	"time"
)

func TestDiagnosticsDeduplicateBySourceSeverityAndMessage(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	ReportDiagnostic("router", DiagnosticWarning, "duplicate route registration")
	ReportDiagnostic("router", DiagnosticWarning, "duplicate route registration")
	ReportDiagnostic("runtime", DiagnosticError, "invalid hook usage")

	diagnostics := GetDiagnostics()
	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 unique diagnostics, got %d", len(diagnostics))
	}
	if diagnostics[0].Count != 2 {
		t.Fatalf("expected duplicate diagnostic count 2, got %d", diagnostics[0].Count)
	}
}

func TestRuntimeInspectCapturesTreeStatsAndHooks(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	rt := &Runtime{}
	rt.profiling = runtimeProfiling{
		renderCalls:          3,
		scheduledRootUpdates: 2,
		scheduledFiberMarks:  5,
		workLoopPasses:       4,
		processedUnits:       9,
		commitCount:          2,
		lastRenderDurationNs: 1_500_000,
		lastCommitDurationNs: 900_000,
	}
	childHooks := &Hooks{
		states: []interface{}{42, 42},
		memos:  []memoizedValue{{value: "memoized"}},
		ids:    []string{"node-1"},
	}
	child := &Fiber{
		typeOf:  func() {},
		dirty:   true,
		hooks:   childHooks,
		effects: []Effect{{}},
	}
	root := &Fiber{
		typeOf: "ROOT",
		child:  child,
	}
	child.parent = root
	childHooks.owner = child
	rt.currentRoot = root

	ReportDiagnostic("runtime", DiagnosticWarning, "test warning")

	snapshot := rt.Inspect()
	if snapshot.Root == nil {
		t.Fatal("expected root snapshot")
	}
	if snapshot.Stats.TotalFibers != 2 {
		t.Fatalf("expected 2 fibers, got %d", snapshot.Stats.TotalFibers)
	}
	if snapshot.Stats.ComponentFibers != 1 {
		t.Fatalf("expected 1 component fiber, got %d", snapshot.Stats.ComponentFibers)
	}
	if snapshot.Stats.DirtyFibers != 1 {
		t.Fatalf("expected 1 dirty fiber, got %d", snapshot.Stats.DirtyFibers)
	}
	if len(snapshot.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic entry, got %d", len(snapshot.Diagnostics))
	}
	if snapshot.Profiling.RenderCalls != 3 || snapshot.Profiling.CommitCount != 2 {
		t.Fatalf("expected profiling snapshot to preserve runtime counters, got %+v", snapshot.Profiling)
	}
	if len(snapshot.Root.Children) != 1 {
		t.Fatalf("expected root child snapshot, got %d children", len(snapshot.Root.Children))
	}
	component := snapshot.Root.Children[0]
	if component.Kind != "component" {
		t.Fatalf("expected component child kind, got %q", component.Kind)
	}
	if component.HookCount < 3 {
		t.Fatalf("expected hook inspection entries, got %d", component.HookCount)
	}
}

func TestRuntimeInspectCollectsHotBranchesAndTiming(t *testing.T) {
	rt := &Runtime{}
	grandchild := &Fiber{typeOf: "span", commitDurationNs: int64(1 * time.Millisecond)}
	child := &Fiber{
		typeOf:            "section",
		commitDurationNs:  int64(2 * time.Millisecond),
		effectDurationNs:  int64(3 * time.Millisecond),
		cleanupDurationNs: int64(1 * time.Millisecond),
		child:             grandchild,
	}
	root := &Fiber{typeOf: "ROOT", child: child}
	child.parent = root
	grandchild.parent = child
	rt.currentRoot = root
	rt.profiling = runtimeProfiling{
		effectExecutions:      4,
		cleanupExecutions:     2,
		lastEffectDurationNs:  int64(3 * time.Millisecond),
		lastCleanupDurationNs: int64(1 * time.Millisecond),
	}

	snapshot := rt.Inspect()
	if snapshot.Root == nil || len(snapshot.Root.Children) != 1 {
		t.Fatal("expected inspected tree with one child")
	}
	branch := snapshot.Root.Children[0]
	if branch.SelfDurationNs != int64(6*time.Millisecond) {
		t.Fatalf("expected branch self duration 6ms, got %d", branch.SelfDurationNs)
	}
	if branch.SubtreeDurationNs != int64(7*time.Millisecond) {
		t.Fatalf("expected branch subtree duration 7ms, got %d", branch.SubtreeDurationNs)
	}
	if snapshot.Profiling.EffectExecutions != 4 || snapshot.Profiling.CleanupExecutions != 2 {
		t.Fatalf("expected effect/cleanup counters in profiling snapshot, got %+v", snapshot.Profiling)
	}
	if len(snapshot.Profiling.HotBranches) == 0 {
		t.Fatal("expected hot branches to be collected")
	}
	if snapshot.Profiling.HotBranches[0].Name != "section" {
		t.Fatalf("expected hottest branch to be section, got %q", snapshot.Profiling.HotBranches[0].Name)
	}
	if snapshot.Profiling.HotBranches[0].SubtreeDurationNs != int64(7*time.Millisecond) {
		t.Fatalf("expected hot branch subtree duration 7ms, got %d", snapshot.Profiling.HotBranches[0].SubtreeDurationNs)
	}
}
