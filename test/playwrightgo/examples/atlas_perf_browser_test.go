//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// Atlas Commerce OS render-path measurement — BROWSER lane.
//
// WHY THIS LANE, GIVEN THE NATIVE ONE EXISTS
//
// The native lane (examples/testing/atlas-perf) can see component bodies,
// element construction, fiber creation and serialization. It CANNOT see the four
// things docs/plans/v5-plan.md names as M2's structural cause: "Reconciliation is
// sliced; deletion, DOM commit, order repair, and layout effects are not." Those
// only exist against a real DOM. M2, M3 and M6 all live here.
//
// WHAT IT MEASURES
//
//   - Cold hydration cost per route, repeated, min/median/max — never a single
//     reading, because this chassis is fanless and the plan records the same
//     build measuring 4 long frames cold and 33 hot.
//   - Long Animation Frames during hydration and during real interactions, with
//     per-script attribution when the browser supports it.
//   - Interaction latency (Event Timing) from REAL Playwright input. Synthetic
//     events carry interactionId 0 and are silently dropped by the observer —
//     that defect already cost this repo one v4 baseline.
//   - A V8 CPU profile of the hydration window, bucketed by Go package. Go's
//     js/wasm build keeps the name section, so profile frames carry
//     internal/runtime and shared/atlas symbol names and the framework-vs-example
//     split is readable in the browser too. This is the same technique as
//     example201_phase_probe_test.go.
//
// WHAT IT CANNOT MEASURE, AND WHY
//
//   - Go-side phase totals (render / diff / commit / effect / cleanup) and GC
//     pause. Those come from gwcruntime.GetGlobalRuntime().Inspect(), which needs
//     an in-app probe registered from main — the pattern
//     examples/testing/v5-load-harness/probe.go uses. Atlas's client does not
//     register one, and this task does not own examples/server/atlas-commerce-os/
//     client. The CPU profile is the substitute: it attributes by symbol instead
//     of by phase.
//
// REPRODUCE:
//
//	go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerf -v -timeout 40m
//
//	# against a server you already have running (skips a ~25 s boot per test,
//	# and works when the tree is mid-edit and `go run` would not compile)
//	ATLAS_PERF_BASE_URL=http://127.0.0.1:8531 \
//	  go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerf -v -timeout 40m

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// -----------------------------------------------------------------------------
// Server handling
// -----------------------------------------------------------------------------

// atlasPerfBaseURL returns the Atlas base URL for a measurement test.
//
// ATLAS_PERF_BASE_URL reuses a server that is already running. That is not a
// convenience: a measurement pass takes many navigations, and booting a fresh
// `go run` per test both wastes ~25 s each time and makes the run impossible
// while another agent has the tree in a non-compiling state. When the variable is
// absent it falls back to the shared spec launcher, which reserves a free port
// and kills its own process tree.
func atlasPerfBaseURL(parseT *testing.T) string {
	parseT.Helper()
	if parseExisting := strings.TrimSpace(os.Getenv("ATLAS_PERF_BASE_URL")); parseExisting != "" {
		parseT.Logf("reusing the Atlas server at %s (ATLAS_PERF_BASE_URL)", parseExisting)
		assertAtlasSpecWASMPresent(parseT, parseExisting)
		return parseExisting
	}
	_, parseFile, _, _ := runtime.Caller(0)
	return startAtlasSpecServer(parseT, examplesRepoRootFromFile(parseFile))
}

// atlasPerfRouteCase is one route to measure, with the session it needs.
type atlasPerfRouteCase struct {
	Name  string
	Route string
	Role  string
	// Why records what this route contributes, so the set can be judged.
	Why string
}

// atlasPerfRouteCases spans the app's real shapes. Deliberately small: each entry
// costs a cold navigation plus repeats, and a long run heats the machine into the
// regime that makes its own numbers untrustworthy.
var atlasPerfRouteCases = []atlasPerfRouteCase{
	{Name: "public-landing", Route: "/", Why: "smallest real tree — isolates boot cost from render cost"},
	{Name: "public-catalog", Route: "/shop", Why: "buyer catalog, widest public tree"},
	{Name: "operator-dashboard", Route: "/app/dashboard", Role: "inventory_manager", Why: "densest payload: 88 comments + transfers + receiving + orders"},
	{Name: "operator-comments", Route: "/app/comments", Role: "inventory_manager", Why: "the 88-item inbox — the largest real list in the app (hypothesis 3)"},
}

// -----------------------------------------------------------------------------
// In-page instrumentation
// -----------------------------------------------------------------------------

// atlasPerfInstallObserversScript is injected via AddInitScript so it runs BEFORE
// any page script, which is the only way to catch the long frames that hydration
// itself produces. An observer created after navigation misses exactly the
// entries this lane is about.
//
// buffered:true is used, and the harness filters by a recorded start time rather
// than trusting the buffer. The v5 load harness's M2 was overstated about sixfold
// by re-counting buffered history per window; the fix there was to drop the
// history, not the buffering, and the same rule applies here.
const atlasPerfInstallObserversScript = `
() => {
  const state = {
    longFrames: [],
    interactions: [],
    longFrameSource: 'none',
    interactionsSupported: false,
  };
  window.__atlasPerf = state;
  if (typeof PerformanceObserver === 'undefined') return;
  const supported = PerformanceObserver.supportedEntryTypes || [];

  if (supported.includes('long-animation-frame')) {
    state.longFrameSource = 'long-animation-frame';
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        state.longFrames.push({
          startTime: entry.startTime,
          duration: entry.duration,
          blockingDuration: entry.blockingDuration || 0,
          styleAndLayoutDuration: entry.styleAndLayoutDuration || 0,
          scripts: (entry.scripts || []).map((s) => ({
            name: s.name,
            invoker: s.invoker,
            duration: s.duration,
            forcedStyleAndLayoutDuration: s.forcedStyleAndLayoutDuration || 0,
          })),
        });
      }
    }).observe({ type: 'long-animation-frame', buffered: true });
  } else if (supported.includes('longtask')) {
    // Weaker instrument: one task, no attribution, no style/layout split. It is
    // recorded as the source so a run measured with it is never silently
    // compared against a run measured with LoAF.
    state.longFrameSource = 'longtask';
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        state.longFrames.push({
          startTime: entry.startTime,
          duration: entry.duration,
          blockingDuration: Math.max(0, entry.duration - 50),
          styleAndLayoutDuration: 0,
          scripts: [],
        });
      }
    }).observe({ type: 'longtask', buffered: true });
  }

  if (supported.includes('event')) {
    state.interactionsSupported = true;
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        // interactionId 0 is not a discrete user interaction. Event Timing only
        // assigns one to TRUSTED events, so a page dispatching its own events
        // records nothing — the defect that produced an n=0 M3 in the v4
        // baseline. Everything measured here comes from real Playwright input.
        if (!entry.interactionId) continue;
        state.interactions.push({
          name: entry.name,
          startTime: entry.startTime,
          duration: entry.duration,
          inputDelay: entry.processingStart - entry.startTime,
          processingTime: entry.processingEnd - entry.processingStart,
          presentationDelay: entry.startTime + entry.duration - entry.processingEnd,
        });
      }
    }).observe({ type: 'event', buffered: true, durationThreshold: 16 });
  }
}
`

// atlasPerfLongFrame mirrors one recorded entry.
type atlasPerfLongFrame struct {
	StartTime              float64 `json:"startTime"`
	Duration               float64 `json:"duration"`
	BlockingDuration       float64 `json:"blockingDuration"`
	StyleAndLayoutDuration float64 `json:"styleAndLayoutDuration"`
	Scripts                []struct {
		Name                         string  `json:"name"`
		Invoker                      string  `json:"invoker"`
		Duration                     float64 `json:"duration"`
		ForcedStyleAndLayoutDuration float64 `json:"forcedStyleAndLayoutDuration"`
	} `json:"scripts"`
}

type atlasPerfInteraction struct {
	Name              string  `json:"name"`
	StartTime         float64 `json:"startTime"`
	Duration          float64 `json:"duration"`
	InputDelay        float64 `json:"inputDelay"`
	ProcessingTime    float64 `json:"processingTime"`
	PresentationDelay float64 `json:"presentationDelay"`
}

