# Feature Backlog

Gap analysis against React/Next/Solid ecosystems (2026-06-11). Ordered by
impact; exactly three active items carry the next-work marker.

## High impact

- [ ] [next] **Selective / progressive hydration (islands)** - hydration is
  whole-tree with per-subtree mismatch fallback. Add a first-class API for
  hydrate-on-visible / hydrate-on-interaction islands, plus budget-driven
  validation around the static-islands example seed. Attacks the measured
  wasm-startup gap vs React directly.
- [ ] [next] **Streaming SSR** - `RenderToString` is synchronous and
  all-or-nothing. Add chunked HTML streaming with out-of-order boundary
  flushing (React `renderToPipeableStream` equivalent) so a slow data
  dependency streams a shell with placeholders instead of stalling TTFB.
- [x] **Real suspension for async data** - `AsyncBoundary` now supports
  render-time `ui.SuspendUntil` / `ui.Await` suspension, runtime fallback
  capture, retry when the async signal resolves, SSR fallback rendering, API
  baseline coverage, and focused native/wasm tests.
- [ ] [next] **TinyGo build profile** - the ~1.4 MB brotli floor is the standard Go
  runtime's price. Hooks-layer reflection/generics probably block TinyGo
  today. Run a one-day feasibility spike first; even a constrained TinyGo
  profile for leaf apps changes the size story.
- [ ] **Route-level code splitting** - Go wasm ships as one binary.
  Multi-binary loading exists in examples (multi-client-binary, benchmark
  worker) but there is no lazy-chunk API for loading route logic on demand.

## Medium impact

- [x] **Hooks-rules analyzer** - `gwc lint` now includes a parser-backed
  `gwc-hooks` pass for `ui`/`state`/`fetch`/`flags`/`router` `Use*` calls
  inside conditionals, loops, or nested functions, with `-skip-hook-rules`
  for explicit opt-out.
- [x] **`UseWebSocket` / `UseEventSource` hooks** - `fetch` now provides
  browser realtime hooks with bounded message/error buffers, capped reconnect
  backoff, heartbeat state, native unsupported stubs, and wasm transport
  tests.
- [x] **Query-cache ergonomics** - `fetch` now provides tag-aware `UseQuery`
  and `LoadQuery`, `InvalidateQueryTag(s)` / `DisposeQueryTag`, paginated
  `UseInfiniteQuery`, and rollback-capable optimistic cache updates on top of
  the existing shared cache.
- [ ] **Browser devtools extension** - devtools exist as in-page panels only.
  Ship a Chrome/Firefox extension (component tree, props/state inspection,
  atom graph, commit profiling). The plugin kernel was shaped for this.
- [ ] **Go-source debugging workflow** - crash reports translate wasm stacks,
  but there is no documented DWARF/source-map workflow for stepping through
  Go source in browser devtools.

## Lower impact / ecosystem

- [ ] **Headless a11y component kit** - package the overlay/focus primitives
  into a menu/combobox/listbox/datepicker/table set instead of examples.
- [ ] **Animation primitives** - transition hooks exist; add spring physics,
  FLIP, and a gesture layer.
- [ ] **Scheduler ergonomics and instrumentation** - keep this as the
  documentation/devtools follow-up for the enterprise priority-lane and
  backpressure work below, rather than tracking a second scheduler
  implementation item.
- [ ] **Server-component-style model** - everything ships to the client; a
  server-only component concept would shrink the wasm binary.
- [ ] **RUM / OpenTelemetry export** - logs carry trace/span IDs but there is
  no OTLP browser exporter.
- [x] **Feature flags / experimentation hooks** - `flags` now provides
  browser-visible feature flags, deterministic weighted experiments, shared
  registry hooks, README coverage, and unit tests.

## Enterprise tier - runtime architecture

