package runtime

import (
	"reflect"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
)

func TestPluginInterposerMapsRuntimeSnapshotDefensively(t *testing.T) {
	source := &FiberSnapshot{
		Name:              "Dashboard",
		Path:              "App > Dashboard",
		Kind:              "component",
		Dirty:             true,
		NeedsUpdate:       true,
		FineGrained:       true,
		ReactiveSource:    "atom:count",
		UpdateOrigin:      "signal",
		EffectCount:       2,
		HookCount:         1,
		Signature:         &ComponentSignature{Name: "Dashboard", Key: "main", HookKinds: []string{"state", "effect"}},
		RenderDurationNs:  10,
		DiffDurationNs:    11,
		CommitDurationNs:  12,
		EffectDurationNs:  13,
		CleanupDurationNs: 14,
		SelfDurationNs:    60,
		SubtreeDurationNs: 90,
		Hooks: []HookSnapshot{{
			Slot:         1,
			Kind:         "state",
			Value:        "42",
			Dependencies: "[]",
			Status:       "stable",
		}},
		Children: []FiberSnapshot{{
			Name:             "button",
			Path:             "App > Dashboard > button",
			Kind:             "host",
			RenderDurationNs: 1,
			SelfDurationNs:   1,
		}},
	}

	mapped := mapRuntimeFiberSnapshot(source)
	if mapped == nil {
		t.Fatal("mapRuntimeFiberSnapshot returned nil for non-nil source")
	}
	if mapped.Name != source.Name || mapped.Path != source.Path || mapped.Kind != source.Kind {
		t.Fatalf("mapped identity mismatch: %#v", mapped)
	}
	if !mapped.Dirty || !mapped.NeedsUpdate || !mapped.FineGrained {
		t.Fatalf("mapped booleans were not preserved: %#v", mapped)
	}
	if mapped.Signature != "Dashboard | key=main | hooks=state > effect" {
		t.Fatalf("Signature = %q", mapped.Signature)
	}
	if len(mapped.Hooks) != 1 || mapped.Hooks[0].Slot != 1 || mapped.Hooks[0].Kind != "state" {
		t.Fatalf("Hooks = %#v", mapped.Hooks)
	}
	if len(mapped.Children) != 1 || mapped.Children[0].Name != "button" {
		t.Fatalf("Children = %#v", mapped.Children)
	}

	source.Hooks[0].Kind = "mutated"
	source.Children[0].Name = "mutated"
	if mapped.Hooks[0].Kind != "state" || mapped.Children[0].Name != "button" {
		t.Fatalf("mapped node aliases source slices: %#v", mapped)
	}
	if mapRuntimeFiberSnapshot(nil) != nil {
		t.Fatal("nil fiber snapshot should map to nil")
	}
}

