package runtime

import (
	"testing"
	"time"
)

// TestProfilingRepresentativeScenarios verifies profiling snapshots stay actionable across representative app shapes.
func TestProfilingRepresentativeScenarios(parseT *testing.T) {
	parseScenarios := []string{
		"large-list",
		"nested-routes",
		"async-dashboard",
		"portal-overlays",
	}
	for _, parseScenario := range parseScenarios {
		parseT.Run(parseScenario, func(parseT2 *testing.T) {
			parseRuntime := buildProfilingScenarioRuntime(parseScenario)
			parseSnapshot := parseRuntime.Inspect()
			if parseSnapshot.Profiling.PhaseTotals.RenderDurationNs <= 0 || parseSnapshot.Profiling.PhaseTotals.CommitDurationNs <= 0 {
				parseT2.Fatalf("expected phase totals for %s, got %+v", parseScenario, parseSnapshot.Profiling.PhaseTotals)
			}
			if len(parseSnapshot.Profiling.ComponentRenders) == 0 {
				parseT2.Fatalf("expected component traces for %s", parseScenario)
			}
			if len(parseSnapshot.Profiling.RecentEvents) == 0 {
				parseT2.Fatalf("expected recent events for %s", parseScenario)
			}
			if parseSnapshot.Profiling.Startup.Mode == "" {
				parseT2.Fatalf("expected startup mode for %s", parseScenario)
			}
		})
	}
}

// BenchmarkProfilingRepresentativeScenarios benchmarks inspection snapshots for representative profiling scenarios.
func BenchmarkProfilingRepresentativeScenarios(parseB *testing.B) {
	parseScenarios := []string{
		"large-list",
		"nested-routes",
		"async-dashboard",
		"portal-overlays",
	}
	for _, parseScenario := range parseScenarios {
		parseB.Run(parseScenario, func(parseB2 *testing.B) {
			parseRuntime := buildProfilingScenarioRuntime(parseScenario)
			parseB2.ReportAllocs()
			for parseB2.Loop() {
				parseSnapshot := parseRuntime.Inspect()
				if parseSnapshot.Profiling.Startup.Mode == "" {
					parseB2.Fatalf("expected startup mode for scenario %s", parseScenario)
				}
			}
		})
	}
}

// buildProfilingScenarioRuntime builds one runtime fixture with profiling data for a named scenario.
func buildProfilingScenarioRuntime(parseScenario string) *Runtime {
	parseComponent := func() *Element { return nil }
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseChild := &Fiber{
		typeOf:            parseComponent,
		parent:            parseRoot,
		renderDurationNs:  int64(3 * time.Millisecond),
		diffDurationNs:    int64(2 * time.Millisecond),
		commitDurationNs:  int64(2 * time.Millisecond),
		effectDurationNs:  int64(1 * time.Millisecond),
		cleanupDurationNs: int64(1 * time.Millisecond),
		updateOrigin:      parseScenario,
	}
	parseRoot.child = parseChild
	parseRuntime := &Runtime{currentRoot: parseRoot}
	parseRuntime.profiling.startupMode = "hydrate"
	parseRuntime.profiling.startupStartedAt = time.Now().Add(-35 * time.Millisecond)
	parseRuntime.profiling.bootstrapReadDurationNs = int64(7 * time.Millisecond)
	parseRuntime.profiling.hydrationDurationNs = int64(9 * time.Millisecond)
	parseRuntime.profiling.startupCommitDurationNs = int64(5 * time.Millisecond)
	parseRuntime.profiling.firstInteractionDurationNs = int64(24 * time.Millisecond)
	parseRuntime.profiling.firstInteractionCaptured = true
	parseRuntime.profiling.firstInteractionEvent = "click"
	parseRuntime.profiling.totalRenderDurationNs = int64(20 * time.Millisecond)
	parseRuntime.profiling.totalDiffDurationNs = int64(11 * time.Millisecond)
	parseRuntime.profiling.totalCommitDurationNs = int64(14 * time.Millisecond)
	parseRuntime.profiling.totalEffectDurationNs = int64(5 * time.Millisecond)
	parseRuntime.profiling.totalCleanupDurationNs = int64(4 * time.Millisecond)
	parseRuntime.profiling.componentRenders = map[string]*componentRenderTrace{
		"App > Dashboard > " + parseScenario: {
			Name:                  "ScenarioView",
			Path:                  "App > Dashboard > " + parseScenario,
			RenderCount:           42,
			RerenderCount:         17,
			LastTrigger:           parseScenario,
			LastRenderDurationNs:  int64(2 * time.Millisecond),
			TotalRenderDurationNs: int64(70 * time.Millisecond),
			TriggerCounts:         map[string]int{parseScenario: 17, "props": 8},
			LastRenderedAt:        time.Date(2026, 3, 25, 16, 0, 0, 0, time.UTC),
		},
	}
	parseRuntime.profiling.events = []ProfilingEvent{
		{
			Domain:     "router",
			Name:       "navigation",
			Phase:      "start",
			Target:     "/" + parseScenario,
			DurationNs: int64(2 * time.Millisecond),
			Timestamp:  "2026-03-25T16:00:00.000Z",
			Fields:     map[string]string{"scenario": parseScenario},
		},
		{
			Domain:     "async",
			Name:       "loader",
			Phase:      "finish",
			Target:     "/" + parseScenario,
			DurationNs: int64(3 * time.Millisecond),
			Timestamp:  "2026-03-25T16:00:00.500Z",
			Fields:     map[string]string{"status": "ok"},
		},
	}
	parseRuntime.profiling.routeStartupBudgets = map[string]*routeStartupBudget{
		"/" + parseScenario: {
			RouteFamily:                     "/" + parseScenario,
			LastRoutePath:                   "/" + parseScenario,
			SampleCount:                     3,
			BootstrapReadDurationTotalNs:    int64(18 * time.Millisecond),
			HydrationDurationTotalNs:        int64(24 * time.Millisecond),
			StartupCommitDurationTotalNs:    int64(15 * time.Millisecond),
			FirstInteractionDurationTotalNs: int64(66 * time.Millisecond),
			WASMTransferBytesTotal:          3 * 1024 * 512,
			WASMDecodedBytesTotal:           3 * 1024 * 1024,
			BootstrapDecodedBytesTotal:      3 * 2048,
			CacheWarmupDurationTotalNs:      int64(6 * time.Millisecond),
			ServiceWorkerOverheadTotalNs:    int64(3 * time.Millisecond),
			InitialRouteDataBytesTotal:      3 * 1024,
		},
	}
	return parseRuntime
}