type atlasPerfSnapshot struct {
	LongFrames            []atlasPerfLongFrame   `json:"longFrames"`
	Interactions          []atlasPerfInteraction `json:"interactions"`
	LongFrameSource       string                 `json:"longFrameSource"`
	InteractionsSupported bool                   `json:"interactionsSupported"`
}

// readAtlasPerfSnapshot pulls the recorded entries out of the page.
func readAtlasPerfSnapshot(parseT *testing.T, parsePage playwright.Page, parseSinceMs float64) atlasPerfSnapshot {
	parseT.Helper()
	parseRaw, parseErr := parsePage.Evaluate(`(since) => {
		const s = window.__atlasPerf;
		if (!s) return JSON.stringify({ longFrames: [], interactions: [], longFrameSource: 'missing', interactionsSupported: false });
		return JSON.stringify({
			longFrames: s.longFrames.filter((f) => f.startTime >= since),
			interactions: s.interactions.filter((i) => i.startTime >= since),
			longFrameSource: s.longFrameSource,
			interactionsSupported: s.interactionsSupported,
		});
	}`, parseSinceMs)
	if parseErr != nil {
		parseT.Fatalf("read atlas perf snapshot: %v", parseErr)
	}
	parseSnapshot := atlasPerfSnapshot{}
	if parseDecodeErr := json.Unmarshal([]byte(parseRaw.(string)), &parseSnapshot); parseDecodeErr != nil {
		parseT.Fatalf("decode atlas perf snapshot: %v", parseDecodeErr)
	}
	return parseSnapshot
}

// atlasPerfNowMs reads the page clock so a measurement window can be bounded
// without trusting the observer buffer.
func atlasPerfNowMs(parseT *testing.T, parsePage playwright.Page) float64 {
	parseT.Helper()
	parseRaw, parseErr := parsePage.Evaluate(`() => performance.now()`)
	if parseErr != nil {
		parseT.Fatalf("read performance.now(): %v", parseErr)
	}
	return atlasPerfAsFloat(parseT, "performance.now()", parseRaw)
}

// atlasPerfLongFrameSummary reduces a set of entries to the shape M2 is stated in.
func atlasPerfLongFrameSummary(parseFrames []atlasPerfLongFrame) (parseCount int, parseWorstMs float64, parseTotalBlockingMs float64, parseStyleLayoutMs float64) {
	for _, parseFrame := range parseFrames {
		// M2 counts frames over 50 ms. LoAF reports every frame over 50 ms
		// already, but the longtask fallback and future threshold changes make
		// the explicit test worth keeping.
		if parseFrame.Duration <= 50 {
			continue
		}
		parseCount++
		if parseFrame.Duration > parseWorstMs {
			parseWorstMs = parseFrame.Duration
		}
		parseTotalBlockingMs += parseFrame.BlockingDuration
		parseStyleLayoutMs += parseFrame.StyleAndLayoutDuration
	}
	return parseCount, parseWorstMs, parseTotalBlockingMs, parseStyleLayoutMs
}

// -----------------------------------------------------------------------------
// Cold hydration
// -----------------------------------------------------------------------------

// atlasPerfHydrationRepeats is the number of cold hydrations per route.
//
// Three, not one: the first navigation of a browser session also pays for
// fetching and instantiating a ~20 MB wasm bundle on a cold HTTP cache, and a
// single reading on a throttling machine is worthless. Three is also the point
// where the run stops being cheap, so it is the honest minimum rather than a
// statistically comfortable number — the report quotes min/median/max, never a
// mean.
const atlasPerfHydrationRepeats = 3

// TestAtlasPerfBrowserColdHydration measures navigation-to-hydrated per route,
// and the long frames hydration itself produces.
//
// This is hypothesis 1: M6 records Initial Render as runtime1's dominant loss
// (-3.236 ms, 0/4 wins). The question is whether a real app's cold hydration has
// the same shape, and what inside it dominates.
func TestAtlasPerfBrowserColdHydration(parseT *testing.T) {
	parseBaseURL := atlasPerfBaseURL(parseT)

	for _, parseCase := range atlasPerfRouteCases {
		parseCase := parseCase
		parseT.Run(parseCase.Name, func(parseSub *testing.T) {
			parseSamples := make([]float64, 0, atlasPerfHydrationRepeats)
			parseFrameCounts := make([]int, 0, atlasPerfHydrationRepeats)
			parseWorstFrames := make([]float64, 0, atlasPerfHydrationRepeats)
			parseSource := "unknown"
			var parseAttribution []atlasPerfLongFrame

			for parseIndex := 0; parseIndex < atlasPerfHydrationRepeats; parseIndex++ {
				// A fresh page per repeat. Reusing one page would leave the
				// previous route's runtime, atoms and fold caches warm, which is a
				// different measurement (route navigation) and is measured
				// separately below.
				withExamplesPage(parseSub, func(parsePage playwright.Page) {
					parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
					if parseErr := parsePage.AddInitScript(playwright.Script{
						Content: playwright.String("(" + atlasPerfInstallObserversScript + ")()"),
					}); parseErr != nil {
						parseSub.Fatalf("install observers: %v", parseErr)
					}
					if parseCase.Role != "" {
						seedAtlasSpecOperatorCookie(parseSub, parsePage, parseBaseURL, parseCase.Role)
					}

					parseStartedAt := atlasPerfHydrationTiming(parseSub, parsePage, parseDiagnostics, parseBaseURL, parseCase.Route)
					parseSamples = append(parseSamples, parseStartedAt)

					parseSnapshot := readAtlasPerfSnapshot(parseSub, parsePage, 0)
					parseSource = parseSnapshot.LongFrameSource
					parseCount, parseWorst, _, _ := atlasPerfLongFrameSummary(parseSnapshot.LongFrames)
					parseFrameCounts = append(parseFrameCounts, parseCount)
					parseWorstFrames = append(parseWorstFrames, parseWorst)
					if parseIndex == 0 {
						parseAttribution = parseSnapshot.LongFrames
					}
				})
			}

			sort.Float64s(parseSamples)
			sort.Float64s(parseWorstFrames)
			sort.Ints(parseFrameCounts)
			parseSub.Logf("%s (%s) — %s", parseCase.Name, parseCase.Route, parseCase.Why)
			parseSub.Logf("  hydration ms   min=%.0f med=%.0f max=%.0f  (n=%d cold pages)",
				parseSamples[0], parseSamples[len(parseSamples)/2], parseSamples[len(parseSamples)-1], len(parseSamples))
			parseSub.Logf("  long frames >50ms  min=%d med=%d max=%d   worst frame med=%.1f max=%.1f ms   instrument=%s",
				parseFrameCounts[0], parseFrameCounts[len(parseFrameCounts)/2], parseFrameCounts[len(parseFrameCounts)-1],
				parseWorstFrames[len(parseWorstFrames)/2], parseWorstFrames[len(parseWorstFrames)-1], parseSource)
			atlasPerfLogFrameAttribution(parseSub, parseAttribution)
		})
	}
}

// atlasPerfHydrationTiming navigates and returns milliseconds from navigation
// start to the point where the client has painted the shell.
//
// It measures against the page's own navigation timing rather than a Go-side
// stopwatch, so Playwright IPC latency is excluded. The completion condition is
// the same one every Atlas spec uses: #app has children AND #atlas-shell-root
// exists. Waiting on route-specific text instead would confuse "not hydrated
// yet" with "hydrated the wrong route".
func atlasPerfHydrationTiming(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseBaseURL string, parseRoute string) float64 {
	parseT.Helper()
	gotoAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, parseRoute)
	parseRaw, parseErr := parsePage.WaitForFunction(`() => {
		const root = document.getElementById("app");
		if (!root || root.children.length === 0) return null;
		if (!document.getElementById("atlas-shell-root")) return null;
		return performance.now();
	}`, nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(atlasSpecHydrationTimeoutMS)})
	if parseErr != nil {
		parseT.Fatalf("route %s never hydrated: %v | %s", parseRoute, parseErr, parseDiagnostics.Summary())
	}
	parseValue, parseErr := parseRaw.JSONValue()
	if parseErr != nil {
		parseT.Fatalf("read hydration timestamp: %v", parseErr)
	}
	return atlasPerfAsFloat(parseT, "hydration timestamp", parseValue)
}

