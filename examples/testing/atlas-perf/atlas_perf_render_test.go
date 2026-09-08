package atlasperf

// Native render-path benchmarks over real Atlas route payloads.
//
// REPRODUCE (all commands from the repository root):
//
//	# shapes + per-route wall clock, one warm render each
//	go test ./examples/testing/atlas-perf/ -run TestAtlasPerfRouteShapes -v
//
//	# per-route throughput and allocations
//	go test ./examples/testing/atlas-perf/ -run '^$' -bench BenchmarkAtlasPerfRouteRender -benchtime 20x
//
//	# CPU + allocation profiles of the whole route set
//	go test ./examples/testing/atlas-perf/ -run '^$' -bench BenchmarkAtlasPerfAllRoutes \
//	    -benchtime 30x -cpuprofile atlas_cpu.out -memprofile atlas_mem.out
//	go tool pprof -top -nodecount=40 atlas_cpu.out
//	go tool pprof -sample_index=alloc_space -top -nodecount=40 atlas_mem.out
//
//	# framework-vs-example attribution (see TestAtlasPerfAttributionRecipe)
//	go test ./examples/testing/atlas-perf/ -run TestAtlasPerfAttributionRecipe -v
//
// THERMAL WARNING: this chassis is fanless. docs/plans/v5-plan.md records the
// same build measuring 4 long frames cold and 33 hot. Never quote a single
// reading; TestAtlasPerfRouteShapes reports run-to-run spread on purpose.

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/design"
)

// TestAtlasPerfRouteShapes renders every fixture and reports what the tree
// actually is, plus wall clock across repeated renders.
//
// This exists because "Atlas render costs N ms" is uninterpretable without the
// tree size behind it, and because a MEDIAN plus a spread is the only honest
// summary on a throttling machine. It reports min / median / max over
// atlasPerfShapeSamples renders and never a single figure.
func TestAtlasPerfRouteShapes(parseT *testing.T) {
	parseFixtures := loadAtlasPerfFixtures(parseT)

	parseT.Logf("%-20s %-30s %-8s %s", "fixture", "route", "renders", "cost (ms) and shape")
	for _, parseFixture := range parseFixtures {
		parsePayload := atlasPerfPayload(parseT, parseFixture)

		// One render outside the timing loop: it validates the subject AND pays
		// the css fold-cache and registry warming, which would otherwise land
		// entirely on sample 0 and be reported as render cost.
		parseMarkup, parseErr := renderAtlasPerfPayload(parsePayload)
		assertAtlasPerfRendered(parseT, parseFixture.Name, parseMarkup, parseErr)

		parseSamples := make([]float64, 0, atlasPerfShapeSamples)
		for parseIndex := 0; parseIndex < atlasPerfShapeSamples; parseIndex++ {
			// A BATCH per sample, not a single render. Windows wall-clock
			// granularity here is ~0.5 ms: timing one render of the landing route
			// reported min=0.00 med=0.00 max=0.51 ms, which is the clock, not the
			// renderer. Dividing a batch pushes the quantum below the signal.
			var parseOut string
			var parseRenderErr error
			parseStart := time.Now()
			for parseRepeat := 0; parseRepeat < atlasPerfShapeBatch; parseRepeat++ {
				parseOut, parseRenderErr = renderAtlasPerfPayload(parsePayload)
				if parseRenderErr != nil {
					break
				}
			}
			parseElapsed := time.Since(parseStart) / atlasPerfShapeBatch
			if parseRenderErr != nil {
				parseT.Fatalf("render %s sample %d: %v", parseFixture.Name, parseIndex, parseRenderErr)
			}
			if parseOut != parseMarkup {
				// Not cosmetic. A render whose output moves between identical
				// inputs means state crossed the render boundary, and every later
				// sample measures a slightly different tree. Reported with the
				// first differing region so the cause is named, not guessed —
				// and reported ONCE per fixture so a systematic difference does
				// not bury the timings.
				if parseIndex == 0 {
					parseT.Logf("  NOT IDEMPOTENT: %s repeat render is %d bytes vs %d; first difference at offset %d:\n    run1: %s\n    run2: %s",
						parseFixture.Name, len(parseOut), len(parseMarkup),
						atlasPerfFirstDifference(parseMarkup, parseOut),
						atlasPerfAround(parseMarkup, atlasPerfFirstDifference(parseMarkup, parseOut)),
						atlasPerfAround(parseOut, atlasPerfFirstDifference(parseMarkup, parseOut)))
				}
			}
			parseSamples = append(parseSamples, float64(parseElapsed.Nanoseconds())/1e6)
		}
		sort.Float64s(parseSamples)
		parseT.Logf("%-20s %-30s %-8d min=%.2f med=%.2f max=%.2f  spread=%.1fx  %s",
			parseFixture.Name,
			parseFixture.Route,
			atlasPerfShapeSamples,
			parseSamples[0],
			parseSamples[len(parseSamples)/2],
			parseSamples[len(parseSamples)-1],
			parseSamples[len(parseSamples)-1]/parseSamples[0],
			atlasPerfMarkupShape(parseMarkup),
		)
	}
}

