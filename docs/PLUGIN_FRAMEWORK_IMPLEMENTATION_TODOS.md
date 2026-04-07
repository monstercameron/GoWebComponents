# Plugin Framework Implementation Todos

This document turns `docs/PLUGIN_FRAMEWORK_PLAN.md` into one implementation backlog that can be executed one todo at a time.

Read first:

- `docs/PLUGIN_FRAMEWORK_PLAN.md`
- `plugin/plugin.go`
- `devtools/devtools_wasm.go`
- `devtools/host_extensions.go`
- `devtools/extension_sections.go`
- `devtools/error_overlay_actions.go`
- `ui/ui.go`

## Current Code Baseline

These observations are from the current repo and should drive the implementation order.

- The existing `plugin` package in `plugin/plugin.go` is an app-owned companion callback host. It is not a framework-owned kernel.
- The existing `plugin.Host` stores raw callback slices for router, fetch, devtools, SSR, and forms concerns on one struct.
- `plugin.Host.DevtoolsSections()` and `plugin.Host.DevtoolsActions()` are live provider reads, but `devtools.ApplyHostExtensions()` in `devtools/host_extensions.go` flattens them into static global state once.
- `devtools.Panel()` in `devtools/devtools_wasm.go` renders `InspectExtensionSections()` directly and has no notion of plugin health, activation, or contribution ordering beyond the current slice order.
- `devtools.ErrorOverlay()` in `devtools/devtools_wasm.go` renders `InspectErrorOverlayActions()` directly and has no kernel-managed action registry.
- The current devtools section and overlay state in `devtools/extension_sections.go` and `devtools/error_overlay_actions.go` is app-owned global mutable state protected by package-level mutexes.
- Runtime inspection already exists for runtime1 in `internal/runtime/inspect.go` through `runtime.GetGlobalRuntime().Inspect()`.
- Router inspection already exists through `router.InspectCurrentRoute()` in `router/router_api.go`, and route loader state is already tracked inside `router/router_state.go`.
- Fetch cache inspection already exists through `fetch.InspectCachedResources()` in `fetch/cache.go`, but there is not yet a first-class kernel service, event ring, or command facade.
- UI bootstrap currently happens in `ui/ui.go` via `ensureInitialized()`, which calls `runtime.InitGlobalRuntime(...)`. This is the most likely first bootstrap point for constructing a core-owned plugin kernel.
- Runtime2 already has reusable status, capability, diagnostics, hydration, snapshot, and patch machinery under `internal/runtime2/`, but there is no stable plugin-facing interposer layer yet.
- Example `examples/99-plugin-host/main.go` demonstrates the current companion host only. It does not demonstrate a core-owned plugin kernel or devtools integration.

## Execution Rules

Use this backlog sequentially.

- Do one unchecked todo at a time.
- Make the smallest correct change that finishes the current todo.
- Run the narrowest validation that proves the todo.
- Update the todo status and add a short note before moving on.
- Do not batch unrelated todos just because they are nearby in the same package.

Status legend:

- `[ ]` not started
- `[~]` in progress
- `[x]` completed
- `[!]` blocked

## Phase 0: Foundation And Bootstrap

### [x] PF-001 Create the kernel package scaffold

Files:

- `internal/pluginruntime/doc.go`
- `internal/pluginruntime/kernel.go`
- `internal/pluginruntime/manifest.go`
- `internal/pluginruntime/context.go`
- `internal/pluginruntime/health.go`
- `internal/pluginruntime/recovery.go`
- `internal/pluginruntime/services.go`
- `internal/pluginruntime/contributions.go`
- `internal/pluginruntime/streams.go`
- `internal/pluginruntime/bootstrap.go`

Work:

- Create the package and file layout from the plan.
- Keep the initial code minimal and compiling.
- Add package comments that clearly separate this package from the public `plugin` companion host.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- The package exists, compiles, and does not yet change production behavior.

Notes:

- Added the capability-level runtime2 inventory to `internal/runtime2/README.md` and grounded the current bridge on `capabilities.go`.

### [x] PF-002 Add the base kernel types

Files:

