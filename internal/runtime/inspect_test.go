package runtime

import (
	"strings"
	"testing"
	"time"
)

func TestDiagnosticsDeduplicateBySourceSeverityAndMessage(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	ReportDiagnostic("router", DiagnosticWarning, "duplicate route registration")
	ReportDiagnostic("router", DiagnosticWarning, "duplicate route registration")
	ReportDiagnostic("runtime", DiagnosticError, "invalid hook usage")
	ReportDiagnosticWithContext("runtime", DiagnosticWarning, "duplicate route registration", "App > Boundary", []string{"App", "Boundary"})

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) != 3 {
		parseT.Fatalf("expected 3 unique diagnostics, got %d", len(parseDiagnostics))
	}
	if parseDiagnostics[0].Count != 2 {
		parseT.Fatalf("expected duplicate diagnostic count 2, got %d", parseDiagnostics[0].Count)
	}
}

func TestReportDiagnosticWithContextFieldsMirrorsStructuredContext(parseT *testing.T) {
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

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) != 1 {
		parseT.Fatalf("expected one diagnostic, got %+v", parseDiagnostics)
	}
	parseDiagnostic := parseDiagnostics[0]
	if parseDiagnostic.Fields["route"] != "/dashboard" ||
		parseDiagnostic.Fields["path"] != "App > Panel" ||
		parseDiagnostic.Fields["component_stack"] != "App > Panel" ||
		parseDiagnostic.Fields["top_frame"] != "app/panel.go:42" ||
		parseDiagnostic.Fields["runtime"] != "render work stopped" {
		parseT.Fatalf("expected diagnostic fields to mirror structured context, got %+v", parseDiagnostic)
	}
}

func TestStrictDiagnosticsEscalatesMatchingRecoverableWarning(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	parsePrevious := CurrentStrictDiagnosticsOptions()
	ConfigureStrictDiagnostics(StrictDiagnosticsOptions{
		Enabled:         true,
		Codes:           []string{"GWC-HYDRATION-FALLBACK"},
		RecoverableOnly: true,
	})
	defer ClearDiagnostics()
	defer ClearLogs()
	defer ConfigureStrictDiagnostics(parsePrevious)

	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected strict diagnostics escalation panic")
		}
		parseMessage, parseOk := parseRecovered.(string)
		if !parseOk || !strings.Contains(parseMessage, "strict diagnostics escalated GWC-HYDRATION-FALLBACK") {
			parseT.Fatalf("expected strict diagnostics panic message, got %T %v", parseRecovered, parseRecovered)
		}
	}()

	ReportDiagnostic("runtime", DiagnosticWarning, "hydration fell back to client rendering for <div>: DOM node <span> did not match expected <div>")
}

func TestStrictDiagnosticsIgnoresNonMatchingWarnings(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	parsePrevious := CurrentStrictDiagnosticsOptions()
	ConfigureStrictDiagnostics(StrictDiagnosticsOptions{
		Enabled:         true,
		Codes:           []string{"GWC-ROUTER-DUPLICATE-ROUTE"},
		RecoverableOnly: true,
	})
	defer ClearDiagnostics()
	defer ClearLogs()
	defer ConfigureStrictDiagnostics(parsePrevious)

	ReportDiagnostic("runtime", DiagnosticWarning, "hydration fell back to client rendering for <div>: DOM node <span> did not match expected <div>")

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) != 1 || parseDiagnostics[0].Code != "GWC-HYDRATION-FALLBACK" || !parseDiagnostics[0].Recoverable {
		parseT.Fatalf("expected non-matching strict config to preserve warning diagnostic, got %+v", parseDiagnostics)
	}
}