func TestPluginInterposerMapsRuntimeStatsProfilingAndHydration(t *testing.T) {
	stats := mapRuntimeStats(InspectionStats{
		TotalFibers:       7,
		DirtyFibers:       2,
		ComponentFibers:   3,
		HostFibers:        4,
		TextFibers:        5,
		FineGrainedFibers: 6,
		HookEntries:       8,
		Effects:           9,
	})
	if stats.TotalFibers != 7 || stats.Effects != 9 || stats.FineGrainedFibers != 6 {
		t.Fatalf("mapRuntimeStats() = %#v", stats)
	}

	profiling := ProfilingSnapshot{
		RenderCalls:                      1,
		ScheduledRootUpdates:             2,
		ScheduledFiberMarks:              3,
		ScheduledGranularMarks:           4,
		WorkLoopPasses:                   5,
		ProcessedUnits:                   6,
		CommitCount:                      7,
		FineGrainedCommits:               8,
		FineGrainedDescendantHostCommits: 9,
		FineGrainedDescendantTextCommits: 10,
		EffectExecutions:                 11,
		CleanupExecutions:                12,
		LastRenderDurationNs:             13,
		LastCommitDurationNs:             14,
		LastEffectDurationNs:             15,
		LastCleanupDurationNs:            16,
		PhaseTotals: ProfilingPhaseTotalsSnapshot{
			RenderDurationNs:  17,
			DiffDurationNs:    18,
			CommitDurationNs:  19,
			EffectDurationNs:  20,
			CleanupDurationNs: 21,
		},
		ComponentRenders: []ComponentRenderTraceSnapshot{{
			Name:                    "Widget",
			Path:                    "App > Widget",
			RenderCount:             3,
			RerenderCount:           2,
			LastTrigger:             "props",
			LastRenderDurationNs:    31,
			TotalRenderDurationNs:   93,
			AverageRenderDurationNs: 31,
			LastRenderedAt:          "now",
			TriggerCounts:           map[string]int{"props": 2},
		}},
		RecentEvents: []ProfilingEvent{{
			Domain:        "runtime",
			Name:          "render",
			Phase:         "commit",
			Target:        "Widget",
			CorrelationID: "corr",
			DurationNs:    44,
			Timestamp:     "later",
			Fields:        map[string]string{"route": "/"},
		}},
		FlamegraphFrames: []FlamegraphFrameSnapshot{{
			Name:              "frame",
			Kind:              "component",
			Path:              "App",
			Depth:             1,
			StartNs:           2,
			DurationNs:        3,
			SelfDurationNs:    4,
			RenderDurationNs:  5,
			DiffDurationNs:    6,
			CommitDurationNs:  7,
			EffectDurationNs:  8,
			CleanupDurationNs: 9,
		}},
		Startup: StartupProfilingSnapshot{
			Mode:                       "hydrate",
			StartedAt:                  "start",
			BootstrapReadDurationNs:    1,
			WASMTransferBytes:          2,
			WASMDecodedBytes:           3,
			BootstrapDecodedBytes:      4,
			CacheWarmupDurationNs:      5,
			ServiceWorkerOverheadNs:    6,
			InitialRouteDataBytes:      7,
			HydrationDurationNs:        8,
			StartupCommitDurationNs:    9,
			FirstInteractionDurationNs: 10,
			FirstInteractionCaptured:   true,
			FirstInteractionEvent:      "click",
			RouteBudgets: []RouteStartupBudgetSnapshot{{
				RouteFamily:                       "/users/:id",
				LastRoutePath:                     "/users/1",
				SampleCount:                       2,
				AverageBootstrapReadDurationNs:    3,
				AverageWASMTransferBytes:          4,
				AverageWASMDecodedBytes:           5,
				AverageBootstrapDecodedBytes:      6,
				AverageCacheWarmupDurationNs:      7,
				AverageServiceWorkerOverheadNs:    8,
				AverageInitialRouteDataBytes:      9,
				AverageHydrationDurationNs:        10,
				AverageStartupCommitDurationNs:    11,
				AverageFirstInteractionDurationNs: 12,
			}},
		},
		HotBranches: []HotBranchSnapshot{{
			Name:              "Hot",
			Kind:              "component",
			Path:              "App > Hot",
			RenderDurationNs:  1,
			DiffDurationNs:    2,
			CommitDurationNs:  3,
			EffectDurationNs:  4,
			CleanupDurationNs: 5,
			SelfDurationNs:    6,
			SubtreeDurationNs: 7,
		}},
	}

	mapped := mapRuntimeProfiling(profiling)
	if mapped.RenderCalls != 1 || mapped.CleanupExecutions != 12 || mapped.PhaseTotals.CleanupDurationNs != 21 {
		t.Fatalf("profiling counters not preserved: %#v", mapped)
	}
	if mapped.ComponentRenders[0].TriggerCounts["props"] != 2 {
		t.Fatalf("ComponentRenders = %#v", mapped.ComponentRenders)
	}
	if mapped.RecentEvents[0].Fields["route"] != "/" || mapped.FlamegraphFrames[0].CleanupDurationNs != 9 {
		t.Fatalf("profiling nested records not preserved: %#v", mapped)
	}
	if mapped.Startup.RouteBudgets[0].AverageFirstInteractionDurationNs != 12 || !mapped.Startup.FirstInteractionCaptured {
		t.Fatalf("startup profiling not preserved: %#v", mapped.Startup)
	}
	if mapped.HotBranches[0].SubtreeDurationNs != 7 {
		t.Fatalf("HotBranches = %#v", mapped.HotBranches)
	}

	profiling.ComponentRenders[0].TriggerCounts["props"] = 99
	profiling.RecentEvents[0].Fields["route"] = "/mutated"
	if mapped.ComponentRenders[0].TriggerCounts["props"] != 2 || mapped.RecentEvents[0].Fields["route"] != "/" {
		t.Fatal("profiling maps should be cloned")
	}

	hydration := mapRuntimeHydration(HydrationDebugSnapshot{
		CorrelationID:        "h1",
		StartedAt:            "s",
		FinishedAt:           "f",
		DurationNs:           15,
		ExistingDOMNodeCount: 1,
		FallbackCount:        2,
		MismatchCount:        3,
		DiscardedNodeCount:   4,
		Strict:               true,
		Failed:               true,
		Failure:              "bad node",
		RecentMessages:       []string{"a", "b"},
	})
	if hydration.CorrelationID != "h1" || hydration.DiscardedNodeCount != 4 || hydration.RecentMessages[1] != "b" {
		t.Fatalf("mapRuntimeHydration() = %#v", hydration)
	}
}