- `internal/pluginruntime/manifest.go`
- `internal/pluginruntime/context.go`
- `internal/pluginruntime/services.go`
- `internal/pluginruntime/contributions.go`

Work:

- Add the core types from the plan: manifest, service keys, contribution kinds, execution classes, activation policy, kernel info, query budgets, snapshot metadata, and subscription or cleanup handles.
- Keep these types internal and stable for the kernel package.
- Do not leak raw runtime structs into these types.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- The base type layer exists and is coherent enough for later service and contribution registries.

Notes:

- Implemented a first runtime2 plugin service in `internal/runtime2/plugininterposer.go`, but it currently exposes normalized capability metadata only. Full runtime-status and diagnostics parity still needs additional runtime2-owned summaries.

### [x] PF-003 Add bootstrap registration and enablement config

Files:

- `internal/pluginruntime/bootstrap.go`
- `internal/pluginruntime/manifest.go`
- `internal/pluginruntime/kernel.go`

Work:

- Add `PluginRegistration` and kernel bootstrap options.
- Support explicit plugin registration by factory function plus manifest.
- Support enable or disable by plugin ID.
- Reject duplicate plugin IDs deterministically.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- Kernel bootstrap can accept a registration list even if no production caller uses it yet.

Notes:

- Added `Runtime2MetaService` and wired it to the existing capability report so richer runtime2-only detail can stay optional.

### [x] PF-004 Add health states and health reports

Files:

- `internal/pluginruntime/health.go`
- `internal/pluginruntime/kernel.go`

Work:

- Implement the plugin health model from the plan: starting, healthy, degraded, quarantined, stopped.
- Add reason codes for startup error, panic, compatibility failure, repeated errors, and budget overrun.
- Add read APIs that devtools can consume later.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- Health state transitions are represented in code and testable without the devtools UI.

Notes:

- Added backend-ID coverage for runtime2 capability snapshots. Full runtime1/runtime2 parity tests for a shared runtime service are still pending because runtime2 does not yet expose the full normalized runtime summary.

### [x] PF-005 Add guarded execution and quarantine primitives

Files:

- `internal/pluginruntime/recovery.go`
- `internal/pluginruntime/kernel.go`
- `internal/pluginruntime/health.go`

Work:

- Implement guarded wrappers for plugin startup, shutdown, contribution reads, command dispatch, and subscription delivery.
- Recover panics and convert them into diagnostics plus health transitions.
- Quarantine individual plugins or contributions without crashing the framework.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- A panicking plugin cannot unwind into framework-owned call sites.

Notes:

- Updated the public companion-host and devtools docs to distinguish the internal kernel from the public `plugin` host, and updated the example messaging for examples `66-devtools-panel` and `99-plugin-host`.

### [x] PF-006 Add kernel diagnostics and plugin event reporting

Files:

- `internal/pluginruntime/health.go`
- `internal/pluginruntime/kernel.go`
- `internal/diagnostics/report.go`

Work:

- Define how the kernel emits plugin lifecycle, panic, quarantine, and budget diagnostics.
- Reuse existing diagnostics conventions where possible instead of inventing a second unrelated reporting style.
- Ensure diagnostics include plugin ID, operation name, and reason.

Validation:

- `go test ./internal/pluginruntime/...`
- `go test ./internal/diagnostics/...`

Exit criteria:

- Kernel events are visible without any devtools UI work yet.

Notes:

- Added kernel benchmarks for service lookup and contribution resolution. Runtime2 already carried substantial benchmark coverage before this pass.

### [x] PF-007 Add the service resolver and service registry skeleton

Files:

- `internal/pluginruntime/services.go`
- `internal/pluginruntime/kernel.go`

Work:

- Implement the typed service resolver shape from the plan.
- Add a registry for built-in services keyed by stable service key.
- Support required and optional services.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- Plugins can resolve typed services from the kernel once service implementations are wired.

Notes:
- Focused package validation and example verification are complete. `go run ./tools/gwc verify -json` from repo root still needs an explicit `-app` because the repo root is not an application entrypoint.
-

