package runtime

import (
	"strings"
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

func TestReportDiagnosticWithContextFieldsMirrorsStructuredContext(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	reportDiagnosticWithContextDetails(
		"runtime",
		DiagnosticError,
		"render failed",
		"App > Panel",
		[]string{"App", "Panel"},
		"app/panel.go:42",
		"render work stopped",
		map[string]string{"route": "/dashboard"},
	)

	diagnostics := GetDiagnostics()
	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %+v", diagnostics)
	}
	diagnostic := diagnostics[0]
	if diagnostic.Fields["route"] != "/dashboard" ||
		diagnostic.Fields["path"] != "App > Panel" ||
		diagnostic.Fields["component_stack"] != "App > Panel" ||
		diagnostic.Fields["top_frame"] != "app/panel.go:42" ||
		diagnostic.Fields["runtime"] != "render work stopped" {
		t.Fatalf("expected diagnostic fields to mirror structured context, got %+v", diagnostic)
	}
}

func TestStrictDiagnosticsEscalatesMatchingRecoverableWarning(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	previous := CurrentStrictDiagnosticsOptions()
	ConfigureStrictDiagnostics(StrictDiagnosticsOptions{
		Enabled:         true,
		Codes:           []string{"GWC-HYDRATION-FALLBACK"},
		RecoverableOnly: true,
	})
	defer ClearDiagnostics()
	defer ClearLogs()
	defer ConfigureStrictDiagnostics(previous)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected strict diagnostics escalation panic")
		}
		message, ok := recovered.(string)
		if !ok || !strings.Contains(message, "strict diagnostics escalated GWC-HYDRATION-FALLBACK") {
			t.Fatalf("expected strict diagnostics panic message, got %T %v", recovered, recovered)
		}
	}()

	ReportDiagnostic("runtime", DiagnosticWarning, "hydration fell back to client rendering for <div>: DOM node <span> did not match expected <div>")
}

