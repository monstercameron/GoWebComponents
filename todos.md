# Feature Backlog

Gap analysis against React/Next/Solid ecosystems (2026-06-11). Ordered by
impact; the recommended next three are marked ⭐.

## High impact

- [ ] ⭐ **Selective / progressive hydration (islands)** — hydration is
  whole-tree with per-subtree mismatch fallback. Add a first-class API for
  hydrate-on-visible / hydrate-on-interaction islands. Attacks the measured
  wasm-startup gap vs React directly; the static-islands example is the seed.
- [ ] ⭐ **Streaming SSR** — `RenderToString` is synchronous and
  all-or-nothing. Add chunked HTML streaming with out-of-order boundary
  flushing (React `renderToPipeableStream` equivalent) so a slow data
  dependency streams a shell with placeholders instead of stalling TTFB.
- [ ] **Real suspension for async data** — `AsyncBoundary` is prop-driven
  (`Pending`/`Error` flags). Components cannot suspend mid-render and resume
  when data arrives; suspension is what enables streamed boundaries and
  removes manual loading-state plumbing from every component.
- [ ] **TinyGo build profile** — the ~1.4 MB brotli floor is the standard Go
  runtime's price. Hooks-layer reflection/generics probably block TinyGo
  today. Run a one-day feasibility spike first; even a constrained TinyGo
  profile for leaf apps changes the size story.
- [ ] **Route-level code splitting** — Go wasm ships as one binary.
  Multi-binary loading exists in examples (multi-client-binary, benchmark
  worker) but there is no lazy-chunk API for loading route logic on demand.

## Medium impact

- [ ] ⭐ **Hooks-rules vet analyzer** — conditional hook calls only fail at
  runtime (now with a contained, reported panic). Build a `go vet` analyzer
  enforcing call-order/top-level rules at compile time. Small effort,
  permanently kills a bug class.
- [ ] **`UseWebSocket` / `UseEventSource` hooks** — `UseChannel` covers Go
  channels and the gRPC bridge covers RPC, but there is no socket/SSE hook
  with reconnect, backoff, and heartbeat.
- [ ] **Query-cache ergonomics** — fetch cache lacks tag-based invalidation,
  pagination/infinite-query helpers, and high-level optimistic updates (the
  mutation queue is lower level than React Query's API).
- [ ] **Browser devtools extension** — devtools exist as in-page panels only.
  Ship a Chrome/Firefox extension (component tree, props/state inspection,
  atom graph, commit profiling). The plugin kernel was shaped for this.
- [ ] **Go-source debugging workflow** — crash reports translate wasm stacks,
  but there is no documented DWARF/source-map workflow for stepping through
  Go source in browser devtools.

## Lower impact / ecosystem

- [ ] **Headless a11y component kit** — package the overlay/focus primitives
  into a menu/combobox/listbox/datepicker/table set instead of examples.
- [ ] **Animation primitives** — transition hooks exist; add spring physics,
  FLIP, and a gesture layer.
- [ ] **Priority-lane scheduling** — the work loop yields on deadlines and
  transitions exist, but there is no interruptible priority model (urgent
  input vs background render) under sustained load.
- [ ] **Server-component-style model** — everything ships to the client; a
  server-only component concept would shrink the wasm binary.
- [ ] **RUM / OpenTelemetry export** — logs carry trace/span IDs but there is
  no OTLP browser exporter.
- [ ] **Feature flags / experimentation hooks** — none exist.

## Enterprise tier — runtime architecture

- [ ] ⭐ **Multi-instance runtimes** — `globalRuntime` is a singleton: two GWC
  apps (or two GWC versions via micro-frontends) on one page share one
  runtime, atom registry, and scheduler, so a crash reset or hot reload in
  one app touches the other. Make `Runtime` instantiable per root with
  isolated atom registries.
- [ ] ⭐ **Priority-lane scheduling with backpressure** — one work queue today:
  background data floods compete equally with keystroke re-renders, nothing
  coalesces update storms (N atom writes → N dispatches), and a started
  render cannot be interrupted. The React-lanes / Solid-scheduler gap.
- [ ] **Bounded internal state** — `uiQueue`, pending-effect lists, and
  loader/diagnostic registries grow with usage; adopt a framework-wide
  policy of bounded ring buffers with eviction, plus instrumentation of
  fiber-tree size and listener counts over time.

## Enterprise tier — contracts & guarantees

- [ ] ⭐ **Public API stability contract** — no compat guard, frozen API
  surface, or deprecation mechanism for `ui`/`router`/`fetch`/`state`. The
  bridge submodule already has `api_compat_guard`; copy the pattern to the
  framework's own public packages.
- [ ] **Specified failure-mode matrix** — crash containment exists but its
  guarantees are informal. Document per phase (render/effect/event/async)
  what state is trustworthy after a contained panic, and pin each cell with
  a test.
- [ ] **Versioned wire protocols** — the hydration bootstrap sidecar,
  hot-reload WebSocket messages, and state snapshots carry no version
  fields or compat negotiation; mismatches fail undiagnosably instead of
  cleanly.
- [ ] **Versioned state-snapshot migration** — hot-reload snapshots restore
  by component path and shape with no schema version or migration hook, so
  snapshots silently drop on refactor.

## Enterprise tier — enforcement & correctness

- [ ] **Threading-model enforcement** — hooks are render-thread-only by
  convention; calling one from a goroutine corrupts state. Detect and
  report at runtime in dev, and enforce at build via a vet analyzer
  (pairs with the hooks-rules analyzer above).
- [ ] **Strict mode** — strict hydration exists, but no general dev-time
  strict mode: double-invoke renders to flush impure components, warn on
  setState-during-render, detect asymmetric effect cleanups.
- [ ] **Deterministic replay** — capture/replay of an update stream for
  reproducing production bugs; profiling already records events
  internally, but nothing exports or replays them.
- [ ] **Migration tooling** — no codemods or upgrade assistant between
  framework versions; the parse-prefix conventions and generated shells
  make mechanical migrations very automatable.

## Enterprise tier — resilience

- [ ] **Fetch circuit breakers / retry policy** — the cache and mutation
  queue exist, but there's no declarative retry/backoff/circuit-breaker
  policy for flaky enterprise networks.
- [ ] **Long-session memory hygiene** — no GC-pressure monitoring or leak
  diagnostics for week-long dashboard sessions, which is where wasm apps
  die quietly.

## Maintenance backlog (carried from the test/perf campaign)

- [ ] Lazy DOM binding — the remaining named lever for the React DOM-ready
  startup gap (syscall/js bridge-bound).
- [ ] Churn benchmark bistability — investigate the bimodal results in the
  render-benchmark churn scenario.
- [ ] Multi-hot-reload state-survival e2e — cover state restoration across
  several consecutive hot reloads in a browser test.
- [ ] Multi-entrypoint deadcode union — `gwc` deadcode analysis should union
  reachability across all entrypoints instead of per-app.
- [ ] Compact-attrs element constructor — element creation fast path for the
  common small-attribute case.
- [ ] Bump GitHub Actions to Node 24-ready versions (`actions/configure-pages`,
  pinned `upload-artifact`) before the June 16, 2026 forced upgrade.