### [x] PF-008 Add the contribution registry, metadata, and ordering

Files:

- `internal/pluginruntime/contributions.go`
- `internal/pluginruntime/kernel.go`

Work:

- Add immutable contribution registration.
- Store contribution metadata: kind, execution class, activation policy, order key, contribution ID, and plugin ownership.
- Apply deterministic ordering by priority and plugin ID.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- The kernel can register and return ordered contributions without any subsystem-specific code yet.

Notes:

-

### [x] PF-009 Add activation policy handling

Files:

- `internal/pluginruntime/kernel.go`
- `internal/pluginruntime/contributions.go`

Work:

- Implement boot, view, session, and opportunistic activation rules.
- Support lazy activation without changing the manifest or contribution shapes later.
- Make activation decisions visible in health or diagnostics.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- The kernel can decide whether one plugin or one contribution is currently active.

Notes:

-

### [x] PF-010 Add cleanup and subscription lifecycle management

Files:

- `internal/pluginruntime/context.go`
- `internal/pluginruntime/streams.go`
- `internal/pluginruntime/kernel.go`

Work:

- Let plugins register cleanup callbacks.
- Track plugin-owned subscriptions and stop them on shutdown or quarantine.
- Ensure shutdown happens in reverse start order.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- Plugin-owned background work cannot outlive kernel-owned lifecycle control.

Notes:

-

### [x] PF-011 Add kernel foundation tests

Files:

- `internal/pluginruntime/*_test.go`

Work:

- Add focused tests for manifest validation, duplicate IDs, required versus optional services, health transitions, guarded recovery, contribution ordering, and cleanup ordering.
- Keep tests small and deterministic.

Validation:

- `go test ./internal/pluginruntime/...`

Exit criteria:

- The kernel foundation is test-covered before any production wiring starts.

Notes:

-

## Phase 1: Legacy Compatibility And Devtools Composition

### [x] PF-012 Split app-owned devtools state from kernel-owned devtools state

Files:

- `devtools/extension_sections.go`
- `devtools/error_overlay_actions.go`
- new kernel-aware registry files under `devtools/`

Work:

- Preserve the existing app-owned `SetExtensionSections` and `SetErrorOverlayActions` state.
- Add separate kernel-owned registries for kernel-managed sections and actions.
- Do not overwrite app-owned state when kernel contributions are present.

Validation:

- `go test ./devtools`

Exit criteria:

- App-owned and kernel-owned devtools state can coexist.

Notes:

-

### [x] PF-013 Replace the current static `ApplyHostExtensions` overwrite path

Files:

- `devtools/host_extensions.go`
- `devtools/devtools_test.go`

Work:

- Change `ApplyHostExtensions` so it no longer replaces all existing devtools state with one host snapshot.
- Either make it compose with app-owned state or clearly downgrade it into a compatibility adapter over the new kernel registries.
- Preserve callable overlay actions and cloned state semantics.

Validation:

- `go test ./devtools`

Exit criteria:

- Host contributions no longer hide pre-existing app-owned sections or actions.

Notes:

-

### [x] PF-014 Make devtools render a composed view instead of reading only one global slice

Files:

- `devtools/devtools_wasm.go`
- `devtools/devtools_stub.go`
- `devtools/types.go`

Work:

- Replace direct `InspectExtensionSections()` and `InspectErrorOverlayActions()` reads with composition helpers that merge app-owned, compatibility-host, and kernel-managed contributions.
- Keep the non-wasm stub behavior coherent.

Validation:

- `go test ./devtools`

Exit criteria:

- Devtools panel and overlay render the merged contribution set from all active sources.

Notes:

-

### [x] PF-015 Rewrite the stale regression tests around host extensions

Files:

- `devtools/devtools_test.go`

Work:

- Replace tests that currently encode replacement semantics.
- Add tests for composition, cleanup, and live provider reevaluation.

Validation:

- `go test ./devtools`

Exit criteria:

- The test suite guards the intended composed and live behavior rather than the old static bridge behavior.

Notes:

-

### [x] PF-016 Add compatibility notes to the package docs

Files:

- `plugin/README.md`
- `devtools/README.md`
- `plugin/doc.go`
- `devtools/doc.go`

Work:

- Document that `plugin` remains the app-owned companion host.
- Document that the new kernel is separate and internal.
- Document `ApplyHostExtensions` as a compatibility seam if it remains.

Validation:

- docs-only

Exit criteria:

- Package docs no longer imply that the existing companion host is the future deep plugin architecture.

Notes:

-

## Phase 2: Service Owners And Interposers For v1alpha1

### [x] PF-017 Add the runtime1 inspection interposer

Files:

- `internal/pluginruntime/interposer.go`
- `internal/runtime/plugininterposer.go`
- `internal/pluginruntime/services.go`

Work:

- Wrap `runtime.GetGlobalRuntime().Inspect()` from `internal/runtime/inspect.go`.
- Normalize runtime tree, stats, profiling, hydration, diagnostics, and logs into the kernel service contract.
- Do not leak `*runtime.Runtime`, `*Fiber`, or mutable slices.

Validation:

- `go test ./internal/runtime/...`
- `go test ./internal/pluginruntime/...`

Exit criteria:

- The kernel can resolve runtime inspection data through a stable service contract on runtime1.

Notes:

-

### [x] PF-018 Add the diagnostics service facade

Files:

- `internal/pluginruntime/services.go`
- `internal/pluginruntime/kernel.go`
- `internal/runtime/plugininterposer.go`

Work:

- Surface kernel diagnostics together with runtime diagnostics in one service family.
- Keep plugin diagnostics attributable by plugin ID and operation.

Validation:

- `go test ./internal/pluginruntime/...`
- `go test ./internal/runtime/...`

Exit criteria:

- Devtools can later consume one diagnostics service instead of reaching into multiple internal stores.

Notes:

-

### [x] PF-019 Add the router interposer for current route inspection

Files:

- `router/plugininterposer.go`
- `internal/pluginruntime/services.go`

Work:

- Wrap `router.InspectCurrentRoute()` from `router/router_api.go`.
- Normalize path, query, params, loader state, redirect info, and metadata.
- Keep route history support separate if it is not cheap yet.

Validation:

- `go test ./router`
- `go test ./internal/pluginruntime/...`

Exit criteria:

- The kernel can resolve current route state without devtools calling router APIs directly.

Notes:

-

### [x] PF-020 Add router command hooks for navigate and revalidate

Files:

- `router/router_api.go`
- `router/plugininterposer.go`
- `internal/pluginruntime/services.go`

Work:

- Expose typed command hooks for navigate and revalidate through the interposer.
- Reuse current public or internal router commands where they already exist.
- Preserve existing router semantics and diagnostics.

Validation:

- `go test ./router`
- `go test ./internal/pluginruntime/...`

Exit criteria:

- The kernel can issue route commands without direct UI-layer coupling.

Notes:

-

### [x] PF-021 Add a loader retry-by-key command path

Files:

- `router/router_state.go`
- `router/router_api.go`
- `router/plugininterposer.go`

Work:

- Add a typed retry path for one specific loader key, since the plan requires explicit loader retry rather than only whole-route revalidate.
- Reuse the existing loader key and loader state machinery already present in `router/router_state.go`.
- Make failure and invalid-key diagnostics clear.

Validation:

- `go test ./router`

Exit criteria:

- The kernel can retry one loader by key in a way that matches the command semantics section of the plan.

Notes:

-

### [x] PF-022 Add the fetch and cache inspection interposer

Files:

- `fetch/plugininterposer.go`
- `internal/pluginruntime/services.go`

Work:

- Wrap `fetch.InspectCachedResources()` from `fetch/cache.go`.
- Normalize cache entries, ready or stale state, owner paths, and resume policy.
- Keep richer request lifecycle data for a later task if it does not exist yet.

Validation:

- `go test ./fetch`
- `go test ./internal/pluginruntime/...`

Exit criteria:

- The kernel can resolve async-data cache state without the devtools package calling `fetch` directly.

Notes:

-

### [x] PF-023 Add fetch cache clear and revalidate command hooks

