package runtime

import (
	"testing"
	"time"
)

func TestDiagnosticsDeduplicateBySourceSeverityAndMessage(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	ReportDiagnostic("router", DiagnosticWarning, "duplicate route registration")
	ReportDiagnostic("router", DiagnosticWarning, "duplicate route registration")
	ReportDiagnostic("runtime", DiagnosticError, "invalid hook usage")
	ReportDiagnosticWithContext("runtime", DiagnosticWarning, "duplicate route registration", "App > Boundary", []string{"App", "Boundary"})

	diagnostics := GetDiagnostics()
	if len(diagnostics) != 3 {
		t.Fatalf("expected 3 unique diagnostics, got %d", len(diagnostics))
	}
	if diagnostics[0].Count != 2 {
		t.Fatalf("expected duplicate diagnostic count 2, got %d", diagnostics[0].Count)
	}
}

func TestRuntimeInspectCapturesTreeStatsAndHooks(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

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
		states:    []interface{}{42, 42},
		memos:     []memoizedValue{{value: "memoized"}},
		ids:       []string{"node-1"},
		signature: []string{"state", "memo", "id"},
	}
	child := &Fiber{
		typeOf:  testSignatureComponent,
		props:   map[string]interface{}{"key": "hero"},
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
	if snapshot.Diagnostics[0].ComponentStack != nil {
		t.Fatalf("expected plain diagnostic to have no component stack, got %+v", snapshot.Diagnostics[0])
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
	if component.Signature == nil {
		t.Fatal("expected component signature")
	}
	if component.Signature.Name != "testSignatureComponent" {
		t.Fatalf("expected component name to round-trip, got %q", component.Signature.Name)
	}
	if component.Signature.Key != "hero" {
		t.Fatalf("expected component key to round-trip, got %q", component.Signature.Key)
	}
	if len(component.Signature.HookKinds) != 3 || component.Signature.HookKinds[0] != "state" || component.Signature.HookKinds[2] != "id" {
		t.Fatalf("expected hook order to be preserved, got %+v", component.Signature.HookKinds)
	}
	if !component.Signature.CompatibleWith(*component.Signature) {
		t.Fatal("expected signature to be compatible with itself")
	}
}

func TestRuntimeInspectCapturesFineGrainedMetadata(t *testing.T) {
	rt := &Runtime{}
	rt.profiling = runtimeProfiling{
		scheduledFiberMarks:    3,
		scheduledGranularMarks: 2,
		commitCount:            1,
		fineGrainedCommits:     2,
	}
	reactive := &Fiber{
		typeOf:         ReactiveTextNodeType,
		fineGrained:    true,
		reactiveAtomID: "count",
		updateOrigin:   "fine-grained",
	}
	root := &Fiber{typeOf: "ROOT", child: reactive}
	reactive.parent = root
	rt.currentRoot = root

	snapshot := rt.Inspect()
	if snapshot.Stats.FineGrainedFibers != 1 {
		t.Fatalf("expected 1 fine-grained fiber, got %d", snapshot.Stats.FineGrainedFibers)
	}
	if snapshot.Profiling.ScheduledGranularMarks != 2 {
		t.Fatalf("expected 2 granular marks, got %d", snapshot.Profiling.ScheduledGranularMarks)
	}
	if snapshot.Profiling.FineGrainedCommits != 2 {
		t.Fatalf("expected 2 fine-grained commits, got %d", snapshot.Profiling.FineGrainedCommits)
	}
	if len(snapshot.Root.Children) != 1 {
		t.Fatalf("expected one child snapshot, got %d", len(snapshot.Root.Children))
	}
	node := snapshot.Root.Children[0]
	if !node.FineGrained {
		t.Fatal("expected inspected node to be marked fine-grained")
	}
	if node.ReactiveSource != "count" {
		t.Fatalf("expected reactive source to round-trip, got %q", node.ReactiveSource)
	}
	if node.UpdateOrigin != "fine-grained" {
		t.Fatalf("expected update origin to round-trip, got %q", node.UpdateOrigin)
	}
}

func TestComponentSignatureCompatibilityUsesIdentityAndHookOrder(t *testing.T) {
	base := ComponentSignature{
		Kind:          "component",
		Name:          "Counter",
		QualifiedName: "github.com/example.Counter",
		Key:           "primary",
		HookKinds:     []string{"state", "effect", "id"},
	}

	compatible := ComponentSignature{
		Kind:          "component",
		Name:          "Counter",
		QualifiedName: "github.com/example.Counter",
		Key:           "primary",
		HookKinds:     []string{"state", "effect", "id"},
	}
	if !base.CompatibleWith(compatible) {
		t.Fatal("expected matching signature to be compatible")
	}

	incompatible := ComponentSignature{
		Kind:          "component",
		Name:          "Counter",
		QualifiedName: "github.com/example.Counter",
		Key:           "primary",
		HookKinds:     []string{"state", "id", "effect"},
	}
	if base.CompatibleWith(incompatible) {
		t.Fatal("expected reordered hooks to be incompatible")
	}
}

func testSignatureComponent() *Element {
	return nil
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

func TestReportDiagnosticWithContextPreservesPathAndStack(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	ReportDiagnosticWithContext("runtime", DiagnosticWarning, "boundary failure", "App > ErrorBoundary > Child", []string{"App", "ErrorBoundary", "Child"})

	diagnostics := GetDiagnostics()
	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic entry, got %d", len(diagnostics))
	}
	if diagnostics[0].Path != "App > ErrorBoundary > Child" {
		t.Fatalf("expected diagnostic path to round-trip, got %q", diagnostics[0].Path)
	}
	if len(diagnostics[0].ComponentStack) != 3 || diagnostics[0].ComponentStack[1] != "ErrorBoundary" {
		t.Fatalf("expected diagnostic component stack to round-trip, got %+v", diagnostics[0].ComponentStack)
	}
}

func TestReportDiagnosticWritesClassifiedLogEntries(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	ReportDiagnostic("runtime", DiagnosticWarning, "slow commit on App took 4.00ms")
	ReportDiagnostic("router", DiagnosticWarning, "ignoring route redirect loop for /login")
	ReportDiagnostic("runtime", DiagnosticError, "invalid hook usage")

	logs := GetLogs()
	if len(logs) != 3 {
		t.Fatalf("expected 3 log entries, got %d", len(logs))
	}
	if logs[0].Classification != DiagnosticPerformance {
		t.Fatalf("expected performance classification, got %+v", logs[0])
	}
	if logs[1].Classification != DiagnosticUnsupportedRecover {
		t.Fatalf("expected unsupported-recovered classification, got %+v", logs[1])
	}
	if logs[2].Classification != DiagnosticCorrectness || logs[2].Level != LogError {
		t.Fatalf("expected correctness error log, got %+v", logs[2])
	}
}
