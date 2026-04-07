# 12 Devtools Testing And Observability

Use this chapter when you need repeatable test harnesses, in-app diagnostics, structured logs, or support-safe capture flows for GoWebComponents apps.

It is the right chapter for:

- public testing helpers under `test/render`, `test/hooks`, `test/router`, `test/ssr`, and `test/browser`
- launcher-driven test lanes through `go run ./tools/gwc test -lane ...`
- in-browser diagnostics through `devtools.Panel(...)`, `devtools.UseSnapshot(...)`, and `devtools.SnapshotNow()`
- error overlays, trace capture, bug bundles, support bundles, and snapshot comparison
- structured logging through `logging.New(...)`, `logging.NewContext(...)`, and browser lifecycle logging through `logging.AttachBrowserConsole(...)`

Use another chapter instead when:

- you need the main `gwc` workflow and release commands first: go to [02 GWC Workflows](02-gwc-workflows.md)
- you need SSR bootstrap or hydration lifecycle details first: go to [09 SSR And Hydration](09-ssr-and-hydration.md)
- you need performance measurement and artifact sizing first: go to [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)

## Overview

The operations story is layered on purpose:

1. focused public test helpers prove behavior against public APIs
2. launcher lanes run the repo and app validation matrix repeatably
3. `devtools` exposes live runtime inspection and debugging surfaces in the browser
4. `logging` provides structured operational events
5. bug and support bundles package debugging data for local or support workflows

Keep these boundaries clear:

- the `test/...` helpers are companion tooling built on public APIs, not privileged runtime hooks
- `devtools` is browser-visible inspection, not a secret-safe telemetry sink
- support bundles must be sanitized before leaving a local debugging context
- logs and diagnostics should help explain failures without turning into raw-data dumps

## Stability Note

These are the main `Supported companion` surfaces:

- `test/render`, `test/hooks`, `test/router`, `test/ssr`, and `test/browser`
- `devtools.Panel(...)`, `devtools.UseSnapshot(...)`, `devtools.SnapshotNow()`, and `devtools.CompareSnapshots(...)`
- `devtools.CaptureTrace(...)`, `devtools.CaptureBugBundle(...)`, `devtools.CaptureSupportDiagnosticBundle(...)`, and the related import/export helpers
- `logging.New(...)`, `logging.NewContext(...)`, `Logger.WithContext(...)`, `Logger.Info(...)`, `Logger.Warn(...)`, `Logger.Error(...)`, `logging.LogContext(...)`, and `logging.AttachBrowserConsole(...)`

Important operational boundaries:

- `devtools.Panel(...)` and `devtools.ErrorOverlay(...)` are browser-only surfaces; on non-browser targets they resolve to empty or nil behavior instead of pretending inspection exists
- `CaptureBugBundle(...)` is for local debugging, while `CaptureSupportDiagnosticBundle(...)` and `SanitizeBugCaptureBundleForSupport(...)` are the support-safe export path
- Playwright remains the real browser runner; the public testing helpers do not replace it

## Minimal Example

Start with the public component test helpers. Keep tests accessibility-first and drive behavior through the rendered UI contract instead of internal state.

```go
package settings_test

import (
	"fmt"
	"testing"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	render "github.com/monstercameron/GoWebComponents/test/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderSaveSettingsButton renders one small stateful control for a consumer-facing fixture test.
func renderSaveSettingsButton() ui.Node {
	getStatus := ui.UseState("Idle")
	handleUserSave := ui.UseEvent(func() {
		getStatus.Set("Saved")
	})

	return h.Main(
		h.Button(h.Type("button"), h.OnClick(handleUserSave), "Save settings"),
		h.P(h.ID("settings-status"), fmt.Sprintf("Status: %s", getStatus.Get())),
	)
}

// TestSaveSettingsButton verifies the public behavior through role and id queries.
func TestSaveSettingsButton(getT *testing.T) {
	getFixture := render.New(getT)
	getFixture.Render(ui.CreateElement(renderSaveSettingsButton))

	getFixture.ByRole("button", "Save settings").Click()

	if getGot := getFixture.ByID("settings-status").Text(); getGot != "Status: Saved" {
		getT.Fatalf("expected save status to update, got %q", getGot)
	}
}
```

Why this is the right first testing shape:

- tests stay on public behavior and public imports
- role-driven queries align better with accessible UI than brittle selector-only checks
- the fixture handles render and event plumbing without depending on repo internals