Files:

- `fetch/cache.go`
- `fetch/plugininterposer.go`
- `internal/pluginruntime/services.go`

Work:

- Add typed command hooks for clearing one cache key and revalidating one cache key.
- Reuse or add the smallest necessary fetch cache APIs to make those operations explicit.
- Keep audit context and error reporting visible.

Validation:

- `go test ./fetch`

Exit criteria:

- The kernel command layer can clear and revalidate one fetch cache entry by key.

Notes:

-

### [x] PF-024 Wire the initial service registry at UI bootstrap

Files:

- `ui/ui.go`
- `internal/pluginruntime/bootstrap.go`
- `internal/pluginruntime/kernel.go`

Work:

- Construct the plugin kernel during `ensureInitialized()` in `ui/ui.go` after runtime initialization.
- Register runtime1, diagnostics, router, and fetch services.
- Keep the bootstrap path safe when devtools is not imported or rendered.

Validation:

- `go test ./ui`
- `go test ./internal/pluginruntime/...`

Exit criteria:

- The core-owned kernel exists during normal client bootstrap.

Notes:

-

### [x] PF-025 Add interposer and service resolver integration tests

Files:

- `internal/pluginruntime/*_test.go`
- `router/*_test.go`
- `fetch/*_test.go`

Work:

- Test required and optional service resolution.
- Test interposer data cloning and normalization.
- Test failure behavior when a service is unavailable.

Validation:

- `go test ./internal/pluginruntime/...`
- `go test ./router ./fetch`

Exit criteria:

- The kernel-service layer is stable before devtools is migrated onto it.

Notes:

-

## Phase 3: Devtools As The First Kernel Consumer

### [x] PF-026 Add kernel-owned devtools contribution types and registries

Files:

- `internal/pluginruntime/contributions.go`
- `devtools/plugin_bridge.go`
- `devtools/plugin_sections.go`
- `devtools/plugin_actions.go`
- `devtools/plugin_views.go`
- `devtools/plugin_timeline.go`
- `devtools/plugin_inspectors.go`

Work:

- Add the internal contribution shapes the plan calls for.
- Start with the v1alpha1 devtools kinds: panel, section, and action.
- Keep timeline and inspector scaffolding if not all of it lands in the first pass.

Validation:

- `go test ./internal/pluginruntime/...`
- `go test ./devtools`

Exit criteria:

- Devtools can ask the kernel for typed contributions instead of global slices only.

Notes:

-

### [x] PF-027 Extend the devtools snapshot with kernel and plugin health

Files:

- `devtools/types.go`
- `devtools/devtools_wasm.go`

Work:

- Add snapshot fields for plugin health, kernel API version, and quarantined diagnostics.
- Keep old fields working for existing callers and examples.
- Do not bloat every snapshot with expensive detail if the data can stay summary-only.

Validation:

- `go test ./devtools`

Exit criteria:

- `SnapshotNow()` can show kernel and plugin state in addition to runtime state.

Notes:

-

### [x] PF-028 Make `devtools.Panel()` resolve sections live through the kernel

Files:

- `devtools/devtools_wasm.go`
- `devtools/plugin_sections.go`
- `devtools/plugin_bridge.go`

Work:

- Replace the current `InspectExtensionSections()` render path with a live contribution query.
- Keep app-owned extension sections composing with kernel-managed sections.
- Preserve the current panel UX unless a specific change is required for the new contribution model.

Validation:

- `go test ./devtools`
- `go test ./devtools -run Panel`

Exit criteria:

- The panel stops depending on the static host-extension bridge for plugin-driven sections.

Notes:

-

### [x] PF-029 Make `devtools.ErrorOverlay()` resolve actions live through the kernel

Files:

- `devtools/devtools_wasm.go`
- `devtools/plugin_actions.go`
- `devtools/plugin_bridge.go`

Work:

- Replace the current `InspectErrorOverlayActions()` render path with a live contribution query.
- Keep app-owned overlay actions composing with kernel-managed actions.
- Preserve issue matching rules by code and source.

Validation:

- `go test ./devtools`
- `go test ./devtools -run Overlay`

