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
3. `devtools` exposes live runtime inspection and debugging surfaces in the browser, including kernel-backed plugin sections
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
- the framework-owned plugin kernel that now feeds part of `devtools` remains internal; the public contract is still the `devtools` package plus the app-owned `plugin.Host` compatibility bridge
- Playwright remains the real browser runner; the public testing helpers do not replace it

## Devtools And The Internal Plugin Kernel

`devtools` is still the supported public companion surface, but the current browser implementation now composes three devtools contribution sources:

1. app-owned explicit devtools state set through the existing `SetExtensionSections(...)` and `SetErrorOverlayActions(...)` hooks
2. app-owned `plugin.Host` compatibility sources registered through `ApplyHostExtensions(...)`
3. kernel-owned contributions resolved live from the internal plugin kernel

That composition matters because the public and internal stories are different:

- `ApplyHostExtensions(...)` is a compatibility bridge for the public `plugin` host, not the deep framework plugin kernel
- kernel-backed sections are additive rather than replacing app-owned sections
- `Snapshot` now includes `Snapshot.Kernel`, which summarizes plugin-kernel API version, plugin health, and recent kernel diagnostic events
- the first shipped built-in kernel plugin contributes kernel health and runtime2 metadata sections

The current internal service families that can feed devtools are intentionally broader than the public `devtools` API:

- runtime tree, diagnostics, profiling, and hydration
- route and loader state
- fetch, cache, and asset inspection
- DOM, style, and event inspection
- security and capture services
- runtime2 metadata and worker-backed region status

Public takeaway:

- embed `devtools.Panel(...)` and `devtools.ErrorOverlay(...)` from app code as before
- use `ApplyHostExtensions(...)` only when your app already owns a `plugin.Host`
- do not depend on `internal/pluginruntime`; that kernel is implementation detail, not application API

## Minimal Example

Start with the public component test helpers. Keep tests accessibility-first and drive behavior through the rendered UI contract instead of internal state.

```go gwc:build
package settings_test

import (
	"fmt"
	"testing"

	h "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	render "github.com/monstercameron/GoWebComponents/v4/test/render"
	"github.com/monstercameron/GoWebComponents/v4/ui"
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

	"github.com/monstercameron/GoWebComponents/v4/devtools"
	h "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/logging"
	"github.com/monstercameron/GoWebComponents/v4/ui"
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

```go gwc:build
package support

import "github.com/monstercameron/GoWebComponents/v4/devtools"

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

## Live Agent Bridge

The live agent bridge is a local development and CI dogfood surface for driving a real wasm app through `gwc mcp` or the matching CLI commands. It is not a production runtime API. Treat it as CDP-equivalent: a leaked token means full control of that dev app session.

Architecture:

```text
MCP client or gwc CLI
        |
        | gwc_sessions / gwc_snapshot / gwc_query / mutating tools
        v
gwc mcp or gwc live command
        |
        | loopback HTTP with ephemeral token
        v
agent hub in the livereload server
        |
        | localhost-only WebSocket, per-session ordering
        v
wasm app opened with ?gwc-dev=agent and built with the gwcagent tag
```

Current tool catalog:

| Tool or command | Mutates app state | Purpose |
| --- | --- | --- |
| `gwc_sessions` / `gwc sessions` | no | list live sessions and choose the newest active session by default |
| `gwc_snapshot` / `gwc snapshot` | no | read a redacted runtime tree snapshot with budget metadata |
| `gwc_query` / `gwc query` | no | resolve semantic selectors to stable refs |
| `gwc_set_atom` / `gwc set-atom` | yes | write an atom value through the bridge payload codec |
| `gwc_emit` / `gwc emit` | yes | dispatch a node event handler by stable ref |
| `gwc_publish` / `gwc publish` | yes | publish a topic event |
| `gwc_navigate` / `gwc navigate` | yes | drive router navigation |
| `gwc_snapshot_diff` / `gwc snapshot-diff` | no | compare two bridge snapshots by stable ref |

Stable refs are opaque addresses returned by `bridge.snapshot` and `bridge.query`. Use them as tokens, not as selectors to parse. A ref is valid only for the session and tree version that produced it; stale refs must fail closed instead of finding a nearby node.

The bridge follows a CRUD-on-inputs rule: agents should drive user-observable inputs, route changes, atoms, and event topics. Direct fiber mutation is intentionally not part of the shipped bridge because it bypasses hooks, scheduler ordering, effect cleanup, and the same invariants real users exercise.

Headless dogfood recipe:

```powershell
go run ./tools/gwc test -lane agent -json
go run ./tools/gwc test -lane agent-browser -json
```

The `agent` lane covers native bridge, runtime, hub, and livereload integration. The `agent-browser` lane runs the ai-chat-wizard Playwright-Go dogfood test when present. The dogfood flow should launch the app through `gwc dev`, open it with `?gwc-dev=agent`, connect through a real hub token, query the composer, set the model atom, emit send, wait, snapshot the thread, and assert the message appears. It must also prove that the same app without the query parameter opens no socket and that reload links a successor session.

Security checklist:

- bind the hub to loopback only
- use one ephemeral token per run and require it on every hub route
- build the wasm app with `gwcagent` only for development or the headless dogfood lane
- keep release-profile artifacts free of bridge strings, symbols, bootstrap token injection, and command registration
- redact snapshots, logs, diagnostics, and crash reports before they leave the page or hub
- bound command history, logs, diagnostics, and crash reports, and report dropped entries
- review [security/agent-bridge-threat-model.md](../../security/agent-bridge-threat-model.md) before adding new bridge verbs

### Crash Containment

A panic that escapes any goroutine or host callback in wasm exits the whole Go program and leaves the page dead. The runtime contains crashes at every boundary it owns instead:

- every framework goroutine (`UseTask`, `UseChannel`, `UseLazyNode`, `UseForm` runners, fetch/cache loaders, debounce/throttle timers) and every `js.FuncOf` host callback (event bridges, timers, promise handlers, worker messages, IndexedDB events) recovers panics and emits one structured `[GWC-RUNTIME-PANIC-*]` console report with `where`/`path`/`error`/`next` fields and app/framework/platform stack buckets, so an agent or developer can locate the crash from the console alone
- a render panic with no error boundary abandons the in-flight render and resets scheduling state; the last committed tree stays mounted and the next clean update renders normally (the page degrades partially instead of dying)
- containment is the default for every runtime configuration; `runtime.Config.ShowRawPanicOutput` opts back into re-panicking with raw output for native debugging
- application background work should use `ui.SafeGo(subject, fn)` instead of the bare `go` statement so app goroutine panics get the same report-and-survive treatment
- hooks are render-goroutine-owned in development and test builds: if a hook is called from a goroutine while another component render fiber is current, the runtime emits `GWC-RUNTIME-HOOK-THREADING` and panics before corrupting hook state. Production-tagged builds compile this owner check out.

The containment contract is phase-specific:

| Phase | Recovery owner | Trustworthy after containment | Do not trust |
| --- | --- | --- | --- |
| Render | nearest `ErrorBoundary` renders fallback; without one, the failed render is abandoned | last committed tree, reset scheduler state, later clean updates | abandoned work-in-progress tree, pending deletions, or effects from the failed render |
| Event | nearest `ErrorBoundary` schedules a fallback update | already committed tree until fallback commits, plus state writes completed before the panic | the remaining handler body or return-value assumptions from the panicking handler |
| Effect | nearest `ErrorBoundary` schedules fallback after the commit that queued the effect | DOM from the completed commit and the scheduled fallback update | the panicking effect body after the panic or any cleanup it would have returned |
| Cleanup | nearest `ErrorBoundary` schedules fallback while deletion and teardown continue | cleared cleanup slot and settled committed tree after fallback | remaining side effects from the panicking cleanup |
| Async goroutine or guarded host callback | no `ErrorBoundary`; `SafeGo`, `GuardCallback`, or the interop guard reports and abandons the task | existing committed UI and runtime scheduling state | task-local state or partial external side effects from the abandoned task |
| Loader, hydration, startup, deferred, or SSR | no `ErrorBoundary`; emit a structured fatal report or return the SSR error | explicitly completed previous commits or returned error payloads | in-flight route, hydration, startup, deferred, or server-render work |

True runtime fatals (for example concurrent map writes or stack exhaustion) cannot be recovered by Go and still terminate the module; containment covers all `panic`-based failures.

## Devtools Plugin Use Cases

The internal kernel exists so `devtools` can grow beyond one static runtime tree snapshot without widening public runtime internals directly.

The main devtools use cases the current architecture is shaped for are:

- runtime tree, hook, profiling, and hydration inspection
- DOM, style, and event inspection
- route, loader, request, and response lifecycle inspection
- worker-backed region, transport, and runtime2 status inspection
- cache, asset, and offline-shell inspection
- capture, replay, support-bundle, and diagnostic export flows
- future theme, DOM-patch, and security-oriented inspection surfaces that still need to remain bounded and auditable

Those use cases are implemented through internal typed services and interposers, not through a public raw callback API. That keeps the public `devtools` surface small while still allowing the framework to ship richer first-party inspection.

## Hot Reload And IDE Workflow

Use hot reload as an explicit development tool, not as a second runtime guarantee.

Practical rules:

- opt in from app code so the development behavior stays visible
- treat preserved state as best-effort and scoped, not as a promise that every runtime resource survives code changes
- restart the shell or remount when component identity, hook ordering, route registration, or browser-bridge contracts change incompatibly
- when a refactor should preserve compatible hot-reload state, increment `hotreload.Config.SnapshotVersion` and provide `SnapshotMigrations` for atom state plus component path or identity aliases
- missing `snapshotVersion` values default to v1; snapshots newer than the configured app schema, missing migrations, or failed migrations restore nothing and surface a diagnostic
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
| Failure surfacing | `devtools.ErrorOverlay`, `SetErrorOverlayActions`, `ApplyHostExtensions` | `Supported companion` | development builds need focused runtime issue presentation and recovery actions | the app needs a generic production toast or alert system | app-owned state and `plugin.Host` compatibility actions compose with kernel-owned actions |
| Snapshot export and diff | `ExportSnapshotJSON`, `CompareSnapshots` | `Supported companion` | one regression needs structured before-or-after inspection | raw log lines already explain the issue | good for tree, route, diagnostics, and profiling drift |
| Trace and bundle capture | `CaptureTrace`, `CaptureBugBundle`, `CaptureSupportDiagnosticBundle`, import/export helpers | `Supported companion` | local replay or support-safe export is needed | the issue can be diagnosed directly from a live panel | sanitize before external sharing |
| Structured logs | `logging.New`, `logging.NewContext`, `Logger.WithContext`, `LogContext`, `Info`, `Warn`, `Error`, `AttachBrowserConsole` | `Supported companion` | operational events should be scoped, structured, reviewable, and correlation-friendly | ad hoc `fmt.Println` is being used as production diagnostics | native output is JSON-line structured; browser output is a structured `console.*` object |

## Design Notes And Boundaries

- The public testing surface lives in companion packages because testing ergonomics should build on public behavior, not privileged runtime hooks.
- The framework does not ship a custom runner. Ordinary Go testing plus Playwright remains the execution model.
- `devtools` is an inspection surface, not a secret-safe telemetry channel. Treat snapshots, logs, and traces as browser-visible.
- kernel-backed devtools sections are still browser-visible inspection. Treat plugin health, runtime2 status, route state, cache state, and capture summaries as diagnostics, not as hidden privileged channels.
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
go run ./tools/gwc build -app .\examples\public\devtools-panel\main.go -root .\examples\public\devtools-panel
go run ./tools/gwc build -app .\examples\public\use-snapshot\main.go -root .\examples\public\use-snapshot
go run ./tools/gwc build -app .\examples\public\snapshot-now\main.go -root .\examples\public\snapshot-now
go run ./tools/gwc build -app .\examples\public\devtools-diagnostics\main.go -root .\examples\public\devtools-diagnostics
go test -tags playwrightgo ./test/playwrightgo/kernelplugindevtools -timeout 5m -v
```

When diagnosing a live bug:

- capture a local bug bundle first
- derive or export the support-safe bundle second
- compare snapshots before and after the failing flow
- confirm the issue is visible in the smallest relevant test lane before widening CI coverage

## Time Travel (`timetravel`)

`timetravel` is the pure engine behind time-travel devtools and undo/redo: a bounded, navigable
`History[T]` of immutable snapshots. It owns no clock, DOM, or runtime.

```go
h := timetravel.New[AppState](100, initial) // capacity; <= 0 means unbounded
h.Record("typed a letter", next)            // truncates any redo branch, evicts oldest past capacity
h.Undo()                                    // (T, ok)
h.Redo()
cur := h.Current()
h.ScrubTo(3)                                // jump to any point; Labels() / Cursor() drive a timeline
```

`timetravel/devpanel.Panel(devpanel.Props{Model: h, OnUndo: ..., OnRedo: ..., OnScrub: ...})` renders
a ready-made scrubber timeline over any `*History[T]` (it satisfies the panel's `Model` interface via
`Labels()`/`Cursor()`).

## Inspecting Values (`ui.UseInspect`)

`ui.UseInspect(label, value)` is the Svelte-style `$inspect`: it logs a labeled value's initial state
and every subsequent change (`old -> new`, by structural equality) through a swappable sink.
`ui.SetInspectSink(fn)` routes those records to devtools, a test buffer, or silences them in
production (it returns a restore func).

```go
ui.UseInspect("cartTotal", total) // logs cartTotal: <initial>, then each change
```

## In-Page Error Overlay (`ui/erroroverlay`)

`erroroverlay.ErrorOverlay(props)` renders a development error as a dismissible, accessible modal
(title, message, actionable "Try: …" hint, optional stack), so a failure is legible in the page, not
only the console. `erroroverlay.FromError(err)` builds `Props` from a Go error; pair it with an error
boundary's fallback.

## Detecting State-Schema Changes On Hot Reload

`hotreload.SchemaChanged(persisted, current)` reports whether a state snapshot's shape (sorted keys +
per-key concrete type) changed, and `hotreload.SchemaFingerprint(snapshot)` returns that shape
fingerprint. A state-preserving hot reload uses this to show a visible "state reset" when the shape
changed instead of silently restoring a persisted snapshot into a mismatched type.

## Topic Pagination
Topic 12 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)