## Production-Shaped Example

For a real app shell, expose a narrow live diagnostics summary inside normal UI, then embed the full devtools panel and error overlay behind development-only controls.

```go
package debug

import (
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderDiagnosticsShell embeds a small summary, the full panel, and the focused error overlay.
func renderDiagnosticsShell() ui.Node {
	getSnapshot := devtools.UseSnapshot(750 * time.Millisecond)
	getOpsLog := logging.New("diagnostics-shell")

	ui.UseEffect(func() func() {
		// Attach structured browser lifecycle logging while this shell is mounted.
		getOpsLog.Info("mount diagnostics shell", "route", getSnapshot.Route.Path)
		return logging.AttachBrowserConsole(logging.BrowserConsoleOptions{
			Scope:           "diagnostics-shell",
			LogNavigation:   true,
			LogWindowErrors: true,
			LogSubmits:      true,
		})
	}, true)

	return ui.Fragment(
		h.Section(
			h.Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
			h.H2("Runtime summary"),
			h.Div(
				h.Class("grid gap-4 md:grid-cols-4"),
				h.P(fmt.Sprintf("route=%s", getSnapshot.Route.Path)),
				h.P(fmt.Sprintf("fibers=%d", getSnapshot.Stats.TotalFibers)),
				h.P(fmt.Sprintf("diagnostics=%d", len(getSnapshot.Diagnostics))),
				h.P(fmt.Sprintf("logs=%d", len(getSnapshot.Logs))),
			),
		),
		ui.CreateElement(devtools.Panel, devtools.PanelProps{
			Title:           "Workspace Devtools",
			InitiallyOpen:   false,
			RefreshInterval: 750 * time.Millisecond,
			MaxDepth:        6,
		}),
		ui.CreateElement(devtools.ErrorOverlay, devtools.ErrorOverlayProps{
			Title:           "Runtime Issues",
			RefreshInterval: 750 * time.Millisecond,
			MaxItems:        4,
		}),
	)
}
```

Why this is the production-shaped baseline:

- the shell exposes lightweight health information without forcing every engineer to open the full panel
- the devtools surfaces stay embedded and app-owned instead of requiring special runtime privileges
- browser logging becomes a mount-scoped tool instead of one uncontrolled global side effect

## Scale-Up Example

When a team needs reproducible troubleshooting, capture a local bug bundle, derive a support-safe bundle from it, and compare snapshots before and after the suspect flow.

```go
package support

import "github.com/monstercameron/GoWebComponents/devtools"

type supportArtifacts struct {
	BugJSON           []byte
	SupportJSON       []byte
	ChangedSections   []string
	CurrentFingerprint string
}

// buildSupportArtifacts captures one local bug bundle, one sanitized support bundle, and one snapshot diff.
func buildSupportArtifacts(getLabel string, getPrevious devtools.Snapshot) (supportArtifacts, error) {
	getCurrent := devtools.SnapshotNow()
	getComparison, getErr := devtools.CompareSnapshots(getPrevious, getCurrent)
	if getErr != nil {
		return supportArtifacts{}, getErr
	}

	getBugBundle := devtools.CaptureBugBundle(getLabel)
	getBugJSON, getErr := devtools.ExportBugCaptureBundleJSON(getBugBundle)
	if getErr != nil {
		return supportArtifacts{}, getErr
	}

	// Keep local replay rich, but only export sanitized support artifacts externally.
	getSupportJSON, getErr := devtools.ExportSupportDiagnosticBundleJSON(getBugBundle)
	if getErr != nil {
		return supportArtifacts{}, getErr
	}

	return supportArtifacts{
		BugJSON:            getBugJSON,
		SupportJSON:        getSupportJSON,
		ChangedSections:    append([]string(nil), getComparison.ChangedSections...),
		CurrentFingerprint: getComparison.CurrentFingerprint,
	}, nil
}

// replayLocalCapture restores one previously captured local bug bundle for interactive replay.
func replayLocalCapture(getBundle devtools.BugCaptureBundle) {
	devtools.ReplayBugCaptureBundle(getBundle)
}
```

How this scales:

- engineers get a rich local replay artifact for debugging
- support or customer-facing flows get the sanitized bundle path by default
- snapshot comparison helps distinguish route, cache, diagnostics, profiling, or coordination changes without diffing raw JSON manually

## Testing Workflow

The recommended testing stack is explicit:

- `test/render` for component fixture tests on `js/wasm`
- `test/hooks` for custom hook behavior without hand-rolled hosts
- `test/router` for params, query, guards, and loader-oriented route assertions
- `test/ssr` for snapshots, bootstrap payload assertions, and hydration-smoke helpers
- `test/browser` for thin browser-environment or coordination harness helpers when browser-like state needs deterministic setup

Prefer this lane split:

- native Go tests for server, pure logic, and non-browser helpers
- `js/wasm` tests for render, hooks, router, and browser-interop logic
- hydration-focused tests when the SSR and client boundary is the thing under risk
- Playwright for real browser behavior such as focus trapping, keyboard flow, multi-surface timing, and end-to-end journeys

## Observability Workflow

Use `devtools` in layers:

- `SnapshotNow()` for one imperative capture from a button, command, or failure hook
- `UseSnapshot(...)` for a small live summary inside normal UI
- `Panel(...)` for the full in-app inspector
- `ErrorOverlay(...)` for focused failure surfacing in development
- `CompareSnapshots(...)` when one regression needs a structured before-and-after comparison
- trace and bundle helpers when the app needs export, replay, or support capture

Use `logging` similarly in layers:

- `logging.New(...)` for feature-scoped structured logs
- `logging.NewContext(...)`, `Logger.WithContext(...)`, or `logging.LogContext(...)` when correlation and trace metadata should be pulled from `context.Context`
- `logging.AttachBrowserConsole(...)` when browser lifecycle and interaction logging is useful in development, diagnostics, or tests

The current record shape is intentionally stable across native and `js/wasm` targets:

- native builds emit one JSON log record per line
- `js/wasm` builds emit the same structured object to `console.*`
- records include `timestamp`, `level`, `severity_text`, `severity_number`, `scope`, `message`, and nested `attributes`
- when context metadata is present, records also include `correlation_id`, `trace_id`, `span_id`, `traceparent`, and `tracestate`
- call sites can stay low ceremony by passing key/value pairs, `logging.Fields`, or `slog.Attr`
- framework-owned unhandled panics on `js/wasm` are also emitted as structured `console.error` records with the same slog-like level metadata, and `ui` initializes the runtime with raw panic rethrow hidden by default so wrapped runtime panics can be reported without tearing down the module

## Hot Reload And IDE Workflow

Use hot reload as an explicit development tool, not as a second runtime guarantee.

Practical rules:

- opt in from app code so the development behavior stays visible
- treat preserved state as best-effort and scoped, not as a promise that every runtime resource survives code changes
- restart the shell or remount when component identity, hook ordering, route registration, or browser-bridge contracts change incompatibly
- keep the editor workflow thin: snippets, `gwc start`, and `gwc dev` should accelerate the documented flow, not replace it

For teams, this means:

- the launcher remains the source of truth for build, serve, and test behavior
- IDE helpers should emit the same public imports and `gwc` commands the manual already documents
- reload-safe state classes should stay small enough to explain in code review

## Actionable Errors And Troubleshooting

The intended diagnostic contract is compact and fix-oriented:

- one stable code
- one plain-language summary
- one `where` or path hint
- one first remediation step
- one manual or package-doc pointer when the fix depends on framework rules

Use this triage order:

1. confirm the smallest failing lane or browser flow
2. read the first concrete error or hydration warning, not just the last panic line
3. identify whether the failure is render, route, hydration, interop, storage, or worker owned
4. only then widen into snapshots, traces, or support bundles

## API Families