Exit criteria:

- Overlay actions are live and kernel-aware instead of static global clones only.

Notes:

-

### [x] PF-030 Implement the first-party internal devtools plugin

Files:

- new internal plugin implementation file under `internal/pluginruntime/` or `devtools/`
- `internal/pluginruntime/bootstrap.go`
- `devtools/plugin_bridge.go`

Work:

- Implement one core-owned devtools plugin that registers panel, section, and action contributions.
- Start by moving current summary sections into contribution methods.
- Keep this internal and first-party only.

Validation:

- `go test ./internal/pluginruntime/...`
- `go test ./devtools`

Exit criteria:

- Devtools is the first real consumer of the kernel and no longer depends on `plugin.Host` for its primary extension story.

Notes:

-

### [x] PF-031 Add devtools kernel integration tests and regressions

Files:

- `devtools/devtools_test.go`
- `devtools/devtools_wasm_test.go`
- `devtools/devtools_panel_wasm_test.go`

Work:

- Add tests for plugin health rendering, composition, quarantine, and recovery.
- Add tests for app-owned plus kernel-owned section and action coexistence.
- Replace tests that still encode overwrite semantics.

Validation:

- `go test ./devtools`

Exit criteria:

- The devtools suite proves the new kernel-backed behavior instead of the old host bridge behavior.

Notes:

-

### [x] PF-032 Update the example story

Files:

- `examples/99-plugin-host/main.go`
- or add a new example under `examples/`

Work:

- Keep example 99 as the companion-host example or rename it clearly.
- Add a separate example that shows the core-owned kernel plus devtools plugin model.
- Make the distinction obvious: companion host for app-owned extensions, kernel for deep framework plugins.

Validation:

- `go run ./tools/gwc verify -app .\\examples\\99-plugin-host\\main.go -root .\\examples\\99-plugin-host`
- verify the new example the same way if one is added

Exit criteria:

- The repo has one example for each extension story instead of implying they are the same system.

Notes:

-

## Phase 4: Extended Service Families Beyond v1alpha1

### [x] PF-033 Add the DOM service baseline

Files:

- `internal/pluginruntime/services.go`
- `internal/platform/plugininterposer.go`
- `ui/plugininterposer.go`
- `devtools/types.go`

Work:

- Start with DOM snapshot, node lookup, highlight, and scroll-into-view.
- Add mutation subscriptions only if the current platform bridge already exposes them cheaply.
- Keep DOM identity and overlay ownership explicit.

Validation:

- focused `go test` for the owning interposer package
- `go test ./devtools`

Exit criteria:

- The kernel can drive a full DOM inspector without exposing renderer-owned mutable nodes.

Notes:

-

### [x] PF-034 Add the style and theme patch service

Files:

- `internal/pluginruntime/services.go`
- `ui/plugininterposer.go`
- `internal/platform/plugininterposer.go`

Work:

- Implement theme snapshot, CSS variable reads, and reversible patch handles.
- Enforce Tier 1 mutation rules first.
- Ensure kernel quarantine removes plugin-owned patches.

Validation:

- focused `go test` for the owning package
- `go test ./devtools`

Exit criteria:

- Dynamic dark mode and related theme plugins can be implemented through patch handles instead of arbitrary DOM takeover.

Notes:

-

### [x] PF-035 Add the event observation service

Files:

- `internal/pluginruntime/services.go`
- `internal/runtime/plugininterposer.go`
- `internal/platform/plugininterposer.go`

Work:

- Add a bounded event ring, filtered subscriptions, and relay-safe records.
- Record capture, target, and bubble metadata where possible.
- Avoid exposing raw event objects.

Validation:

- focused `go test` for the owning packages

Exit criteria:

- Devtools and other plugins can inspect event flows without hot-path regressions.

Notes:

-

### [x] PF-036 Add the asset and cache-storage service

Files:

- `internal/pluginruntime/services.go`
- new asset interposer files in the owning package

Work:

- Add release manifest, cache-storage summary, asset events, invalidate, refresh, and warmup command support.
- Keep the first implementation summary-oriented if full browser integration is not cheap yet.