func TestRuntimeInspectCapturesTreeStatsAndHooks(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseRt := &Runtime{}
	parseRt.profiling = runtimeProfiling{
		renderCalls:          3,
		scheduledRootUpdates: 2,
		scheduledFiberMarks:  5,
		workLoopPasses:       4,
		processedUnits:       9,
		commitCount:          2,
		lastRenderDurationNs: 1_500_000,
		lastCommitDurationNs: 900_000,
	}
	parseChildHooks := &Hooks{
		states:    []interface{}{42, 42},
		memos:     []memoizedValue{{value: "memoized"}},
		ids:       []string{"node-1"},
		signature: []string{"state", "memo", "id"},
	}
	parseChild := &Fiber{
		typeOf:  testSignatureComponent,
		props:   map[string]interface{}{"key": "hero"},
		dirty:   true,
		hooks:   parseChildHooks,
		effects: []Effect{{}},
	}
	parseRoot := &Fiber{
		typeOf: "ROOT",
		child:  parseChild,
	}
	parseChild.parent = parseRoot
	parseChildHooks.owner = parseChild
	parseRt.currentRoot = parseRoot

	ReportDiagnostic("runtime", DiagnosticWarning, "test warning")

	parseSnapshot := parseRt.Inspect()
	if parseSnapshot.Root == nil {
		parseT.Fatal("expected root snapshot")
	}
	if parseSnapshot.Stats.TotalFibers != 2 {
		parseT.Fatalf("expected 2 fibers, got %d", parseSnapshot.Stats.TotalFibers)
	}
	if parseSnapshot.Stats.ComponentFibers != 1 {
		parseT.Fatalf("expected 1 component fiber, got %d", parseSnapshot.Stats.ComponentFibers)
	}
	if parseSnapshot.Stats.DirtyFibers != 1 {
		parseT.Fatalf("expected 1 dirty fiber, got %d", parseSnapshot.Stats.DirtyFibers)
	}
	if len(parseSnapshot.Diagnostics) != 1 {
		parseT.Fatalf("expected 1 diagnostic entry, got %d", len(parseSnapshot.Diagnostics))
	}
	if parseSnapshot.Diagnostics[0].ComponentStack != nil {
		parseT.Fatalf("expected plain diagnostic to have no component stack, got %+v", parseSnapshot.Diagnostics[0])
	}
	if parseSnapshot.Profiling.RenderCalls != 3 || parseSnapshot.Profiling.CommitCount != 2 {
		parseT.Fatalf("expected profiling snapshot to preserve runtime counters, got %+v", parseSnapshot.Profiling)
	}
	if len(parseSnapshot.Root.Children) != 1 {
		parseT.Fatalf("expected root child snapshot, got %d children", len(parseSnapshot.Root.Children))
	}
	parseComponent := parseSnapshot.Root.Children[0]
	if parseComponent.Kind != "component" {
		parseT.Fatalf("expected component child kind, got %q", parseComponent.Kind)
	}
	if parseComponent.Path != "ROOT > testSignatureComponent" {
		parseT.Fatalf("expected component path to round-trip, got %q", parseComponent.Path)
	}
	if parseComponent.HookCount < 3 {
		parseT.Fatalf("expected hook inspection entries, got %d", parseComponent.HookCount)
	}
	if parseComponent.Signature == nil {
		parseT.Fatal("expected component signature")
	}
	if parseComponent.Signature.Name != "testSignatureComponent" {
		parseT.Fatalf("expected component name to round-trip, got %q", parseComponent.Signature.Name)
	}
	if parseComponent.Signature.Key != "hero" {
		parseT.Fatalf("expected component key to round-trip, got %q", parseComponent.Signature.Key)
	}
	if len(parseComponent.Signature.HookKinds) != 3 || parseComponent.Signature.HookKinds[0] != "state" || parseComponent.Signature.HookKinds[2] != "id" {
		parseT.Fatalf("expected hook order to be preserved, got %+v", parseComponent.Signature.HookKinds)
	}
	if !parseComponent.Signature.CompatibleWith(*parseComponent.Signature) {
		parseT.Fatal("expected signature to be compatible with itself")
	}
}

