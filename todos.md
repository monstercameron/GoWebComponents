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