| Family | Primary APIs | Stability | Use this when | Do not use this when | Notes |
| --- | --- | --- | --- | --- | --- |
| Component fixture tests | `test/render.New`, `Render`, `ByRole`, `ByID`, `Click` | `Supported companion` | you need deterministic component or interaction tests on `js/wasm` | the behavior really needs a full browser | prefer role and name queries first |
| Hook tests | `test/hooks.RenderHook`, `Act` | `Supported companion` | a custom hook needs isolated state assertions | the hook behavior is inseparable from a full rendered subtree | keep hook tests small and direct |
| Router tests | `test/router.NewHash`, `Register`, `SetPath`, `Render`, `Params`, `Query` | `Supported companion` | route params, query, and rendering should be asserted together | you need real browser history, focus timing, or navigation UX | use browser tests for the final route UX |
| SSR and hydration tests | `test/ssr.Render`, `RequirePayload`, `LoadStaticExport`, hydration smoke helpers | `Supported companion` | SSR HTML, bootstrap payloads, or static exports need direct verification | the risk is purely runtime interaction after boot | keep SSR delivery tests native when possible |
| Browser harness helpers | `test/browser.Install`, `NewCoordinationHarness` | `Supported companion` | browser-like environment or coordination setup should stay deterministic in tests | Playwright is already giving you the real browser path you need | useful for interop, worker, and cross-tab harnesses |
| Launcher validation | `gwc test`, `gwc lint` | `Supported companion` | you need repeatable local or CI validation lanes | one focused package test is enough | use the smallest lane that covers the current change |
| Live diagnostics | `devtools.UseSnapshot`, `devtools.SnapshotNow`, `devtools.Panel` | `Supported companion` | engineers need runtime tree, route, log, or diagnostic inspection | production users should see developer inspection surfaces | browser-only companion surface |
| Failure surfacing | `devtools.ErrorOverlay`, `SetErrorOverlayActions` | `Supported companion` | development builds need focused runtime issue presentation and recovery actions | the app needs a generic production toast or alert system | keep it app-owned and development-focused |
| Snapshot export and diff | `ExportSnapshotJSON`, `CompareSnapshots` | `Supported companion` | one regression needs structured before-or-after inspection | raw log lines already explain the issue | good for tree, route, diagnostics, and profiling drift |
| Trace and bundle capture | `CaptureTrace`, `CaptureBugBundle`, `CaptureSupportDiagnosticBundle`, import/export helpers | `Supported companion` | local replay or support-safe export is needed | the issue can be diagnosed directly from a live panel | sanitize before external sharing |
| Structured logs | `logging.New`, `logging.NewContext`, `Logger.WithContext`, `LogContext`, `Info`, `Warn`, `Error`, `AttachBrowserConsole` | `Supported companion` | operational events should be scoped, structured, reviewable, and correlation-friendly | ad hoc `fmt.Println` is being used as production diagnostics | native output is JSON-line structured; browser output is a structured `console.*` object |

## Design Notes And Boundaries

- The public testing surface lives in companion packages because testing ergonomics should build on public behavior, not privileged runtime hooks.
- The framework does not ship a custom runner. Ordinary Go testing plus Playwright remains the execution model.
- `devtools` is an inspection surface, not a secret-safe telemetry channel. Treat snapshots, logs, and traces as browser-visible.
- Bug bundles are richer than support bundles by design. Local engineers may need replay fidelity; external support flows should default to sanitized export.
- `logging.AttachBrowserConsole(...)` is useful for development and diagnostics, but it should be an explicit operational choice, not an always-on default.
- Performance debugging uses both devtools and benchmarks: snapshots explain where, benchmarks prove how much.

## Common Failure Modes

- writing brittle selector-driven tests when role or name queries would survive markup refactors better
- using arbitrary sleeps in tests instead of fixture or harness settlement helpers
- mixing native and `js/wasm` responsibilities so the wrong runner owns the failure
- assuming the devtools panel or overlay will exist on non-browser targets
- exporting unsanitized bug bundles to support or customer channels
- treating logs or diagnostics as secret-safe and including cookies, tokens, or raw payloads
- relying on devtools snapshots as a benchmark replacement instead of pairing them with real measurement
- enabling broad browser-console logging in production without reviewing the operational exposure

## Validation

Use the smallest relevant slices first:

```powershell
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc test -lane hydration
go run ./tools/gwc test -lane browser
go run ./tools/gwc lint -root .
go test ./devtools ./logging ./test/render ./test/hooks ./test/router ./test/ssr ./test/browser
go run ./tools/gwc build -app .\examples\66-devtools-panel\main.go -root .\examples\66-devtools-panel
go run ./tools/gwc build -app .\examples\67-use-snapshot\main.go -root .\examples\67-use-snapshot
go run ./tools/gwc build -app .\examples\68-snapshot-now\main.go -root .\examples\68-snapshot-now
go run ./tools/gwc build -app .\examples\69-devtools-diagnostics\main.go -root .\examples\69-devtools-diagnostics
```

When diagnosing a live bug:

- capture a local bug bundle first
- derive or export the support-safe bundle second
- compare snapshots before and after the failing flow
- confirm the issue is visible in the smallest relevant test lane before widening CI coverage

## Topic Pagination
Topic 12 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)