// atlasPerfShapeSamples is small on purpose: enough for a median and a spread,
// short enough that the run does not itself heat the machine into the regime it
// is trying to characterize.
const atlasPerfShapeSamples = 15

// atlasPerfShapeBatch is how many renders make up one timing sample. See the
// clock-granularity note in the loop above.
const atlasPerfShapeBatch = 20

// BenchmarkAtlasPerfRouteRender measures each route independently so a profile
// can be attributed to one payload shape.
func BenchmarkAtlasPerfRouteRender(parseB *testing.B) {
	parseFixtures := loadAtlasPerfFixtures(parseB)
	for _, parseFixture := range parseFixtures {
		parseFixture := parseFixture
		parseB.Run(parseFixture.Name, func(parseSub *testing.B) {
			parsePayload := atlasPerfPayload(parseSub, parseFixture)
			parseMarkup, parseErr := renderAtlasPerfPayload(parsePayload)
			assertAtlasPerfRendered(parseSub, parseFixture.Name, parseMarkup, parseErr)
			parseSub.ReportMetric(float64(len(parseMarkup)), "markup_bytes")
			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			for parseIndex := 0; parseIndex < parseSub.N; parseIndex++ {
				if _, parseRenderErr := renderAtlasPerfPayload(parsePayload); parseRenderErr != nil {
					parseSub.Fatalf("render: %v", parseRenderErr)
				}
			}
		})
	}
}

// BenchmarkAtlasPerfAllRoutes cycles the whole route set in one benchmark, which
// is the right subject for a PROFILE: a profile of one route attributes cost to
// that route's screen component, while cycling spreads the sample over the
// framework paths every route shares.
func BenchmarkAtlasPerfAllRoutes(parseB *testing.B) {
	parseFixtures := loadAtlasPerfFixtures(parseB)
	parsePayloads := make([]atlas.Payload, 0, len(parseFixtures))
	for _, parseFixture := range parseFixtures {
		parsePayload := atlasPerfPayload(parseB, parseFixture)
		parseMarkup, parseErr := renderAtlasPerfPayload(parsePayload)
		assertAtlasPerfRendered(parseB, parseFixture.Name, parseMarkup, parseErr)
		parsePayloads = append(parsePayloads, parsePayload)
	}
	parseB.ResetTimer()
	parseB.ReportAllocs()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseErr := renderAtlasPerfPayload(parsePayloads[parseIndex%len(parsePayloads)]); parseErr != nil {
			parseB.Fatalf("render: %v", parseErr)
		}
	}
}

