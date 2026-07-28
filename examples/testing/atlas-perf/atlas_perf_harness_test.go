package atlasperf

// Atlas Commerce OS render-path measurement harness — NATIVE lane.
//
// WHY THIS EXISTS
//
// docs/PRODUCTION_READINESS.md records that "no real application exercises v5,
// so nothing has validated it outside two synthetic harnesses". Atlas is now
// that application. This package measures where the framework's time and
// allocations go when a REAL app's component tree is rendered, with REAL route
// payloads captured from the running server.
//
// WHAT THIS LANE CAN AND CANNOT SEE — read before quoting a number
//
// The Atlas server does NOT render markup. Verified:
//
//	grep -c RenderToString examples/server/atlas-commerce-os/server/*.go   # 0 in every file
//
// It ships an empty <div id="app"></div> plus a JSON bootstrap payload and
// paints everything from wasm. So "SSR" in Atlas means SSR of the DATA
// BOOTSTRAP, not of HTML. ui.RenderToString over atlas.App is therefore a
// harness of our own construction, not a reproduction of a server code path.
//
// That still measures the parts of the render path we most need attributed:
//
//	IN SCOPE : component-body execution, hook machinery, ui.CreateElement /
//	           html element construction, props handling, fiber creation,
//	           child reconciliation into the SSR tree, HTML serialization,
//	           and every allocation those make.
//	OUT OF SCOPE : DOM commit, deletion, order repair, layout effects, and the
//	           browser scheduler. Those are exactly what docs/plans/v5-plan.md
//	           names as M2's structural cause, and they only exist in the
//	           browser. See atlas_perf_browser_test.go (playwrightgo lane).
//
// SINGLE-FLIGHT: internal/runtime/reconciler.go tracks the component being
// rendered in PACKAGE globals (currentFiber, currentFiberOwnerGoroutineID), so
// native rendering is single-flight per PROCESS. Every render here goes through
// renderAtlasPerfPayload, which holds a mutex. Consequence for measurement:
// this lane cannot say anything about parallel-render throughput or contention.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// atlasPerfFixtureDir holds the captured bootstrap payloads, one JSON file per
// route. Regenerate with atlas_perf_capture_test.go.
const atlasPerfFixtureDir = "fixtures"

// atlasPerfRenderMu serializes native renders for the reason documented at the
// top of this file. Not an optimization guard — a correctness requirement.
var atlasPerfRenderMu sync.Mutex

// atlasPerfFixture is one captured route payload.
type atlasPerfFixture struct {
	// Name is the fixture file's basename without extension; it is also the
	// benchmark sub-name, so it must stay stable across captures.
	Name string
	// Route is the path the payload was captured from.
	Route string
	// BootstrapJSON is the verbatim contents of the page's
	// <script id="__ATLAS_BOOTSTRAP__"> element.
	BootstrapJSON []byte
}

// atlasPerfCapturedFixture is the on-disk envelope. The bootstrap is stored as
// a raw message so a capture is byte-identical to what the browser received —
// re-marshalling it would silently reorder keys and change the measured decode
// cost.
type atlasPerfCapturedFixture struct {
	Name      string          `json:"name"`
	Route     string          `json:"route"`
	Role      string          `json:"role,omitempty"`
	Bootstrap json.RawMessage `json:"bootstrap"`
	// Shape is a human-readable note about payload size (e.g. "88 comments"),
	// recorded at capture time so a stale fixture is obvious in a report.
	Shape string `json:"shape,omitempty"`
}