// atlasPerfAsFloat coerces a value that crossed the JSON boundary.
//
// Playwright decodes a JS number through JSON, and a whole-number result arrives
// as int, not float64. A type assertion to float64 alone fails on exactly the
// fast pages — the ones whose timings happen to land on an integer millisecond —
// which is a measurement bug that only shows up sometimes.
func atlasPerfAsFloat(parseT *testing.T, parseWhat string, parseValue interface{}) float64 {
	parseT.Helper()
	switch parseTyped := parseValue.(type) {
	case float64:
		return parseTyped
	case float32:
		return float64(parseTyped)
	case int:
		return float64(parseTyped)
	case int64:
		return float64(parseTyped)
	case json.Number:
		parseParsed, parseErr := parseTyped.Float64()
		if parseErr != nil {
			parseT.Fatalf("%s was %q, not a number: %v", parseWhat, parseTyped.String(), parseErr)
		}
		return parseParsed
	default:
		parseT.Fatalf("%s was %T, not a number", parseWhat, parseValue)
		return 0
	}
}

// atlasPerfBootBreakdown splits the boot window into the pieces that could each
// be the dominant cost, so "initial render is slow" can be resolved into a cause.
type atlasPerfBootBreakdown struct {
	// DocumentMs is navigationStart -> responseEnd for the HTML: the server's
	// share. Atlas sends an empty #app plus a bootstrap payload, so this is
	// small by construction and is measured to prove it is not the cost.
	DocumentMs float64 `json:"documentMs"`
	// WasmTransferBytes and WasmDurationMs come from Resource Timing for the
	// .wasm request. transferSize 0 with a nonzero encodedBodySize means the
	// bundle was served from cache.
	WasmTransferBytes float64 `json:"wasmTransferBytes"`
	WasmEncodedBytes  float64 `json:"wasmEncodedBytes"`
	WasmDurationMs    float64 `json:"wasmDurationMs"`
	WasmStartMs       float64 `json:"wasmStartMs"`
	WasmEndMs         float64 `json:"wasmEndMs"`
	// FirstPaintMs and FirstContentfulPaintMs bound when anything appeared.
	FirstPaintMs           float64 `json:"firstPaintMs"`
	FirstContentfulPaintMs float64 `json:"firstContentfulPaintMs"`
	// HydratedMs is when #atlas-shell-root existed with children in #app.
	HydratedMs float64 `json:"hydratedMs"`
}

// atlasPerfReadBootBreakdown must be called AFTER hydration completed.
func atlasPerfReadBootBreakdown(parseT *testing.T, parsePage playwright.Page, parseHydratedMs float64) atlasPerfBootBreakdown {
	parseT.Helper()
	parseRaw, parseErr := parsePage.Evaluate(`() => {
		const nav = performance.getEntriesByType("navigation")[0] || {};
		const wasm = performance.getEntriesByType("resource").filter((r) => r.name.endsWith(".wasm"));
		const biggest = wasm.sort((a, b) => (b.encodedBodySize || 0) - (a.encodedBodySize || 0))[0] || {};
		const paints = performance.getEntriesByType("paint");
		const fp = paints.find((p) => p.name === "first-paint");
		const fcp = paints.find((p) => p.name === "first-contentful-paint");
		return JSON.stringify({
			documentMs: nav.responseEnd || 0,
			wasmTransferBytes: biggest.transferSize || 0,
			wasmEncodedBytes: biggest.encodedBodySize || 0,
			wasmDurationMs: biggest.duration || 0,
			wasmStartMs: biggest.startTime || 0,
			wasmEndMs: (biggest.startTime || 0) + (biggest.duration || 0),
			firstPaintMs: fp ? fp.startTime : 0,
			firstContentfulPaintMs: fcp ? fcp.startTime : 0,
		});
	}`)
	if parseErr != nil {
		parseT.Fatalf("read boot breakdown: %v", parseErr)
	}
	parseBreakdown := atlasPerfBootBreakdown{}
	if parseDecodeErr := json.Unmarshal([]byte(parseRaw.(string)), &parseBreakdown); parseDecodeErr != nil {
		parseT.Fatalf("decode boot breakdown: %v", parseDecodeErr)
	}
	parseBreakdown.HydratedMs = parseHydratedMs
	return parseBreakdown
}

func (parseB atlasPerfBootBreakdown) String() string {
	parseCacheNote := "network"
	if parseB.WasmTransferBytes == 0 && parseB.WasmEncodedBytes > 0 {
		parseCacheNote = "HTTP cache"
	}
	return fmt.Sprintf(
		"document(responseEnd)=%.0f  wasm[%s %.1f MB]=%.0f..%.0f (%.0f ms)  FP=%.0f FCP=%.0f  hydrated=%.0f  "+
			"post-fetch tail=%.0f ms",
		parseB.DocumentMs,
		parseCacheNote,
		parseB.WasmEncodedBytes/(1024*1024),
		parseB.WasmStartMs, parseB.WasmEndMs, parseB.WasmDurationMs,
		parseB.FirstPaintMs, parseB.FirstContentfulPaintMs,
		parseB.HydratedMs,
		parseB.HydratedMs-parseB.WasmEndMs,
	)
}

// TestAtlasPerfBrowserWarmCacheHydration separates the wasm bundle's transfer
// cost from the framework's own boot cost.
//
// This is the load-bearing half of hypothesis 1. A cold-cache hydration of Atlas
// fetches a ~20 MB development bundle, so ANY per-route hydration number taken
// that way is dominated by transfer and instantiation and says nothing about
// reconciliation. Reloading inside the same browser context serves the bundle
// from the HTTP cache; the residue is document + instantiate + payload decode +
// first render + first commit.
//
// The "post-fetch tail" it reports — hydrated minus the wasm request's end — is
// the closest browser-side bound on framework boot cost available without an
// in-app probe.
func TestAtlasPerfBrowserWarmCacheHydration(parseT *testing.T) {
	parseBaseURL := atlasPerfBaseURL(parseT)

	for _, parseCase := range atlasPerfRouteCases {
		parseCase := parseCase
		parseT.Run(parseCase.Name, func(parseSub *testing.T) {
			withExamplesPage(parseSub, func(parsePage playwright.Page) {
				parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
				if parseErr := parsePage.AddInitScript(playwright.Script{
					Content: playwright.String("(" + atlasPerfInstallObserversScript + ")()"),
				}); parseErr != nil {
					parseSub.Fatalf("install observers: %v", parseErr)
				}
				if parseCase.Role != "" {
					seedAtlasSpecOperatorCookie(parseSub, parsePage, parseBaseURL, parseCase.Role)
				}

				// Load 1 primes the HTTP cache and is reported separately, never
				// averaged in with the warm loads.
				parseColdMs := atlasPerfHydrationTiming(parseSub, parsePage, parseDiagnostics, parseBaseURL, parseCase.Route)
				parseColdBreakdown := atlasPerfReadBootBreakdown(parseSub, parsePage, parseColdMs)
				parseSub.Logf("%s (%s)", parseCase.Name, parseCase.Route)
				parseSub.Logf("  cold cache : %s", parseColdBreakdown)

				parseWarm := make([]atlasPerfBootBreakdown, 0, atlasPerfHydrationRepeats)
				parseTails := make([]float64, 0, atlasPerfHydrationRepeats)
				parseHydrated := make([]float64, 0, atlasPerfHydrationRepeats)
				parseFrameCounts := make([]int, 0, atlasPerfHydrationRepeats)
				for parseIndex := 0; parseIndex < atlasPerfHydrationRepeats; parseIndex++ {
					parseSince := atlasPerfNowMs(parseSub, parsePage)
					_ = parseSince
					parseMs := atlasPerfHydrationTiming(parseSub, parsePage, parseDiagnostics, parseBaseURL, parseCase.Route)
					parseBreakdown := atlasPerfReadBootBreakdown(parseSub, parsePage, parseMs)
					parseWarm = append(parseWarm, parseBreakdown)
					parseTails = append(parseTails, parseBreakdown.HydratedMs-parseBreakdown.WasmEndMs)
					parseHydrated = append(parseHydrated, parseMs)
					// A navigation resets performance timing, so every entry the
					// observers hold belongs to this load only.
					parseSnapshot := readAtlasPerfSnapshot(parseSub, parsePage, 0)
					parseCount, _, _, _ := atlasPerfLongFrameSummary(parseSnapshot.LongFrames)
					parseFrameCounts = append(parseFrameCounts, parseCount)
				}
				for parseIndex, parseBreakdown := range parseWarm {
					parseSub.Logf("  warm load %d: %s", parseIndex+1, parseBreakdown)
				}
				sort.Float64s(parseTails)
				sort.Float64s(parseHydrated)
				sort.Ints(parseFrameCounts)
				parseSub.Logf("  WARM hydrated ms  min=%.0f med=%.0f max=%.0f", parseHydrated[0], parseHydrated[len(parseHydrated)/2], parseHydrated[len(parseHydrated)-1])
				parseSub.Logf("  WARM post-fetch tail ms  min=%.0f med=%.0f max=%.0f   long frames>50ms med=%d",
					parseTails[0], parseTails[len(parseTails)/2], parseTails[len(parseTails)-1], parseFrameCounts[len(parseFrameCounts)/2])
				parseSub.Logf("  cold-vs-warm hydrated: %.0f -> %.0f ms (%.0f%% of the cold number was bundle transfer)",
					parseColdMs, parseHydrated[len(parseHydrated)/2], 100*(1-parseHydrated[len(parseHydrated)/2]/parseColdMs))
			})
		})
	}
}