func TestRuntimeInspectCapturesFineGrainedMetadata(parseT *testing.T) {
	parseRt := &Runtime{}
	parseRt.profiling = runtimeProfiling{
		scheduledFiberMarks:              3,
		scheduledGranularMarks:           2,
		commitCount:                      1,
		fineGrainedCommits:               2,
		fineGrainedDescendantHostCommits: 5,
		fineGrainedDescendantTextCommits: 7,
	}
	parseReactive := &Fiber{
		typeOf:         ReactiveTextNodeType,
		fineGrained:    true,
		reactiveAtomID: "count",
		updateOrigin:   "fine-grained",
	}
	parseRoot := &Fiber{typeOf: "ROOT", child: parseReactive}
	parseReactive.parent = parseRoot
	parseRt.currentRoot = parseRoot

	parseSnapshot := parseRt.Inspect()
	if parseSnapshot.Stats.FineGrainedFibers != 1 {
		parseT.Fatalf("expected 1 fine-grained fiber, got %d", parseSnapshot.Stats.FineGrainedFibers)
	}
	if parseSnapshot.Profiling.ScheduledGranularMarks != 2 {
		parseT.Fatalf("expected 2 granular marks, got %d", parseSnapshot.Profiling.ScheduledGranularMarks)
	}
	if parseSnapshot.Profiling.FineGrainedCommits != 2 {
		parseT.Fatalf("expected 2 fine-grained commits, got %d", parseSnapshot.Profiling.FineGrainedCommits)
	}
	if parseSnapshot.Profiling.FineGrainedDescendantHostCommits != 5 {
		parseT.Fatalf("expected 5 descendant host commits, got %d", parseSnapshot.Profiling.FineGrainedDescendantHostCommits)
	}
	if parseSnapshot.Profiling.FineGrainedDescendantTextCommits != 7 {
		parseT.Fatalf("expected 7 descendant text commits, got %d", parseSnapshot.Profiling.FineGrainedDescendantTextCommits)
	}
	if len(parseSnapshot.Root.Children) != 1 {
		parseT.Fatalf("expected one child snapshot, got %d", len(parseSnapshot.Root.Children))
	}
	parseNode := parseSnapshot.Root.Children[0]
	if !parseNode.FineGrained {
		parseT.Fatal("expected inspected node to be marked fine-grained")
	}
	if parseNode.ReactiveSource != "count" {
		parseT.Fatalf("expected reactive source to round-trip, got %q", parseNode.ReactiveSource)
	}
	if parseNode.UpdateOrigin != "fine-grained" {
		parseT.Fatalf("expected update origin to round-trip, got %q", parseNode.UpdateOrigin)
	}
}