- [ ] **Multi-instance runtimes** - `globalRuntime` is a singleton: two GWC
  apps (or two GWC versions via micro-frontends) on one page share one
  runtime, atom registry, and scheduler, so a crash reset or hot reload in
  one app touches the other. Make `Runtime` instantiable per root with
  isolated atom registries.
- [ ] **Priority-lane scheduling with backpressure** - one work queue today:
  background data floods compete equally with keystroke re-renders, nothing
  coalesces update storms (N atom writes -> N dispatches), and a started
  render cannot be interrupted. The React-lanes / Solid-scheduler gap.
- [ ] **Bounded internal state** - `uiQueue`, pending-effect lists, and
  loader/diagnostic registries grow with usage; adopt a framework-wide
  policy of bounded ring buffers with eviction, plus instrumentation of
  fiber-tree size and listener counts over time.

## Enterprise tier - contracts & guarantees

- [x] **Public API stability contract** - `tools/api_compat_guard` now pins
  exported `fetch`/`flags`/`router`/`state`/`ui` symbols for native and wasm
  targets, allowing additive changes while failing removals unless the
  baseline is intentionally updated.
- [ ] **Specified failure-mode matrix** - crash containment exists but its
  guarantees are informal. Document per phase (render/effect/event/async)
  what state is trustworthy after a contained panic, and pin each cell with
  a test.
- [ ] **Versioned wire protocols** - the hydration bootstrap sidecar,
  hot-reload WebSocket messages, and state snapshots carry no version
  fields or compat negotiation; mismatches fail undiagnosably instead of
  cleanly.
- [ ] **Versioned state-snapshot migration** - hot-reload snapshots restore
  by component path and shape with no schema version or migration hook, so
  snapshots silently drop on refactor.

## Enterprise tier - enforcement & correctness

- [ ] **Threading-model enforcement** - hooks are render-thread-only by
  convention; calling one from a goroutine corrupts state. Detect and
  report at runtime in dev, and enforce at build via a vet analyzer
  (pairs with the hooks-rules analyzer above).
- [ ] **Strict mode** - strict hydration exists, but no general dev-time
  strict mode: double-invoke renders to flush impure components, warn on
  setState-during-render, detect asymmetric effect cleanups.
- [ ] **Deterministic replay** - capture/replay of an update stream for
  reproducing production bugs; profiling already records events
  internally, but nothing exports or replays them.
- [ ] **Migration tooling** - no codemods or upgrade assistant between
  framework versions; the parse-prefix conventions and generated shells
  make mechanical migrations very automatable.

## Enterprise tier - resilience

- [ ] **Fetch circuit breakers / retry policy** - the cache and mutation
  queue exist, but there's no declarative retry/backoff/circuit-breaker
  policy for flaky enterprise networks.
- [ ] **Long-session memory hygiene** - no GC-pressure monitoring or leak
  diagnostics for week-long dashboard sessions, which is where wasm apps
  fail quietly.

## Maintenance backlog (carried from the test/perf campaign)

- [ ] Lazy DOM binding - the remaining named lever for the React DOM-ready
  startup gap (syscall/js bridge-bound).
- [ ] Churn benchmark bistability - investigate the bimodal results in the
  render-benchmark churn scenario.
- [ ] Multi-hot-reload state-survival e2e - cover state restoration across
  several consecutive hot reloads in a browser test.
- [ ] Multi-entrypoint deadcode union - `gwc` deadcode analysis should union
  reachability across all entrypoints instead of per-app.
- [ ] Compact-attrs element constructor - element creation fast path for the
  common small-attribute case.
- [x] Bump GitHub Actions to Node 24-ready versions before the June 16, 2026
  forced upgrade.
  Updated workflow references on 2026-06-11 to current Node 24-ready major
  lines for `actions/checkout`, `actions/setup-go`, `actions/configure-pages`,
  `actions/upload-pages-artifact`, `actions/deploy-pages`,
  `actions/upload-artifact`, `actions/attest-build-provenance`, and
  `softprops/action-gh-release`.