// atlasPerfLogFrameAttribution prints per-script attribution for the worst long
// frames, which is the whole reason LoAF is preferred over longtask.
func atlasPerfLogFrameAttribution(parseT *testing.T, parseFrames []atlasPerfLongFrame) {
	parseT.Helper()
	parseSorted := append([]atlasPerfLongFrame(nil), parseFrames...)
	sort.Slice(parseSorted, func(parseI, parseJ int) bool {
		return parseSorted[parseI].Duration > parseSorted[parseJ].Duration
	})
	for parseIndex, parseFrame := range parseSorted {
		if parseIndex >= 5 || parseFrame.Duration <= 50 {
			break
		}
		parseT.Logf("    frame %.1f ms  blocking=%.1f styleLayout=%.1f  at t=%.0f",
			parseFrame.Duration, parseFrame.BlockingDuration, parseFrame.StyleAndLayoutDuration, parseFrame.StartTime)
		for _, parseScript := range parseFrame.Scripts {
			parseT.Logf("        script %.1f ms  forcedStyleLayout=%.1f  invoker=%s  %s",
				parseScript.Duration, parseScript.ForcedStyleAndLayoutDuration, parseScript.Invoker, parseScript.Name)
		}
	}
}

// -----------------------------------------------------------------------------
// CPU profile of hydration
// -----------------------------------------------------------------------------

// TestAtlasPerfBrowserHydrationCPUProfile captures a V8 CPU profile across a cold
// Atlas hydration and buckets the samples by Go package.
//
// Go's js/wasm build keeps the name section, so wasm frames carry their Go symbol
// names. That makes the framework-vs-example split readable in the browser, which
// is the one thing the native lane cannot confirm about commit-phase work.
//
// The profiler is started BEFORE navigation, because the interesting work is
// instantiation plus first render and both happen before any DOM assertion could
// fire.
func TestAtlasPerfBrowserHydrationCPUProfile(parseT *testing.T) {
	parseBaseURL := atlasPerfBaseURL(parseT)

	for _, parseCase := range atlasPerfRouteCases {
		parseCase := parseCase
		parseT.Run(parseCase.Name, func(parseSub *testing.T) {
			withExamplesPage(parseSub, func(parsePage playwright.Page) {
				parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
				if parseErr := parsePage.AddInitScript(playwright.Script{
					Content: playwright.String("(" + atlasPerfInstallObserversScript + ")()"),
				}); parseErr != nil {
					parseSub.Fatalf("install observers: %v", parseErr)
				}
				if parseCase.Role != "" {
					seedAtlasSpecOperatorCookie(parseSub, parsePage, parseBaseURL, parseCase.Role)
				}

				parseSession, parseErr := parsePage.Context().NewCDPSession(parsePage)
				if parseErr != nil {
					parseSub.Fatalf("cdp session: %v", parseErr)
				}
				if _, parseEnableErr := parseSession.Send("Profiler.enable", nil); parseEnableErr != nil {
					parseSub.Fatalf("profiler enable: %v", parseEnableErr)
				}
				// 100 µs. Hydration is a ~1 s window, so the default 1 ms
				// interval would yield ~1000 samples spread over instantiation,
				// decode and render — too coarse to separate them.
				if _, parseIntervalErr := parseSession.Send("Profiler.setSamplingInterval", map[string]interface{}{"interval": 100}); parseIntervalErr != nil {
					parseSub.Fatalf("profiler interval: %v", parseIntervalErr)
				}
				if _, parseStartErr := parseSession.Send("Profiler.start", nil); parseStartErr != nil {
					parseSub.Fatalf("profiler start: %v", parseStartErr)
				}

				atlasPerfHydrationTiming(parseSub, parsePage, parseDiagnostics, parseBaseURL, parseCase.Route)

				parseResult, parseStopErr := parseSession.Send("Profiler.stop", nil)
				if parseStopErr != nil {
					parseSub.Fatalf("profiler stop: %v", parseStopErr)
				}
				atlasPerfReportProfile(parseSub, parseCase.Name+" cold hydration", parseResult)
			})
		})
	}
}

// atlasPerfAssertOperatorShell proves the operator rail is present WITHOUT
// reading layout.
//
// Reading layout is not neutral on this page. A run that called
// assertAtlasSpecOperatorSessionVisible (which uses Playwright InnerText, and so
// forces style and layout) then found 9 anchors where the previous statement had
// counted 33, repeatably; the same page sampled for six seconds with no layout
// read never left 33. So the observation changed the subject, and any number
// taken after it describes a different tree.
func atlasPerfAssertOperatorShell(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) {
	parseT.Helper()
	parseRaw, parseErr := parsePage.Evaluate(`() => JSON.stringify({
		anchors: document.querySelectorAll("a[href]").length,
		operatorLinks: document.querySelectorAll("a[href^='/app/']").length,
		shell: !!document.getElementById("atlas-shell-root"),
		path: location.pathname,
	})`)
	if parseErr != nil {
		parseT.Fatalf("read operator shell state: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	var parseState struct {
		Anchors       int    `json:"anchors"`
		OperatorLinks int    `json:"operatorLinks"`
		Shell         bool   `json:"shell"`
		Path          string `json:"path"`
	}
	if parseDecodeErr := json.Unmarshal([]byte(parseRaw.(string)), &parseState); parseDecodeErr != nil {
		parseT.Fatalf("decode operator shell state: %v", parseDecodeErr)
	}
	if !parseState.Shell || parseState.OperatorLinks < 5 {
		parseT.Fatalf("not on an authenticated operator shell: path=%s shellRoot=%v anchors=%d operatorLinks=%d | %s",
			parseState.Path, parseState.Shell, parseState.Anchors, parseState.OperatorLinks, parseDiagnostics.Summary())
	}
	parseT.Logf("operator shell confirmed: path=%s anchors=%d operatorLinks=%d",
		parseState.Path, parseState.Anchors, parseState.OperatorLinks)
}

// TestAtlasPerfBrowserShellStability samples the operator shell after hydration
// and reports whether it stays put.
//
// It exists because the steady-state profile could not be driven at all until
// this was measured. Attempting six in-app route swaps on a hydrated
// /app/dashboard failed with the nav rail simply not being there: a probe that
// read 33 anchors, then started polling, saw 0 anchors and then 9 for 189
// consecutive polls over 10 s, with zero console errors and zero page errors.
// The first guess was that the V8 profiler was to blame; moving the swap ahead
// of the profiler reproduced it identically, which refuted that.
//
// So this test asks the question directly: what does the shell look like at
// t+0.2 s, t+0.4 s … after it hydrates? A perf harness that cannot state whether
// its subject is stable cannot interpret any of its own numbers.
func TestAtlasPerfBrowserShellStability(parseT *testing.T) {
	parseBaseURL := atlasPerfBaseURL(parseT)

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
		seedAtlasSpecOperatorCookie(parseT, parsePage, parseBaseURL, "inventory_manager")
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/dashboard", 0)

		parseRaw, parseErr := parsePage.Evaluate(`async () => {
			const samples = [];
			for (let i = 0; i < 30; i++) {
				samples.push({
					t: Math.round(performance.now()),
					anchors: document.querySelectorAll("a[href]").length,
					internalRail: document.querySelectorAll("a[href^='/app/']").length,
					shell: !!document.getElementById("atlas-shell-root"),
					appChildren: document.getElementById("app") ? document.getElementById("app").children.length : -1,
					bodyChars: (document.body.innerText || "").length,
					path: location.pathname,
				});
				await new Promise((r) => setTimeout(r, 200));
			}
			return JSON.stringify(samples);
		}`)
		if parseErr != nil {
			parseT.Fatalf("sample shell stability: %v | %s", parseErr, parseDiagnostics.Summary())
		}
		var parseSamples []struct {
			T            int    `json:"t"`
			Anchors      int    `json:"anchors"`
			InternalRail int    `json:"internalRail"`
			Shell        bool   `json:"shell"`
			AppChildren  int    `json:"appChildren"`
			BodyChars    int    `json:"bodyChars"`
			Path         string `json:"path"`
		}
		if parseDecodeErr := json.Unmarshal([]byte(parseRaw.(string)), &parseSamples); parseDecodeErr != nil {
			parseT.Fatalf("decode stability samples: %v", parseDecodeErr)
		}
		parseT.Log("/app/dashboard shell over 6 s after hydration (200 ms sampling):")
		parseLastAnchors := -1
		for _, parseSample := range parseSamples {
			if parseSample.Anchors != parseLastAnchors {
				parseT.Logf("  t=%5d ms  anchors=%3d  /app/ links=%3d  shellRoot=%v  #app children=%d  bodyChars=%d  path=%s",
					parseSample.T, parseSample.Anchors, parseSample.InternalRail, parseSample.Shell,
					parseSample.AppChildren, parseSample.BodyChars, parseSample.Path)
				parseLastAnchors = parseSample.Anchors
			}
		}
		parseT.Logf("browser diagnostics: %s", parseDiagnostics.Summary())
	})
}