func TestInspectIncludesHydrationDebugSnapshot(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseStartedAt := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	parseFinishedAt := parseStartedAt.Add(42 * time.Millisecond)
	parseRt := &Runtime{
		lastHydrationMetrics: HydrationMetrics{
			CorrelationID:        "hydrate-42",
			StartedAt:            parseStartedAt,
			FinishedAt:           parseFinishedAt,
			Duration:             42 * time.Millisecond,
			DurationNs:           parseFinishedAt.Sub(parseStartedAt).Nanoseconds(),
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

	parseSnapshot := parseRt.Inspect()
	if parseSnapshot.Hydration.CorrelationID != "hydrate-42" {
		parseT.Fatalf("expected hydration correlation id, got %+v", parseSnapshot.Hydration)
	}
	if parseSnapshot.Hydration.StartedAt != "2026-03-25T12:00:00.000Z" || parseSnapshot.Hydration.FinishedAt != "2026-03-25T12:00:00.042Z" {
		parseT.Fatalf("expected hydration timestamps to be formatted, got %+v", parseSnapshot.Hydration)
	}
	if parseSnapshot.Hydration.DurationNs != int64(42*time.Millisecond) {
		parseT.Fatalf("expected hydration duration, got %+v", parseSnapshot.Hydration)
	}
	if parseSnapshot.Hydration.ExistingDOMNodeCount != 12 || parseSnapshot.Hydration.FallbackCount != 2 || parseSnapshot.Hydration.MismatchCount != 3 || parseSnapshot.Hydration.DiscardedNodeCount != 5 {
		parseT.Fatalf("expected hydration counters to round-trip, got %+v", parseSnapshot.Hydration)
	}
	if !parseSnapshot.Hydration.Strict || !parseSnapshot.Hydration.Failed || parseSnapshot.Hydration.Failure == "" {
		parseT.Fatalf("expected strict failed hydration details, got %+v", parseSnapshot.Hydration)
	}
	if len(parseSnapshot.Hydration.RecentMessages) != 2 {
		parseT.Fatalf("expected only hydration diagnostics, got %+v", parseSnapshot.Hydration.RecentMessages)
	}
}

func TestInspectHooksIncludesSlotsDependenciesAndEffectStatus(parseT *testing.T) {
	parseCallback := func() {}
	parseHooks := &Hooks{
		states:       []interface{}{"draft", "draft"},
		memos:        []memoizedValue{{value: "memoized", deps: []interface{}{"team", 3}}},
		callbacks:    []callbackValue{{fn: parseCallback, deps: []interface{}{"search"}}},
		deps:         [][]interface{}{{"theme", true}},
		cleanups:     []func(){func() {}},
		effectEpochs: []int{4},
	}

	parseInspected := inspectHooks(parseHooks)
	if len(parseInspected) != 4 {
		parseT.Fatalf("expected 4 hook snapshots, got %+v", parseInspected)
	}

	var parseMemoHook, parseCallbackHook, parseEffectHook HookSnapshot
	for _, parseHook := range parseInspected {
		switch parseHook.Kind {
		case "memo":
			parseMemoHook = parseHook
		case "callback":
			parseCallbackHook = parseHook
		case "effect":
			parseEffectHook = parseHook
		}
	}
	if parseMemoHook.Slot != 0 || parseMemoHook.Dependencies != `"team", 3` {
		parseT.Fatalf("expected memo hook slot and deps, got %+v", parseMemoHook)
	}
	if parseCallbackHook.Slot != 0 || parseCallbackHook.Dependencies != `"search"` {
		parseT.Fatalf("expected callback hook slot and deps, got %+v", parseCallbackHook)
	}
	if parseEffectHook.Slot != 0 || parseEffectHook.Dependencies != `"theme", true` || parseEffectHook.Status != "cleanup=registered epoch=4" {
		parseT.Fatalf("expected effect hook lifecycle metadata, got %+v", parseEffectHook)
	}
}

func TestComponentSignatureCompatibilityUsesIdentityAndHookOrder(parseT *testing.T) {
	parseBase := ComponentSignature{
		Kind:          "component",
		Name:          "Counter",
		QualifiedName: "github.com/example.Counter",
		Key:           "primary",
		HookKinds:     []string{"state", "effect", "id"},
	}

	parseCompatible := ComponentSignature{
		Kind:          "component",
		Name:          "Counter",
		QualifiedName: "github.com/example.Counter",
		Key:           "primary",
		HookKinds:     []string{"state", "effect", "id"},
	}
	if !parseBase.CompatibleWith(parseCompatible) {
		parseT.Fatal("expected matching signature to be compatible")
	}

	parseIncompatible := ComponentSignature{
		Kind:          "component",
		Name:          "Counter",
		QualifiedName: "github.com/example.Counter",
		Key:           "primary",
		HookKinds:     []string{"state", "id", "effect"},
	}
	if parseBase.CompatibleWith(parseIncompatible) {
		parseT.Fatal("expected reordered hooks to be incompatible")
	}
}

func testSignatureComponent() *Element {
	return nil
}

func TestRuntimeInspectCollectsHotBranchesAndTiming(parseT *testing.T) {
	parseRt := &Runtime{}
	parseGrandchild := &Fiber{typeOf: "span", commitDurationNs: int64(1 * time.Millisecond)}
	parseChild := &Fiber{
		typeOf:            "section",
		renderDurationNs:  int64(2 * time.Millisecond),
		diffDurationNs:    int64(1 * time.Millisecond),
		commitDurationNs:  int64(2 * time.Millisecond),
		effectDurationNs:  int64(3 * time.Millisecond),
		cleanupDurationNs: int64(1 * time.Millisecond),
		child:             parseGrandchild,
	}
	parseRoot := &Fiber{typeOf: "ROOT", child: parseChild}
	parseChild.parent = parseRoot
	parseGrandchild.parent = parseChild
	parseRt.currentRoot = parseRoot
	parseRt.profiling = runtimeProfiling{
		effectExecutions:      4,
		cleanupExecutions:     2,
		lastEffectDurationNs:  int64(3 * time.Millisecond),
		lastCleanupDurationNs: int64(1 * time.Millisecond),
	}

	parseSnapshot := parseRt.Inspect()
	if parseSnapshot.Root == nil || len(parseSnapshot.Root.Children) != 1 {
		parseT.Fatal("expected inspected tree with one child")
	}
	parseBranch := parseSnapshot.Root.Children[0]
	if parseBranch.SelfDurationNs != int64(9*time.Millisecond) {
		parseT.Fatalf("expected branch self duration 9ms, got %d", parseBranch.SelfDurationNs)
	}
	if parseBranch.SubtreeDurationNs != int64(10*time.Millisecond) {
		parseT.Fatalf("expected branch subtree duration 10ms, got %d", parseBranch.SubtreeDurationNs)
	}
	if parseSnapshot.Profiling.EffectExecutions != 4 || parseSnapshot.Profiling.CleanupExecutions != 2 {
		parseT.Fatalf("expected effect/cleanup counters in profiling snapshot, got %+v", parseSnapshot.Profiling)
	}
	if len(parseSnapshot.Profiling.HotBranches) == 0 {
		parseT.Fatal("expected hot branches to be collected")
	}
	if parseSnapshot.Profiling.HotBranches[0].Name != "section" {
		parseT.Fatalf("expected hottest branch to be section, got %q", parseSnapshot.Profiling.HotBranches[0].Name)
	}
	if parseSnapshot.Profiling.HotBranches[0].SubtreeDurationNs != int64(10*time.Millisecond) {
		parseT.Fatalf("expected hot branch subtree duration 10ms, got %d", parseSnapshot.Profiling.HotBranches[0].SubtreeDurationNs)
	}
	if parseSnapshot.Profiling.HotBranches[0].RenderDurationNs != int64(2*time.Millisecond) || parseSnapshot.Profiling.HotBranches[0].DiffDurationNs != int64(1*time.Millisecond) {
		parseT.Fatalf("expected hot branch render/diff attribution, got %+v", parseSnapshot.Profiling.HotBranches[0])
	}
	if len(parseSnapshot.Profiling.FlamegraphFrames) == 0 {
		parseT.Fatal("expected flamegraph frames to be collected")
	}
	parseFrame := parseSnapshot.Profiling.FlamegraphFrames[0]
	if parseFrame.Name != "section" || parseFrame.Depth != 0 || parseFrame.DurationNs != int64(10*time.Millisecond) {
		parseT.Fatalf("expected first flamegraph frame to map section timings, got %+v", parseFrame)
	}
}

func TestReportDiagnosticWithContextPreservesPathAndStack(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	ReportDiagnosticWithContext("runtime", DiagnosticWarning, "boundary failure", "App > ErrorBoundary > Child", []string{"App", "ErrorBoundary", "Child"})

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) != 1 {
		parseT.Fatalf("expected one diagnostic entry, got %d", len(parseDiagnostics))
	}
	if parseDiagnostics[0].Path != "App > ErrorBoundary > Child" {
		parseT.Fatalf("expected diagnostic path to round-trip, got %q", parseDiagnostics[0].Path)
	}
	if len(parseDiagnostics[0].ComponentStack) != 3 || parseDiagnostics[0].ComponentStack[1] != "ErrorBoundary" {
		parseT.Fatalf("expected diagnostic component stack to round-trip, got %+v", parseDiagnostics[0].ComponentStack)
	}
}

func TestReportDiagnosticWritesClassifiedLogEntries(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	ReportDiagnostic("runtime", DiagnosticWarning, "slow commit on App took 4.00ms")
	ReportDiagnostic("router", DiagnosticWarning, "ignoring route redirect loop for /login")
	ReportDiagnostic("runtime", DiagnosticError, "invalid hook usage")

	parseLogs := GetLogs()
	if len(parseLogs) != 3 {
		parseT.Fatalf("expected 3 log entries, got %d", len(parseLogs))
	}
	if parseLogs[0].Classification != DiagnosticPerformance {
		parseT.Fatalf("expected performance classification, got %+v", parseLogs[0])
	}
	if parseLogs[1].Classification != DiagnosticUnsupportedRecover {
		parseT.Fatalf("expected unsupported-recovered classification, got %+v", parseLogs[1])
	}
	if parseLogs[2].Classification != DiagnosticCorrectness || parseLogs[2].Level != LogError {
		parseT.Fatalf("expected correctness error log, got %+v", parseLogs[2])
	}
}

func TestReportDiagnosticAddsStableMetadataForKnownFailures(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	ReportDiagnostic("runtime", DiagnosticWarning, "hydration text mismatch for p: server \"Server\" client \"Client\"")
	ReportLogWithFields("router", LogError, DiagnosticCorrectness, "route loader failed", "", nil)

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) != 1 {
		parseT.Fatalf("expected one diagnostic, got %d", len(parseDiagnostics))
	}
	if parseDiagnostics[0].Code != "GWC-HYDRATION-TEXT-MISMATCH" {
		parseT.Fatalf("expected hydration code, got %+v", parseDiagnostics[0])
	}
	if parseDiagnostics[0].Docs == "" || parseDiagnostics[0].Remediation == "" {
		parseT.Fatalf("expected hydration diagnostic docs and remediation, got %+v", parseDiagnostics[0])
	}
	if !parseDiagnostics[0].Recoverable {
		parseT.Fatalf("expected hydration warning to be marked recoverable, got %+v", parseDiagnostics[0])
	}

	parseLogs := GetLogs()
	if len(parseLogs) == 0 {
		parseT.Fatal("expected log entry")
	}
	parseLast := parseLogs[len(parseLogs)-1]
	if parseLast.Code != "GWC-ROUTER-LOADER-FAILED" {
		parseT.Fatalf("expected loader log code, got %+v", parseLast)
	}
	if parseLast.Docs == "" || parseLast.Remediation == "" || parseLast.Recoverable {
		parseT.Fatalf("expected loader log metadata, got %+v", parseLast)
	}
}