func TestPluginInterposerDiagnosticsLogsAndBudget(t *testing.T) {
	diagnostics := []Diagnostic{
		{
			Source:         "runtime",
			Severity:       DiagnosticError,
			Classification: DiagnosticCorrectness,
			Code:           "GWC001",
			Docs:           "docs",
			Remediation:    "fix it",
			Recoverable:    true,
			TopFrame:       "frame",
			Consequence:    "stale UI",
			Message:        "failed",
			Count:          2,
			Path:           "App",
			ComponentStack: []string{"App", "Child"},
			Fields:         map[string]string{"id": "1"},
		},
		{Code: "GWC002"},
	}
	logs := []LogEntry{
		{
			Domain:         "runtime",
			Level:          LogWarn,
			Classification: DiagnosticPerformance,
			Code:           "LOG001",
			Docs:           "docs",
			Remediation:    "memoize",
			Recoverable:    true,
			TopFrame:       "frame",
			Consequence:    "slow render",
			Message:        "slow",
			Timestamp:      "ts",
			CorrelationID:  "corr",
			Fields:         map[string]string{"duration": "10"},
		},
		{Code: "LOG002"},
	}

	mappedDiagnostics := mapRuntimeDiagnostics(diagnostics, pluginruntime.QueryBudget{MaxItems: 1})
	if len(mappedDiagnostics) != 1 || mappedDiagnostics[0].Code != "GWC001" {
		t.Fatalf("mapRuntimeDiagnostics() = %#v", mappedDiagnostics)
	}
	if !reflect.DeepEqual(mappedDiagnostics[0].ComponentStack, []string{"App", "Child"}) {
		t.Fatalf("ComponentStack = %#v", mappedDiagnostics[0].ComponentStack)
	}
	mappedLogs := mapRuntimeLogs(logs, pluginruntime.QueryBudget{MaxItems: 1})
	if len(mappedLogs) != 1 || mappedLogs[0].Code != "LOG001" || mappedLogs[0].Level != "warn" {
		t.Fatalf("mapRuntimeLogs() = %#v", mappedLogs)
	}

	diagnostics[0].ComponentStack[0] = "mutated"
	diagnostics[0].Fields["id"] = "mutated"
	logs[0].Fields["duration"] = "mutated"
	if mappedDiagnostics[0].ComponentStack[0] != "App" || mappedDiagnostics[0].Fields["id"] != "1" {
		t.Fatal("diagnostics should be defensively cloned")
	}
	if mappedLogs[0].Fields["duration"] != "10" {
		t.Fatal("logs should be defensively cloned")
	}

	values := []int{1, 2, 3}
	if got := applyRuntimeBudget(values, pluginruntime.QueryBudget{}); len(got) != 3 {
		t.Fatalf("zero budget should keep all values: %#v", got)
	}
	budgeted := applyRuntimeBudget(values, pluginruntime.QueryBudget{MaxItems: 2})
	if !reflect.DeepEqual(budgeted, []int{1, 2}) {
		t.Fatalf("budgeted values = %#v", budgeted)
	}
	values[0] = 99
	if budgeted[0] != 1 {
		t.Fatal("truncated budget result should not alias source prefix")
	}
}

func TestPluginInterposerCloneHelpers(t *testing.T) {
	if cloneRuntimeIntMap(nil) != nil || cloneRuntimeStringMap(map[string]string{}) != nil {
		t.Fatal("empty maps should clone to nil")
	}
	ints := map[string]int{"a": 1}
	strings := map[string]string{"b": "2"}
	clonedInts := cloneRuntimeIntMap(ints)
	clonedStrings := cloneRuntimeStringMap(strings)
	ints["a"] = 9
	strings["b"] = "mutated"
	if clonedInts["a"] != 1 || clonedStrings["b"] != "2" {
		t.Fatalf("clone helpers alias source maps: %#v %#v", clonedInts, clonedStrings)
	}
}