// loadAtlasPerfFixtures reads every fixture in atlasPerfFixtureDir.
//
// It FAILS rather than skips when the directory is empty. A perf harness that
// silently measures nothing is the failure mode docs/PRODUCTION_READINESS.md
// documents for M10 ("every one of them fails by producing nothing"), so the
// missing-fixture case names the capture command instead.
func loadAtlasPerfFixtures(parseT testing.TB) []atlasPerfFixture {
	parseT.Helper()
	parseEntries, parseErr := os.ReadDir(atlasPerfFixtureDir)
	if parseErr != nil {
		parseT.Fatalf("read %s: %v\ncapture fixtures first:\n  %s", atlasPerfFixtureDir, parseErr, atlasPerfCaptureCommand)
	}
	var parseFixtures []atlasPerfFixture
	for _, parseEntry := range parseEntries {
		if parseEntry.IsDir() || !strings.HasSuffix(parseEntry.Name(), ".json") {
			continue
		}
		parseRaw, parseReadErr := os.ReadFile(filepath.Join(atlasPerfFixtureDir, parseEntry.Name()))
		if parseReadErr != nil {
			parseT.Fatalf("read fixture %s: %v", parseEntry.Name(), parseReadErr)
		}
		parseCaptured := atlasPerfCapturedFixture{}
		if parseDecodeErr := json.Unmarshal(parseRaw, &parseCaptured); parseDecodeErr != nil {
			parseT.Fatalf("decode fixture %s: %v", parseEntry.Name(), parseDecodeErr)
		}
		if len(parseCaptured.Bootstrap) == 0 {
			parseT.Fatalf("fixture %s has an empty bootstrap; recapture with:\n  %s", parseEntry.Name(), atlasPerfCaptureCommand)
		}
		parseFixtures = append(parseFixtures, atlasPerfFixture{
			Name:          parseCaptured.Name,
			Route:         parseCaptured.Route,
			BootstrapJSON: append([]byte(nil), parseCaptured.Bootstrap...),
		})
	}
	if len(parseFixtures) == 0 {
		parseT.Fatalf("no fixtures in %s; capture them first:\n  %s", atlasPerfFixtureDir, atlasPerfCaptureCommand)
	}
	sort.Slice(parseFixtures, func(parseI, parseJ int) bool {
		return parseFixtures[parseI].Name < parseFixtures[parseJ].Name
	})
	return parseFixtures
}

// atlasPerfPayload turns a captured bootstrap into the atlas.Payload the client
// hands to the runtime.
//
// This goes through the app's OWN decode path (ui.UnmarshalSSRBootstrap then
// atlas.PayloadFromSSRBootstrap) rather than hand-building a Payload, so the
// benchmark measures the payload the app actually renders — including the
// JSON round trips that path performs.
func atlasPerfPayload(parseT testing.TB, parseFixture atlasPerfFixture) atlas.Payload {
	parseT.Helper()
	parseBootstrap, parseErr := ui.UnmarshalSSRBootstrap(parseFixture.BootstrapJSON)
	if parseErr != nil {
		parseT.Fatalf("unmarshal bootstrap for %s: %v", parseFixture.Name, parseErr)
	}
	parsePayload := atlas.PayloadFromSSRBootstrap(parseBootstrap)
	if strings.TrimSpace(parsePayload.Route.Path) == "" {
		parseT.Fatalf("fixture %s decoded to an empty route path; the bootstrap shape changed", parseFixture.Name)
	}
	return parsePayload
}

// renderAtlasPerfPayload renders one payload to markup.
//
// atlas.App is handed to the runtime as component + props. Calling
// atlas.App(payload) directly runs its hooks with no fiber installed and panics
// with GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT — the same mistake that shipped in
// client/main.go and white-screened the browser, per
// shared/atlas/app_render_matrix_test.go.
func renderAtlasPerfPayload(parsePayload atlas.Payload) (string, error) {
	atlasPerfRenderMu.Lock()
	defer atlasPerfRenderMu.Unlock()
	return ui.RenderToString(ui.CreateElement(atlas.App, parsePayload))
}

// assertAtlasPerfRendered fails when a render produced markup that is not the
// app.
//
// A benchmark whose subject silently renders an error screen still produces a
// tidy ns/op, and that number describes the error screen. Every measuring
// function in this package validates its output at least once outside the timed
// loop.
func assertAtlasPerfRendered(parseT testing.TB, parseName string, parseMarkup string, parseErr error) {
	parseT.Helper()
	if parseErr != nil {
		parseT.Fatalf("render %s: %v", parseName, parseErr)
	}
	if !strings.Contains(parseMarkup, "atlas-shell-root") {
		parseT.Fatalf("render %s produced markup without atlas-shell-root (%d bytes); the benchmark would be measuring the wrong tree",
			parseName, len(parseMarkup))
	}
}

// atlasPerfCaptureCommand is the one-line recipe printed by every
// missing-fixture failure.
const atlasPerfCaptureCommand = `ATLAS_PERF_CAPTURE=1 go test ./examples/testing/atlas-perf/ -run TestAtlasPerfCaptureFixtures -v`

// atlasPerfMarkupShape reports cheap structural facts about rendered markup so
// a report can state what size of tree a number describes.
func atlasPerfMarkupShape(parseMarkup string) string {
	return fmt.Sprintf("bytes=%d elements≈%d classes≈%d",
		len(parseMarkup),
		strings.Count(parseMarkup, "<"),
		strings.Count(parseMarkup, ` class="`),
	)
}
