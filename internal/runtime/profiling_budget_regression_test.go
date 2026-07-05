package runtime

import (
	"testing"
	"time"
)

type profilingBudgetFixture struct {
	parseName                          string
	parseRouteFamily                   string
	parseRoutePath                     string
	parseStartupMode                   string
	parseBootstrapReadDurationNs       int64
	parseHydrationDurationNs           int64
	parseStartupCommitDurationNs       int64
	parseFirstInteractionDurationNs    int64
	parseWASMTransferBytes             int64
	parseWASMDecodedBytes              int64
	parseBootstrapDecodedBytes         int64
	parseCacheWarmupDurationNs         int64
	parseServiceWorkerOverheadNs       int64
	parseInitialRouteDataBytes         int64
	parseRenderCount                   int
	parseRerenderCount                 int
	parseTotalRenderDurationNs         int64
	parseTotalCommitDurationNs         int64
	parseMaxBootstrapReadDurationNs    int64
	parseMaxHydrationDurationNs        int64
	parseMaxFirstInteractionDurationNs int64
	parseMaxRerenderRatioPercent       int
}

// TestProfilingBudgetFixturesForStartupHydrationAndRerender verifies budget fixtures for representative app flows.
func TestProfilingBudgetFixturesForStartupHydrationAndRerender(parseT *testing.T) {
	parseFixtures := buildProfilingBudgetFixtures()
	for _, parseFixture := range parseFixtures {
		parseT.Run(parseFixture.parseName, func(parseT2 *testing.T) {
			parseRuntime := buildProfilingBudgetRuntime(parseFixture)
			parseSnapshot := parseRuntime.Inspect()
			parseStartup := parseSnapshot.Profiling.Startup
			if parseStartup.Mode != parseFixture.parseStartupMode {
				parseT2.Fatalf("expected startup mode %q, got %q", parseFixture.parseStartupMode, parseStartup.Mode)
			}
			if parseStartup.BootstrapReadDurationNs > parseFixture.parseMaxBootstrapReadDurationNs {
				parseT2.Fatalf("expected bootstrap read <= %dns for %s, got %dns", parseFixture.parseMaxBootstrapReadDurationNs, parseFixture.parseName, parseStartup.BootstrapReadDurationNs)
			}
			if parseStartup.HydrationDurationNs > parseFixture.parseMaxHydrationDurationNs {
				parseT2.Fatalf("expected hydration <= %dns for %s, got %dns", parseFixture.parseMaxHydrationDurationNs, parseFixture.parseName, parseStartup.HydrationDurationNs)
			}
			if parseStartup.FirstInteractionDurationNs > parseFixture.parseMaxFirstInteractionDurationNs {
				parseT2.Fatalf("expected first interaction <= %dns for %s, got %dns", parseFixture.parseMaxFirstInteractionDurationNs, parseFixture.parseName, parseStartup.FirstInteractionDurationNs)
			}
			if len(parseStartup.RouteBudgets) != 1 {
				parseT2.Fatalf("expected one route budget for %s, got %d", parseFixture.parseName, len(parseStartup.RouteBudgets))
			}
			parseBudget := parseStartup.RouteBudgets[0]
			if parseBudget.RouteFamily != parseFixture.parseRouteFamily {
				parseT2.Fatalf("expected route budget family %q, got %q", parseFixture.parseRouteFamily, parseBudget.RouteFamily)
			}
			parseRerenderRatioPercent := buildProfilingRerenderRatioPercent(parseSnapshot.Profiling.ComponentRenders)
			if parseRerenderRatioPercent > parseFixture.parseMaxRerenderRatioPercent {
				parseT2.Fatalf("expected rerender ratio <= %d%% for %s, got %d%%", parseFixture.parseMaxRerenderRatioPercent, parseFixture.parseName, parseRerenderRatioPercent)
			}
		})
	}
}