// TestAtlasPerfBrowserSteadyStateCPUProfile profiles the STEADY-STATE render
// path: an app that has already booted, doing in-app route swaps.
//
// This is the profile that matters for M2 and for M6's update categories, and it
// is a different measurement from the cold-hydration profile above. A cold
// hydration is dominated by one-time costs — wasm compilation, Go package
// initializers, the first payload decode — none of which any reconciler change
// touches. Profiling only the cold path and then recommending reconciler work
// would be exactly the reasoning docs/PRODUCTION_READINESS.md records being
// wrong by 15x.
//
// The swap loop alternates two routes with very different payload sizes (the
// 88-row inbox and the dense dashboard) so mount, unmount, deletion and order
// repair all run repeatedly.
func TestAtlasPerfBrowserSteadyStateCPUProfile(parseT *testing.T) {
	parseBaseURL := atlasPerfBaseURL(parseT)

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
		if parseErr := parsePage.AddInitScript(playwright.Script{
			Content: playwright.String("(" + atlasPerfInstallObserversScript + ")()"),
		}); parseErr != nil {
			parseT.Fatalf("install observers: %v", parseErr)
		}
		seedAtlasSpecOperatorCookie(parseT, parsePage, parseBaseURL, "inventory_manager")
		// /app/inventory, not /app/dashboard: the dashboard has no text input at
		// all (measured — the focus helper reported "no text input on
		// .../app/dashboard"), so there is nothing there to drive a keystroke
		// re-render with. The inventory surface carries the filter field.
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/inventory", 0)
		atlasSpecSettleBriefly(parsePage)
		atlasPerfAssertOperatorShell(parseT, parsePage, parseDiagnostics)

		parseSession, parseErr := parsePage.Context().NewCDPSession(parsePage)
		if parseErr != nil {
			parseT.Fatalf("cdp session: %v", parseErr)
		}
		if _, parseEnableErr := parseSession.Send("Profiler.enable", nil); parseEnableErr != nil {
			parseT.Fatalf("profiler enable: %v", parseEnableErr)
		}
		if _, parseIntervalErr := parseSession.Send("Profiler.setSamplingInterval", map[string]interface{}{"interval": 200}); parseIntervalErr != nil {
			parseT.Fatalf("profiler interval: %v", parseIntervalErr)
		}

		// WHAT IS PROFILED HERE, AND WHY IT IS NOT ROUTE SWAPS
		//
		// The first design profiled repeated in-app route swaps. It could not be
		// made to run. After a client-side swap to /app/comments the operator nav
		// rail is present when page.WaitForFunction polls for it and ABSENT to a
		// single-shot query taken moments later — Playwright Locators and
		// main-world page.Evaluate agreed it was absent, so this is not a
		// selector-engine artefact. Anchor counts on a swapped-to route were
		// observed at 29, then 9, then 0, with zero console errors and zero page
		// errors, and it reproduced with the profiler removed. Cause unidentified;
		// it is recorded as an open question in the report rather than guessed at.
		//
		// Typing is profiled instead, and it is a better subject for the question
		// M6 poses anyway: keystroke-to-re-render is the fine-grained update
		// category runtime1 already WINS, so this profile shows where time goes on
		// the path that works, to be read against the cold-hydration profile of
		// the path that does not.
		parseSince := atlasPerfNowMs(parseT, parsePage)
		atlasPerfFocusFirstInput(parseT, parsePage, parseDiagnostics)
		if _, parseStartErr := parseSession.Send("Profiler.start", nil); parseStartErr != nil {
			parseT.Fatalf("profiler start: %v", parseStartErr)
		}
		const parseKeystrokes = 40
		if parseTypeErr := parsePage.Keyboard().Type(strings.Repeat("desk ", parseKeystrokes/5), playwright.KeyboardTypeOptions{
			Delay: playwright.Float(25),
		}); parseTypeErr != nil {
			parseT.Fatalf("type into the first text input: %v | %s", parseTypeErr, parseDiagnostics.Summary())
		}
		parseResult, parseStopErr := parseSession.Send("Profiler.stop", nil)
		if parseStopErr != nil {
			parseT.Fatalf("profiler stop: %v", parseStopErr)
		}

		parseSnapshot := readAtlasPerfSnapshot(parseT, parsePage, parseSince)
		parseCount, parseWorst, parseBlocking, parseStyleLayout := atlasPerfLongFrameSummary(parseSnapshot.LongFrames)
		parseT.Logf("steady-state: %d keystrokes into the inventory filter field", parseKeystrokes)
		parseT.Logf("  long frames>50ms=%d worst=%.1f ms totalBlocking=%.1f ms totalStyleLayout=%.1f ms",
			parseCount, parseWorst, parseBlocking, parseStyleLayout)
		atlasPerfLogFrameAttribution(parseT, parseSnapshot.LongFrames)
		atlasPerfReportProfile(parseT, fmt.Sprintf("steady state, %d keystrokes", parseKeystrokes), parseResult)
		parseT.Logf("browser diagnostics: %s", parseDiagnostics.Summary())
	})
}