func TestRuntimeInspectCapturesExtendedProfilingSurface(parseT *testing.T) {
	parseRt := &Runtime{
		currentRoot: &Fiber{typeOf: "ROOT"},
	}
	parseRt.profiling.totalRenderDurationNs = int64(11 * time.Millisecond)
	parseRt.profiling.totalDiffDurationNs = int64(7 * time.Millisecond)
	parseRt.profiling.totalCommitDurationNs = int64(5 * time.Millisecond)
	parseRt.profiling.totalEffectDurationNs = int64(3 * time.Millisecond)
	parseRt.profiling.totalCleanupDurationNs = int64(2 * time.Millisecond)
	parseRt.profiling.startupMode = "hydrate"
	parseRt.profiling.startupStartedAt = time.Date(2026, 3, 24, 15, 4, 5, 0, time.UTC)
	parseRt.profiling.bootstrapReadDurationNs = int64(4 * time.Millisecond)
	parseRt.profiling.hydrationDurationNs = int64(8 * time.Millisecond)
	parseRt.profiling.startupCommitDurationNs = int64(3 * time.Millisecond)
	parseRt.profiling.firstInteractionDurationNs = int64(21 * time.Millisecond)
	parseRt.profiling.firstInteractionCaptured = true
	parseRt.profiling.firstInteractionEvent = "event"
	parseRt.profiling.routeStartupBudgets = map[string]*routeStartupBudget{
		"/reports/*": {
			RouteFamily:                     "/reports/*",
			LastRoutePath:                   "/reports/7",
			SampleCount:                     2,
			BootstrapReadDurationTotalNs:    int64(6 * time.Millisecond),
			HydrationDurationTotalNs:        int64(10 * time.Millisecond),
			StartupCommitDurationTotalNs:    int64(8 * time.Millisecond),
			FirstInteractionDurationTotalNs: int64(40 * time.Millisecond),
		},
	}
	parseRt.RecordProfilingEvent(ProfilingEvent{
		Domain:     "router",
		Name:       "navigation",
		Phase:      "start",
		Target:     "/reports",
		DurationNs: int64(1 * time.Millisecond),
		Fields: map[string]string{
			"mode": "push",
		},
	})

	parseSnapshot := parseRt.Inspect()
	if parseSnapshot.Profiling.PhaseTotals.RenderDurationNs != int64(11*time.Millisecond) {
		parseT.Fatalf("expected render phase total to round-trip, got %+v", parseSnapshot.Profiling.PhaseTotals)
	}
	if parseSnapshot.Profiling.PhaseTotals.DiffDurationNs != int64(7*time.Millisecond) {
		parseT.Fatalf("expected diff phase total to round-trip, got %+v", parseSnapshot.Profiling.PhaseTotals)
	}
	if len(parseSnapshot.Profiling.RecentEvents) != 1 {
		parseT.Fatalf("expected one profiling event, got %+v", parseSnapshot.Profiling.RecentEvents)
	}
	parseEvent := parseSnapshot.Profiling.RecentEvents[0]
	if parseEvent.Domain != "router" || parseEvent.Name != "navigation" || parseEvent.Phase != "start" || parseEvent.Target != "/reports" {
		parseT.Fatalf("unexpected profiling event payload: %+v", parseEvent)
	}
	if parseEvent.Timestamp == "" {
		parseT.Fatalf("expected profiling event timestamp, got %+v", parseEvent)
	}
	if parseEvent.Fields["mode"] != "push" {
		parseT.Fatalf("expected profiling event fields to round-trip, got %+v", parseEvent.Fields)
	}
	if parseSnapshot.Profiling.Startup.Mode != "hydrate" || parseSnapshot.Profiling.Startup.BootstrapReadDurationNs != int64(4*time.Millisecond) || !parseSnapshot.Profiling.Startup.FirstInteractionCaptured {
		parseT.Fatalf("expected startup profiling to round-trip, got %+v", parseSnapshot.Profiling.Startup)
	}
	if len(parseSnapshot.Profiling.Startup.RouteBudgets) != 1 || parseSnapshot.Profiling.Startup.RouteBudgets[0].RouteFamily != "/reports/*" || parseSnapshot.Profiling.Startup.RouteBudgets[0].AverageFirstInteractionDurationNs != int64(20*time.Millisecond) {
		parseT.Fatalf("expected route startup budgets to round-trip, got %+v", parseSnapshot.Profiling.Startup.RouteBudgets)
	}
}