// buildProfilingBudgetFixtures returns representative app budget fixtures.
func buildProfilingBudgetFixtures() []profilingBudgetFixture {
	return []profilingBudgetFixture{
		{
			parseName:                          "small-app",
			parseRouteFamily:                   "/",
			parseRoutePath:                     "/",
			parseStartupMode:                   "hydrate",
			parseBootstrapReadDurationNs:       int64(4 * time.Millisecond),
			parseHydrationDurationNs:           int64(6 * time.Millisecond),
			parseStartupCommitDurationNs:       int64(3 * time.Millisecond),
			parseFirstInteractionDurationNs:    int64(16 * time.Millisecond),
			parseWASMTransferBytes:             220 * 1024,
			parseWASMDecodedBytes:              560 * 1024,
			parseBootstrapDecodedBytes:         1600,
			parseCacheWarmupDurationNs:         int64(2 * time.Millisecond),
			parseServiceWorkerOverheadNs:       int64(1 * time.Millisecond),
			parseInitialRouteDataBytes:         900,
			parseRenderCount:                   14,
			parseRerenderCount:                 2,
			parseTotalRenderDurationNs:         int64(11 * time.Millisecond),
			parseTotalCommitDurationNs:         int64(6 * time.Millisecond),
			parseMaxBootstrapReadDurationNs:    int64(8 * time.Millisecond),
			parseMaxHydrationDurationNs:        int64(10 * time.Millisecond),
			parseMaxFirstInteractionDurationNs: int64(28 * time.Millisecond),
			parseMaxRerenderRatioPercent:       30,
		},
		{
			parseName:                          "routed-mid-sized-app",
			parseRouteFamily:                   "/workspace/*",
			parseRoutePath:                     "/workspace/reports/weekly",
			parseStartupMode:                   "hydrate",
			parseBootstrapReadDurationNs:       int64(10 * time.Millisecond),
			parseHydrationDurationNs:           int64(18 * time.Millisecond),
			parseStartupCommitDurationNs:       int64(8 * time.Millisecond),
			parseFirstInteractionDurationNs:    int64(38 * time.Millisecond),
			parseWASMTransferBytes:             630 * 1024,
			parseWASMDecodedBytes:              2 * 1024 * 1024,
			parseBootstrapDecodedBytes:         3800,
			parseCacheWarmupDurationNs:         int64(4 * time.Millisecond),
			parseServiceWorkerOverheadNs:       int64(3 * time.Millisecond),
			parseInitialRouteDataBytes:         2300,
			parseRenderCount:                   31,
			parseRerenderCount:                 8,
			parseTotalRenderDurationNs:         int64(28 * time.Millisecond),
			parseTotalCommitDurationNs:         int64(15 * time.Millisecond),
			parseMaxBootstrapReadDurationNs:    int64(16 * time.Millisecond),
			parseMaxHydrationDurationNs:        int64(24 * time.Millisecond),
			parseMaxFirstInteractionDurationNs: int64(52 * time.Millisecond),
			parseMaxRerenderRatioPercent:       40,
		},
		{
			parseName:                          "production-shaped-app",
			parseRouteFamily:                   "/workspace/canvas/*",
			parseRoutePath:                     "/workspace/canvas/board/42",
			parseStartupMode:                   "hydrate",
			parseBootstrapReadDurationNs:       int64(15 * time.Millisecond),
			parseHydrationDurationNs:           int64(26 * time.Millisecond),
			parseStartupCommitDurationNs:       int64(11 * time.Millisecond),
			parseFirstInteractionDurationNs:    int64(50 * time.Millisecond),
			parseWASMTransferBytes:             920 * 1024,
			parseWASMDecodedBytes:              3 * 1024 * 1024,
			parseBootstrapDecodedBytes:         5400,
			parseCacheWarmupDurationNs:         int64(6 * time.Millisecond),
			parseServiceWorkerOverheadNs:       int64(4 * time.Millisecond),
			parseInitialRouteDataBytes:         4200,
			parseRenderCount:                   52,
			parseRerenderCount:                 16,
			parseTotalRenderDurationNs:         int64(44 * time.Millisecond),
			parseTotalCommitDurationNs:         int64(24 * time.Millisecond),
			parseMaxBootstrapReadDurationNs:    int64(24 * time.Millisecond),
			parseMaxHydrationDurationNs:        int64(34 * time.Millisecond),
			parseMaxFirstInteractionDurationNs: int64(70 * time.Millisecond),
			parseMaxRerenderRatioPercent:       45,
		},
	}
}