func TestStrictDiagnosticsIgnoresNonMatchingWarnings(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	previous := CurrentStrictDiagnosticsOptions()
	ConfigureStrictDiagnostics(StrictDiagnosticsOptions{
		Enabled:         true,
		Codes:           []string{"GWC-ROUTER-DUPLICATE-ROUTE"},
		RecoverableOnly: true,
	})
	defer ClearDiagnostics()
	defer ClearLogs()
	defer ConfigureStrictDiagnostics(previous)

	ReportDiagnostic("runtime", DiagnosticWarning, "hydration fell back to client rendering for <div>: DOM node <span> did not match expected <div>")

	diagnostics := GetDiagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Code != "GWC-HYDRATION-FALLBACK" || !diagnostics[0].Recoverable {
		t.Fatalf("expected non-matching strict config to preserve warning diagnostic, got %+v", diagnostics)
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
	if component.Path != "ROOT > testSignatureComponent" {
		t.Fatalf("expected component path to round-trip, got %q", component.Path)
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
		scheduledFiberMarks:              3,
		scheduledGranularMarks:           2,
		commitCount:                      1,
		fineGrainedCommits:               2,
		fineGrainedDescendantHostCommits: 5,
		fineGrainedDescendantTextCommits: 7,
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
	if snapshot.Profiling.FineGrainedDescendantHostCommits != 5 {
		t.Fatalf("expected 5 descendant host commits, got %d", snapshot.Profiling.FineGrainedDescendantHostCommits)
	}
	if snapshot.Profiling.FineGrainedDescendantTextCommits != 7 {
		t.Fatalf("expected 7 descendant text commits, got %d", snapshot.Profiling.FineGrainedDescendantTextCommits)
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

func TestInspectIncludesHydrationDebugSnapshot(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	startedAt := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(42 * time.Millisecond)
	rt := &Runtime{
		lastHydrationMetrics: HydrationMetrics{
			CorrelationID:        "hydrate-42",
			StartedAt:            startedAt,
			FinishedAt:           finishedAt,
			Duration:             42 * time.Millisecond,
			DurationNs:           finishedAt.Sub(startedAt).Nanoseconds(),
			ExistingDOMNodeCount: 12,
			FallbackCount:        2,
			MismatchCount:        3,
			DiscardedNodeCount:   5,
			Strict:               true,
			Failed:               true,
			Failure:              "hydration mismatch forced subtree replacement",
		},
	}

	ReportDiagnostic("runtime", DiagnosticWarning, "hydration text mismatch at App > Hero")
	ReportDiagnostic("runtime", DiagnosticWarning, "hydration fallback at App > Sidebar")
	ReportDiagnostic("runtime", DiagnosticWarning, "unrelated warning")

	snapshot := rt.Inspect()
	if snapshot.Hydration.CorrelationID != "hydrate-42" {
		t.Fatalf("expected hydration correlation id, got %+v", snapshot.Hydration)
	}
	if snapshot.Hydration.StartedAt != "2026-03-25T12:00:00.000Z" || snapshot.Hydration.FinishedAt != "2026-03-25T12:00:00.042Z" {
		t.Fatalf("expected hydration timestamps to be formatted, got %+v", snapshot.Hydration)
	}
	if snapshot.Hydration.DurationNs != int64(42*time.Millisecond) {
		t.Fatalf("expected hydration duration, got %+v", snapshot.Hydration)
	}
	if snapshot.Hydration.ExistingDOMNodeCount != 12 || snapshot.Hydration.FallbackCount != 2 || snapshot.Hydration.MismatchCount != 3 || snapshot.Hydration.DiscardedNodeCount != 5 {
		t.Fatalf("expected hydration counters to round-trip, got %+v", snapshot.Hydration)
	}
	if !snapshot.Hydration.Strict || !snapshot.Hydration.Failed || snapshot.Hydration.Failure == "" {
		t.Fatalf("expected strict failed hydration details, got %+v", snapshot.Hydration)
	}
	if len(snapshot.Hydration.RecentMessages) != 2 {
		t.Fatalf("expected only hydration diagnostics, got %+v", snapshot.Hydration.RecentMessages)
	}
}

func TestInspectHooksIncludesSlotsDependenciesAndEffectStatus(t *testing.T) {
	callback := func() {}
	hooks := &Hooks{
		states:       []interface{}{"draft", "draft"},
		memos:        []memoizedValue{{value: "memoized", deps: []interface{}{"team", 3}}},
		callbacks:    []callbackValue{{fn: callback, deps: []interface{}{"search"}}},
		deps:         [][]interface{}{{"theme", true}},
		cleanups:     []func(){func() {}},
		effectEpochs: []int{4},
	}

	inspected := inspectHooks(hooks)
	if len(inspected) != 4 {
		t.Fatalf("expected 4 hook snapshots, got %+v", inspected)
	}

	var memoHook, callbackHook, effectHook HookSnapshot
	for _, hook := range inspected {
		switch hook.Kind {
		case "memo":
			memoHook = hook
		case "callback":
			callbackHook = hook
		case "effect":
			effectHook = hook
		}
	}
	if memoHook.Slot != 0 || memoHook.Dependencies != `"team", 3` {
		t.Fatalf("expected memo hook slot and deps, got %+v", memoHook)
	}
	if callbackHook.Slot != 0 || callbackHook.Dependencies != `"search"` {
		t.Fatalf("expected callback hook slot and deps, got %+v", callbackHook)
	}
	if effectHook.Slot != 0 || effectHook.Dependencies != `"theme", true` || effectHook.Status != "cleanup=registered epoch=4" {
		t.Fatalf("expected effect hook lifecycle metadata, got %+v", effectHook)
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
		renderDurationNs:  int64(2 * time.Millisecond),
		diffDurationNs:    int64(1 * time.Millisecond),
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
	if branch.SelfDurationNs != int64(9*time.Millisecond) {
		t.Fatalf("expected branch self duration 9ms, got %d", branch.SelfDurationNs)
	}
	if branch.SubtreeDurationNs != int64(10*time.Millisecond) {
		t.Fatalf("expected branch subtree duration 10ms, got %d", branch.SubtreeDurationNs)
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
	if snapshot.Profiling.HotBranches[0].SubtreeDurationNs != int64(10*time.Millisecond) {
		t.Fatalf("expected hot branch subtree duration 10ms, got %d", snapshot.Profiling.HotBranches[0].SubtreeDurationNs)
	}
	if snapshot.Profiling.HotBranches[0].RenderDurationNs != int64(2*time.Millisecond) || snapshot.Profiling.HotBranches[0].DiffDurationNs != int64(1*time.Millisecond) {
		t.Fatalf("expected hot branch render/diff attribution, got %+v", snapshot.Profiling.HotBranches[0])
	}
	if len(snapshot.Profiling.FlamegraphFrames) == 0 {
		t.Fatal("expected flamegraph frames to be collected")
	}
	frame := snapshot.Profiling.FlamegraphFrames[0]
	if frame.Name != "section" || frame.Depth != 0 || frame.DurationNs != int64(10*time.Millisecond) {
		t.Fatalf("expected first flamegraph frame to map section timings, got %+v", frame)
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

func TestReportDiagnosticAddsStableMetadataForKnownFailures(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	ReportDiagnostic("runtime", DiagnosticWarning, "hydration text mismatch for p: server \"Server\" client \"Client\"")
	ReportLogWithFields("router", LogError, DiagnosticCorrectness, "route loader failed", "", nil)

	diagnostics := GetDiagnostics()
	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %d", len(diagnostics))
	}
	if diagnostics[0].Code != "GWC-HYDRATION-TEXT-MISMATCH" {
		t.Fatalf("expected hydration code, got %+v", diagnostics[0])
	}
	if diagnostics[0].Docs == "" || diagnostics[0].Remediation == "" {
		t.Fatalf("expected hydration diagnostic docs and remediation, got %+v", diagnostics[0])
	}
	if !diagnostics[0].Recoverable {
		t.Fatalf("expected hydration warning to be marked recoverable, got %+v", diagnostics[0])
	}

	logs := GetLogs()
	if len(logs) == 0 {
		t.Fatal("expected log entry")
	}
	last := logs[len(logs)-1]
	if last.Code != "GWC-ROUTER-LOADER-FAILED" {
		t.Fatalf("expected loader log code, got %+v", last)
	}
	if last.Docs == "" || last.Remediation == "" || last.Recoverable {
		t.Fatalf("expected loader log metadata, got %+v", last)
	}
}

func TestRuntimeInspectCapturesExtendedProfilingSurface(t *testing.T) {
	rt := &Runtime{
		currentRoot: &Fiber{typeOf: "ROOT"},
	}
	rt.profiling.totalRenderDurationNs = int64(11 * time.Millisecond)
	rt.profiling.totalDiffDurationNs = int64(7 * time.Millisecond)
	rt.profiling.totalCommitDurationNs = int64(5 * time.Millisecond)
	rt.profiling.totalEffectDurationNs = int64(3 * time.Millisecond)
	rt.profiling.totalCleanupDurationNs = int64(2 * time.Millisecond)
	rt.profiling.startupMode = "hydrate"
	rt.profiling.startupStartedAt = time.Date(2026, 3, 24, 15, 4, 5, 0, time.UTC)
	rt.profiling.bootstrapReadDurationNs = int64(4 * time.Millisecond)
	rt.profiling.hydrationDurationNs = int64(8 * time.Millisecond)
	rt.profiling.startupCommitDurationNs = int64(3 * time.Millisecond)
	rt.profiling.firstInteractionDurationNs = int64(21 * time.Millisecond)
	rt.profiling.firstInteractionCaptured = true
	rt.profiling.firstInteractionEvent = "event"
	rt.RecordProfilingEvent(ProfilingEvent{
		Domain:     "router",
		Name:       "navigation",
		Phase:      "start",
		Target:     "/reports",
		DurationNs: int64(1 * time.Millisecond),
		Fields: map[string]string{
			"mode": "push",
		},
	})

	snapshot := rt.Inspect()
	if snapshot.Profiling.PhaseTotals.RenderDurationNs != int64(11*time.Millisecond) {
		t.Fatalf("expected render phase total to round-trip, got %+v", snapshot.Profiling.PhaseTotals)
	}
	if snapshot.Profiling.PhaseTotals.DiffDurationNs != int64(7*time.Millisecond) {
		t.Fatalf("expected diff phase total to round-trip, got %+v", snapshot.Profiling.PhaseTotals)
	}
	if len(snapshot.Profiling.RecentEvents) != 1 {
		t.Fatalf("expected one profiling event, got %+v", snapshot.Profiling.RecentEvents)
	}
	event := snapshot.Profiling.RecentEvents[0]
	if event.Domain != "router" || event.Name != "navigation" || event.Phase != "start" || event.Target != "/reports" {
		t.Fatalf("unexpected profiling event payload: %+v", event)
	}
	if event.Timestamp == "" {
		t.Fatalf("expected profiling event timestamp, got %+v", event)
	}
	if event.Fields["mode"] != "push" {
		t.Fatalf("expected profiling event fields to round-trip, got %+v", event.Fields)
	}
	if snapshot.Profiling.Startup.Mode != "hydrate" || snapshot.Profiling.Startup.BootstrapReadDurationNs != int64(4*time.Millisecond) || !snapshot.Profiling.Startup.FirstInteractionCaptured {
		t.Fatalf("expected startup profiling to round-trip, got %+v", snapshot.Profiling.Startup)
	}
}

func TestRuntimeInspectCapturesPerComponentRenderTracing(t *testing.T) {
	rt := &Runtime{}
	root := &Fiber{typeOf: "ROOT"}
	componentFn := func() *Element {
		return nil
	}
	initial := &Fiber{
		typeOf: componentFn,
		parent: root,
	}
	root.child = initial

	if _, handled, _ := rt.renderFunctionComponent(initial); handled {
		t.Fatal("expected initial component render not to panic")
	}

	rerender := &Fiber{
		typeOf:       componentFn,
		parent:       root,
		alternate:    initial,
		updateOrigin: "local-state",
	}
	root.child = rerender
	rt.currentRoot = root

	if _, handled, _ := rt.renderFunctionComponent(rerender); handled {
		t.Fatal("expected rerender component render not to panic")
	}

	snapshot := rt.Inspect()
	if len(snapshot.Profiling.ComponentRenders) != 1 {
		t.Fatalf("expected one component trace entry, got %+v", snapshot.Profiling.ComponentRenders)
	}
	trace := snapshot.Profiling.ComponentRenders[0]
	if trace.RenderCount != 2 || trace.RerenderCount != 1 {
		t.Fatalf("expected render/rerender counters to round-trip, got %+v", trace)
	}
	if trace.LastTrigger != "local-state" || trace.TriggerCounts["mount"] != 1 || trace.TriggerCounts["local-state"] != 1 {
		t.Fatalf("expected trigger attribution to round-trip, got %+v", trace)
	}
	if trace.TotalRenderDurationNs < 0 || trace.AverageRenderDurationNs < 0 {
		t.Fatalf("expected non-negative render durations, got %+v", trace)
	}
}

func TestRecordFirstInteractionCapturesStartupLatency(t *testing.T) {
	rt := &Runtime{}
	rt.profiling.startupStartedAt = time.Now().Add(-25 * time.Millisecond)
	rt.profiling.startupMode = "render"

	rt.recordFirstInteraction("event")
	if !rt.profiling.firstInteractionCaptured {
		t.Fatal("expected first interaction to be captured")
	}
	if rt.profiling.firstInteractionDurationNs <= 0 {
		t.Fatalf("expected positive first interaction duration, got %d", rt.profiling.firstInteractionDurationNs)
	}
	if rt.profiling.firstInteractionEvent != "event" {
		t.Fatalf("expected first interaction event tag, got %q", rt.profiling.firstInteractionEvent)
	}
	if len(rt.profiling.events) == 0 {
		t.Fatal("expected first interaction profiling event")
	}
	last := rt.profiling.events[len(rt.profiling.events)-1]
	if last.Name != "startup.first_interaction" || last.Phase != "finish" {
		t.Fatalf("expected startup first interaction profiling event, got %+v", last)
	}
}
