# V6 Wails desktop integration plan

Status: Windows-first integration implemented and reviewed. CLI scaffolding/build/package, durable state, cancellation and two-window native smoke pass; this remains experimental, with manual OS-interaction and release gates open. See the [final verification](v6-wails-integration-verification.md) and root backlog for exact completion status and limits.

Scope decision (2026-09-08): the user selected **Windows first** for the initial release. The macOS/Linux build, compatibility and distribution work below is deferred; no support on those platforms is implied by portable adapter tests.
Execution status: [WAILS-001 through WAILS-024 in the root backlog](../../todos.md#v6-wails-desktop-integration) are the authoritative TODOs; milestone descriptions below define scope and exit gates.
Date: 2026-09-08. Baseline: branch `v6`, commit `ce42a871`.
The existing untracked `tools/uicodegen/` directory is outside this plan.

## Objective and architecture

Make GWC applications buildable as desktop applications while retaining their Go/Wasm components, hooks, typed CSS, routing, and DOM renderer. Use Wails as the native host and service transport. Desktop support is opt-in; browser users should not acquire a native Wails dependency or platform toolchain requirement.

There are two compilation targets and two Go runtimes:

1. Native Go host: application lifecycle, windows, OS operations, durable storage, background services.
2. Go `js/wasm` frontend: GWC renderer and components, loaded with the matching `wasm_exec.js` inside a WebView.
3. Generated Wails JavaScript bindings connect the frontend to registered native services. Shared Go DTO packages share source definitions, not pointers, runtime state, or memory.

Recommendation: prototype against Wails `v3.0.0-beta.17`, the newest release listed during this inspection. Pin the CLI, Go dependency, generated runtime/bindings, and task templates together. This is a prerelease evaluation baseline, not a declaration of production compatibility. Recheck the pinned release's actual APIs during implementation; rolling documentation can differ.

Do not rewrite the DOM renderer or attempt to call native `ui.Render`. Begin with one Windows window, a local UI, one typed service method, one native dialog, and one progress event. Multi-window, signing, and durable state are subsequent milestones.

## Inspected integration map

All paths below are relative to the repository root. Symbols identify the integration point even if line numbers change.

| Area | Existing code and evidence | Planned integration |
| --- | --- | --- |
| Rendering | `ui/ui.go`: `ensureInitialized` constructs the Wasm DOM, event, scheduler, and browser-state adapters; `Render` mounts the UI. `ui/ui_native.go`: `Render` panics as unsupported on the server. | Keep frontend on `js/wasm` and use the existing renderer inside Wails. Host code must not call the native Render stub. No renderer changes expected unless a WebView probe exposes an incompatibility. |
| Module calls | `interop/interop_wasm.go`: `ImportModule` imports ES modules, invokes named exports, converts arguments/results, and awaits returned promises. `interop/interop_module.go`: public `Module.Call` and `Dispose`. | First prototype can import a generated Wails service module directly. Build typed Go wrappers over this path before considering a new generator. Avoid manually maintaining Wails numeric method IDs. |
| Import bootstrap/CSP | `ImportModule` uses `__gwcImportModule` when supplied, otherwise constructs a function with the JavaScript `Function` constructor. Only tests currently supply the helper in the searched Go/JS/HTML files. | Supply an external bootstrap script defining the import helper before Wasm starts, avoiding the Function-constructor fallback under a restrictive desktop CSP. Test Wasm compilation under the chosen CSP separately. |
| Async lifecycle | `ui/ui_async.go`: `UseTask`/`UseTaskCtx` run work on a goroutine, cancel on unmount, and suppress stale results. | Use these existing hooks for service calls. Never await a service inside a render or synchronous event callback. Do not introduce another task state machine. |
| Cancellation/errors | `interop/interop_convert_wasm.go`: `awaitValue` stops waiting when context ends, but does not cancel the underlying promise; JS callbacks are released on settlement. Cancellation is currently wrapped as `CodePromiseRejected`. `interop/interop.go` already defines cancelled, timeout, remote, encode, and decode codes. | Preserve the Wails cancellable request handle in the desktop adapter, map context cancellation/deadline distinctly, and forward cancellation where the pinned runtime supports it. Prove backend termination separately from UI cancellation. Bound the lifetime of callbacks for never-settling calls. |
| Events | `events/events.go`: process-local topic registry and effect-owned `UseTopic` subscriptions. `interop/interop.go`: `Subscription` has an unexported cancel field. | Explicitly subscribe to native Wails events and return an idempotent unsubscribe function. A desktop package cannot construct interop.Subscription directly; use its own subscription type or function. No implicit forwarding of every local GWC topic. |
| Wasm build | `tools/gwc/release_build.go`: `runBuild`, `resolveBuildConfig`, `executeBuild`, and `buildCommandForProfile`; the normal Go profile builds with `buildWasmGoEnv`. | Reuse this stage to generate frontend assets, then invoke a separate native Wails build. Do not reinterpret existing `build` as a native build. Scope environment variables separately for each child process. |
| Boot assets | `tools/gwc/start_render.go`: `renderScaffoldHTML` loads wasm_exec.js and calls instantiateStreaming without a fallback. `tools/gwc/start.go`: `resolveWasmExecPath` and scaffold runtime-asset copying. | Desktop-specific bootstrap loads the Wails adapter before Wasm, uses matching runtime assets, reports visible startup errors, and handles non-streaming instantiation when MIME/engine support requires it. Package HTML, CSS, JS modules, Wasm, fonts, and other referenced assets locally. |
| Development | `tools/gwc/dev.go`: `runDev` launches the livereload tool and takes a singleton lock by host/port. `tools/livereload/livereload.go`: Wasm build environment and watcher/server implementation. | Desktop dev must supervise frontend rebuild/reload and native restart with one owner per file set. Filter generated outputs to prevent loops; close child processes on exit. Verify Wails runtime routes through the dev asset path. |
| Configuration/scaffolding | `tools/gwc/start.go`: schema-v1 scaffold metadata with one app/HTML/Wasm target. `tools/gwc/dev.go`: metadata loading/normalization. `tools/gwc/start_render.go`: generated source and metadata. `tools/runnerconfig/core.go`: machine-specific path overrides. | Add explicit optional desktop target metadata; retain old web defaults. Keep app identity and target configuration in project metadata, not user-level runner path overrides. Introduce desktop templates without changing web templates' behavior. |
| CLI discovery | `tools/gwc/main.go`: command registry, dispatch, JSON envelopes, human help. `tools/gwc/agentic_help.go`: separate command descriptions. `tools/gwc/agentic_mcp.go`: MCP command exposure. | Register desktop subcommands consistently, with correct mutation/long-running metadata and JSON errors. Inspect MCP allowlists/schema before exposing desktop build/run; native execution must not be labelled read-only. |
| Release/tests | `tools/gwc/release_build.go`: executeRelease produces Wasm artifacts and reports. `tools/gwc/test_lanes.go`: explicit test-lane dispatch and normalization. | Preserve web release semantics; add a desktop package manifest and explicit desktop lane. A browser test passing is not proof that the packaged native WebView works. |
| Persistence | `db/sqlite/open_native.go`: non-memory storage uses `os.TempDir()/gwc-sqlite`, described as a native test approximation. `open_wasm.go` has a separate browser driver. | Native desktop storage needs an explicit application data path and lifecycle. Merely placing the frontend in Wails does not switch its Wasm database to the native driver. Keep durable operations behind a host service. |
| Durable UI state | `kvstate/backend.go`: public PersistenceBackend interface with Load/Save/Delete/Keys. `kvstate/watch.go`: BroadcastChannel notifications are separate from CRUD. | Later provide an opt-in service-backed PersistenceBackend. Cross-window invalidation needs an explicit Wails event path; changing the storage backend alone does not solve synchronization. |
| Routing | `router/router.go`: NewHashRouter; `router/router_api.go`: hash router default. | Use hash routing for the first desktop app. History routing requires verified asset fallback behavior; OS deep links require a separate host-to-router translation. |

## Dependency and package boundaries

Start with an isolated example module at proposed `examples/desktop/wails-counter/`, containing native and Wasm entrypoints in different packages. This keeps Wails out of the root module during the feasibility stage. Native host dependencies must not be imported into the frontend package graph.

Suggested example structure (new paths, not existing features):

```text
examples/desktop/wails-counter/
  go.mod                       # pinned Wails and GWC requirements
  cmd/desktop/main.go           # native entrypoint
  frontend/main.go              # js && wasm entrypoint
  internal/services/           # native-only service implementation
  contracts/                   # portable request/response DTOs
  assets/assets.go              # embed declaration owns its dist subtree
  assets/dist/                 # assembled local frontend, generated bindings
  Taskfile.yml                 # reproducible build stages
  README.md
```

Build the assets before compiling their embedding package. Binding generation may inspect native packages that embed assets: use only the pinned Wails template's supported bootstrap/generation ordering, and explicitly test a clean checkout with no dist directory. An already-populated developer directory must not conceal a generation cycle.

After the spike, propose a `desktop/` frontend adapter in the root GWC module, depending on `interop` and (only for hook helpers) `ui`, without importing native Wails packages. Keep Wails service registration in the host app initially. Extract a separately versioned native adapter module only when repeated host code warrants it.

The current module remains `github.com/monstercameron/GoWebComponents/v5` even though the branch is `v6`. Do not silently change import paths as part of this integration. Until a coordinated v6 module migration, examples use the declared module path. Also verify standalone module resolution: the root go.mod contains local replacements for agenthub and GoGRPCBridge, and dependency replacements do not propagate to consuming modules. A contributor-linked example must document any actually required replacements rather than assuming they propagate.

## Proposed API and behavior

The names below were original design sketches. Consult `desktop/README.md` and
the Windows desktop reference for the implemented API and command signatures.

- `desktop.GetCapabilities(ctx)` returns host/adapter version, platform, and supported operations. An ordinary browser or native SSR call returns a structured unavailable error. A JS host marker is feature detection, not authorization.
- Generated or handwritten typed service clients accept context and return typed DTOs/errors. Start with a handwritten counter client backed by `interop.ImportModule` and `Module.Call`; defer general code generation until the contract is proven.
- `desktop.Subscribe[T](topic, handler)` returns an idempotent unsubscribe function and error. Native events use a deliberate namespace. Close/unmount/reload releases callbacks; queueing and coalescing rules are explicit for frequent progress events.
- `desktop.UseEvent[T]` can later wrap subscription setup/cleanup with existing hooks. The initial example can use UseEffect directly.
- Keep native dialogs, clipboard, external links, and window operations in a small explicit surface. Do not turn arbitrary frontend strings into shell commands or filesystem operations.

Serialization must cover nil vs absent values, nested DTOs, errors, byte slices, time values, and integers outside JavaScript's exact numeric range. Use strings for wide identifiers and explicit wire representations for binary/time data. Do not promise arbitrary Go types across the bridge. Avoid transferring whole databases or large file buffers; use bounded operations, pages, or progress events.

Calls need distinct errors for unavailable host, unknown binding, invalid payload, remote failure, cancellation, deadline, and closed window. Native cancellation is cooperative and cannot undo completed writes. The adapter must not report a canceled UI wait as proof that a native mutation did not happen.

## Planned workflow

Command family: `gwc desktop init`, `doctor`, `dev`, `build`, and `package` is now implemented for the contributor-linked Windows template and under final verification. Existing web build/release behavior remains separate. The current dev supervisor rebuilds/restarts; frontend-only reload remains a follow-up optimization.

Production pipeline:

1. Resolve/validate the project target and pinned tool versions.
2. Generate native service bindings using the supported Wails task order.
3. Compile GWC frontend with GOOS=js and GOARCH=wasm; copy matching wasm_exec.js.
4. Assemble local assets and verify every referenced asset/binding exists.
5. Compile the native host with explicit target environment, embedding those assets.
6. Run the native smoke test, then package; signing is a separate configured release stage.

Do not require Vite/npm for the first prototype. Try generated JavaScript modules and Wails' served runtime directly. If the pinned generator/runtime requires additional tooling, record the concrete dependency and adjust the template; do not claim a no-npm workflow before a clean build proves it.

Development pipeline: prefer Wails' task supervisor with a custom GWC frontend build task. Reuse GWC's existing livereload only if the spike proves the routes and process ownership work cleanly. Shared-contract edits regenerate bindings and rebuild both sides; frontend edits rebuild Wasm and reload the window; native edits restart the host. Generated files must not trigger endless rebuilds. State preservation across native restart is not a first-stage guarantee.

## Sequential implementation backlog and exit gates

Execute one item at a time from the linked root backlog, record evidence, then advance. The milestones below describe scope; completion checkboxes live only in the backlog to avoid divergent status.

### D0 — Pin and verify the environment

- Inspect the pinned Wails release's template, runtime exports, cancellation mechanism, asset handler and binding-generation options; record exact versions.
- Check Windows Go/WebView2/build prerequisites without changing global Go environment. Validate the isolated example module can resolve its actual dependency graph.
- Record an existing browser counter baseline: startup, output size and behavior. No performance targets inferred from Wails marketing numbers.

Exit: reproducible environment instructions and a confirmed build/generation order, including clean-checkout behavior.

### D1 — Prove one desktop application

- Build a minimal local-assets GWC counter using ui.Render, typed CSS and hash routing in one Wails window.
- Add a typed native service round trip, a native open-file dialog, and a native-to-UI progress event. Keep dialog cancel distinct from failure.
- Load generated JS through interop.ImportModule with an external import helper. Verify adapter/runtime readiness before enabling native controls.
- Exercise cold start without a dev server/network, asset MIME/fallback handling, resizing, keyboard input, visible boot failure, and clean window close.

Exit: runnable Windows executable plus recorded actual-WebView evidence. Existing browser UI still works. No renderer rewrite and no root Wails dependency unless evidence forces a documented revision.

### D2 — Define and harden the frontend adapter

- Extract the smallest desktop package, including js/wasm implementation, matching native unavailable stubs, and injectable transport for unit tests.
- Implement typed payload/error mapping, capability/version handshake, and explicit native cancellation using the verified Wails API. Decide whether a narrow reusable interop extension is needed after testing the existing Module.Call limitation.
- Verify cancel-before-call, in-flight cancel, timeout, rejection, late completion, repeated mount/unmount, window close, and a promise that never settles. Cleanup must be bounded and must not invoke released JS callbacks.
- Add event unsubscribe and burst behavior tests. Confirm local GWC events do not accidentally cross the native bridge.

Exit: native service tests, Wasm transport contract tests, and native-WebView lifecycle tests pass. Document precisely which operations can cancel backend work.

### D3 — Add reproducible desktop tooling

- Add desktop target metadata and validation, with fixtures proving old schema-v1 web projects retain their behavior. Choose an additive field or explicit schema migration after reviewing normalizeScaffoldMetadata.
- Add isolated desktop templates with portable DTOs, pinned dependencies, local assets, and a clean build path. Do not reuse or modify the untracked uicodegen work without inspecting and scoping it separately.
- Add desktop init/doctor/build commands, JSON output, help, command metadata, and applicable MCP registration. Diagnose only desktop prerequisites for desktop targets.
- Add supervised dev rebuild/reload/restart with output exclusions, readiness, process cleanup, and explicit port-conflict handling. Do not inherit web singleton takeover behavior without reviewing desktop process ownership.

Exit: a generated app builds and runs from a new directory; frontend/native/DTO edits trigger the correct actions; failures leave no orphan host/server; existing scaffold/build/dev tests remain green.

### D4 — Durable state and multiple windows

- Add an explicit app-data location for native persistence, with path injection for tests. Do not adopt the temp-directory approximation as production storage.
- Implement a service-backed kvstate.PersistenceBackend as an opt-in integration, defining conflict/version behavior and restart durability.
- Add a configurable invalidation source to kvstate or an equivalent narrow adapter so native commits can refresh other windows. Preserve browser BroadcastChannel behavior and avoid event echo loops.
- Test two independent windows: UI state is window-local; shared durable state is owned by the backend; reopen/reload resynchronizes state.

Exit: persistence survives application restart at the documented location, conflicting edits are handled deterministically, and window close does not leak subscriptions or database resources.

### D5 — Desktop compatibility and distribution

- Add an explicit desktop test lane and separate CI workflow with real Windows, macOS and Linux WebView validation. Record environment limitations instead of treating skipped native tests as a pass.
- Verify focus, keyboard/IME, accessibility, dialogs, fonts, DPI, CSS, routing, clipboard and external-link behavior. Probe workers/storage separately; PWA installation/service workers are not prerequisites for the initial desktop target.
- Review content policy and navigation: trusted packaged assets, external links outside the privileged view, explicit service registration, input validation, and no production dev/agent endpoints. Determine actual Wasm/script CSP requirements on each engine.
- Add package metadata, artifact hashes and tool versions; configure platform signing/notarization and installer/runtime requirements. Keep updates, mobile and deep links out of the initial release unless separately scoped.
- Measure cold start, packaged size, memory per window, UI responsiveness and bridge latency with representative payloads. Set budgets from measured data and product needs.

Exit: packaged artifacts pass native smoke tests on each claimed platform; documentation distinguishes tested capabilities from unsupported or untested ones. Production support is not claimed from browser tests alone.

## Validation strategy and open questions

This planning change needs documentation/path/diff checks, not a full code test run. Implementation requires focused native tests for DTO/service/storage semantics, Wasm tests for transport/callback lifecycle, and actual Wails window tests for boot and OS behavior. Existing browser/SSR/build/scaffold lanes protect their respective surfaces when touched. Root `go test ./...` will not cover a nested example module; CI must invoke that module explicitly.

Resolve these questions during D0/D1 before freezing APIs:

- Does the pinned generated JS work directly with ImportModule and the served runtime, without a bundler?
- What API cancels a generated Wails promise, and what context does the native service observe on cancellation/window close?
- Does the pinned asset handler supply the expected Wasm MIME and nested module routes on all targets?
- Can Wails' dev supervisor directly run GWC's frontend build/watch task without conflicting reload ownership?
- Which browser APIs differ in the actual installed WebViews? Cross-origin isolation and SharedArrayBuffer cannot be assumed.
- Which public module/version strategy will the broader v6 release adopt? This plan intentionally does not migrate the module path.

## Sources and inspection checkpoint

Official external references checked on 2026-09-08:

- [Wails beta.17 release](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.17) — pinned evaluation candidate; prerelease status.
- [Wails architecture](https://v3.wails.io/concepts/architecture/) — native host/WebView separation and asset packaging.
- [Go-frontend bridge](https://v3.wails.io/concepts/bridge/) — generated bindings, serialization, events and call lifecycle; verify details against the pin.
- [Desktop compatibility status](https://v3.wails.io/status/) — beta platform commitments.
- [Frontend integration](https://v3.wails.io/guides/dev/frontend-frameworks/) — custom frontend integration.
- [Cross-platform builds](https://v3.wails.io/guides/build/cross-platform/) — native toolchains and packaging constraints.

Original planning checkpoint: inspected renderer, module interop, promise lifecycle, task hooks, local events, build/release, boot templates, dev supervisor, metadata, CLI registration, test dispatch, SQLite and kvstate extension points. At that checkpoint only this plan existed.

Implementation began later on 2026-09-08 using the user-requested Luna subagents and final Astra verification. Wails is now a pinned submodule; the isolated Windows example is under `examples/desktop/wails-counter/`. Follow the root TODO completion tracker and its evidence links for current results rather than treating this design's proposed APIs as implemented.