// BenchmarkAtlasPerfPayloadDecode isolates atlas.PayloadFromSSRBootstrap.
//
// It is here because the decode is NOT free and it is NOT framework cost: the
// path JSON-marshals then JSON-unmarshals the payload (bootstrap.go decodeInto),
// and shared/atlas/page.go's decode[T] repeats a full JSON round trip on every
// read of the page data. Separating it keeps that cost from being reported as
// reconciliation.
func BenchmarkAtlasPerfPayloadDecode(parseB *testing.B) {
	parseFixtures := loadAtlasPerfFixtures(parseB)
	for _, parseFixture := range parseFixtures {
		parseFixture := parseFixture
		parseB.Run(parseFixture.Name, func(parseSub *testing.B) {
			parseSub.ReportMetric(float64(len(parseFixture.BootstrapJSON)), "bootstrap_bytes")
			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			for parseIndex := 0; parseIndex < parseSub.N; parseIndex++ {
				_ = atlasPerfPayload(parseSub, parseFixture)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Typed-CSS class folding (hypothesis 4)
// -----------------------------------------------------------------------------

// BenchmarkAtlasPerfDesignClassWarm measures design.Class on an already-folded
// rule set — the steady-state per-render cost once the fold cache is warm.
//
// css.New canonicalizes on EVERY call (css/rule.go canonicalize: builds a bucket
// map, sorts groups, serializes) before it can consult newCache, so a cache hit
// is not free. This benchmark is what a converted component pays per element per
// render.
func BenchmarkAtlasPerfDesignClassWarm(parseB *testing.B) {
	parseBundles := atlasPerfDesignBundles()
	for _, parseCase := range parseBundles {
		parseCase := parseCase
		parseB.Run(parseCase.Name, func(parseSub *testing.B) {
			parseWarm := design.Class(parseCase.Bundles...)
			if parseWarm == "" {
				parseSub.Fatalf("design.Class(%s) folded to an empty class", parseCase.Name)
			}
			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			for parseIndex := 0; parseIndex < parseSub.N; parseIndex++ {
				if design.Class(parseCase.Bundles...) == "" {
					parseSub.Fatal("empty class")
				}
			}
		})
	}
}

// BenchmarkAtlasPerfDesignClassCold measures the first fold of a rule set: the
// canonicalize + hash + CSS-text build + registry emit that the warm path skips.
//
// css.Reset() between iterations is process-wide, so this benchmark must not run
// concurrently with anything else in the package — it holds the render mutex for
// that reason even though it renders nothing.
func BenchmarkAtlasPerfDesignClassCold(parseB *testing.B) {
	parseBundles := atlasPerfDesignBundles()
	for _, parseCase := range parseBundles {
		parseCase := parseCase
		parseB.Run(parseCase.Name, func(parseSub *testing.B) {
			atlasPerfRenderMu.Lock()
			defer atlasPerfRenderMu.Unlock()
			parseSub.ReportAllocs()
			for parseIndex := 0; parseIndex < parseSub.N; parseIndex++ {
				parseSub.StopTimer()
				css.Reset()
				parseSub.StartTimer()
				if design.Class(parseCase.Bundles...) == "" {
					parseSub.Fatal("empty class")
				}
			}
			parseSub.StopTimer()
			css.Reset()
		})
	}
}

// atlasPerfDesignBundleCase pairs a name with a rule bundle set that a real
// converted Atlas element would fold.
type atlasPerfDesignBundleCase struct {
	Name    string
	Bundles [][]css.Rule
}

// atlasPerfDesignBundles picks bundle combinations of the sizes real markup uses:
// one bundle for a leaf, and a multi-bundle composition for a container, which is
// the shape design/doc.go documents.
func atlasPerfDesignBundles() []atlasPerfDesignBundleCase {
	return []atlasPerfDesignBundleCase{
		{Name: "single-bundle-leaf", Bundles: [][]css.Rule{design.CatalogTitle()}},
		{Name: "two-bundle-control", Bundles: [][]css.Rule{design.ButtonPrimary(), design.Link()}},
		{Name: "four-bundle-container", Bundles: [][]css.Rule{
			design.CatalogRow(), design.CatalogIdentity(), design.CatalogSummary(), design.CatalogPrice(),
		}},
	}
}

// TestAtlasPerfDesignFoldReach records whether typed-CSS folding is on Atlas's
// render path at all, which decides whether the fold benchmarks above describe a
// current cost or a future one.
//
// This is a measurement, not an assumption: shared/atlas is mid-conversion, and
// the answer changes as other agents land work. The check is a build-graph fact,
// so it stays true regardless of which route is rendered.
func TestAtlasPerfDesignFoldReach(parseT *testing.T) {
	parseFixtures := loadAtlasPerfFixtures(parseT)
	parsePayload := atlasPerfPayload(parseT, parseFixtures[0])

	// A fresh sink, then one render. Every css.New fold emits into the native
	// buffer sink exactly once, so the emitted-class count after a render is the
	// number of distinct typed-CSS folds that render performed. Zero means
	// folding is not on this path at all.
	atlasPerfRenderMu.Lock()
	css.Reset()
	atlasPerfRenderMu.Unlock()
	parseBefore := len(css.HarvestedClasses())

	parseMarkup, parseErr := renderAtlasPerfPayload(parsePayload)
	assertAtlasPerfRendered(parseT, parseFixtures[0].Name, parseMarkup, parseErr)
	parseAfter := len(css.HarvestedClasses())
	parseStyleBytes := len(css.StyleBlock())

	parseT.Logf("typed-CSS classes emitted by one Atlas render of %s: before=%d after=%d delta=%d styleBlock=%s",
		parseFixtures[0].Name, parseBefore, parseAfter, parseAfter-parseBefore, atlasPerfFormatBytes(parseStyleBytes))
	if parseAfter == parseBefore {
		parseT.Log("VERDICT: typed-CSS folding is NOT on the Atlas render path in this working tree. " +
			"The fold benchmarks above describe what conversion WILL cost, not what it costs now. " +
			"Cross-check that is independent of any single route:\n" +
			"  grep -rn 'v5/css' --include=*.go examples/server/atlas-commerce-os | grep -v _test\n" +
			"  -> only server/server.go, and only to call design.Install() once at boot.")
	} else {
		parseT.Logf("VERDICT: typed-CSS folding IS on the Atlas render path — %d folds per render of this route. "+
			"Multiply by BenchmarkAtlasPerfDesignClassWarm to bound the steady-state cost.", parseAfter-parseBefore)
	}
}

// -----------------------------------------------------------------------------
// Attribution
// -----------------------------------------------------------------------------

// TestAtlasPerfAttributionRecipe prints the exact commands that produce the
// framework-vs-example split, and the reason each one is shaped the way it is.
//
// It is a test rather than a comment so the recipe is executable and shows up in
// `-v` output next to the numbers it explains. The split itself is done by pprof
// because pprof is the tool that owns it; re-implementing aggregation in Go would
// add a second thing that can be wrong.
func TestAtlasPerfAttributionRecipe(parseT *testing.T) {
	parseT.Log(atlasPerfAttributionRecipe)
	parseT.Logf("go=%s GOOS=%s GOARCH=%s NumCPU=%d", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
}

const atlasPerfAttributionRecipe = `
FRAMEWORK-VS-EXAMPLE ATTRIBUTION

1. Capture profiles over the whole route set (not one route — see
   BenchmarkAtlasPerfAllRoutes):

     go test ./examples/testing/atlas-perf/ -run '^$' -bench BenchmarkAtlasPerfAllRoutes \
       -benchtime 30x -cpuprofile atlas_cpu.out -memprofile atlas_mem.out

2. Split by package. pprof's -top prints one line per function with the package
   path in the symbol, so grouping is a text fold over that:

     go tool pprof -top -nodecount=100000 atlas_cpu.out
     go tool pprof -sample_index=alloc_space -top -nodecount=100000 atlas_mem.out

   Buckets that matter, in the order a reader should care about them:

     internal/runtime   framework: fibers, reconciliation, hooks, SSR walk
     ui, html           framework: element construction, props, serialization
     css                framework: typed-CSS folding
     shared/atlas       EXAMPLE: component bodies (P5.1 measured ~11% of a pass)
     shared/design      EXAMPLE: design-system bundles
     encoding/json      EXAMPLE: payload decode, and page.go's decode[T]
     runtime.*          Go runtime: GC, map/slice growth, memmove

3. Exclude this harness. Anything under examples/testing/atlas-perf is
   scaffolding. docs/PRODUCTION_READINESS.md records a review that named
   cloneElementProps as 46-56% of framework allocations when the real profile put
   it at 3.14% flat behind benchmark scaffolding at 55%. Report the scaffolding
   share explicitly; do not net it out silently.

4. Cool the machine between runs and take at least three. The plan records the
   same build measuring 4 long frames cold and 33 hot on this chassis.
`

// atlasPerfFormatBytes keeps report tables readable without pulling a dependency.
func atlasPerfFormatBytes(parseBytes int) string {
	switch {
	case parseBytes >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(parseBytes)/float64(1<<20))
	case parseBytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(parseBytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", parseBytes)
	}
}

// atlasPerfFirstDifference returns the byte offset where two renders diverge, or
// -1 when they are identical.
func atlasPerfFirstDifference(parseLeft string, parseRight string) int {
	parseLimit := len(parseLeft)
	if len(parseRight) < parseLimit {
		parseLimit = len(parseRight)
	}
	for parseIndex := 0; parseIndex < parseLimit; parseIndex++ {
		if parseLeft[parseIndex] != parseRight[parseIndex] {
			return parseIndex
		}
	}
	if len(parseLeft) == len(parseRight) {
		return -1
	}
	return parseLimit
}

// atlasPerfAround extracts a readable window around an offset.
func atlasPerfAround(parseText string, parseOffset int) string {
	if parseOffset < 0 {
		return ""
	}
	parseStart := parseOffset - 60
	if parseStart < 0 {
		parseStart = 0
	}
	parseEnd := parseOffset + 60
	if parseEnd > len(parseText) {
		parseEnd = len(parseText)
	}
	return parseText[parseStart:parseEnd]
}