// atlasPerfFocusFirstInput focuses the first text input without reading layout.
func atlasPerfFocusFirstInput(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) {
	parseT.Helper()
	parseRaw, parseErr := parsePage.Evaluate(`() => {
		const field = document.querySelector("input[type=text], input[type=search], input:not([type])");
		if (!field) return "";
		field.focus();
		return field.getAttribute("name") || field.getAttribute("placeholder") || "(unnamed input)";
	}`)
	if parseErr != nil {
		parseT.Fatalf("focus the first text input: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	parseName, _ := parseRaw.(string)
	if parseName == "" {
		parseT.Fatalf("no text input on %s to type into | %s", parsePage.URL(), parseDiagnostics.Summary())
	}
	parseT.Logf("typing into %s", parseName)
}

// atlasPerfReportProfile ranks a CPU profile by self time and buckets it by Go
// package.
//
// The bucketing is the point. A flat list of 45 function names does not answer
// "how much of this is the framework"; a package fold does, and it is the same
// question docs/PRODUCTION_READINESS.md shows a review getting wrong by 15x when
// it guessed instead of measuring.
func atlasPerfReportProfile(parseT *testing.T, parseLabel string, parseResult interface{}) {
	parseT.Helper()
	parseJSON, parseErr := json.Marshal(parseResult)
	if parseErr != nil {
		parseT.Fatalf("marshal profile: %v", parseErr)
	}
	var parsePayload struct {
		Profile struct {
			Nodes []struct {
				HitCount  int `json:"hitCount"`
				CallFrame struct {
					FunctionName string `json:"functionName"`
					URL          string `json:"url"`
				} `json:"callFrame"`
			} `json:"nodes"`
			StartTime float64 `json:"startTime"`
			EndTime   float64 `json:"endTime"`
		} `json:"profile"`
	}
	if parseDecodeErr := json.Unmarshal(parseJSON, &parsePayload); parseDecodeErr != nil {
		parseT.Fatalf("decode profile: %v", parseDecodeErr)
	}

	parseTotal := 0
	parseByFunction := map[string]int{}
	parseByBucket := map[string]int{}
	for _, parseNode := range parsePayload.Profile.Nodes {
		if parseNode.HitCount == 0 {
			continue
		}
		parseName := parseNode.CallFrame.FunctionName
		if parseName == "" {
			parseName = "(anonymous)"
		}
		parseTotal += parseNode.HitCount
		parseByFunction[parseName] += parseNode.HitCount
		parseByBucket[atlasPerfBucketForSymbol(parseName)] += parseNode.HitCount
	}
	if parseTotal == 0 {
		parseT.Fatalf("%s: profile recorded zero samples — the profiler was not running over the measured window", parseLabel)
	}

	parseDurationMs := (parsePayload.Profile.EndTime - parsePayload.Profile.StartTime) / 1000
	parseT.Logf("%s — %d samples over %.0f ms", parseLabel, parseTotal, parseDurationMs)

	type entry struct {
		Name string
		Hits int
	}
	parseBuckets := make([]entry, 0, len(parseByBucket))
	for parseName, parseHits := range parseByBucket {
		parseBuckets = append(parseBuckets, entry{Name: parseName, Hits: parseHits})
	}
	sort.Slice(parseBuckets, func(parseI, parseJ int) bool { return parseBuckets[parseI].Hits > parseBuckets[parseJ].Hits })
	parseT.Log("  by bucket (self time):")
	for _, parseBucket := range parseBuckets {
		parseT.Logf("    %6.2f%% %7d  %s", 100*float64(parseBucket.Hits)/float64(parseTotal), parseBucket.Hits, parseBucket.Name)
	}

	parseFunctions := make([]entry, 0, len(parseByFunction))
	for parseName, parseHits := range parseByFunction {
		parseFunctions = append(parseFunctions, entry{Name: parseName, Hits: parseHits})
	}
	sort.Slice(parseFunctions, func(parseI, parseJ int) bool { return parseFunctions[parseI].Hits > parseFunctions[parseJ].Hits })
	parseT.Log("  top functions (self time):")
	for parseIndex, parseFunction := range parseFunctions {
		if parseIndex >= 30 {
			break
		}
		parseT.Logf("    %6.2f%% %7d  %s", 100*float64(parseFunction.Hits)/float64(parseTotal), parseFunction.Hits, parseFunction.Name)
	}
}

// atlasPerfBucketForSymbol maps a profile symbol to an attribution bucket.
//
// TWO THINGS THIS GOT WRONG THE FIRST TIME, both worth keeping in the comment
// because they are silent failures:
//
//  1. Go's js/wasm name section MANGLES package paths: "/" becomes "_" and "-"
//     becomes "__", so internal/runtime appears as
//     "github.com_monstercameron_GoWebComponents_v5_internal_runtime.…" and
//     shared/atlas as "…examples_server_atlas__commerce__os_shared_atlas.…".
//     Matching on "internal/runtime" therefore matched NOTHING and every
//     framework frame fell into the default bucket. The fix is to normalize "/"
//     to "_" in the needle, not to guess the mangling.
//  2. The order of the tests matters: shared/atlas must be checked before the
//     generic v5 prefix, because an Atlas symbol's path contains the module path
//     too and would otherwise be attributed to the framework.
func atlasPerfBucketForSymbol(parseSymbol string) string {
	// Normalize the HAYSTACK once so both the mangled wasm form and any
	// slash-bearing form match the same underscore needles.
	parseNormalized := strings.ReplaceAll(parseSymbol, "/", "_")
	switch {
	case strings.Contains(parseNormalized, "shared_atlas") || strings.Contains(parseNormalized, "shared_design") ||
		strings.Contains(parseNormalized, "atlas__commerce__os"):
		return "EXAMPLE  atlas app (shared/atlas, shared/design, client)"
	case strings.Contains(parseNormalized, "internal_runtime"):
		return "FRAMEWORK internal/runtime (fibers, reconcile, commit, hooks)"
	case strings.Contains(parseNormalized, "GoWebComponents_v5_ui") || strings.Contains(parseNormalized, "GoWebComponents_v5_html"):
		return "FRAMEWORK ui + html (elements, props, serialization)"
	case strings.Contains(parseNormalized, "GoWebComponents_v5_css"):
		return "FRAMEWORK css (typed-CSS folding)"
	case strings.Contains(parseNormalized, "GoWebComponents_v5"):
		return "FRAMEWORK other v5 packages"
	case strings.Contains(parseNormalized, "encoding_json"):
		return "STDLIB encoding/json (payload decode)"
	case strings.Contains(parseNormalized, "syscall_js") || parseSymbol == "wasm-to-js" || parseSymbol == "js-to-wasm" ||
		parseSymbol == "loadString" || parseSymbol == "loadValue" || parseSymbol == "loadSliceOfValues" ||
		parseSymbol == "storeValue" || parseSymbol == "makeValue" || parseSymbol == "mem" ||
		parseSymbol == "setInt64" || parseSymbol == "getInt64" || parseSymbol == "strPtr":
		// The Go<->JS boundary, both halves: the Go side (syscall/js) and the
		// wasm_exec.js glue that marshals across it. Kept as one bucket because
		// no DOM write happens without paying both.
		return "GO<->JS BRIDGE (syscall/js + wasm_exec glue)"
	case strings.HasPrefix(parseSymbol, "runtime.") || strings.HasPrefix(parseNormalized, "runtime_") ||
		parseSymbol == "wasm_pc_f_loop" || parseSymbol == "wasm_export_run" || parseSymbol == "wasm_export_resume":
		return "GO RUNTIME (GC, alloc, scheduler, wasm trampoline)"
	case strings.Contains(parseNormalized, "regexp"):
		// Broken out rather than folded into stdlib: regexp compilation at boot
		// is a package-init cost, not a render cost, and conflating them would
		// inflate whatever bucket absorbed it.
		return "STDLIB regexp (package-init compilation)"
	case strings.HasSuffix(parseSymbol, ".init") || strings.Contains(parseSymbol, ".init.") ||
		strings.Contains(parseSymbol, "map.init") || strings.HasPrefix(parseSymbol, "unicode."):
		// Go package initializers. A boot-only cost, addressable by pruning
		// dependencies rather than by touching the render path — so it must not
		// be allowed to inflate a render bucket.
		return "PACKAGE INIT (Go package initializers)"
	case strings.HasPrefix(parseSymbol, "reflect."):
		return "STDLIB reflect"
	case strings.HasPrefix(parseSymbol, "strings.") || strings.HasPrefix(parseSymbol, "bytes.") ||
		strings.HasPrefix(parseSymbol, "internal_bytealg") || strings.HasPrefix(parseSymbol, "sort.") ||
		strings.HasPrefix(parseSymbol, "fmt.") || strings.HasPrefix(parseSymbol, "strconv.") ||
		strings.HasPrefix(parseSymbol, "internal_strconv"):
		return "STDLIB strings/sort/fmt/strconv"
	case parseSymbol == "(garbage collector)" || parseSymbol == "(program)" || parseSymbol == "(idle)" ||
		parseSymbol == "(root)" || parseSymbol == "(anonymous)" || parseSymbol == "(parser)":
		return "V8 " + parseSymbol
	case strings.HasPrefix(parseSymbol, "wasm-function"):
		// Unnamed wasm frames. If this bucket is large the name section was
		// stripped and the whole attribution above is unreliable; say so rather
		// than presenting a confident split built on it.
		return "UNATTRIBUTED wasm-function[n] (no name section)"
	default:
		return "STDLIB + other Go"
	}
}

func atlasPerfShortSymbol(parseSymbol string) string {
	if len(parseSymbol) <= 24 {
		return parseSymbol
	}
	return parseSymbol[:24] + "…"
}

// -----------------------------------------------------------------------------
// Interactions
// -----------------------------------------------------------------------------

// TestAtlasPerfBrowserInteractions drives real Atlas interactions on one warm
// page and reports long frames plus interaction latency for each.
//
// One page, in sequence, on purpose: this is the "warm app, user does things"
// regime M2 and M3 are about, and it is a different regime from cold hydration.
// The interactions are chosen to exercise different framework paths — a
// client-side route swap (mount + unmount + commit), a filter over a list
// (reconcile + order repair), and a form round trip (effects + navigation).
func TestAtlasPerfBrowserInteractions(parseT *testing.T) {
	parseBaseURL := atlasPerfBaseURL(parseT)

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
		if parseErr := parsePage.AddInitScript(playwright.Script{
			Content: playwright.String("(" + atlasPerfInstallObserversScript + ")()"),
		}); parseErr != nil {
			parseT.Fatalf("install observers: %v", parseErr)
		}
		seedAtlasSpecOperatorCookie(parseT, parsePage, parseBaseURL, "inventory_manager")

		// Warm the app once. Everything measured below is post-hydration, so
		// including the boot frame would put a one-off cost into every scenario.
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/dashboard", 0)
		atlasSpecSettleBriefly(parsePage)

		parseScenarios := []struct {
			Name string
			Why  string
			Run  func()
		}{
			{
				Name: "route-swap dashboard -> comments",
				Why:  "client-side navigation: unmount + mount + full commit of the 88-row inbox",
				Run: func() {
					atlasPerfClickNavTo(parseT, parsePage, parseDiagnostics, "/app/comments")
					waitForAtlasSpecCondition(parseT, parsePage, parseDiagnostics,
						"the comments inbox has rows",
						`() => document.querySelectorAll('[data-comment-id]').length > 0 || (document.body.innerText || '').includes('comment')`)
				},
			},
			{
				Name: "route-swap comments -> inventory",
				Why:  "second navigation, warm runtime — separates first-swap cost from steady-state swap cost",
				Run: func() {
					atlasPerfClickNavTo(parseT, parsePage, parseDiagnostics, "/app/inventory")
					atlasSpecSettleBriefly(parsePage)
				},
			},
			{
				Name: "typing into the first text input",
				Why:  "keystroke-per-render path; real keypresses so Event Timing assigns an interactionId",
				Run:  func() { atlasPerfTypeIntoFirstInput(parseT, parsePage) },
			},
			{
				Name: "route-swap inventory -> settings",
				Why:  "form surface mount",
				Run: func() {
					atlasPerfClickNavTo(parseT, parsePage, parseDiagnostics, "/app/settings")
					atlasSpecSettleBriefly(parsePage)
				},
			},
		}

		parseAllInteractions := []atlasPerfInteraction{}
		for _, parseScenario := range parseScenarios {
			parseSince := atlasPerfNowMs(parseT, parsePage)
			parseScenario.Run()
			atlasSpecSettleBriefly(parsePage)
			parseSnapshot := readAtlasPerfSnapshot(parseT, parsePage, parseSince)
			parseCount, parseWorst, parseBlocking, parseStyleLayout := atlasPerfLongFrameSummary(parseSnapshot.LongFrames)
			parseT.Logf("%s — %s", parseScenario.Name, parseScenario.Why)
			parseT.Logf("  long frames>50ms=%d worst=%.1f ms blocking=%.1f ms styleLayout=%.1f ms  interactions=%d",
				parseCount, parseWorst, parseBlocking, parseStyleLayout, len(parseSnapshot.Interactions))
			atlasPerfLogFrameAttribution(parseT, parseSnapshot.LongFrames)
			parseAllInteractions = append(parseAllInteractions, parseSnapshot.Interactions...)
		}

		atlasPerfReportInteractionLatency(parseT, parseAllInteractions)
		// Diagnostics last: a console error during a perf run invalidates the run
		// as surely as it invalidates a functional one, and the summary is the
		// only thing that names the cause.
		parseT.Logf("browser diagnostics: %s", parseDiagnostics.Summary())
	})
}