Validation:

- focused `go test` for the owning packages

Exit criteria:

- Asset-management plugins have a typed service instead of reaching through ad hoc cache or service-worker paths.

Notes:

-

### [x] PF-037 Add the security inspection and policy services

Files:

- `internal/pluginruntime/services.go`
- new security owner files

Work:

- Implement advisory finding generation first.
- Add typed policy evaluation for request, export, DOM patch, storage write, and route transitions.
- Ensure decisions are auditable and never hidden.

Validation:

- focused `go test` for the owning packages

Exit criteria:

- Security plugins can warn, redact, require confirmation, or block through typed contexts only.

Notes:

-

### [x] PF-038 Add the capture and replay service

Files:

- `internal/pluginruntime/capture.go`
- `devtools/trace_capture.go`
- `devtools/bug_capture.go`
- `devtools/support_bundle.go`

Work:

- Route capture sessions through the kernel.
- Keep support bundle redaction rules intact.
- Preserve current export or replay behavior while moving ownership into kernel-managed services.

Validation:

- `go test ./devtools`
- focused tests for the capture package

Exit criteria:

- Capture and replay stop being one-off devtools helpers and become kernel-backed services.

Notes:

-

### [x] PF-039 Add tests for DOM, theme, event, security, asset, and capture services

Files:

- new `*_test.go` files across owning packages

Work:

- Add narrow tests for service cloning, patch removal, bounded rings, redaction, policy decisions, and capture lifecycle.
- Do not wait until the end to add coverage.

Validation:

- focused package tests

Exit criteria:

- Every newly added service family has direct coverage for its invariants.

Notes:

-

## Phase 5: Runtime2 Interposers And Normalization

### [x] PF-040 Write the runtime2 interposer inventory

Files:

- `internal/runtime2/README.md`
- new internal notes file if needed

Work:

- Survey the runtime2 surfaces already present: capability report, diagnostics snapshot, runtime status, hydration attach helper, snapshot envelope, and patch machinery.
- Write down the exact stable contract each runtime2 surface can satisfy now.
- Identify the gaps that still require new runtime2 code.

Validation:

- docs-only or small internal tests

Exit criteria:

- Runtime2 interposer work is based on real available surfaces, not assumptions.

Notes:

-

### [x] PF-041 Add the first runtime2 status and diagnostics interposer

Files:

- `internal/runtime2/plugininterposer.go`
- `internal/pluginruntime/interposer.go`

Work:

- Start with the runtime2 surfaces that already exist: capability reporting, hydration status, post-hydration attach state, fallback mode, downgrade state, and diagnostics snapshots.
- Normalize these into the minimum stable runtime service contract.

Validation:

- `go test ./internal/runtime2/...`
- `go test ./internal/pluginruntime/...`

Exit criteria:

- The kernel can resolve a minimum runtime service contract from runtime2 without requiring full parity yet.

Notes:
- `internal/runtime2/plugininterposer.go` now merges capability reporting with a provider-backed status snapshot, and `ui/plugininterposer_runtime2.go` supplies normalized region status, hydration, downgrade, timing, and redacted diagnostic data from tracked parallel-region adapters.
-

### [x] PF-042 Add runtime2 optional capability negotiation

Files:

- `internal/runtime2/plugininterposer.go`
- `internal/pluginruntime/services.go`

Work:

- Expose richer runtime2-only detail through optional capability checks instead of inflating the baseline contract.
- Use the existing capability report machinery in `internal/runtime2/capabilities.go`.

Validation:

- `go test ./internal/runtime2/...`
- `go test ./internal/pluginruntime/...`

Exit criteria:

- Runtime2 can expose more detail where available without forcing runtime1 to fake the same cost profile.

Notes:

-

### [x] PF-043 Add runtime1 versus runtime2 normalization tests

Files:

- `internal/pluginruntime/*_test.go`
- `internal/runtime2/*_test.go`

Work:

- Add parity tests for minimum stable summaries.
- Add downgrade tests where runtime2 supports richer detail but runtime1 does not.
- Add snapshot metadata tests that prove truncation, optional capability, and backend ID reporting.