// buildProfilingBudgetRuntime builds one runtime seeded from one budget fixture.
func buildProfilingBudgetRuntime(parseFixture profilingBudgetFixture) *Runtime {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseChild := &Fiber{
		typeOf:           func() *Element { return nil },
		parent:           parseRoot,
		renderDurationNs: parseFixture.parseTotalRenderDurationNs,
		commitDurationNs: parseFixture.parseTotalCommitDurationNs,
		updateOrigin:     "fixture." + parseFixture.parseName,
	}
	parseRoot.child = parseChild
	parseRuntime := &Runtime{currentRoot: parseRoot}
	parseRuntime.profiling.startupMode = parseFixture.parseStartupMode
	parseRuntime.profiling.startupStartedAt = time.Now().Add(-time.Duration(parseFixture.parseFirstInteractionDurationNs))
	parseRuntime.profiling.bootstrapReadDurationNs = parseFixture.parseBootstrapReadDurationNs
	parseRuntime.profiling.startupWASMTransferBytes = parseFixture.parseWASMTransferBytes
	parseRuntime.profiling.startupWASMDecodedBytes = parseFixture.parseWASMDecodedBytes
	parseRuntime.profiling.startupBootstrapDecodedBytes = parseFixture.parseBootstrapDecodedBytes
	parseRuntime.profiling.startupCacheWarmupDurationNs = parseFixture.parseCacheWarmupDurationNs
	parseRuntime.profiling.startupServiceWorkerOverheadNs = parseFixture.parseServiceWorkerOverheadNs
	parseRuntime.profiling.startupInitialRouteDataBytes = parseFixture.parseInitialRouteDataBytes
	parseRuntime.profiling.hydrationDurationNs = parseFixture.parseHydrationDurationNs
	parseRuntime.profiling.startupCommitDurationNs = parseFixture.parseStartupCommitDurationNs
	parseRuntime.profiling.firstInteractionDurationNs = parseFixture.parseFirstInteractionDurationNs
	parseRuntime.profiling.firstInteractionCaptured = true
	parseRuntime.profiling.firstInteractionEvent = "pointerdown"
	parseRuntime.profiling.totalRenderDurationNs = parseFixture.parseTotalRenderDurationNs
	parseRuntime.profiling.totalCommitDurationNs = parseFixture.parseTotalCommitDurationNs
	parseRuntime.profiling.componentRenders = map[string]*componentRenderTrace{
		"App > " + parseFixture.parseName: {
			Name:                  "App",
			Path:                  "App > " + parseFixture.parseName,
			RenderCount:           parseFixture.parseRenderCount,
			RerenderCount:         parseFixture.parseRerenderCount,
			LastTrigger:           "state",
			LastRenderDurationNs:  int64(2 * time.Millisecond),
			TotalRenderDurationNs: parseFixture.parseTotalRenderDurationNs,
			TriggerCounts:         map[string]int{"state": parseFixture.parseRerenderCount, "mount": 1},
			LastRenderedAt:        time.Date(2026, 3, 25, 17, 30, 0, 0, time.UTC),
		},
	}
	parseRuntime.profiling.routeStartupBudgets = map[string]*routeStartupBudget{
		parseFixture.parseRouteFamily: {
			RouteFamily:                     parseFixture.parseRouteFamily,
			LastRoutePath:                   parseFixture.parseRoutePath,
			SampleCount:                     1,
			BootstrapReadDurationTotalNs:    parseFixture.parseBootstrapReadDurationNs,
			HydrationDurationTotalNs:        parseFixture.parseHydrationDurationNs,
			StartupCommitDurationTotalNs:    parseFixture.parseStartupCommitDurationNs,
			FirstInteractionDurationTotalNs: parseFixture.parseFirstInteractionDurationNs,
			WASMTransferBytesTotal:          parseFixture.parseWASMTransferBytes,
			WASMDecodedBytesTotal:           parseFixture.parseWASMDecodedBytes,
			BootstrapDecodedBytesTotal:      parseFixture.parseBootstrapDecodedBytes,
			CacheWarmupDurationTotalNs:      parseFixture.parseCacheWarmupDurationNs,
			ServiceWorkerOverheadTotalNs:    parseFixture.parseServiceWorkerOverheadNs,
			InitialRouteDataBytesTotal:      parseFixture.parseInitialRouteDataBytes,
		},
	}
	return parseRuntime
}

// buildProfilingRerenderRatioPercent calculates the rerender ratio percentage for one trace set.
func buildProfilingRerenderRatioPercent(parseTraces []ComponentRenderTraceSnapshot) int {
	parseRenderCount := 0
	parseRerenderCount := 0
	for _, parseTrace := range parseTraces {
		parseRenderCount += parseTrace.RenderCount
		parseRerenderCount += parseTrace.RerenderCount
	}
	if parseRenderCount <= 0 {
		return 0
	}
	return parseRerenderCount * 100 / parseRenderCount
}