// atlasPerfClickNavTo clicks the in-app link for a path.
//
// href-attribute selector, never a text filter: Playwright text filters
// substring-match, so "FR" matches "StoreFRont" and a nav test can click the
// wrong element while looking correct.
// It tries the exact href first, then a suffix match (Atlas builds some hrefs
// with query strings appended), and on failure DUMPS the hrefs that are actually
// present. A nav probe that fails with "timeout waiting for a[href=...]" costs a
// full debugging cycle to learn what the page offered instead.
func atlasPerfClickNavTo(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parsePath string) {
	parseT.Helper()
	parseCandidates := []string{
		fmt.Sprintf(`a[href="%s"]`, parsePath),
		fmt.Sprintf(`a[href^="%s?"]`, parsePath),
		fmt.Sprintf(`a[href^="%s"]`, parsePath),
	}
	for _, parseSelector := range parseCandidates {
		parseLocator := parsePage.Locator(parseSelector).First()
		parseCount, parseCountErr := parsePage.Locator(parseSelector).Count()
		if parseCountErr != nil || parseCount == 0 {
			continue
		}
		if parseErr := parseLocator.Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(atlasSpecActionTimeoutMS),
		}); parseErr != nil {
			parseT.Fatalf("click %s: %v | %s", parseSelector, parseErr, parseDiagnostics.Summary())
		}
		return
	}
	parseT.Fatalf("no in-app link to %s on %s. Links present: %s | %s",
		parsePath, parsePage.URL(), atlasPerfVisibleHrefs(parseT, parsePage), parseDiagnostics.Summary())
}

// atlasPerfClickNavViaEvaluate clicks an in-app link from the page's MAIN world.
//
// WHY A SECOND CLICK PATH EXISTS — this cost a debugging cycle and is not
// documented anywhere else in this repo:
//
//	While CDP `Profiler.start` is active, Playwright's LOCATOR engine stops
//	matching. Locator(...).Count() returns 0 for a selector that
//	page.Evaluate(document.querySelectorAll) resolves to 33 elements in the same
//	instant — measured, not inferred (the probe that found it logged both numbers
//	back to back). Playwright resolves selectors in an isolated utility world;
//	main-world Evaluate is unaffected.
//
// Consequence for measurement, stated because it is a real limitation: an
// element.click() from script is an UNTRUSTED event, so Event Timing assigns it
// no interactionId and a profiled run records no interaction latency. That is why
// interaction latency is measured in TestAtlasPerfBrowserInteractions, which uses
// real Playwright input and no profiler, and the profiled test measures only
// where the time goes.
func atlasPerfClickNavViaEvaluate(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parsePath string) {
	parseT.Helper()
	// ONE async evaluate that polls and clicks inside the page.
	//
	// This shape is not stylistic. A synchronous `() => {...}` evaluate that read
	// document.querySelectorAll("a") returned ZERO anchors while the statement
	// immediately before it — an identical query issued through a separate
	// evaluate — returned 33, on a shell that a 6-second sampling probe showed to
	// be completely stable at 33. The failure reproduced with and without the V8
	// profiler running, with and without CSS attribute selectors, and with and
	// without evaluate arguments; the only thing that fixed it was doing the look
	// and the click inside a single async evaluate that yields to the event loop
	// between attempts. Root cause unidentified — recorded as a harness hazard
	// rather than explained, because guessing at it is how a measurement pass
	// produces a confident wrong answer.
	parseExpression := fmt.Sprintf(`async () => {
		const path = %q;
		for (let attempt = 0; attempt < 100; attempt++) {
			const anchors = Array.from(document.querySelectorAll("a"));
			let target = null;
			for (const anchor of anchors) {
				const href = anchor.getAttribute("href") || "";
				if (href === path) { target = anchor; break; }
				if (target === null && href.indexOf(path) === 0) { target = anchor; }
			}
			if (target !== null) {
				target.click();
				return JSON.stringify({ found: true, attempts: attempt + 1, anchors: anchors.length });
			}
			await new Promise((resolve) => setTimeout(resolve, 50));
		}
		return JSON.stringify({ found: false, attempts: 100, anchors: document.querySelectorAll("a").length });
	}`, parsePath)
	parseRaw, parseErr := parsePage.Evaluate(parseExpression)
	if parseErr != nil {
		parseT.Fatalf("click %s via evaluate: %v | %s", parsePath, parseErr, parseDiagnostics.Summary())
	}
	parseText, parseIsText := parseRaw.(string)
	if !parseIsText {
		parseT.Fatalf("click %s via evaluate returned %T, not the JSON report string | %s",
			parsePath, parseRaw, parseDiagnostics.Summary())
	}
	var parseReport struct {
		Found    bool `json:"found"`
		Attempts int  `json:"attempts"`
		Anchors  int  `json:"anchors"`
	}
	if parseDecodeErr := json.Unmarshal([]byte(parseText), &parseReport); parseDecodeErr != nil {
		parseT.Fatalf("decode click report %q: %v", parseText, parseDecodeErr)
	}
	if !parseReport.Found {
		parseT.Fatalf("no in-app link to %s after %d in-page attempts; last look saw %d anchors at %s | %s",
			parsePath, parseReport.Attempts, parseReport.Anchors, parsePage.URL(), parseDiagnostics.Summary())
	}
}

// atlasPerfVisibleHrefs lists the internal hrefs on the page, deduplicated.
func atlasPerfVisibleHrefs(parseT *testing.T, parsePage playwright.Page) string {
	parseT.Helper()
	parseRaw, parseErr := parsePage.Evaluate(`() => {
		const seen = new Set();
		for (const a of document.querySelectorAll("a[href^='/']")) seen.add(a.getAttribute("href"));
		return Array.from(seen).sort().join(" ");
	}`)
	if parseErr != nil {
		return "(could not read hrefs: " + parseErr.Error() + ")"
	}
	parseText, _ := parseRaw.(string)
	return parseText
}

// atlasPerfTypeIntoFirstInput types real keystrokes into whichever text input the
// current surface offers.
//
// Deliberately not tied to a named field: Atlas is mid-conversion and field names
// move, and a perf probe that fails because a placeholder changed reports nothing
// about performance. If there is no input, it says so rather than passing
// silently.
func atlasPerfTypeIntoFirstInput(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	parseLocator := parsePage.Locator(`input[type="text"], input[type="search"], input:not([type])`).First()
	parseCount, parseErr := parsePage.Locator(`input[type="text"], input[type="search"], input:not([type])`).Count()
	if parseErr != nil {
		parseT.Fatalf("count text inputs: %v", parseErr)
	}
	if parseCount == 0 {
		parseT.Log("  (no text input on this surface — typing scenario measured nothing, and is reported as such)")
		return
	}
	if parseClickErr := parseLocator.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(atlasSpecActionTimeoutMS),
	}); parseClickErr != nil {
		parseT.Fatalf("focus the first text input: %v", parseClickErr)
	}
	// Real keyboard input. page.Keyboard.Type dispatches trusted events, which is
	// what Event Timing requires; a JS-dispatched input event records nothing.
	if parseTypeErr := parsePage.Keyboard().Type("desk", playwright.KeyboardTypeOptions{
		Delay: playwright.Float(60),
	}); parseTypeErr != nil {
		parseT.Fatalf("type into the first text input: %v", parseTypeErr)
	}
}

// atlasPerfReportInteractionLatency reports M3's statistics, and refuses to quote
// a percentile the sample cannot support.
//
// The v4 baseline produced n=56 and its guard correctly refused p95. Repeating
// that refusal here is the difference between a number and a number-shaped
// guess.
func atlasPerfReportInteractionLatency(parseT *testing.T, parseInteractions []atlasPerfInteraction) {
	parseT.Helper()
	if len(parseInteractions) == 0 {
		parseT.Log("interaction latency: n=0 — nothing to report. If this is unexpected, the events were not " +
			"trusted (Event Timing assigns interactionId only to real input) or every interaction was under " +
			"the 16 ms durationThreshold.")
		return
	}
	parseDurations := make([]float64, 0, len(parseInteractions))
	parseWorst := atlasPerfInteraction{}
	for _, parseInteraction := range parseInteractions {
		parseDurations = append(parseDurations, parseInteraction.Duration)
		if parseInteraction.Duration > parseWorst.Duration {
			parseWorst = parseInteraction
		}
	}
	sort.Float64s(parseDurations)
	parseT.Logf("interaction latency: n=%d  median=%.0f ms  max=%.0f ms  (Chrome rounds duration to 8 ms)",
		len(parseDurations), parseDurations[len(parseDurations)/2], parseDurations[len(parseDurations)-1])
	parseT.Logf("  worst interaction %q: total=%.0f inputDelay=%.0f processing=%.0f presentation=%.0f ms",
		parseWorst.Name, parseWorst.Duration, parseWorst.InputDelay, parseWorst.ProcessingTime, parseWorst.PresentationDelay)
	switch {
	case len(parseDurations) >= 200:
		parseT.Logf("  p95=%.0f ms (n>=200, quotable against M3's 50 ms gate)", parseDurations[(len(parseDurations)*95)/100])
	default:
		parseT.Logf("  p95 NOT QUOTED: n=%d is below the 200 the tail-reliability guard requires. "+
			"The max above is gated independently and is quotable.", len(parseDurations))
	}
}

// -----------------------------------------------------------------------------
// The 88-item inbox (hypothesis 3)
// -----------------------------------------------------------------------------

// TestAtlasPerfBrowserCommentsInboxShape answers whether the largest real list in
// the app is actually rendered in full, and whether the repo's virtualization
// package is anywhere near it.
//
// This matters because "the 88-item inbox produces long frames" is only a
// framework finding if all 88 rows really reach the DOM. If Atlas paginates or
// truncates, the long frames come from somewhere else and virtualization is not
// the lever.
func TestAtlasPerfBrowserCommentsInboxShape(parseT *testing.T) {
	parseBaseURL := atlasPerfBaseURL(parseT)

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
		if parseErr := parsePage.AddInitScript(playwright.Script{
			Content: playwright.String("(" + atlasPerfInstallObserversScript + ")()"),
		}); parseErr != nil {
			parseT.Fatalf("install observers: %v", parseErr)
		}
		seedAtlasSpecOperatorCookie(parseT, parsePage, parseBaseURL, "inventory_manager")
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/comments", 0)
		atlasSpecSettleBriefly(parsePage)

		parseRaw, parseErr := parsePage.Evaluate(`() => {
			const shell = document.getElementById("atlas-shell-root");
			const all = document.querySelectorAll("*").length;
			const inShell = shell ? shell.querySelectorAll("*").length : 0;
			// Rows are counted several ways because the markup is mid-conversion
			// and one selector going stale must not read as "no rows".
			const byData = document.querySelectorAll("[data-comment-id]").length;
			const rows = document.querySelectorAll("tr").length;
			const items = document.querySelectorAll("li").length;
			const articles = document.querySelectorAll("article").length;
			return JSON.stringify({
				documentElements: all,
				shellElements: inShell,
				commentIdNodes: byData,
				tableRows: rows,
				listItems: items,
				articles: articles,
				bodyTextLength: (document.body.innerText || "").length,
			});
		}`)
		if parseErr != nil {
			parseT.Fatalf("measure inbox shape: %v", parseErr)
		}
		var parseShape struct {
			DocumentElements int `json:"documentElements"`
			ShellElements    int `json:"shellElements"`
			CommentIDNodes   int `json:"commentIdNodes"`
			TableRows        int `json:"tableRows"`
			ListItems        int `json:"listItems"`
			Articles         int `json:"articles"`
			BodyTextLength   int `json:"bodyTextLength"`
		}
		if parseDecodeErr := json.Unmarshal([]byte(parseRaw.(string)), &parseShape); parseDecodeErr != nil {
			parseT.Fatalf("decode inbox shape: %v", parseDecodeErr)
		}
		parseT.Logf("/app/comments DOM: documentElements=%d shellElements=%d bodyText=%d chars",
			parseShape.DocumentElements, parseShape.ShellElements, parseShape.BodyTextLength)
		parseT.Logf("  row candidates: [data-comment-id]=%d tr=%d li=%d article=%d",
			parseShape.CommentIDNodes, parseShape.TableRows, parseShape.ListItems, parseShape.Articles)
		parseT.Logf("  payload holds 88 comments (see the captured fixture's shape field). " +
			"If no row selector reaches 88, the surface truncates or paginates and the 88 rows never hit the DOM.")
	})
}