func TestRuntimeInspectCapturesPerComponentRenderTracing(parseT *testing.T) {
	parseRt := &Runtime{}
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseComponentFn := func() *Element {
		return nil
	}
	parseInitial := &Fiber{
		typeOf: parseComponentFn,
		parent: parseRoot,
	}
	parseRoot.child = parseInitial

	if _, parseHandled, _ := parseRt.renderFunctionComponent(parseInitial); parseHandled {
		parseT.Fatal("expected initial component render not to panic")
	}

	parseRerender := &Fiber{
		typeOf:       parseComponentFn,
		parent:       parseRoot,
		alternate:    parseInitial,
		updateOrigin: "local-state",
	}
	parseRoot.child = parseRerender
	parseRt.currentRoot = parseRoot

	if _, parseHandled2, _ := parseRt.renderFunctionComponent(parseRerender); parseHandled2 {
		parseT.Fatal("expected rerender component render not to panic")
	}

	parseSnapshot := parseRt.Inspect()
	if len(parseSnapshot.Profiling.ComponentRenders) != 1 {
		parseT.Fatalf("expected one component trace entry, got %+v", parseSnapshot.Profiling.ComponentRenders)
	}
	parseTrace := parseSnapshot.Profiling.ComponentRenders[0]
	if parseTrace.RenderCount != 2 || parseTrace.RerenderCount != 1 {
		parseT.Fatalf("expected render/rerender counters to round-trip, got %+v", parseTrace)
	}
	if parseTrace.LastTrigger != "local-state" || parseTrace.TriggerCounts["mount"] != 1 || parseTrace.TriggerCounts["local-state"] != 1 {
		parseT.Fatalf("expected trigger attribution to round-trip, got %+v", parseTrace)
	}
	if parseTrace.TotalRenderDurationNs < 0 || parseTrace.AverageRenderDurationNs < 0 {
		parseT.Fatalf("expected non-negative render durations, got %+v", parseTrace)
	}
}

func TestRecordFirstInteractionCapturesStartupLatency(parseT *testing.T) {
	parseRt := &Runtime{}
	parseRt.profiling.startupStartedAt = time.Now().Add(-25 * time.Millisecond)
	parseRt.profiling.startupMode = "render"

	parseRt.recordFirstInteraction("event")
	if !parseRt.profiling.firstInteractionCaptured {
		parseT.Fatal("expected first interaction to be captured")
	}
	if parseRt.profiling.firstInteractionDurationNs <= 0 {
		parseT.Fatalf("expected positive first interaction duration, got %d", parseRt.profiling.firstInteractionDurationNs)
	}
	if parseRt.profiling.firstInteractionEvent != "event" {
		parseT.Fatalf("expected first interaction event tag, got %q", parseRt.profiling.firstInteractionEvent)
	}
	if len(parseRt.profiling.events) == 0 {
		parseT.Fatal("expected first interaction profiling event")
	}
	parseLast := parseRt.profiling.events[len(parseRt.profiling.events)-1]
	if parseLast.Name != "startup.first_interaction" || parseLast.Phase != "finish" {
		parseT.Fatalf("expected startup first interaction profiling event, got %+v", parseLast)
	}
}