Validation:

- `go test ./internal/pluginruntime/...`
- `go test ./internal/runtime2/...`

Exit criteria:

- The kernel proves contract stability across backends instead of relying on documentation only.

Notes:
- Added runtime2 provider tests in `internal/runtime2/plugininterposer_test.go` and backend-normalization tests in `ui/parallel_region_test.go` that verify region ordering, budget truncation, capability metadata, and stable field mapping from tracked runtime2 adapters.
-

## Phase 6: Documentation, Benchmarks, And Final Verification

### [x] PF-044 Update the docs to match the implemented architecture

Files:

- `docs/PLUGIN_FRAMEWORK_PLAN.md`
- `docs/PLUGIN_FRAMEWORK_IMPLEMENTATION_TODOS.md`
- `plugin/README.md`
- `devtools/README.md`
- any affected reference manual pages

Work:

- Mark which parts of the plan are implemented versus planned.
- Document the v1alpha1 public posture clearly.
- Keep companion-host docs and kernel docs distinct.

Validation:

- docs-only

Exit criteria:

- Repo docs describe the actual implementation state instead of just the intended end state.

Notes:

-

### [x] PF-045 Add kernel and interposer benchmark coverage

Files:

- `internal/pluginruntime/*_benchmark_test.go`
- `devtools/*_benchmark_test.go`
- `internal/runtime2/*_bench_test.go` or new compare benches

Work:

- Add startup, service lookup, contribution resolution, route command, cache command, and normalization benchmarks.
- Add regression thresholds where practical.
- Keep benchmarks aligned with the performance contract in the plan.

Validation:

- `go test -run ^$ -bench . ./internal/pluginruntime/...`
- targeted benchmark runs for devtools and runtime2

Exit criteria:

- The kernel performance model is measured, not just asserted.

Notes:

-

### [x] PF-046 Run the full verification pass and close the implementation backlog

Files:

- all touched files

Work:

- Run narrow package tests first.
- Run broader repo verification once the feature is stable.
- Resolve remaining docs, examples, and benchmark gaps.
- Capture residual risks if any part of the plan remains intentionally deferred.

Suggested validation sequence:

- `go test ./internal/pluginruntime/...`
- `go test ./devtools ./plugin ./router ./fetch`
- `go test ./internal/runtime/...`
- `go test ./internal/runtime2/...`
- `go test ./ui`
- `go run ./tools/gwc test -lane unit -lane wasm -lane hydration -lane browser`
- `go run ./tools/gwc verify -json`

Exit criteria:

- The kernel exists in code, devtools is its first real consumer, legacy companion behavior remains intentional, and the repo verifies cleanly.

Notes:

- Focused package validation passed with `go test ./internal/pluginruntime/... ./devtools ./plugin ./router ./fetch ./internal/runtime/... ./internal/runtime2/... ./ui -timeout 5m`.
- Kernel benchmark coverage passed with `go test -run ^$ -bench . ./internal/pluginruntime/... -timeout 5m`.
- Example verification passed with `go run ./tools/gwc verify -app .\examples\66-devtools-panel\main.go -root .\examples\66-devtools-panel` and `go run ./tools/gwc verify -app .\examples\111-kernel-plugin-devtools\main.go -root .\examples\111-kernel-plugin-devtools`.
- Focused browser validation passed with `go test -tags playwrightgo ./test/playwrightgo/kernelplugindevtools -run TestKernelPluginDevtoolsExampleBrowser -timeout 5m -v`.
- `go run ./tools/gwc test -lane unit -lane wasm -lane hydration -lane browser` still fails for unrelated repo issues outside the plugin framework pass: missing `third_party/GoGRPCBridge`, generated-output drift in `examples/13-browser-compiler/template_lowering`, missing `scripts/livereload-client.txt` in a scaffold dev-server smoke path, and unsupported Tailwind helper expectations on `windows/arm64`.

## Checkpoint Template

Use this after each todo.

- completed todo:
- files changed:
- validation run:
- result:
- residual risk:
- next suggested todo:
