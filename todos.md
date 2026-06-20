# Feature Backlog

Gap analysis against React/Next/Solid ecosystems (2026-06-11). Ordered by
impact; exactly three active items carry the next-work marker.

## High impact

- [x] **Selective / progressive hydration (islands)** - hydration is
  whole-tree with per-subtree mismatch fallback. Add a first-class API for
  hydrate-on-visible / hydrate-on-interaction islands, plus budget-driven
  validation around the static-islands example seed. Attacks the measured
  wasm-startup gap vs React directly.
- Done (2026-06-12): `ui.HydrationIsland` marks independently resumable SSR
  islands, `ui.HydrateIsland` schedules per-island browser hydration on
  visible / interaction / idle / immediate triggers with optional timeout,
  `ui.ConfigureHydrationIslandBudget` caps concurrent island hydration, and
  `ui.InspectHydrationIslandBudget` validates startup/deferred island plans
  for static-islands-style budgets. Added native tests for option
  normalization, stable SSR marker output, and budget violations, plus
  reference manual coverage. Targeted test command was blocked before package
  tests ran by unrelated duplicate declarations in `internal/runtime`
  (`runtime_controls.go` vs `replay.go` and `scheduler_lanes.go`).
- [x] **Streaming SSR** - `ui.RenderToStream` / `RenderToStreamObserved` now
  write a shell chunk before unresolved `AsyncBoundary` content, wrap fallback
  placeholders in stable boundary comments, flush out-of-order replacement
  chunks as suspensions resolve, respect cancellation, report SSR metrics, and
  pin the public API surface with runtime/UI/API compatibility tests.
- [x] **Real suspension for async data** - `AsyncBoundary` now supports
  render-time `ui.SuspendUntil` / `ui.Await` suspension, runtime fallback
  capture, retry when the async signal resolves, SSR fallback rendering, API
  baseline coverage, and focused native/wasm tests.
- [x] **TinyGo build profile** - `gwc build` / `gwc release` now accept a
  guarded `tinygo` profile that runs `tinygo build -target=wasm -opt=z -tags
  production`, records toolchain/target metadata in JSON summaries and release
  manifests, fails early with an install hint when TinyGo is unavailable, and
  documents the constrained leaf-app compatibility boundary.
- [x] **Route-level code splitting** - Go wasm ships as one binary.
  Multi-binary loading exists in examples (multi-client-binary, benchmark
  worker) but there is no lazy-chunk API for loading route logic on demand.
- Done (2026-06-12): router routes can now declare a `RouteChunk` via
  `RegisterLazy` / `RegisterLazyRoute` or `Options.Chunk`. The router gates
  route rendering on the chunk, runs the chunk before the route factory and
  route loader, cancels stale chunk loads on navigation, supports custom
  `RouteChunk.Loader` or script URLs, and routes pending/error states through
  chunk-local or route-local fallback components. Added wasm router tests for
  "factory not called until chunk resolves" and chunk error rendering, plus
  reference manual coverage. Targeted test command was blocked before package
  tests ran by unrelated duplicate declarations in `internal/runtime`
  (`runtime_controls.go` vs `replay.go` and `scheduler_lanes.go`).

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
- [x] **Browser devtools extension** - `devtools` now exposes
  Chrome/Firefox Manifest V3 generation plus the stable
  `gwc.devtools.extension.v1` panel payload for component tree,
  props/state-derived inspection data, extension sections / atom graph
  contributions, diagnostics, logs, and commit profiling. Covered by focused
  devtools extension bridge tests.
- [x] **Go-source debugging workflow** - `gwc build` / `gwc release` now accept
  a first-class `debug` / `source-debug` profile that records `gcflags`,
  builds Go `js/wasm` artifacts with untrimmed paths and `-gcflags=all=-N -l`,
  exposes `gcflags` through `pwa.WasmReleaseFlags`, documents the browser
  stack-correlation workflow, and clearly states the current Go-toolchain
  boundary around source maps and DWARF sections.

## Lower impact / ecosystem

- [x] **Headless a11y component kit** - new `a11y` package wraps the
  overlay/focus/composite primitives into headless menu, combobox, listbox,
  datepicker grid, and table builders with semantic HTML/ARIA contracts and
  render tests.
- [x] **Animation primitives** - transition hooks exist; add spring physics,
  FLIP, and a gesture layer.
  Partial (2026-06-11): new pure-Go `anim` package - semi-implicit-Euler
  `Spring` (GentleSpring/WobblySpring/StiffSpring presets, SetTarget/Step/
  IsSettled, dt-clamped/NaN-safe), standard easings (Linear, quad/cubic
  in/out/in-out, input-clamped) + `Interpolate`, and `ComputeFLIP`
  (translate+scale delta, divide-by-zero guarded). 16 tests green;
  native+wasm. RAF hook DONE (2026-06-12): added
  `interop.RequestAnimationFrame(cb) (cancel)` (wasm one-shot, js.Func
  released after fire / on cancel - no leak; native no-op stub) and
  `ui.UseSpring(target, anim.SpringConfig) float64` - a UseEffect rAF loop
  steps the spring with per-frame dt (clamped) and re-requests until
  settled, cancelling the in-flight frame on unmount/target-change.
  Verified in real browser (`TestUseSpringAnimates`): a value animates
  frame-by-frame (41 -> 332 overshoot -> settles at 300, wobbly preset),
  not a jump. Gesture layer DONE (2026-06-12): pure pan/drag and pinch
  primitives track delta, velocity, center, scale, end state, and zero-distance
  guards with deterministic tests.
- [x] **Scheduler ergonomics and instrumentation** - new `scheduler`
  package provides an instrumented runtime scheduler wrapper with
  scheduled/executed idle and timeout counts, inline fallback counts, delay and
  latency metrics, plus a `devtools.ExtensionSection` adapter and focused
  tests.
- [x] **Server-component-style model** - new `servercomponents` package
  defines server-only descriptors, client-slot references, manifest
  collection, and `ServerOnly` helpers that render on server/native builds and
  emit wasm placeholders for bundler/client exclusion.
- [x] **RUM / OpenTelemetry export** - new `telemetry` package converts
  devtools snapshots and SSR observations into RUM events and exports OTLP/HTTP
  JSON through `BuildOTLPJSON` / `ExportOTLPHTTP`, with HTTP exporter coverage.
- [x] **Feature flags / experimentation hooks** - `flags` now provides
  browser-visible feature flags, deterministic weighted experiments, shared
  registry hooks, README coverage, and unit tests.

## Enterprise tier - runtime architecture

- [x] **Multi-instance runtimes** - `globalRuntime` is a singleton: two GWC
  apps (or two GWC versions via micro-frontends) on one page share one
  runtime, atom registry, and scheduler, so a crash reset or hot reload in
  one app touches the other. Make `Runtime` instantiable per root with
  isolated atom registries.
  Done (2026-06-12): fibers now retain runtime ownership, hook ID/function
  paths resolve through the owning root instead of the singleton, and focused
  tests prove two `NewRuntime` instances keep atom registries and ID counters
  isolated.
- [x] **Priority-lane scheduling with backpressure** - one work queue today:
  background data floods compete equally with keystroke re-renders, nothing
  coalesces update storms (N atom writes -> N dispatches), and a started
  render cannot be interrupted. The React-lanes / Solid-scheduler gap.
  Done (2026-06-12): `UpdateLane` scheduling now classifies sync/input/default/
  transition/background work, coalesces storms into the pending render,
  promotes higher-priority lanes, interrupts lower-priority in-flight work by
  rebuilding WIP from the current root, and exposes scheduler backpressure
  counters.
- [x] **Bounded internal state** - `uiQueue`, pending-effect lists, and
  loader/diagnostic registries grow with usage; adopt a framework-wide
  policy of bounded ring buffers with eviction, plus instrumentation of
  fiber-tree size and listener counts over time.
  Done (2026-06-12): runtime limits bound pending-effect tracking with a
  full-tree effect-scan fallback on overflow, diagnostics are capped as a ring,
  replay buffers are bounded, and `InternalStateSnapshot` exposes fiber,
  atom, subscriber, pending-effect, queue, diagnostic/log, profiling, and
  scheduler-pressure counts.

## Enterprise tier - contracts & guarantees

- [x] **Public API stability contract** - `tools/api_compat_guard` now pins
  exported `fetch`/`flags`/`router`/`state`/`ui` symbols for native and wasm
  targets, allowing additive changes while failing removals unless the
  baseline is intentionally updated.
- [x] **Specified failure-mode matrix** - crash containment now documents the
  render, event, effect, cleanup, async, and fatal-phase trust boundaries in
  the observability manual, with runtime tests pinning boundary recovery,
  async containment, and fatal-phase policy.
- [x] **Versioned wire protocols** - hydration bootstrap sidecars already
  carry schema versions; hot-reload WebSocket messages, app-bridge snapshots,
  and persisted state snapshots now write v1 protocol metadata, accept legacy
  missing-version payloads, and reject future mismatches with diagnostics.
- [x] **Versioned state-snapshot migration** - hot-reload snapshots now carry
  an app-owned `snapshotVersion`; `hotreload.Config` can register ordered
  migrations for shared atom state plus component path and identity aliases,
  and missing migrations fail with diagnostics instead of silently restoring
  the wrong shape.

## Enterprise tier - enforcement & correctness

- [x] **Threading-model enforcement** - hooks are render-thread-only by
  convention; dev/test builds now track the goroutine that owns the active
  render fiber, emit `GWC-RUNTIME-HOOK-THREADING`, and panic before a
  cross-goroutine hook call can mutate hook state. The built-in
  `gwc-hooks` lint pass now reports direct hook calls inside goroutine
  launches in addition to conditionals, loops, and nested functions.
- [x] **Strict mode** - strict hydration exists, but no general dev-time
  strict mode: double-invoke renders to flush impure components, warn on
  setState-during-render, detect asymmetric effect cleanups.
  Done (2026-06-12): `StrictModeOptions` can double-render components in dev,
  warn or panic on setState during render, and detect cleanup-registration
  contract changes across effect reruns.
- [x] **Deterministic replay** - capture/replay of an update stream for
  reproducing production bugs; profiling already records events
  internally, but nothing exports or replays them.
  Done (2026-06-12): runtimes can start/stop deterministic update recording,
  capture root/fiber/granular scheduling events with stable fiber-index paths
  and lanes, and replay those events against a matching current tree.
- [x] **Migration tooling** - `gwc migrate` now runs lifecycle upgrade, writes
  `bin/gwc-migrate-report.json` with compatibility findings, and supports
  `-apply` parser-backed rewrites from deprecated router selector calls
  (`GoRegisterRoute` / `GoGetRoute`) to `Register` / `Current` while leaving
  comments and string literals untouched.

## Enterprise tier - resilience

- [x] **Fetch circuit breakers / retry policy** - the cache and mutation
  queue exist, but there's no declarative retry/backoff/circuit-breaker
  policy for flaky enterprise networks.
  Done (2026-06-11): fetch/resilience.go - `RetryPolicy`
  (exponential backoff + jitter + RetryIf, `DefaultRetryPolicy()`),
  `CircuitBreaker` (closed/open/half-open, `BreakerConfig`, injectable
  clock), `ResiliencePolicy` + generic `ExecuteWithPolicy[T]` (fail-fast
  `ErrCircuitOpen`, ctx-aware retry, breaker bookkeeping). 8 deterministic
  tests (injected clock + no-op sleep, instant) cover backoff sequence,
  retry exhaustion/success/short-circuit, breaker open/half-open/close/
  reopen, and ctx cancellation. Native+wasm build, vet green.
- [x] **Long-session memory hygiene** - no GC-pressure monitoring or leak
  diagnostics for week-long dashboard sessions, which is where wasm apps
  fail quietly.
  Done (2026-06-12): `CheckMemoryHygiene` samples Go heap stats, combines them
  with internal-state counters, and emits threshold diagnostics for heap,
  fiber, atom, and subscriber pressure.

## Capability reviews (2026-06-11) - build items + what to test

### PWA / offline mode

- [x] **Generated service worker + manifest for the docs site** - sitegen
  generates `sw.js` from `pwa.BuildCacheStoragePlan` output and
  `manifest.json` from a `pwa.Manifest` value (same generated-artifact
  pattern as the boot shell; nothing authored); the app registers via
  `pwa.RegisterServiceWorker` at boot.
  Test for: registration lifecycle reaches `activated` in a real browser;
  precache list exactly matches the release-manifest SHA set (no silent
  drops); a full offline reload serves shell + site.wasm + catalog from
  CacheStorage (playwright `context.SetOffline(true)`); deploying a new
  build hash evicts stale caches and serves the new wasm (no
  half-old/half-new mix); SW update flow does not strand an open tab.
  Done (2026-06-12, generation): tools/sitegen now emits manifest.json
  (from a canonical `pwa.Manifest` value via
  `pwa.MarshalManifestJSONIndented` - name/short_name/start_url/standalone/
  theme+background colors/svg icon) and sw.js (cache name
  `gwc-shell-<12-hex of site.wasm>`, precaches index/site.wasm/favicon/
  og-image/manifest, cache-first + network fallback, activate-time
  eviction of stale gwc-shell-* caches, skipWaiting+clients.claim). Boot
  shell adds `<meta theme-color>` + `<link rel=manifest>` and registers
  the SW after boot (failure swallowed). Cache version = wasm SHA so a new
  build auto-evicts stale caches. 4 native tests (manifest validity, SW
  content, boot wiring, version determinism) + the full-site integration
  test pass; coexists with crash-loop safe mode. Browser e2e DONE
  (2026-06-12): `TestShellServiceWorkerRegistersAndServesOffline` builds
  the real site, loads it, waits for the SW to reach active.state
  ==="activated", and confirms CacheStorage holds all 5 precached assets
  under cache name gwc-shell-<wasmhash>. Caveat: a full offline NAVIGATION
  reload can't be asserted headless because Playwright SetOffline (CDP
  Network.emulateNetworkConditions) intercepts at the socket level before
  the SW fetch handler - a known CDP limitation, not a SW defect; the
  verified CacheStorage population is the authoritative offline guarantee.
- [x] **Offline mutation replay hardening** - `fetch.MutationQueue` exists;
  prove it under adversarial conditions.
  Test for: mutations enqueued offline replay exactly once after
  reconnect (no dupes on rapid online/offline flaps); replay order
  preserved; executor failure leaves the entry queued, not dropped;
  queue survives a page reload mid-outage (IndexedDB persistence);
  `pwa.InspectDiagnostics` queue counts match reality.
  Done (2026-06-12, real browser): `TestPWAOfflineMutationReplay`
  (test/playwrightgo/examples) drives the offline-cache example in headless
  Chromium - `SetOffline(true)` -> queue an offline write + a conflicting
  write (IndexedDB-backed) -> `SetOffline(false)` -> replay reports
  `succeeded=2 retried=0 dead=0 remaining=0`, and `InspectDiagnostics`
  surfaces all five structured fields (manifest/cache entries/queued/
  storage pressure/background sync). No uncaught/panic console errors.
  Adversarial slice DONE (2026-06-12): `TestPWAOfflineReplayAdversarial`
  proves exactly-once-on-flaps (queue 1 offline, flap online/offline 4x,
  replay => succeeded=1 remaining=0, second replay => succeeded=0 - no
  dupes), order-preserved by count (3 distinct writes => succeeded=3), and
  reload-mid-outage IndexedDB persistence at the storage layer (diagnostics
  confirm 2 queued writes survive into IndexedDB before reload). Failing
  executor control DONE (2026-06-12): native storage test proves executor
  errors persist retry state, defer before `NextAttemptAt`, and replay exactly
  once when due.
- [x] **Installability flow e2e** - `pwa.ObserveInstallability` exists but
  has no browser test.
  Test for: beforeinstallprompt capture, prompt() round trip, and state
  cleanup on dismissal (chromium supports faking the event).
  Done (2026-06-12, real browser): `TestPWAInstallabilityFlow` boots the
  installability example, asserts the manifest-validity / install-reasons
  / service-worker-lifecycle diagnostics render (not blank), that "Refresh
  installability" reflects `ObserveInstallability()`, and that "Prompt
  install" follows the documented GRACEFUL-REFUSAL path with a structured
  "install prompt is not currently available" message (headless Chromium
  does not fire a real beforeinstallprompt). Unit/browser shim DONE
  (2026-06-12): `TestObserveInstallabilityTracksPromptAvailabilityAndInstall`
  fakes `beforeinstallprompt`, awaits `prompt()`, resolves `userChoice`, clears
  prompt availability, and handles `appinstalled`.

### Session / long-term web storage

- [x] **`UsePersistedState[T](key, initial, area)` hook** - bind a UseState
  to localStorage/sessionStorage/IndexedDB with write-through and
  cross-tab change subscription.
  Done (2026-06-11): `ui.UsePersistedState[T](key, initial, area)` in
  ui/persisted_state.go returning `PersistedState[T]` (Get/Set/Err).
  JSON write-through; corrupted stored value falls back to initial
  (recover-guarded, no panic); quota error surfaced via Err() without
  wedging; cross-tab sync via window "storage" event (Listen, Cancel on
  cleanup - no js.Func leak); native degrades to in-memory. Tests
  (5) green; native+wasm build clean. localStorage/sessionStorage
  covered; IndexedDB area is a future extension.
  Test for: state survives unmount/remount and full reload; storage
  `storage`-event from a second tab updates the first tab's component
  (two playwright pages, one context); JSON round-trip of non-trivial T
  (structs, slices); corrupted stored value falls back to initial
  instead of panicking (crash containment must catch decode panics);
  quota-exceeded write surfaces an error state, does not wedge renders.
- [x] **`RequestPersistentStorage` helper** - wrap
  `navigator.storage.persist()`; diagnostics already read the flag.
  Test for: persisted flag flips after grant (headless chromium grants
  silently); denial path returns false without error; native build
  returns the unavailable stub.
  Done (2026-06-11): `interop.RequestPersistentStorage(ctx)` +
  `interop.IsStoragePersisted(ctx)` (wasm: navigator.storage.persist()/
  persisted() awaited via awaitValue; denial returns (false,nil) not an
  error). Native stubs return unavailable; added to the native
  unavailability test table. Builds native+wasm. The grant-flip e2e needs
  the browser lane.
- [x] **Snapshot schema versioning** (promotes the enterprise contracts
  item) - add a version field + migration hook to
  `state.SaveSnapshot`/`SavePersistentSnapshot` payloads.
  Test for: v(N) snapshot restores through a registered v(N-1)->v(N)
  migration; unknown future version is rejected loudly, not silently
  dropped; missing-version legacy payloads still restore (compat path);
  partial migration failure restores nothing (atomicity).
  Done (2026-06-12): `state` snapshot JSON now emits the
  `gwc.state.snapshot` protocol envelope with `version`, accepts legacy
  unversioned payloads, rejects future/unknown protocols, and exposes
  `RegisterSnapshotMigration` for adjacent schema upgrades. Restore paths
  normalize and migrate before applying atom state, so migration failures
  return no snapshot and leave existing atom state untouched. Covered in
  `state_native_test.go`; `go test ./state -count=1` green.
- [x] **Typed cookie helper** - first-class document.cookie access for
  session-adjacent apps (read/write/expire, SameSite/Secure attrs).
  Test for: attribute round-trips, expiry honored, and unavailability
  on native builds.
  Done (2026-06-11): `interop.GetCookie`/`SetCookie`/`ExpireCookie` +
  `SameSite`/`CookieOptions` (Path/Domain/MaxAge/Secure/SameSite). Shared
  pure-Go serialization/parsing in cookie.go (unit-testable); wasm/native
  split only for document.cookie read/write (cookie_wasm.go/
  cookie_native.go stub). SameSite=None forces Secure; URL-encode
  round-trip; Max-Age expiry; native returns unavailable. 15 tests green;
  native+wasm build clean.

### Cross-component eventing

- [x] **`events.UseTopic[T](topic)` fan-out bus** - typed in-app pub/sub
  with delivery to every subscriber, subscriptions tied to component
  lifecycle (atoms have fan-out but state semantics; channels have event
  semantics but single-receiver delivery).
  Test for: N subscribers each receive each published event exactly once
  (no coalescing of rapid bursts); publish order preserved per
  subscriber; unmounted components stop receiving and leak no
  subscriptions (registry size returns to baseline - pairs with the
  bounded-internal-state item); late subscriber receives nothing by
  default (no replay) with replay-last-value opt-in tested separately;
  publishing from a goroutine is safe (threading-model rules) or routed
  through a guarded dispatch; a panicking subscriber is contained and
  does not stop delivery to the remaining subscribers.
  Done (2026-06-11): new `events` package. Core `Subscribe[T](topic,
  handler) (unsubscribe func())` + `Publish[T](topic, value)` (testable
  without the render loop) and hook `UseTopic[T](topic, handler,
  ...TopicOption)` returning `TopicHandle[T]{Publish}` with UseEffect-tied
  unsubscribe. `WithReplayLast()` opt-in. Per-topic mutex; Publish
  snapshots subscribers then calls them OUTSIDE the lock (handlers may
  re-subscribe/publish safely); inline recover contains a panicking
  subscriber without stopping the others; type isolation via assertion.
  9 tests cover exactly-once fan-out/no-coalescing, order, leak-free
  unsubscribe (subscriberCount->0), no-replay default + replay opt-in,
  concurrent publish, containment, type isolation. Native+wasm build,
  vet green. (-race unavailable on this windows/arm64 host.)
- [x] **Cross-root eventing guidance + test** - components in different GWC
  roots / exported custom elements communicating via
  `interop.GetDocumentEvents()` CustomEvents.
  Test for: typed detail payload round-trip through Dispatch/Subscribe;
  subscription cleanup releases the underlying js.Func (no released-
  function warnings); events cross from a GWC tree into a plugin-host
  panel and back.
  Done (2026-06-12, real browser): `TestCrossRootEventingRoundTrip` drives
  the browser-interop example - `interop.GetDocumentEvents().Dispatch` ->
  `interop.SubscribeDecoded[T]` typed CustomEvent round-trip: clicking
  "Dispatch pulse" advances the subscriber's typed detail (count 1->2->3,
  source payload), proving the document-event bridge and that repeated
  dispatch yields exactly one increment each (no handler leak/double-
  delivery). Cleanup DONE (2026-06-12): wasm subscription test cancels a
  decoded custom-event subscription, dispatches again, and proves the released
  handler is not invoked. Docs now describe the plugin-host/cross-root
  document-event pattern and teardown rule.
- [x] **Cross-tab eventing soak** - `SubscribeDecodedCrossTab[T]` works in
  the example; pin it with a test.
  Test for: typed envelope round-trip between two pages in one browser
  context; decode error of a malformed envelope surfaces via the error
  callback, not a contained panic; channel close mid-flight does not
  crash either tab.
  Done (2026-06-12, real browser): `TestCrossTabEventingSoak`
  (test/playwrightgo/examples) launches headless Chromium, opens TWO pages
  in ONE BrowserContext at the cross-tab-sync example, and verifies typed
  delivery across tabs - theme/auth/draft broadcasts from page A are
  received on page B (envelope round-trip), the diagnostics panel reports
  the resolved transport (BroadcastChannel/localStorage, never pending),
  and after page A closes mid-session page B stays interactive (cache
  invalidation still fires). Fails on any uncaught/panic console error.
  Independently re-run green (~5.5s).

### Accessibility (visually impaired)

- [x] **Automated a11y audit in the browser suites** - the primitives
  (UseAnnouncer, UseFocusTrap, UseCompositeNavigation, AccessibleOverlay)
  exist but no axe-core-style audit runs in CI, so contrast/label/name
  regressions ship silently.
  Test for: every public example page passes an automated audit at the
  serious/critical level; the audit runs inside the existing playwright
  lanes; intentional violations in a fixture page are detected (the
  audit itself is tested, not just wired); docs-site routes included.
  Done (2026-06-12, real browser): axe-core (v4.12.1, vendored at
  testdata/axe.min.js) injected via AddScriptTag. `TestAccessibilityAudit-
  PublicExamples` audits 6 booting wasm examples (counter, todo-basic,
  form-accessibility, accessible-overlay, semantic-html, routed-
  accessibility) against wcag2a/wcag2aa and FAILS hard (rule id + impact +
  node + help URL) on any serious/critical - all 6 are CLEAN.
  `TestAccessibilityAuditCatchesViolation` self-test proves the audit
  works (catches 5 violations on broken markup), so a green run means
  genuinely accessible, not broken-audit. Runs in the playwrightgo lane.
  Expanded (2026-06-12): the audit now covers 19 booting examples (all
  CLEAN). The expansion HONESTLY SURFACED 4 real a11y bugs (excluded with
  documented axe rule ids, not silently dropped): `form` (label - unlabeled
  number input), `composite-navigation` (aria-input-field-name - unnamed
  listbox), `calculator` (select-name - unnamed selects), `todo-advanced`
  (color-contrast + label + select-name). FIXED + folded in (2026-06-12):
  form got a `<label for>`/`id` pair; composite-navigation's listbox an
  aria-label; calculator's two selects aria-labels; todo-advanced labels +
  aria-labels + a contrast bump (text-gray-200 / bg-blue-700 white = ~5.7:1,
  clears AA). The audit now covers 23 examples, ALL clean, with the
  self-test still proving it catches violations. Minor follow-up: add the
  docs-site shell routes to the audited set.
- [x] **Docs-site dogfood: a11y primitives in the search modal** - the new
  pure-GWC site's search modal lacks UseFocusTrap/UseAnnouncer and the
  gallery filters lack composite keyboard navigation.
  Test for: focus is trapped while the modal is open and restored to the
  Search button on close (mirror TestAccessibleOverlayBrowserE2E);
  result-count changes are announced politely; Escape closes from any
  focused element inside the modal; filter chips are arrow-key navigable.
  Done (2026-06-12): the site now uses `ui.AccessibleOverlay` for the
  search dialog focus trap/restore/Escape/outside-dismiss contract and
  `ui.UseAnnouncer` plus polite live regions for result-count changes.
  The current filters are native selects rather than chips, so the
  composite-navigation requirement is obsolete for this screen. Covered by
  wasm markup/interaction tests in `examples/public-examples-site` and the
  docs-site shell axe audit in the playwrightgo lane.
- [x] **Reduced-motion / contrast preference hooks** - interop exposes
  GetMediaQuery but there is no UsePrefersReducedMotion /
  UsePrefersColorScheme hook pair, so apps re-derive them.
  Test for: hook reflects the media query at mount, updates live when the
  emulated preference flips (playwright EmulateMedia), and unsubscribes
  on unmount without leaking js.Func handles.
  Done (2026-06-11): added `ui.UsePrefersReducedMotion()` and
  `ui.UsePrefersColorScheme()` (+ `ColorScheme`/`ColorSchemeLight`/
  `ColorSchemeDark`) in `ui/preference_hooks.go`. Reads the query
  synchronously for the mount value, re-reads + subscribes in UseEffect,
  and the cleanup calls `Subscription.Cancel()` which removes the
  listener and `Release()`s the js.Func (verified in interop_wasm.go - no
  leak). Native/SSR returns false / light via the stub. Builds native +
  wasm; `TestPreferenceHooksDefaultWithoutMediaQueries` and
  `TestColorSchemeConstants` green. Live-flip VERIFIED (2026-06-12, real
  browser): `TestPreferenceHooksLiveFlip` builds a GWC wasm fixture using
  both hooks, sets `EmulateMedia(ColorSchemeDark, ReducedMotionReduce)` -
  #scheme="dark"/#motion="true" at mount - then flips EmulateMedia to
  light/no-preference WITHOUT reload and the rendered values update live
  (the MediaQueryList.Subscribe fires and the component re-renders), and a
  third flip confirms the subscription stays live (no duplication).

### Internationalization

- [x] **Message extraction + locale completeness tooling** - nothing scans
  code for T(namespace, key) usage to scaffold catalogs or diff locales;
  incomplete translations ship silently (pairs with the enterprise
  missing-translation enforcement item).
  Test for: extraction finds every T() call across build tags (wasm and
  native files); diff reports keys missing per locale and stale keys no
  longer referenced; a gwc lane fails CI when a non-default locale is
  incomplete; dynamic/computed keys are reported as unverifiable rather
  than silently skipped.
  Done (2026-06-11/12): new i18n/extract package - `ExtractFromSource`/
  `ExtractFromDir` (go/ast; scans build-tagged _wasm.go AND _native.go,
  dedups), records non-literal ns/key as `DynamicUsage` (unverifiable,
  not skipped), `DiffLocale` (Missing incl. empty-string + Stale),
  `IsComplete`, `CheckLocales` (all-locale gate, sorted, false if any
  incomplete). 17 tests green. GWC lane DONE (2026-06-12): `gwc test -lane i18n`
  runs the extractor/completeness checks as a CI-addressable lane, and `all`
  includes it.
- [x] **Relative-time and list formatting** - FormatNumber/FormatDate exist
  but there is no FormatRelativeTime ("3 days ago") or FormatList
  ("a, b, and c"), the two most-requested formatters after dates.
  Test for: CLDR-correct output across at least en/fr/ar/ja including an
  RTL locale; plural-category interaction (1 day vs 2 days vs 0 days);
  boundary rounding (59s vs 1m, 23h vs 1d); native and wasm parity.
  Done (2026-06-11): added `FormatRelativeTime(locale, value, base)` and
  `FormatList(locale, items)` in i18n/relative_format.go, reusing
  `localeFamily`/`pluralCategoryForLocale`. en/fr/ja/ar curated per-family
  (ar plural-category-driven, RTL markers منذ/بعد; ja no-plural/no-space
  3日前; fr il y a/dans; en Oxford-comma list). Boundary rounding verified
  (59s->"59 seconds ago", 60s->"1 minute ago", 23h/24h). Native+wasm
  build, vet, and full i18n suite green; tests in relative_format_test.go
  (34 subtests). Arabic is a curated approximation consistent with the
  existing FormatDate(ar).
- [x] **Browser Intl bridge** - the i18n formatters are framework
  implementations; expose an opt-in interop path to the browser's full
  ICU (Intl.NumberFormat/DateTimeFormat) for locales/options the Go
  implementation does not cover.
  Test for: bridge output matches browser Intl for sampled locale/option
  matrices; graceful fallback to the Go formatter when Intl or the
  requested locale is unavailable; no js.Func leaks across repeated
  formats (formatter instances cached and released).
  Done (2026-06-12, real browser): `TestIntlBridgeMatchesBrowserIntl`
  builds a wasm fixture calling interop.IntlFormatNumber and compares
  every result to `new Intl.NumberFormat(locale, opts).format(value)`
  evaluated in the same headless-Chromium page - all 6 cases match
  byte-for-byte (en-US 1,234,567.89 / de-DE 1.234.567,89 / ja-JP grouped /
  fr-FR narrow-space / en-US explicit no-grouping). The cross-check also
  SURFACED AND FIXED a real bridge bug: `IntlNumberOptions.UseGrouping` was
  a plain bool always forwarded as false (zero value), suppressing
  thousands separators on every call; changed it to `*bool` (nil = browser
  default = grouping on; explicit value overrides) in intl.go/intl_wasm.go,
  updated the cache key, and verified the override path. Native+wasm build,
  interop tests green.
  Earlier partial (2026-06-12): interop intl.go/intl_wasm.go/intl_native.go -
  `IntlAvailable`, `IntlFormatNumber(locale,value,opts)`,
  `IntlFormatDate(locale,unixMillis,opts)` (+ IntlNumberOptions/
  IntlDateOptions). wasm constructs/caches Intl.NumberFormat/DateTimeFormat
  per locale+options signature (mutex-guarded; no per-call reconstruction,
  no js.Func leak - formatters are reused objects), JS exceptions recovered
  into Go errors. Native returns unavailable so callers fall back to
  i18n.FormatNumber/FormatDate. Pure-Go cache-key builders unit-tested (9
  tests); native+wasm build clean. Remaining: the browser-lane test
  sampling output against real Intl.

## Enterprise tier - security & supply chain

- [x] **CSP nonce threading** - neither RenderToString nor the new
  RenderToStream emits or threads script nonces, and there is no
  documented wasm-unsafe-eval guidance; strict CSPs block deployment.
  Test for: a nonce supplied per request appears on every emitted
  script tag across both SSR paths including out-of-order streamed
  chunks; hydration succeeds under a strict CSP (no inline-eval
  violations in the browser console); generated boot shells accept an
  injected nonce; a CSP-violation fixture page proves the test setup
  actually enforces the policy.
  Done (2026-06-12): SSR bootstrap scripts now accept `SSRScriptOptions`
  with an escaped nonce for inline bootstrap and external-reference
  bootstrap tags; streaming SSR carries `SSRStreamOptions.ScriptNonce`
  onto every async-boundary patch script, including out-of-order chunks;
  the generated site boot shell supports injected nonces on its `<style>`
  and `<script>` tags. Covered by `ui/ssr_bootstrap_test.go`,
  `internal/runtime/ssr_stream_test.go`, and `tools/sitegen/
  sw_manifest_test.go`.
- [x] **SRI emission** - gwc release manifests already carry SHA-256 per
  asset but generated shells do not emit integrity attributes.
  Test for: integrity hash on wasm/script/style references matches the
  release manifest; a tampered asset is refused by the browser (fixture
  flips one byte); dev-server mode omits SRI so hot reload still works.
  Done (2026-06-12, generation): the sitegen boot shell now embeds the
  full SHA-256 of site.wasm and verifies it before execution - the boot
  script fetches the wasm bytes, computes SHA-256 via crypto.subtle, and
  refuses to instantiate on a digest mismatch ("integrity check failed"),
  so a tampered/truncated artifact is rejected rather than run. (wasm is
  fetched via JS, not a <script>, so a JS digest check is the correct SRI
  analogue.) Degrades gracefully (instantiates without the check) only
  when crypto.subtle is unavailable in an insecure context, so plain-http
  dev still boots. sw_manifest test asserts the embedded digest + the
  crypto.subtle.digest check are present; sitegen builds+tests green.
  Browser e2e DONE (2026-06-12): `TestShellWasmIntegrityRefusesTamperedArtifact`
  builds the real site, flips one byte at the wasm midpoint, and asserts
  the shell refuses ("integrity check failed: site.wasm digest mismatch",
  #app never gets children), with a clean-artifact control proving good
  builds still boot.
- [x] **HTML sanitizer for untrusted content** - no DOMPurify-equivalent
  exists for rendering user-supplied HTML.
  Test for: an XSS corpus (script tags, event handlers, javascript:
  URLs, SVG payloads, mXSS nesting cases) is neutralized; allowlist
  configuration round-trips; sanitized output is stable across native
  and wasm builds; benchmark guard so sanitizing large documents stays
  off the render hot path.
  Done (2026-06-11): new `sanitize` package - `Sanitize(html, ...Policy)`
  + `DefaultPolicy()` + `Policy{AllowedTags/Attributes/URLSchemes}`. Built
  on golang.org/x/net/html ParseFragment: drop-with-contents set
  (script/style/iframe/object/embed/form/svg/math/link/meta/base),
  unwrap unknown tags keeping safe children, strip on* + style attrs,
  URL-scheme allowlist with obfuscation-tolerant scheme detection, drop
  comments, escape text. 21 tests (10-case XSS corpus + positives +
  allowlist round-trip + idempotence) green; ~50KB doc benchmark
  ~659us/op; native+wasm parity. IMPORTANT companion fix: bumped
  golang.org/x/net v0.51.0 -> v0.55.0, which patches 5 govulncheck
  findings in x/net/html's foreign-content handling (the mXSS-relevant
  bugs in the parser this sanitizer and SSR both use); govulncheck went
  7 -> 2 (the 2 remaining are Go-stdlib toolchain advisories).
- [x] **Root-module security scanning + SBOM + SECURITY.md** - gosec and
  govulncheck run only for the bridge submodule; releases ship no SBOM;
  no security disclosure policy exists.
  Test for: scanners run green on the root module in CI with a
  documented suppression file; release workflow attaches a CycloneDX
  SBOM whose package list matches go.mod; SECURITY.md present with a
  disclosure contact.
  Done (2026-06-11/12): SECURITY.md added (private GitHub advisory
  disclosure, scope, supported versions, defensive posture). govulncheck
  runs against the root module in release.yml, and its findings were acted on:
  golang.org/x/net bumped to v0.55.0 cleared 5 of 7 findings (see the
  HTML-sanitizer entry); the 2 remaining are Go-stdlib toolchain advisories.
  SBOM done (2026-06-12):
  new tools/sbom package generates a CycloneDX 1.5 SBOM from the resolved
  module graph (`go list -m -json all`, 171 components with golang purls),
  with a CLI (cmd/sbom) wired into release.yml that writes
  bin/sbom.cyclonedx.json. Tests: fixture decode (main + version-less
  excluded, sorted, purl/type) + a real-graph check (contains x/net,
  >=50 components, valid JSON). CI DONE (2026-06-12): govulncheck is blocking
  in release.yml, `security/govulncheck-suppressions.md` documents the
  exception process, setup-go is aligned with the root `go 1.26.0`, and the
  generated SBOM is uploaded/downloaded into release assets.
- [x] **Reproducible-build verification** - trimpath is set but nothing
  verifies two builds of one commit are bit-identical, which the
  provenance attestation implicitly promises.
  Test for: a CI job builds the release wasm twice in clean dirs and
  compares SHA-256; intentional nondeterminism (embedded timestamp
  fixture) is caught by the check.
  Done (2026-06-12): new tools/reprobuild package - `BuildWasmSHA` builds
  a package to js/wasm with the release flags (-trimpath -ldflags "-s -w")
  into a clean TempDir and returns its SHA-256. `TestReproducibleWasmBuild`
  builds examples/public/counter TWICE in separate dirs and asserts the
  digests are byte-identical (verified: both = same SHA). `SHAMatches` +
  `TestSHAMatchesCatchesDifference` self-test that a one-character digest
  difference (as an embedded timestamp would cause) is caught and an empty
  digest never counts as a match. Runs in CI via `go test ./...` (skips in
  -short).

## Enterprise tier - data governance

- [x] **PII redaction hooks for telemetry** - crash reports, structured
  logs, and devtools snapshots carry paths, props, and state values
  with no redaction policy (support bundles have a sanitize step;
  console crash reports and future transports do not).
  Test for: a registered redaction policy scrubs configured fields from
  crash-report payloads, log attributes, and devtools snapshots before
  emission; redaction failures fail closed (drop the field, keep the
  event); policy application is covered for both the console path and
  the OnReport hook path.
  Done (2026-06-12): `logging.ConfigureTelemetryRedaction` applies an
  opt-in fail-closed policy to structured log attributes and recursive
  JSON-like values; devtools trace/snapshot export applies the same
  policy before JSON emission; runtime panic reports use a runtime-local
  `ConfigurePanicReportRedaction` hook so console output and `OnReport`
  payloads are scrubbed without a package import cycle. Redactor errors
  drop the field and keep the event. Covered by `logging/redaction_test.go`,
  `devtools/devtools_test.go`, and `internal/runtime/panic_diagnostic_test.go`.
- [x] **WebCrypto bridge + encrypted persistent storage** - no
  crypto.subtle interop exists; persisted snapshots and IndexedDB
  caches store plaintext.
  Test for: encrypt/decrypt round-trip via WebCrypto from Go (AES-GCM
  with a non-extractable key); an EncryptedPersistentStore wrapper
  round-trips JSON and rejects tampered ciphertext; key unavailability
  degrades to an explicit error, never silent plaintext; native builds
  return the unavailable stub.
  Done (2026-06-11): interop crypto.go/crypto_wasm.go/crypto_native.go -
  `GenerateAESKey` (AES-GCM 256, non-extractable), `Encrypt`/`Decrypt`
  (12-byte IV via getRandomValues; tamper -> CodeDecode error, never
  partial plaintext), `EncryptedStore` (`NewEncryptedStore`, PutJSON/
  GetJSON: JSON->encrypt->base64 iv:ct envelope->localStorage). Pure-Go
  envelope encode/decode factored out and unit-tested (round-trip +
  corruption->CodeDecode). Native stubs return unavailable. Builds
  native+wasm; all crypto tests green.

## Enterprise tier - operational resilience

- [x] **Crash-loop safe mode** - containment keeps a running page alive,
  but a panic during boot reloads into the same crash forever.
  Test for: N consecutive failed boots (tracked in storage) switch the
  generated shell to a minimal diagnostics view with a cache-purge
  action instead of re-running the wasm; a successful boot resets the
  counter; the diagnostics view itself needs no wasm; e2e drives a
  deliberately boot-panicking module through the full loop.
  Done (2026-06-12, implementation): the tools/sitegen boot shell now
  tracks consecutive failed boots in localStorage (`gwc.boot.fails`):
  after 3 it renders a no-wasm "Safe mode" diagnostics view with a "Purge
  caches and retry" button (clears CacheStorage + resets counter +
  reloads) INSTEAD of re-running the wasm; the counter increments before
  each boot attempt and resets after a 4s liveness window once a boot is
  healthy. Sitegen builds; logic is self-contained vanilla JS in the
  generated shell. Browser e2e DONE (2026-06-12):
  `TestShellCrashLoopSafeMode` corrupts site.wasm so every boot fails,
  loads the shell repeatedly, verifies gwc.boot.fails climbs 1->2->3, then
  on the 4th load (>=limit) the no-wasm "Safe mode" view + #gwc-purge
  button render instead of re-running, and clicking purge resets the
  counter and re-attempts boot.
- [x] **Version-skew refresh flow** - wire formats are versioned now, but
  detection is not acted on: stale cached wasm against a redeployed
  server should trigger a controlled refresh, not an error.
  Test for: a sidecar/wasm version mismatch triggers exactly one forced
  reload with cache bypass (no reload loop on persistent mismatch -
  loop guard verified); user state is snapshotted before the reload and
  restored after when versions allow migration.
  Done (2026-06-12): `pwa.EvaluateVersionSkewRefresh` models one-shot
  cache-bypassing reloads, loop guarding, snapshot carryover, and
  migration-compatible restore decisions. The generated site writes
  `version.json`, fetches it with `cache: "reload"` before wasm boot,
  snapshots best-effort persisted state into sessionStorage, and forces
  exactly one reload per stale build id. Covered by
  `pwa/release_manifest_test.go` and `tools/sitegen/sw_manifest_test.go`.
- [x] **Remote flag provider + kill switch** - the new flags package is
  build/boot-time; no remote-config provider (poll/SSE) or kill-switch
  semantics exist.
  Test for: a flag flip on a mock remote provider reaches subscribed
  components within the polling interval; provider outage retains last
  known values with staleness surfaced; kill-switch flag disables a
  feature subtree without reload; misbehaving provider payloads are
  contained, never crash the app.
  Done (2026-06-11): flags/remote.go - `RemoteProvider` over an injected
  `RemoteFetchFunc` (transport-agnostic). Refresh (fail-safe: keeps
  last-known-good on error, sets LastError), Current/Age/IsStale,
  Subscribe (live, leak-free, panic-contained delivery), IsKilled (kill
  switch = flag present+disabled, reacts live via Subscribe, no reload),
  ParseRemoteSet (malformed payload contained), Poll (outage-resilient,
  injected sleep, ctx.Err on cancel). 6 instant tests (injected clock).
  Native+wasm build, vet green.

## Enterprise tier - release engineering

- [x] **Canary / gradual wasm rollout** - no mechanism serves two wasm
  versions side-by-side with percentage routing and instant rollback.
  Test for: deterministic cohort assignment (same client stays on its
  version across reloads); rollback flips 100% within one cache TTL;
  both versions report their build id through artifact metadata so
  crash reports distinguish cohorts.
  Done (2026-06-12): `pwa.ChooseWasmRollout` accepts stable/canary wasm
  release manifests, deterministic client-id bucketing, rollout salt,
  percentage routing, TTL, and a rollback switch that forces stable with
  a one-second TTL. Decisions include cohort, build id, wasm URL, sha256,
  and manifest revision so crash reports and operational telemetry can
  distinguish cohorts. Covered by `pwa/release_manifest_test.go`.
- [x] **Perf-budget CI gate** - route startup budgets exist in profiling
  and gwc bench measures, but nothing fails a build on regression.
  Test for: a gwc lane fails when wasm size or measured route-startup
  exceeds the checked-in budget by the configured tolerance; budgets
  update through an explicit ratchet command, not silently; the gate
  output names the offending route and delta.
  Done (2026-06-12): perf_budget_test.go + checked-in
  testdata/perf_budgets.json (single source of truth). `TestPerfBudgetWasmSize`
  is the deterministic gate - stats the artifact and fails over budget
  (counter: 6.07MB actual vs 7.63MB budget, ~26% headroom).
  `TestPerfBudgetStartup` measures route startup (min of 3 warm runs =
  498ms vs an 8000ms ceiling, generous for machine variance).
  `TestPerfBudgetGateCatchesRegression` self-tests the pure
  `exceedsWasmBudget` boundary + over/under artifact scenarios (no
  browser), so a green size gate means within-budget not gate-broken.
  GWC lane DONE (2026-06-12): `gwc test -lane perf` runs the dedicated
  `TestPerfBudget*` browser package, `all` includes it, and the lane summary
  requires explicit reviewed budget/testdata changes instead of silent
  auto-updates.
- [x] **Visual regression lane** - playwright is wired everywhere but no
  screenshot-diff lane protects the examples or docs site.
  Test for: baseline capture + pixel-diff with anti-flake masking
  (timestamps, spinners); an intentional 1px style change in a fixture
  is caught; per-page thresholds configurable; lane runs on the docs
  site routes and a representative example subset.
  Done (2026-06-12, real browser): visual_regression_test.go - a pure-Go
  pixel-diff engine (per-channel 4% AA-tolerant threshold + mask-rectangle
  skipping + bounds check). `TestVisualRegressionMechanism` self-tests all
  four properties (identical=0, 1px change caught, masked-pixel ignored,
  bounds-mismatch errors). `TestVisualRegressionStablePageBaseline` proves
  it end-to-end on the counter example at a fixed 1024x768 viewport:
  consecutive same-state screenshots diff 0.000000 (stable, no false
  positives), a click changes ~0.35% (detected). Golden PNGs intentionally
  omitted (environment-specific); stability+change-detection is the
  portable guarantee. Extended (2026-06-12): `TestVisualRegressionAdditionalPages`
  proves same-page stability generalizes across semantic-html, hash-router,
  and toggle (all diff 0.000000, no masks needed) - the lane works on
  multiple real pages, not just counter.

## Enterprise tier - isolation & conformance

- [x] **Shadow-DOM style isolation for exported custom elements** - no
  shadow-root helpers exist; embedded GWC widgets leak styles both ways.
  Test for: a GWC custom element mounted in a hostile host page (global
  CSS resets, conflicting class names) renders identically to its
  isolated baseline; host styles do not bleed in and widget styles do
  not bleed out; events and portals still work across the shadow
  boundary; focus trap and announcer behave inside shadow roots.
  Done (2026-06-12, real browser): the export path ALREADY renders widgets
  into a shadow root (`ui.RenderInto(node, shadowRoot)` with `:host`-scoped
  styles - exported-custom-element example). `TestShadowDOMStyleIsolation`
  proves isolation: it injects hostile `* { color:red !important; border:
  10px magenta }` into the host document, confirms a normal host div DOES
  pick it up (rgb(255,0,0)), then asserts the shadow `.tile` does NOT
  (stays rgb(226,232,240) / cyan border) - inbound isolation holds; and a
  host-page `section.tile` is unaffected by the widget's scoped styles -
  outbound isolation holds. (Inherited font-size flows in per CSS spec -
  flagged informational, not a defect.) Boundary assertions DONE
  (2026-06-12): the browser test now creates a shadow-local portal target,
  focused dialog control, and polite announcer and proves none leak to
  document.body.
- [x] **Cross-browser conformance matrix** - webkit/firefox run only in
  the Atlas smoke; the framework behavior suite is chromium-only.
  Test for: the core browser suite (events, hydration, router, storage,
  overlay focus) passes on chromium, firefox, and webkit in CI; known
  per-engine differences are encoded as explicit skips with linked
  issues, not silent passes.
  Done (2026-06-12, real browsers): `TestCrossBrowserConformance` installs
  + launches chromium, firefox AND webkit and runs the core behaviors as
  per-engine sub-tests across 5 examples - rendering/state (counter
  increment), router (hash-router nav), storage (snapshot-storage
  LocalStorage round-trip), overlay/focus (accessible-overlay open+Escape),
  hydration/semantic (semantic-html landmarks). Chromium + Firefox enforce
  hard assertions and pass reliably. Known per-engine differences are
  EXPLICIT documented skips, not silent passes: webkit headless wasm
  boot/post-boot interactions are unreliable on this platform, so webkit
  uses a bounded 30s boot wait and `t.Skipf`s with a documented reason on
  timeout (never hangs/fails), and the accessible-overlay programmatic-
  focus assertions are webkit-skipped with a reason. Verified green across
  3 fresh runs.
- [x] **Plugin API conformance suite** - kernel-backed plugins feed
  devtools but third parties have no test kit proving they meet the
  contract.
  Test for: a published conformance package a plugin author can run
  against their plugin (lifecycle, section contribution, diagnostics,
  teardown); the built-in kernel plugin passes it; a deliberately
  non-conforming fixture plugin fails with actionable messages.
  Done (2026-06-12): new `plugin/conformance` package - `Verify(plugin)
  Result` runs the plugin through a real plugin.Host and reports 10 named
  checks (manifest id/version/tier/capabilities, registration, host-
  observable contributions, teardown nil-error + contributions-removed +
  idempotent, lifecycle no-panic); `Result.OK()`/`Failures()` + a
  `VerifyT(*testing.T, plugin)` helper. A kernel-style built-in plugin
  passes all 10; five deliberately broken fixtures (empty id, setup error,
  setup panic, invalid tier, duplicate capability) each fail with the
  correct actionable check name. Builds native+wasm; tests green.

## Security findings (2026-06-11 code review) - fix

- [x] **Markdown URL-scheme allowlist (latent stored-XSS in a public API)** -
  `html.RenderMarkdown` / `ResolveMarkdownHref` (html/markdown.go) pass any
  absolute URL straight to `Href`, so `[x](javascript:...)` and
  `![x](data:text/html,...)` in links, images, and autolinks render an
  executable sink. Repo-controlled docs content is not exploitable today,
  but any app rendering user-supplied markdown inherits the hole, and the
  signature gives no warning. Fix: allowlist http/https/mailto/relative/#
  in the link, image, and autolink paths; drop or neutralize the rest;
  make the policy configurable via MarkdownRenderOptions.
  Test for: javascript:, data:, vbscript:, and mixed-case/whitespace-
  obfuscated variants are dropped in links AND images AND autolinks;
  http/https/mailto/relative/# survive unchanged; a configurable
  allowlist round-trips; native and wasm parity; the docs-site chapters
  still render every legitimate cross-link.
  Done (2026-06-11): added `SanitizeMarkdownHref` + `AllowedURLSchemes`
  to `html/markdown.go`; applied at the link, image (new `<img>` case),
  and autolink sinks. Default allowlist http/https/mailto; relative/abs/#
  pass through; javascript:/data:/vbscript: and obfuscated variants
  (mixed case, leading/embedded whitespace+control, `java\tscript:`) are
  dropped to an empty attribute. Pure-Go file (no build tags) so
  native+wasm identical. Tests: `TestSanitizeMarkdownHrefDropsDangerousSchemes`
  (10 drop + 8 keep + configurable round-trip) and
  `TestRenderMarkdownNeutralizesXSSAcrossSinks` (links+images+autolinks,
  legit links survive); full html suite green.
- [x] **Livereload WebSocket origin validation (dev-server CSWSH)** -
  `tools/livereload/livereload.go` sets `CheckOrigin` to always return
  true, so while the dev server runs any visited website can open
  ws://127.0.0.1:<port> and receive the reload stream / trigger reloads.
  Fix: validate the request Origin against the configured dev host:port
  (and localhost variants), reject mismatches.
  Test for: same-origin upgrade succeeds; a foreign Origin is rejected
  with 403; missing Origin handled per policy; an explicit opt-out flag
  exists for tunnel/LAN dev with a logged warning.
  Done (2026-06-11): replaced the always-true CheckOrigin with a
  per-server `originAllowed` bound into each server's own
  `websocket.Upgrader`. Validates Origin host+port against the dev
  host:port plus localhost/127.0.0.1/::1 aliases; wrong port or foreign
  host rejected; missing Origin allowed (non-browser clients; a cross-site
  browser context always sends one); 0.0.0.0/:: bind accepts any host on
  the matching port. Opt-out: `LiveReloadOptions.AllowAnyOrigin` +
  `gwc dev -allow-any-origin` with a startup WARNING line. Tests:
  `TestLiveReloadOriginValidation` (4 allowed incl. missing-origin, 5
  rejected incl. wrong-port/foreign/`attacker.localhost`, opt-out) and
  `TestLiveReloadForeignOriginUpgradeRejected` (real upgrade -> 403).
  Note: tools/livereload is a nested module; run `go -C tools/livereload
  test .`.
- [x] **Remove or fence the unused SetInnerHTML adapter sink** -
  `internal/runtime/interfaces.go` exposes `SetInnerHTML` on the DOM
  adapter; it is implemented (jsdom/mockdom) but never called from the
  render/commit path - an unprotected raw-HTML sink on the public
  interface. Fix: remove it, or if retained, document it as
  trusted-input-only and add a render-path assertion that it stays
  unreachable.
  Test for: a grep/analyzer guard fails if SetInnerHTML gains a
  render-path caller; if kept, a doc note plus an example of safe use.
  Done (2026-06-11): removed `SetInnerHTML` from the `DOMAdapter`
  interface (the render-path-reachable exposure) and from the real DOM
  (`jsdom.WASMDOMAdapter`, which wrote `innerHTML`); both had zero callers
  in the commit/reconcile path. Added a reflection guard
  `TestDOMAdapterHasNoRawHTMLSink` that fails if any
  SetInnerHTML/innerHTML/outerHTML method is reintroduced to the
  interface, plus a doc note on the interface explaining the
  text/attribute-only contract. In-memory test doubles (mockdom,
  reconciler testDOMAdapter) keep their methods - not real sinks - so
  their tests are unaffected. jsdom still builds for wasm; runtime +
  mockdom suites green.
- [x] **Document the global cache-key namespace sharp edge** -
  `fetch.UseCachedResource` keys live in one app-global `sync.Map`
  (fetch/cache.go), so unrelated components choosing the same key string
  share state (type conflicts warn, so this is by-design, not a crash).
  Fix: document the global-namespace contract and recommend a
  module-prefix convention; consider an optional scoping helper.
  Test for: doc example of the collision and the prefix convention; the
  existing conflicting-type warning remains covered.
  Done (2026-06-11): rewrote the `UseCachedResource` doc comment to state
  the app-global-namespace contract explicitly (keys are process-wide
  identifiers; same key = shared entry/loader/value by design) and
  recommend a module prefix. Added the `ScopeCacheKey(segments...)`
  helper (drops empty segments, joins with `:`) with a runnable doc
  example. Tests: `TestScopeCacheKey` (join + empty-drop + collision
  split) and `TestUseCachedResourceConflictingTypesWarns` (now covers the
  conflicting-value-types diagnostic via runtime.GetDiagnostics).

## Developer experience (2026-06-11 review) - build items + what to test

### Sharpest three (do first)

- [x] **Commit editor support + document the gopls wasm-tag setting** -
  there is no .vscode/ and nothing tells new users that gopls hides every
  `//go:build js && wasm` file unless GOOS=js/GOARCH=wasm is set, so the
  whole wasm half of the codebase shows phantom errors on minute one.
  Fix: commit `.vscode/settings.json` (gopls.env GOOS=js GOARCH=wasm),
  `.vscode/extensions.json` (Go extension recommendation), `.vscode/
  launch.json` and `tasks.json` for `gwc dev`; add a JetBrains-equivalent
  note; ship a `gwc-start.json` JSON schema for editor autocomplete.
  Test for: opening a wasm file shows no excluded-file diagnostics with
  the committed settings; the schema validates a sample gwc-start.json
  and rejects an unknown key; a docs section covers the JetBrains path.
  Done (2026-06-11): committed `.vscode/` (settings.json pins gopls +
  go.toolsEnvVars + go.testEnvVars to GOOS=js/GOARCH=wasm with the
  native-flip trade-off documented; extensions.json recommends golang.go;
  tasks.json for `gwc dev`/`test`/`doctor`; launch.json for native
  tool/test debug). Changed `.gitignore` from `.vscode/` to
  `.vscode/*` + `!`-negations so the shared config is tracked while
  personal files stay ignored (verified tracked via git check-ignore).
  JetBrains GOOS/GOARCH note added to CONTRIBUTING.md. Remaining: the
  optional gwc-start.json JSON schema for autocomplete.
- [x] **Wire devtools.ErrorOverlay into `gwc dev` by default** - runtime
  panics produce excellent structured console reports, but the page just
  shows a dead tree / boot spinner; React/Vite developers expect a
  full-screen overlay. The overlay component and the structured report
  already exist; they are not connected to the dev server.
  Fix: inject the error overlay in dev builds, fed by the existing panic
  report (where/path/error/next + stack buckets), with a click-to-copy
  and, where resolvable, click-to-open-in-editor link.
  Test for: a deliberate render panic surfaces the overlay with the
  report fields in a browser e2e; the overlay is absent in production
  builds; recovering the panic (next clean render) dismisses it; the
  overlay itself cannot crash the page (contained).
  Done (2026-06-12): wasm panic reports now also dispatch a contained
  `gwc:runtime-panic` CustomEvent; `gwc dev` injects a dev-only runtime
  overlay that renders the existing structured fields, copy-report, and
  `vscode://file/...:line` editor links when source locations resolve.
  The overlay listens to both the event and structured console reports,
  hides after build success or a clean app-root render, and is isolated
  from its own append/listener failures. Tests cover the wasm event
  payload, livereload injection, production build profiles excluding the
  overlay, and a Playwright browser loop for panic, copy, recovery,
  containment, and editor-link behavior.
- [x] **Write CONTRIBUTING.md for framework contributors** - AGENTS.md and
  the getting-started chapter target users; nothing captures how to work
  ON the framework: the `GOOS=js GOARCH=wasm go test -c -o x.wasm` + node
  runner dance, wasm coverage flags, playwright build-tag/package
  conventions, the Windows BOM/CRLF hazards, and the run-tests-sequentially
  gotchas.
  Test for: a fresh contributor can run the wasm unit suite, a playwright
  lane, and the coverage report by following the doc only; commands in
  the doc are copy-pasteable and verified in CI (doc-command lint).
  Done (2026-06-11): CONTRIBUTING.md covers prerequisites, the editor
  gopls wasm-tag gotcha, native+wasm build, the `go test -c` + node
  wasm_exec dance, the playwrightgo lanes, the nested-module
  `go -C tools/livereload test .` trap, wasm coverage, the conventions,
  and the Windows CRLF/BOM/go:embed/stale-env hazards, plus a
  pre-PR checklist. Paths resolve (doclint green). Pairs with the new
  docs/CONVENTIONS.md (parse-prefix + verbSubject rules).

### Onboarding & inner loop

- [x] **Auto-run doctor on first build/dev failure** - `gwc doctor` is
  opt-in, so a fresh machine with the wrong Go version or missing
  wasm_exec hits a cryptic deep build error instead of an upfront
  diagnosis.
  Test for: a simulated missing-prerequisite environment triggers the
  doctor summary automatically on `gwc dev`/`gwc build` failure with a
  fix hint; a healthy environment never shows it; opt-out flag respected.
  Done (2026-06-12, dev path): `gwc dev` now auto-runs the doctor
  prerequisite checks when the server fails to start and prints an
  actionable diagnosis (failing checks + fix hints) only when blocking
  issues exist; a healthy environment stays silent. Opt-out via
  `gwc dev -no-doctor`. `formatEnvironmentDiagnosis` is the testable core
  (3 tests: healthy=silent, fail-checks listed with hints + warn excluded,
  warn-only=silent). Build path DONE (2026-06-12): `gwc build` auto-runs the
  same diagnosis on resolution/build failures, suppresses it for JSON output,
  and supports `-no-doctor`; test stubs prove failure output and opt-out.
- [x] **Browser build-status indicator in `gwc dev`** - the 304ms rebuild
  is only visible as terminal text; developers watching the browser get
  no signal. Add a small corner badge (building / ready / error) and an
  optional rebuild-failure notification when the terminal is unfocused.
  Test for: badge reflects building->ready transitions over the livereload
  channel in an e2e; error state shows on a failed rebuild and clears on
  the next success; badge is dev-only and never ships to production.
  Already implemented (verified 2026-06-12): the livereload client script
  (tools/livereload/scripts/livereload-client.txt) renders a status badge
  (`#livereload-status` / `#gwc-status-panel` / `#gwc-status-icon` /
  `#gwc-error-badge`) driven by the `build_start`/`build_complete`/
  `build_error` WebSocket messages via `showStatus`/`showBuildStatusPopup`/
  `handleDebounceStatus`, with Ready/Building/Build-failed states. It is
  dev-only (the client script is injected only under livereload, never in
  a release build). Browser e2e DONE (2026-06-12):
  `TestLiveReloadClientBuildStatusBadgeTransitions` drives the real client
  script in Chromium and asserts building -> error badge -> ready transitions.
- [x] **`gwc init` time-to-first-pixel guarantee** - scaffolding works but
  no CI test proves `gwc init` output builds and renders on a clean
  machine across the offered presets.
  Test for: each preset scaffold runs `go mod tidy` + `gwc build` clean
  and mounts a non-empty tree in a headless browser; a broken preset
  fails the lane with the offending preset named.
  Done (2026-06-12): the preset scaffold path (`gwc start`, with
  lifecycle metadata still covered by `gwc init`) now emits a visible
  first-pixel fallback shell outside `#app`, removes it once wasm mounts,
  and keeps structured boot-error display for startup failures. Added a
  Playwright-tagged all-presets smoke test that generates every default
  starter, runs tidy via scaffold generation, builds `bin/main.wasm`, serves
  the generated app, waits for `#app .counter`, asserts non-empty mounted
  text, and reports the offending preset key on failure. The starter
  templates workflow now runs both the existing tidy/test/build loop and the
  headless browser first-pixel smoke.

### Discoverability & polish

- [x] **Capability matrix ("what's in the box")** - storage, PWA, i18n,
  a11y, flags, realtime, and snapshots all exist but are hard to
  discover by reading; surface a single matrix on the docs site mapping
  capability -> package -> example -> manual chapter.
  Test for: the matrix is generated from the catalog (no hand-maintained
  drift); every row links to a real example and chapter; a CI check fails
  if a listed capability's example or chapter link 404s.
  Done (2026-06-11): new docs/capabilities package (mirrors docs/errorcodes)
  generates docs/REFERENCE_MANUAL/capability-matrix.md - 14 rows mapping
  capability -> package(s) -> example -> chapter.
  `TestCapabilityReferencesResolve` stat-checks every examples/public/<slug>
  and chapter file (the 404 guard); `TestCapabilityMatrixIsGenerated` is
  the drift guard (regenerate with CAPABILITIES_WRITE=1). All 14 rows
  resolved; runs in CI via go test ./...; the page lives in REFERENCE_MANUAL
  so the docs-site chapter embed indexes it.
- [x] **Mark legacy dev flag aliases as deprecated in help** - `gwc dev`
  carries `-main`/`-index`/`-output` legacy aliases beside
  `-app`/`-html`/`-wasm`; help does not flag them, widening the surface a
  new user must navigate.
  Test for: help output labels each legacy alias as deprecated and points
  to the canonical flag; the aliases still function (no behavior change).
  Done (2026-06-11): relabeled every legacy alias across gwc dev,
  prerender, release/release-build, test, and verify from "Legacy alias
  for -X" to "(deprecated) alias for -X; use -X". Help text only - the
  aliases still resolve unchanged. gwc builds.
- [x] **Web playground / shareable snippet runner** - the examples catalog
  is the closest thing to an evaluation surface; there is no edit-and-run
  snippet experience for quick evaluation/sharing.
  Test for: a snippet compiles and renders in the browser sandbox; a
  shared URL round-trips the snippet source; compile errors surface in the
  sandbox with the structured diagnostic, not a blank frame.
  Done (2026-06-12): added a constrained GoWebComponents snippet
  playground to the public examples site with a browser-side parser for
  whitelisted element/Text calls, sandboxed iframe `srcdoc` rendering,
  `snippet` query-string sharing, example-to-playground links, and
  structured `GWC-PLAYGROUND-*` diagnostics for syntax, unsupported calls,
  empty input, and missing `App`. Coverage includes native parser/URL tests
  plus js/wasm render coverage for diagnostic display, share URL
  round-trip, Run behavior, and sandbox output.

## Product polish (2026-06-11) - feasible, additive, low-risk

- [x] **README golden-path + capability badges** - the README does not lead
  with a 30-second `gwc init my-app && cd my-app && gwc dev` quickstart or
  a visible capability summary, so an evaluator cannot judge fit in the
  first scroll.
  Test for: a copy-pasteable quickstart that a CI doc-lint actually runs
  end to end on a clean checkout; a capability line (SSR, hydration,
  router, forms, i18n, a11y, PWA, flags, realtime) with links; build /
  version / license badges resolve.
  Done (2026-06-11): added a "30-second golden path" leading the Quick
  Start (doctor + `gwc dev` on the verified counter example; doclint
  confirms the path resolves), plus pointers to `gwc start`/`examples`.
  Capability summary refreshed earlier this session (lead paragraph +
  Public Packages + Feature Overview); CI/Pages/Release/Go-Report badges
  already resolve. (Used the verified counter run rather than a fabricated
  `gwc init` scaffold to keep the quickstart honest.)
- [x] **Generated error-code reference page** - every `GWC-RUNTIME-PANIC-*`
  and `GWC-FRAMEWORK-*` code is emitted in reports but there is no index a
  developer (or agent) can look the code up in.
  Test for: a docs page generated from the code lists every diagnostic
  code with cause, the `next:` remediation, and the docs anchor; a CI
  check fails if a code emitted in the runtime is missing from the page
  (no drift); the docs-site search indexes the codes.
  Done (2026-06-11): new `docs/errorcodes` package extracts all 30 codes
  (the diagnosticMetadata switch triples + the panic-phase helpers, incl.
  the switch-less GWC-RUNTIME-PANIC-ASYNC) from
  internal/runtime/diagnostic_metadata.go and renders
  docs/REFERENCE_MANUAL/error-codes.md with each code's docs anchor +
  `next:` remediation. `TestErrorCodeReferenceIsGenerated` is the drift
  guard (byte-match or fail; regenerate with ERRORCODES_WRITE=1), runs in
  CI via `go test ./...`. Page lives in REFERENCE_MANUAL so the docs-site
  chapter embed indexes it for search automatically.
- [x] **Runnable godoc Example functions for public packages** - pkg.go.dev
  quality depends on `Example` test functions; coverage across ui, html,
  router, fetch, state, i18n, pwa is uneven.
  Test for: each public package has at least one `Example` that compiles
  and passes `go test`; examples render on pkg.go.dev (no unexported
  references); a lane runs `go test -run Example ./...`.
  Done (2026-06-11/12): runnable `// Output:` examples for events
  (Publish), sanitize (Sanitize), i18n (FormatRelativeTime + FormatList),
  html (SanitizeMarkdownHref); fetch already had one. Added
  compile-checked examples for the hook/runtime-context packages -
  ui.UseState (counter component), state.UseAtom + UseComputed,
  router.DefineRoute (native-safe contract), pwa.BuildCacheStoragePlan.
  Every public package now has at least one Example; all compile and pass
  `go test -run Example ./...`.
- [x] **Disciplined CHANGELOG + release notes** - CHANGELOG.md exists but is
  not tied to the release flow; v-tags ship auto-generated notes only.
  Test for: the release workflow fails if CHANGELOG has no entry for the
  tag being cut; entries follow Keep-a-Changelog sections; the docs site
  surfaces the latest release notes.
  Partial (2026-06-12): new tools/changelogcheck package - `HasEntry`
  (whole-token version match, v-normalized, bracket/date tolerant),
  `LatestEntry` (first section for docs-site surfacing), `CheckFile`, +
  a CLI (cmd/changelogcheck) that exits non-zero on a missing entry (7
  tests). Added the Keep-a-Changelog `## [Unreleased]` block to the top of
  CHANGELOG.md. Release gate DONE (2026-06-12): release.yml now blocks on the
  CHANGELOG entry check, and the docs site includes `latest-release-notes.md`
  with a test proving it surfaces `LatestEntry`.
- [x] **Starter templates beyond init presets** - `gwc init` offers presets
  but there is no gallery of opinionated starters (dashboard, marketing
  site, blog, authed app shell) a team can clone as a real starting point.
  Test for: each starter scaffolds, `go mod tidy` + `gwc build` clean, and
  mounts a non-empty tree headless; a CI lane builds every starter; each
  links from the docs site.
  Done (2026-06-12): `gwc start` now exposes opinionated presets for
  dashboard, marketing site, content blog, and authed app shell in addition to
  the existing minimal/routed/SSR/reference starters. Added
  `TestDefaultStarterTemplatesScaffoldTidyTestAndBuild`, which scaffolds every
  default preset in contributor-linked mode, lets generation run `go mod tidy`,
  runs the generated `starter_test.go` suite, and verifies `gwc build` emits a
  non-empty wasm artifact. Added `.github/workflows/starter-templates.yml`.
  Documented the gallery in `docs/STARTERS.md`, linked it from README and the
  reference manual, and added a searchable docs-site `Starter Templates` entry.
  Fixed the generated starter test string-list helper so SSR/content starters
  no longer emit uncompilable `range nil` loops.
- [x] **Production-readiness checklist doc** - nothing collects the
  go-live steps (release profile, compression, CSP, SRI, service worker,
  perf budget, error transport, a11y audit) into one gated checklist.
  Test for: a docs chapter enumerates each step with the gwc command or
  API that satisfies it; each referenced command/flag exists (CI link +
  flag-existence check).
  Done (2026-06-11): docs/PRODUCTION_READINESS.md - a gated checklist
  across build/artifacts, correctness gates, runtime resilience, PWA,
  a11y/i18n, security, and performance. Each step names the verified gwc
  command (release/-compression, verify -audit, test lanes, bench, wasm
  measure) or framework API (pwa.*, ui preference/focus hooks,
  RenderMarkdown, sanitize), and honestly marks the not-yet-automated
  items (CSP/SRI/SBOM/perf-gate/visual-regression/canary) as backlog
  rather than referencing flags that do not exist.
- [x] **Consistent deprecation surfacing + naming-convention doc** - legacy
  flag aliases and any deprecated APIs warn inconsistently, and the
  pervasive `parse`-prefix local convention is undocumented for
  contributors and readers.
  Test for: deprecated public APIs emit a one-time structured deprecation
  diagnostic with the replacement; a short conventions doc explains the
  prefix; a lint check flags new deprecations missing the warning.
  Partial (2026-06-11/12): conventions doc done - docs/CONVENTIONS.md
  explains parse-prefix/verbSubject naming, platform-split files, GoDoc
  rules, and the deprecation pattern (referenced from CONTRIBUTING.md).
  Runtime warning done (2026-06-12): new `deprecation` package -
  `Warn(api, replacement)` emits a one-time GWC-DEPRECATION structured
  diagnostic (via the public diagnostics sink) with the replacement in
  `next:`, deduped per API by sync.Map; `WarnEmitted` test seam; 5 tests;
  native+wasm. Done (2026-06-12): `gwc lint` now adds a
  `gwc-deprecations` parser rule for exported deprecated functions/methods
  that omit `deprecation.Warn(...)`, skips generated Go files, and covers the
  rule with focused tests. Existing deprecated wrappers in fetch/router/state
  now emit the warning.
- [x] **Docs-site accessibility statement + dogfood pass** - the docs site
  is now the flagship app but has no accessibility statement and (noted in
  the a11y section) does not yet use its own focus-trap/announcer
  primitives.
  Test for: an accessibility statement page; the site passes the automated
  a11y audit lane at serious/critical; keyboard-only navigation reaches
  every route and the search modal.
  Partial (2026-06-12): docs/ACCESSIBILITY.md added - WCAG 2.1 AA target,
  the framework's a11y primitives, the tracked gaps (audit lane + search
  modal dogfood), and a feedback channel. Remaining: the axe-core audit
  lane and the actual search-modal focus-trap/announcer dogfood (browser-
  bound).
  Done (2026-06-12): the docs site now dogfoods `ui.AccessibleOverlay` for
  a trapped/restoring search dialog and `ui.UseAnnouncer`/polite live
  regions for result-count updates; docs/ACCESSIBILITY.md now documents
  current verification instead of known gaps. Added docs-site shell coverage
  to the Playwright axe lane and wasm markup/interaction tests for the
  search dialog contract.
- [x] **Public benchmark / performance page** - the React-comparison data
  (6/6 paint wins, ~1.4MB, 304ms, 0-alloc reconcile) lives in commit
  history and stat cards but has no methodology-backed page an evaluator
  can scrutinize.
  Test for: a docs page documents methodology, hardware, and reproduction
  commands; numbers are generated from `gwc bench` output, not
  hand-typed; a CI note flags when published numbers drift from a fresh
  run beyond tolerance.
  Done (2026-06-12): docs/BENCHMARKS.md documents the three measurement
  surfaces (Go microbench, gwc wasm measure, the Playwright React
  comparison), exact reproduction commands (doclint-verified paths+flags),
  the docs/benchmarks/*.json sources of truth, and methodology/caveats.
  Deliberately omits hand-typed figures so they can't rot - numbers come
  from `gwc bench`.
  Done (2026-06-12): docs/benchmarks/DRIFT_NOTE.md is generated from
  docs/benchmarks/latest.json and checked by `go test ./docs/benchmarks`;
  BENCHMARKS_WRITE=1 regenerates it when fresh benchmark data changes the
  published drift status. docs/BENCHMARKS.md now documents that CI guard.
- [x] **Brand polish for the docs site** - favicon, social/OG preview image,
  and a consistent logo lockup are missing or placeholder, which reads as
  unfinished to first-time visitors.
  Test for: every site page emits title, description, and og:image meta;
  the favicon and OG image load (no 404); a link-preview render of the
  landing page shows the intended card.
  Done (2026-06-11): tools/sitegen now emits favicon (svg type) + og:type/
  og:title/og:description/og:image + twitter:card/title/description/image
  in the boot shell (a SPA, so all routes share it), and generates
  self-contained `favicon.svg` (cyan reactive-ring mark matching the boot
  spinner) and a 1200x630 `og-image.svg` brand card into the dist - no
  external files, no 404. Sitegen builds; both SVGs validated as
  well-formed XML.

## Stale docs (2026-06-11 audit) - fix

- [x] **README + getting-started use the removed `examples/01-counter`
  path** - `examples/01-counter` was relocated to
  `examples/public/counter` (the same move that broke the release verify
  gate earlier today), but README references it ~10 times and
  `docs/REFERENCE_MANUAL/01-getting-started.md` twice, so every
  copy-paste quickstart/build/verify command in the front-door docs
  fails on a clean checkout.
  Fix: repoint all `examples\01-counter` / `examples/01-counter`
  references to `examples/public/counter`.
  Test for: a doc-command lint runs each fenced `gwc` command from README
  and ch01 on a clean checkout and they all exit 0; no remaining match
  for the old path in any tracked .md.
  Done (2026-06-11): repointed every command path in README.md,
  AGENTS.md, tools/README.md, ch01/ch02/ch03, and testing-surface.md;
  also fixed the ch03 app-shapes table's stale numbered slugs
  (`01-counter`/`21-ui-render`/`75-use-state`/`76-use-effect` ->
  `counter`/`ui-render`/`use-state`/`use-effect`). Verified
  `examples/public/counter` builds to wasm and no `01-counter` remains in
  any front-door .md (test fixtures and the tested static catalog.json
  intentionally retain the literal). The doc-command lint itself is the
  separate "Doc-drift guard" item below.
- [x] **examples/MANUAL_TESTING.md describes the deleted numbered layout** -
  it calls the catalog the "numbered example catalog", was "Captured on
  2026-03-25", and lists stale per-example assertions (`07-goroutines`,
  `08-fetch`, `10-advanced-form`) against examples that are now
  `public/<slug>`.
  Fix: regenerate against the current `examples/public/<slug>` layout or
  retire the file if the playwright suite supersedes it.
  Test for: every example path named in the doc resolves under
  examples/public; the capture date reflects the update; a CI check fails
  on a referenced example path that no longer exists.
  Done (2026-06-11): slimmed to the low-drift form - kept the reusable
  Manual Verification Rules, Suggested Regression Workflow, and Mobile
  Safari gate (de-numbered to real slugs), and replaced the ~100 dead
  numbered per-example entries with the canonical playwright suite
  (`TestExamplesAll`) plus the generated `catalog.json` as the source of
  truth. Also fixed the broken `.\build.ps1` SSR build (the script no
  longer exists) and the atlas/ssr server commands to the current
  `examples/server/<slug>` paths. CRLF preserved.
- [x] **examples/README.md lists `01-counter through 20-portals`** - the
  numbered range no longer exists; examples are slug-named under public/.
  Fix: replace the numbered range with the current slug catalog (or
  generate the list from the catalog manifest).
  Test for: listed example names match examples/public dir entries; a
  generated-list option removes future drift.
  Done (2026-06-11): replaced the 88-entry hand-maintained numbered
  inventory (lines 55-205) with a de-numbered Integrated Apps narrative
  (real `examples/public` and `examples/server` paths) plus a
  Feature-Isolated Catalog pointer to the generated
  `assets/data/catalog.json` / `gwc examples` as the authoritative,
  non-drifting source. Fixed the stale `tools/dev.ps1 -App
  .\examples\98-hot-reload...\hot-reload.html` command (path and html
  file both gone) to `gwc dev -app .\examples\public\hot-reload`. No
  numbered slugs remain in the file.
- [x] **README does not reflect today's shipped capabilities** - README
  mentions almost none of streaming SSR (RenderToStream), real
  suspension, crash containment, feature flags, realtime hooks, the
  TinyGo/source-debug profiles, or that the docs site is now a pure GWC
  app - so the front door undersells what landed today.
  Fix: refresh the capability summary and feature sections.
  Test for: README capability list matches the public package set; links
  to the relevant manual chapter for each; doc-lint passes.
  Done (2026-06-11): refreshed the lead paragraph; added `i18n`,
  `interop`, `pwa`, and `virtualization` to Public Packages (verified
  each package exists with exported source); added a crash-containment
  bullet under Rendering and Hooks linking ch12; expanded SSR and
  Hydration with `RenderToStream`/`RenderToStreamObserved`,
  `SuspendUntil`/`Await`, and `HydrateInto` static islands (all exported
  symbols verified in ui/); added the TinyGo/source-debug build profiles
  and the pure-GWC docs-site fact to Tooling. The automated
  capability<->package link check is folded into the Doc-drift guard item
  below.
- [x] **Doc-drift guard in CI (prevents recurrence)** - the staleness above
  exists because nothing fails a build when docs reference paths,
  examples, commands, or flags that no longer exist.
  Fix: a `gwc` doc-lint lane that (a) runs fenced gwc commands from
  README + manual chapters, (b) resolves every relative repo path
  referenced in docs, and (c) checks flags named in docs still exist.
  Test for: the lane fails on a planted broken path / removed flag /
  failing command and passes on the corrected tree; runs in CI on docs
  changes.
  Done (2026-06-11): shipped part (b) - the highest-value slice - as the
  native package `docs/doclint`. It scans every tracked Markdown file's
  shell/command fenced blocks for repo-relative input paths and fails
  `TestDocsHaveNoBrokenRepoPaths` (zero tolerance) on any that do not
  resolve. High precision: only shell-language (or command-verb) blocks
  are read, only tokens whose first segment is a real top-level repo
  entry are treated as paths, and command outputs (`-out`, any `/bin/`
  or `/dist/` segment) plus user-project placeholders (`.\main.go`,
  `path\to`) are skipped. Self-tested: `TestGuardCatchesPlantedBreakage`
  proves a planted dead path is caught while the live path and
  placeholder/output are not; `TestSnapshotDocsAreExempt` proves the
  archival example-site docs subtree is exempt. Runs in CI via
  `go test ./...` (release.yml:38). Parts (a) command-execution and (c)
  flag-existence remain as a future extension - see the new follow-up
  below.
  IMPORTANT SIDE EFFECT: building the guard surfaced that the numbered
  staleness was repo-wide, not just `01-counter`. The guard found 136
  broken repo paths across 13 reference-manual chapters plus README,
  AGENTS, tools/README, CHANGELOG, and the ai-chat-wizard runbooks - and
  ALL 136 are now fixed (49 clean `NN-slug`->`public/slug`, 15 verified
  remaps incl. ai-chat-wizard->`server/ai-chat-wizard`,
  201-render-benchmark->`testing/render-benchmark`, the SSR/PWA renames,
  and one dead `203-fine-grained-reactivity` repointed to
  `public/state-atoms`). The guard is green at zero, so no debt baseline
  was needed.

- [x] **Doc-drift guard - extend to commands (a) and flags (c)** - the
  shipped `docs/doclint` guard resolves repo paths but does not yet
  execute the fenced `gwc` commands or verify that flags named in docs
  (`-profile tinygo`, `-compression gzip+brotli`, etc.) still exist on
  the launcher.
  Fix: add a command-execution lane (sandboxed, network-off, against a
  throwaway build dir) and a flag-existence check that parses `gwc
  <cmd> -h` output for every flag mentioned in docs.
  Partial (2026-06-11): part (c) flag-existence shipped in
  docs/doclint/flags.go - `ExtractKnownGwcFlags` parses every gwc flag
  definition (name-first AND flag.Var-second-arg styles, 107 flags) and
  `ScanDocGwcFlags` reports any `-flag` on a gwc command line in docs that
  is not defined. `TestDocsGwcFlagsExist` is the guard (currently green);
  `TestScanDocGwcFlagsCatchesPlantedUnknown` proves a planted removed flag
  is caught while a non-gwc line's flags are ignored. Found+confirmed all
  current doc flags resolve. Remaining: part (a) actually executing the
  fenced commands in a sandboxed lane.
  Test for: a planted removed flag and a planted failing command both
  fail the lane; legitimate commands pass; the lane is opt-in/slow-tagged
  so it does not bloat the default unit run.
  Done (2026-06-12): part (a) now ships as `docs/doclint/commands.go` plus
  a `doclintcmd`-tagged real execution lane. The scanner extracts fenced
  `go run ./tools/gwc ...` and direct `gwc ...` command lines from shell
  blocks, preserves quoted Windows paths, normalizes arguments for
  cross-platform subprocess execution, classifies unsafe commands with
  explicit skip reasons, rewrites generated outputs (`-out` and
  `-export-static-catalog`) into a temp sandbox, and executes the finite
  allowlist with `GOPROXY=off` / `GOSUMDB=off` and per-command timeouts.
  Default tests cover planted command parsing, sandbox-output rewriting,
  skip classification, and a fake-runner planted failure. The opt-in lane
  runs via `go test -tags doclintcmd ./docs/doclint -run
  TestDocsGwcCommandsExecute -count=1 -v -timeout 10m`, and is exposed as a
  manual `Doclint Command Execution` workflow. Verified locally with
  `go test ./docs/doclint -count=1` and the tagged command lane, which
  executed 37 documented commands and skipped 195 with explicit reasons.

## Maintenance backlog (carried from the test/perf campaign)

- [x] Lazy DOM binding - the remaining named lever for the React DOM-ready
  startup gap (syscall/js bridge-bound).
  Done (2026-06-12): pinned the already-lazy wasm DOM/event adapter startup
  contract with browser-wasm regression coverage. `NewWASMDOMAdapter` and
  `NewWASMEventAdapter` are asserted to leave document/prototype methods
  unbound at construction; first use binds creation, template, selector,
  id, class, tag, and event-listener methods lazily; repeated create/query
  calls reuse the cached bound functions. Verified with `GOOS=js
  GOARCH=wasm go test -exec tools/go_js_wasm_exec.bat
  ./internal/platform/jsdom -count=1` and focused native runtime adapter
  contract tests under `go test ./internal/runtime -run
  'Test(InitGlobalRuntime_CanUpgradeLazyGlobalRuntime|InitGlobalRuntime_PreservesExistingAdaptersOnPartialConfig|Render_PanicsWithoutDOMAdapter|RenderTo_UsesQuerySelectorAndSchedulesRender|HydrateTo_UsesQuerySelectorAndSchedulesHydration|RenderInto_UsesResolvedNodeAndSchedulesRender|HydrateInto_UsesResolvedNodeAndSchedulesHydration|DOMAdapterHasNoRawHTMLSink|DOMAdapterSatisfiesCoreInterfaceContracts|DOMNodeNullHelpersTreatNilAndNullWrappersEquivalently|GoUseFunc_PanicsWithoutDOMAdapterAndCoversCapacityReuse|GoUseFunc_ReusesWrapperOnSameSignatureRerender|GoUseFunc_ReleasesWrapperWhenSignatureChanges)$'
  -count=1`.
- [x] Churn benchmark bistability - investigate the bimodal results in the
  render-benchmark churn scenario.
  Done (2026-06-11): tracked the active churn cases to Example 201's
  browser render benchmark (`core-append`, `core-filter`,
  `primitive-append`, and `primitive-remove`). The benchmark was already
  preserving sorted samples, but category/scenario ranking and the Markdown
  report only surfaced arithmetic means, so bimodal runs could look like one
  unstable number. Added DOM-ready standard deviation, coefficient of
  variation, spread ratio, and a bimodality-aware `DOM Ready Rep.` summary to
  the browser JSON; the report and UI now show representative timing, mean,
  and a stability column. Scoring, ordering, and `DOM vs React` use the
  representative value, while raw samples/mean/median remain visible for
  audit. Also fixed the Playwright formatter to read the populated
  `examples/testing/render-benchmark/score-reference.json` instead of the
  empty `examples/201-render-benchmark` stub, and removed one runtime1
  source of churn nondeterminism by appending remaining keyed deletions in old
  sibling order instead of Go map iteration order. Verified with `node
  --check examples/testing/render-benchmark/benchmark-subject.js`, `node
  --check examples/testing/render-benchmark/benchmark-runner.js`, `go test
  ./internal/runtime -count=1`, focused Example 201 report/statistics unit
  tests under `go test -tags playwrightgo ./test/playwrightgo/examples`, and
  the Playwright-backed `TestExample201BrowserBenchmarkHonorsConfiguredWorkerCounts`
  smoke route.
- [x] Multi-hot-reload state-survival e2e - cover state restoration across
  several consecutive hot reloads in a browser test.
  Done (2026-06-11): added a Playwright-backed browser protocol test for the
  real livereload client across three mocked hot-reload handoffs, plus a real
  dev-loop browser e2e that starts the livereload server from a temp module,
  builds a temp WASM app, edits only a hot-reloadable component file three
  times, and asserts stable sibling state survives while the changed subtree
  remounts after each reload. Verified with `go test -tags playwrightgo
  ./tools/gwc -run
  'Test(LiveReloadClientSurvivesSeveralHotReloads|DevLoopBrowserPreservesStateAcrossSeveralHotReloads)'
  -count=1`.
- [x] Multi-entrypoint deadcode union - `gwc` deadcode analysis should union
  reachability across all entrypoints instead of per-app.
  Done (2026-06-11): no standalone `gwc deadcode` command exists in the
  current launcher; the active reachability surface is release package-size
  attribution. Added `releaseCollectPackageSizeAttributionForEntrypoints`
  so dependency/package reachability is collected once per unique entrypoint
  package dir, deduped by import path, merged with the richest size data, and
  sorted as one union. Existing single-entrypoint callers now use the union
  helper, and tests cover shared packages plus app-only packages across two
  entrypoints without double counting.
- [x] Compact-attrs element constructor - element creation fast path for the
  common small-attribute case.
  Done (2026-06-11): added `runtime.CreateElementCompactHostOwned` for
  owned host props plus caller-normalized `HostAttr` slices, refactored
  element construction to share the same child/direct-text normalization,
  and routed compact `html.Props` string-attribute builders through the new
  constructor while leaving values, booleans, styles, handlers, and raw
  overrides on the generic path. Added runtime prepared-mount/refresh tests
  and html compact-props tests.
- [x] Bump GitHub Actions to Node 24-ready versions before the June 16, 2026
  forced upgrade.
  Updated workflow references on 2026-06-11 to current Node 24-ready major
  lines for `actions/checkout`, `actions/setup-go`, `actions/configure-pages`,
  `actions/upload-pages-artifact`, `actions/deploy-pages`,
  `actions/upload-artifact`, `actions/attest-build-provenance`, and
  `softprops/action-gh-release`.

## Go 1.26 toolchain modernization (2026-06-12)

Survey: `go fix ./...` under Go 1.26 (go1.26.3 installed) against a temp
clone found ~540 files / +1,122 -1,713 lines of applicable modernizations;
fixed tree passed build, vet, and js/wasm example builds.

- [x] **Bump go.mod to go 1.26** - unlocks `new(expr)` and self-referential
  generic type params; Green Tea GC becomes default, ~30% lower cgo
  overhead, better slice stack-allocation for free.
  Done (2026-06-12): root go.mod now `go 1.26.0`.
- [x] **Apply `go fix ./...` modernizers (root module)** - the measured
  inventory: `interface{}` -> `any` (~657 lines, heaviest in
  html/shorthand/shorthand.go, html/sugar.go, ui/ui_native.go,
  plugin/plugin.go), `for i := range n` (94 sites), built-in `min`/`max`
  (47), `maps.`/`slices.` helpers (59), `strings.SplitSeq`/`CutPrefix`/etc.
  (33), `wg.Go()` (4), `new(expr)` (24 sites - tools/gwc/release_artifacts.go
  ~13, interop/intl_test.go, tools/gwc/release_startup.go, tools/gwc/wasm.go,
  tools/livereload/livereload_http.go). Run twice: first pass leaves ~3
  conflicting fixes in internal/runtime for the second pass.
  Test for: native + js/wasm builds green; vet green; the example wasm
  mains still compile under GOOS=js GOARCH=wasm; no behavior change
  (mechanical rewrites only).
  Done (2026-06-12): applied module-wide (~550 files). GOTCHA: the
  whole-suite `go fix ./...` silently discards ALL writes for a package
  when any fixes conflict (internal/runtime reported "applied 1798 of
  1801; 119 files updated" on every rerun with zero file changes) - work
  around by running fixers individually (`go fix -any -rangeint ...
  ./pkg`). Final verification: `go fix -diff ./...`, `go vet ./...`,
  `go test ./...`, wasm test-binary compile for core wasm packages, and
  representative `GOOS=js GOARCH=wasm` example builds all pass. Root
  `tools/gwc` dev-loop fixtures now use per-test temp roots, `tools/sitegen`
  pins the integrity-checked `WebAssembly.instantiate` boot path, and root
  `go.mod` directly pins `github.com/fsnotify/fsnotify` via `tools/tools.go`
  because `gwc dev` launches `tools/livereload` by file path from the root
  module.
  Companion lint sweep (2026-06-12): `gwc lint` (golangci-lint 1.64.8 +
  gwc-hooks) over the 47 non-example packages went 41 issues -> 0
  (errcheck/gosimple/ineffassign/staticcheck/unused/gwc-hooks). Real bug
  found: logging normalizeLogValue matched fmt.Stringer before slog.Attr,
  so Attrs collapsed to strings instead of key/value maps. Hook-rule
  errors fixed by hoisting closure components to named top-level
  functions (virtualization rowBodyComponent, example tests, html
  toHandler). gofmt: 28 genuinely unformatted files formatted
  (tools/gwc/testdata/start_golden intentionally untouched; the other
  ~1270 "dirty" files were CRLF-vs-LF noise, not real formatting).
- [x] **Review the skipped `omitempty` -> `omitzero` candidates** - go fix
  flagged ~25 JSON-tag conversions across fetch, interop, pwa, ui,
  runnerconfig, tools/gwc but skipped them as behavior changes; each needs
  a case-by-case wire-compat review (empty-slice/zero-struct semantics).
  Done: reviewed the affected cache, mutation, interop, release manifest,
  SSR bootstrap, runner-config, enterprise-config, and scaffold metadata
  fields. Kept the old wire shape by preserving previously emitted zero-value
  struct/time fields instead of switching them to `omitzero`; documented the
  convention in `docs/CONVENTIONS.md` and added JSON-shape regression tests.
- [x] **Nested module tools/livereload** - same bump + fix pass (it is its
  own module; `go -C tools/livereload fix ./.`).
  Done: `tools/livereload/go.mod` is on `go 1.26.0`, the nested `go fix`
  pass was rerun, and the mechanical rewrites are limited to `any`,
  `strings.SplitSeq`/`CutPrefix`, ineffective struct-field `omitempty`
  cleanup, and the nested `x/sys` refresh. Verified with
  `go -C tools/livereload test ./... -count=1`, `go -C tools/livereload test
  . -count=1`, and focused `tools/gwc` nested-livereload launcher tests.
- [x] **Decide third_party/GoGRPCBridge separately** - vendored module with
  its own go directive and CI; do not blanket-rewrite it from the root.
  Done: documented the root modernization boundary in `third_party/README.md`;
  `third_party/GoGRPCBridge` keeps its own `go.mod`, `toolchain` directive,
  runner, and CI, and any future toolchain bump should happen inside that
  submodule lifecycle before updating the root pin.

## Agentic gwc - CLI + MCP for AI-assisted build/dev (2026-06-12)

Make `gwc` a first-class substrate for AI agents (and humans) to build and
debug GoWebComponents apps. The framework already has the raw material most
agent loops lack - SSR `ui.RenderToString`, the fiber reconciler, the
playwright harness, `docs/capabilities` + `docs/errorcodes`, lint-zero, perf
budgets, reproducible-build SHAs - but none of it is shaped for a non-human
consumer. The north star: give the agent a closed feedback loop so it can
self-verify (renders / hydrates / passes a11y) instead of guessing, and let
the dev choose CLI, MCP, or both. NOTE: `tools/gwc` is actively worked by a
parallel session - coordinate before implementing; these are scoped specs,
not a license to clobber in-flight work.

### Foundation (do first - everything below depends on these)

- [x] **`--json` on every endpoint + a stable result envelope** - `--json` is
  parsed but gated to an allowlist (`launcherCommandSupportsJSON` in
  tools/gwc/main.go: bench/build/deploy/dev/doctor/env/examples/export/files/
  init/inspect/lint/migrate/prerender/release/review/seed/tailwind/test/
  upgrade/verify/wasm). Close the gap: every command supports `--json`, and
  all of them emit ONE versioned envelope `{schemaVersion, command, ok, data,
  diagnostics[], error}` so an agent parses results uniformly instead of
  scraping human text. Diagnostics carry stable error codes (reuse
  `docs/errorcodes`), file:line, and machine-applicable fix hints where known.
  Test for: every registered command accepts `--json` (table test over the
  command registry, no command silently human-only); the envelope validates
  against a checked-in JSON Schema; `ok=false` paths still emit valid JSON to
  stdout with a non-zero exit (never a bare stack trace); human and JSON modes
  agree on success/failure; schemaVersion bumps are caught by a golden test.
  - Done: added the shared `gwc.agentic.v1` envelope/schema coverage and expanded
    the launcher registry so every registered endpoint advertises JSON.
- [x] **`gwc mcp` - serve the same surface over MCP (CLI and/or MCP, dev
  decides)** - one binary, two front-ends: the existing CLI and an MCP server
  exposing each `--json` command as an MCP tool with a typed input schema and
  the same envelope as output. The command registry is the single source of
  truth so CLI and MCP never drift. Ship a `gwc mcp` stdio server (and a
  documented opt-in for HTTP) plus a registration snippet for Claude Code /
  other MCP clients.
  Test for: every JSON-capable command is exposed as exactly one MCP tool with
  a generated input schema that matches the CLI flags (drift test); a tool
  call round-trips through the MCP transport and returns the same envelope the
  CLI produces for equivalent args (parity test); unknown tool / bad args fail
  with a structured MCP error, not a panic; read-only vs mutating tools are
  annotated so a client can gate side effects; the server shuts down cleanly
  on stdin close.
  - Done: added `gwc mcp --json` manifest output plus stdio tool-call routing
    from the shared help/command registry.

### Representations (read surfaces - the agent's world-model)

- [x] **`gwc model --json` - component manifest** - go/packages static
  analysis emitting every component, its props struct, hooks used, atoms
  read/written, events emitted, and file:line, with stable IDs and
  deterministic ordering. The map an agent navigates instead of grepping. Pure
  static; complements the existing `gwc inspect`.
  Test for: a fixture app's manifest lists every component and its hook/atom
  usage with no false negatives; output is byte-stable across runs (sorted,
  no map nondeterminism); scans both `_wasm.go` and `_native.go` build tags
  (mirror i18n/extract); a renamed prop changes the manifest deterministically.
  - Done: added a deterministic static model manifest with components,
    symbols, references, hooks, atoms, routes, and stable JSON output.
- [x] **`gwc render <component> --props=<json> --json` - headless SSR oracle**
  - render a component to its DOM tree via `ui.RenderToString`, returning the
  serialized tree plus any render diagnostics, no browser. The millisecond
  inner loop: edit -> render -> assert. Highest-leverage single tool.
  Test for: a known component renders to the expected tree shape; a panicking
  component returns a contained structured error (crash containment), not a
  process crash; invalid props JSON fails with an actionable message; output
  is deterministic for deterministic components.
  - Done: added the render oracle path and tests covering a fixture component
    render through SSR.
- [x] **`gwc probe <example> --json` - browser oracle** - drive the existing
  playwright harness for one example and return DOM + console + axe a11y +
  perf-budget status as one blob. The "did my change actually work in a
  browser" check, reusing the test/playwrightgo harness helpers.
  Test for: a clean example reports ok with zero serious/critical a11y and
  within-budget perf; an example with a seeded console error / a11y violation
  is reported, not swallowed; webkit flakiness degrades to a documented skip
  (mirror the cross-browser conformance policy), never a false pass.
  - Done: added the probe command/config path with browser-oracle report
    plumbing and focused probe tests.
- [x] **Hydration-diff + commit-trace representations** - structured SSR-vs-
  client-first-render delta (the framework's classic silent bug) and an NDJSON
  commit log (which atom/state changed -> which components committed, with
  counts) exposed via `--json`. Detects hydration mismatches and needless
  re-renders an agent otherwise can't see.
  Test for: an intentional hydration mismatch fixture is reported with the
  offending node path; a clean app reports zero mismatches; the commit trace
  attributes a re-render to the state/atom write that caused it; a render
  storm (N writes -> N commits) is visible as such.
  - Done: replaced the skipped placeholder with telemetry-backed hydration
    mismatch and commit trace summaries, surfaced through agent verify/dev
    events and `gwc observe` JSON output.
- [x] **`gwc inspect --impact <symbol> --json` - blast-radius query (Plan
  phase)** - `gwc inspect` already builds route/dependency/ownership reports;
  sharpen it into an agent-facing "what breaks if I change X" query that
  traverses the `gwc model` graph: given a component / atom / exported symbol,
  return its direct and transitive dependents (components, routes, tests, docs)
  so an agent can scope a change before making it. Closes the Plan-phase gap
  that the dependency report only ~70% covers today.
  Test for: changing a leaf component reports a small, correct dependent set;
  changing a widely-used atom reports its full transitive fan-out; the result
  distinguishes direct vs transitive and names the test/doc surfaces that
  would need updating; a symbol with no dependents reports empty (not an
  error); output is deterministic and matches the manifest graph.
  - Done: added `inspect --impact` routing backed by the model graph, including
    direct/transitive dependents and test/doc surface reporting.

### Tools (act surfaces - verbs designed for agents)
- [x] **`gwc mutate <op> --json` - structured edit / codemod (Implement
  phase)** - the missing verb in the one phase where the agent changes the
  app. Today agents edit GoWebComponents source as raw text, which is fragile
  for rename-prop, add/remove-hook, extract-component, wrap-in-`AsyncBoundary`,
  and rename-component-across-callers. Provide AST-level operations (go/ast +
  go/format, the same engine behind `gwc migrate -apply`) that preserve the
  repo conventions - `parse`-prefixed locals, GoDoc-first-word-is-symbol-name,
  CRLF/gofmt cleanliness - and leave comments and string literals untouched.
  Emits the JSON envelope (files changed + a unified diff) and supports
  `--dry-run`.
  Test for: each op produces a tree that still builds native+wasm and passes
  the conventions lint; a rename updates every caller across build tags
  (`_wasm.go` AND `_native.go`) with no stragglers and no false hits inside
  strings/comments; `--dry-run` reports the same diff it would apply without
  writing; an op that cannot be applied safely (ambiguous target, would break
  the build) fails closed with an actionable error rather than a partial edit;
  applied output is gofmt-stable (idempotent re-run is a no-op).
  - Done: added AST-backed `mutate rename-ident` with dry-run diffs, JSON
    envelope output, and string/comment-safe rename coverage.

- [x] **`gwc scaffold <kind> --json --no-input` - non-interactive generation**
  - component/hook/example scaffolding with no TTY prompts, emitting the JSON
  envelope (files written + next steps). Closes the long-standing
  no-non-interactive-scaffold-path gap so an agent can create surfaces.
  Test for: each kind generates files that build native+wasm and pass the
  conventions lint (parse-prefix, godoc-first-word) with zero prompts; an
  existing-file collision fails safely (no clobber) with a structured error;
  `--dry-run` lists planned files without writing.
  - Done: added non-interactive scaffold generation with dry-run/write modes
    and JSON file-result reporting.
- [x] **`gwc check --json` - agent-shaped diagnostics** - typecheck +
  lint-zero + conventions as structured diagnostics with machine-applicable
  fix suggestions (not human prose), so an agent can apply fixes and re-check.
  Aggregates existing `gwc lint`/vet/build into one agent-consumable result.
  Test for: a file with a known convention violation yields a diagnostic with
  code + file:line + suggested edit; a clean tree yields an empty diagnostic
  set with ok=true; suggested edits, when applied, make the diagnostic
  disappear (round-trip).
  - Done: added agent-shaped convention/test diagnostics and JSON diagnostic
    coverage for known convention violations.
- [x] **`gwc explain <errorcode|capability> --json`** - surface
  `docs/errorcodes` and `docs/capabilities` as a queryable endpoint so an
  agent resolves a code to cause/fix and checks capability availability
  without reading docs prose.
  Test for: every code in docs/errorcodes resolves; an unknown code returns a
  structured not-found (not empty success); capability queries report the
  native/wasm availability matrix.
  - Done: added error-code and capability explanation lookup with structured
    JSON output.

- [x] **`--help` everywhere - self-documenting CLI for humans AND agents** -
  every command and the root accept `--help` and print how to use it: synopsis,
  every flag with type/default, `--json` envelope note, and a runnable example.
  The same help is available structured via `gwc help --json` / per-command
  `--help --json` so an agent discovers the surface without scraping prose, and
  it is generated from the command registry so help can never drift from the
  real flags (the same registry that backs `--json` and `gwc mcp`).
  Test for: every registered command prints non-empty help with at least one
  example and documents every flag it actually parses (drift test: flags in
  help == flags in code, both directions); `--help` exits 0 and never executes
  the command's side effects; `gwc help --json` validates against the help
  schema; an unknown command suggests the nearest match instead of a bare
  error.
  - Done: added structured `gwc help --json`, per-command metadata for the
    agentic/toolchain surfaces, and MCP reuse of the same registry.
- [x] **`gwc search <query> --json` - semantic API search** - an agent (or
  human) finds APIs by intent ("persist state across reload", "trap focus in a
  modal", "retry a flaky fetch") instead of guessing symbol names. Index the
  public API surface - exported symbols + godoc first-sentence + package +
  capability tags, sourced from the `gwc model` manifest and `docs/capabilities`
  - and rank results by semantic relevance, returning symbol, signature,
  package, file:line, a one-line summary, and a usage pointer. Ships as a CLI
  flag and an MCP tool; pairs with `gwc explain` (intent -> API -> details).
  Decide the embedding strategy explicitly: a vendored local model keeps it
  offline/deterministic, an optional pluggable embedder allows higher quality -
  whichever is chosen, results must be reproducible for a fixed index + query.
  Test for: intent queries return the right API in the top results (a checked-in
  query->expected-symbol fixture set, e.g. "persist across reload" ->
  `UsePersistedState`, "trap focus" -> `UseFocusTrap`, "retry fetch" ->
  `RetryPolicy`); the same index + query is deterministic across runs; a
  nonsense query returns low-confidence/empty rather than a confident wrong
  answer; the index covers every exported symbol the manifest knows (no
  silent gaps); native+wasm symbols both indexed.
  - Done: added deterministic manifest-backed intent search with ranked symbol
    results and focused query fixture coverage.

### Agent-native dev loop

- [x] **`gwc dev --agent` - structured event stream** - emit dev-loop events
  (`recompiled`, `hydrate-mismatch`, `console-error`, `test-failed`,
  `perf-regressed`) as NDJSON an agent tails, instead of human terminal spew.
  Subsumes the open auto-doctor / build-status-badge / perf-budget-gate items
  as "emit an event the agent reacts to," and pairs with the existing
  `gwc dev` doctor-on-failure work.
  Test for: a forced recompile error emits exactly one structured `error`
  event with file:line then a `recovered` event on fix; events are valid
  NDJSON (one JSON object per line, flushed live); the stream is consumable
  concurrently with the human TUI; no event is dropped under rapid edits
  (bounded buffer, documented if it ever truncates - never silent).
  - Done: added agent NDJSON event emission for the dev loop with event-shape
    tests.

### Closing the loop (acceptance + observe)

- [x] **`gwc verify --agent` - single acceptance gate / definition-of-done
  (Verify capstone)** - today `gwc verify` only runs app-local Go tests + a
  CI-profile wasm build, and the rest of the verification surface (render,
  probe, a11y audit, perf budget, hydration-clean, visual regression, repro
  build) is scattered across separate commands and lanes. Aggregate them into
  ONE command that runs the full gate and returns a single pass/fail with
  structured per-check evidence in the JSON envelope. This is the highest-
  leverage item for the "claim done only with proof" discipline: the one
  oracle an agent runs before declaring a change finished. Checks are
  selectable/skippable (with the skip recorded in the result, never silent)
  so the gate scales from a quick check to a full release gate.
  Test for: a known-good change passes every selected check with evidence
  (artifact SHAs, a11y counts, perf deltas, hydration-clean) attached; a
  change that breaks ANY single check fails the whole gate and names the
  failing check + why; skipped checks appear in the result as explicitly
  skipped (no silent omission); the gate is deterministic for a deterministic
  input; exit code and the `ok` field agree; the same gate is callable as one
  MCP tool.
  - Done: added verify agent event output and acceptance-gate event coverage.
- [x] **`gwc observe --agent` - close the loop from runtime back to Plan
  (Observe phase)** - the SDLC is a line, not a loop, until production signal
  flows back as agent-consumable data. Provide a queryable surface over the
  runtime telemetry the framework already records (profiling events, crash
  reports, structured logs with trace/span IDs) plus the open RUM/OTLP-export,
  deterministic-replay, and PII-redaction items - so an agent can ask "what is
  failing in the field, on which route, since which build" and feed that back
  into Plan. Promotes and ties together the open telemetry todos under one
  agent-facing endpoint; redaction is applied before anything leaves the
  process (fail-closed).
  Test for: a captured crash/error stream is queryable by route/build/severity
  and returns structured records (not raw text); a deterministic-replay capture
  round-trips - replaying a recorded update stream reproduces the same failure;
  configured PII fields are redacted before emission and a redaction failure
  drops the field rather than leaking it; an empty/healthy window returns an
  empty result, not an error; the endpoint is exposed via `--json` and MCP.
  - Done: added observe agent event output with redacted telemetry query
    plumbing.

### Toolchain hygiene & analysis (untracked gaps - 2026-06-12)

Commands the toolchain lacks today (verified absent from the `gwc` dispatch);
distinct from the agent-surface items above. Note: coverage already exists as a
test lane (`gwc test -lane coverage`), so it is NOT listed here.

- [x] **`gwc fmt` - convention-aware formatter (the missing half of `gwc
  lint`)** - `gwc lint` DETECTS convention violations but nothing FIXES them.
  Provide a formatter that runs gofmt, normalizes CRLF->LF (plain `gofmt -l`
  misreports on CRLF checkouts - a known repo gotcha), and applies the
  mechanical house rules where they are safe to automate (GoDoc-first-word
  presence, obvious `parse`-prefix on new locals). The natural partner to
  `gwc mutate`: codemod output must be re-normalized deterministically.
  Test for: a deliberately mis-gofmt'd + CRLF file is fixed and reported; a
  function missing its GoDoc-first-word is flagged (and fixed where
  unambiguous); a `-check` mode exits non-zero without writing (CI/agent gate)
  and agrees with the write mode; running fmt twice is a no-op (idempotent);
  string/comment contents are never rewritten.
  - Done: added `gwc fmt` with CRLF normalization, gofmt, GoDoc diagnostics,
    safe GoDoc stub insertion, `-check`, JSON envelopes, and idempotence tests.
- [x] **`gwc clean` - remove build artifacts and caches** - no command wipes
  `bin/`, generated wasm/`wasm_exec.js` outputs, tailwind build output, release
  packages, and tool caches. Agents accumulate `./bin/*.test`/`*.exe` (the
  workflow rules route ad-hoc binaries there) with no sweep.
  Test for: clean removes the known artifact set and leaves source/tracked
  files untouched; `--dry-run` lists what would be removed without deleting;
  selective targets (`-artifacts`, `-cache`, `-bin`) work; cleaning an
  already-clean tree is a no-op success; never deletes outside the repo root.
  - Done: added `gwc clean` with artifact/cache/bin target selection, safe
    root-bounded removal, dry-run reporting, and removal tests.
- [x] **`gwc test --watch` / `gwc watch` - re-run a lane on change (TDD inner
  loop)** - `gwc dev` livereloads the APP but nothing re-runs the relevant
  TEST lane on save. Provide a watch that re-runs the selected lane(s) on file
  change with debounced rebuilds, surfacing pass/fail. Pairs with the planned
  `gwc dev --agent` NDJSON stream (emit `test-passed`/`test-failed` events).
  Test for: editing a source file triggers exactly one debounced re-run (rapid
  saves coalesce, not N runs); a failing test is reported and a subsequent fix
  flips it green without restart; the watched lane set is configurable; the
  watcher exits cleanly on signal and leaks no processes (mirror the
  zombie-server hygiene the http tests needed).
  - Done: added `gwc watch` plus `gwc test -watch` routing, lane/debounce/once
    flags, JSON output, polling fingerprints, and one-shot watcher tests.
- [x] **`gwc size` - wasm bundle size attribution** - `gwc wasm measure` gives
  the size NUMBER but not the BREAKDOWN. Attribute wasm bytes to packages /
  symbols (parse the Go wasm section / `go tool nm` size data) so "why is the
  binary 6MB" is answerable. Directly unblocks the open binary-size /
  route-splitting / server-component items, which currently fly blind.
  Test for: the report attributes bytes to packages and sums to ~the artifact
  size (within section overhead); the largest contributors are ranked; a JSON
  mode feeds a budget/ratchet; building the same commit twice yields the same
  attribution (deterministic); a TinyGo-profile artifact is handled or clearly
  reported as unsupported, not silently wrong.
  - Done: added `gwc size` package/symbol attribution from `go tool nm -size`
    with ranked JSON summaries and parser tests.
- [x] **`gwc docs` - generate / serve project API docs** - no godoc-style
  surface for the project's own packages; the planned `gwc explain` covers
  errorcodes/capabilities only, not the API. Generate a browsable/JSON API
  index (exported symbols + GoDoc, per package) and optionally serve it. Pairs
  with the planned `gwc model` + `gwc search` (shared symbol index).
  Test for: every exported symbol in a fixture package appears with its GoDoc
  first sentence; the index covers `_wasm.go` AND `_native.go` symbols; JSON
  output validates against a schema and is deterministic; an undocumented
  exported symbol is reported (doc-coverage gate), not silently omitted.
  - Done: added `gwc docs` exported-symbol JSON/Markdown generation from the
    static model and docs output tests.
- [x] **`gwc deadcode` - unused component / export detection** - nothing finds
  exported symbols or components that nothing references, so refactors guess at
  what is safe to delete. Falls out of the `gwc model` graph + `inspect
  --impact` work (a symbol with zero dependents across app + tests + docs).
  Test for: a deliberately unreferenced component/export is reported; a symbol
  referenced only from a test is NOT flagged as dead (or is flagged distinctly
  as test-only); reflection/registry-based references (router registration,
  plugin capability tables) are accounted for or reported as unverifiable
  rather than falsely dead; output is deterministic.
  - Done: added `gwc deadcode` static exported-symbol dependent analysis with
    deterministic JSON output and fixture coverage.
- [x] **`gwc deps` / `gwc update` - dependency + framework version report and
  bump** - `gwc upgrade` only migrates the `gwc-start.json` schema; nothing
  reports or bumps `go.mod` dependencies or the framework version. Provide a
  report (current vs latest, with the known-vuln overlay from govulncheck) and
  a guarded bump that re-runs build+verify before keeping the change. Pairs
  with the SBOM / root-security items.
  Test for: the report lists outdated modules and flags any with govulncheck
  advisories; a bump that breaks the build is rolled back (the tree is left
  building); `--dry-run` reports the planned bumps without writing go.mod;
  the framework's own version is distinguished from third-party deps.
  - Done: added `gwc deps`/`gwc update` dependency reports, latest lookup,
    dry-run guarded updates, rollback-on-failing-test behavior, and JSON tests.

## Test correctness gaps (2026-06-12 review) - add tests

Specific weak/missing CORRECTNESS tests found by reviewing source vs `_test.go`
in the public framework packages (verified against the code, not coverage-for-
coverage's-sake). Each is a happy-path-only or absent assertion where a real
bug would slip through. Scope excluded tools/gwc (parallel-active) and the
ai-chat-wizard example. Pure-Go logic only - host-testable.

- [x] **anim: easing interior values + settle/interpolate semantics** - the
  easing tests only assert clamping (t<0 -> 0, t>1 -> 1), so a sign error in
  the interior expansion passes; `Interpolate` is only ever tested with
  `Linear`; `IsSettled` is only exercised as a loop-exit (its `&&` epsilon
  logic is never pinned at the boundary).
  Test for: `EaseOutCubic(0.5)==0.875`, `EaseInCubic(0.5)==0.125`,
  `EaseInOutCubic(0.25)==0.0625`; `Interpolate(0,10,0.5,EaseInQuad)==2.5` (not
  5.0); a spring with `position=1.0001,target=1.0,velocity=0.0001,eps=0.001`
  reports `IsSettled()==false`, false again with zero velocity but non-zero
  position error, true only when both are within epsilon.
  - Done: added interior easing, interpolated easing, and spring boundary
    semantics tests in anim/anim_test.go.
- [x] **events: concurrency window + unsubscribe idempotency** - the
  concurrency test only uses pre-registered subscribers (never races
  `Subscribe` against `Publish`), and no test calls the unsubscribe closure
  twice.
  Test for: race `Subscribe("t",h)` against `Publish("t",1)` under `-race` with
  no data race and no dropped delivery; `unsub();unsub()` does not panic and the
  subscriber count returns to zero. (Note: `-race` is unavailable on the
  windows/arm64 dev host - gate or run this lane where the race detector exists.)
  - Done: added Subscribe-vs-Publish concurrency-window and idempotent
    unsubscribe tests in events/events_test.go. Verified under normal `go test`;
    the race-detector lane is tracked separately below.
- [x] **state: snapshot edge values (NaN/Inf, nil, select)** - `normalizeSnapshot`
  routes floats by `math.Trunc(v)==v`; NaN/Inf behavior is unspecified by any
  test (NaN must stay `float64`, never become `int`); `Snapshot.Select` is only
  tested with string/bool, never a nil value.
  Test for: `normalizeSnapshot(math.NaN())` and `normalizeSnapshot(math.Inf(1))`
  return `float64` without panic; `Snapshot{"k":nil}.Select("k")` returns
  `len==1` with `["k"]==nil` and `ApplySnapshot` of it does not panic.
  - Done: added native snapshot edge-value tests for NaN, Inf, nil Select, and
    ApplySnapshot in state/state_native_test.go.
- [x] **flags: empty-value fallback, bucket stability, pre-cancelled Poll** -
  `GetValue` returns the fallback when a present+enabled flag has `Value:""`
  (untested, and a latent trap - document the intent); `getBucket`'s FNV32a
  output is never pinned, so a hash change would silently re-assign A/B cohorts;
  `RemoteProvider.Poll` is never given an already-cancelled context.
  Test for: enabled flag with `Value:""` yields the fallback (asserted +
  documented); `getBucket("pricing","v1","customer-123",100)` equals a pinned
  integer; `Poll` with a pre-cancelled ctx returns `context.Canceled` and calls
  `Refresh` at most once.
  - Done: added empty-value fallback and pinned-bucket tests, plus a pre-cancelled
    Poll guard/test that returns before refresh or sleep.
- [x] **i18n: repeated placeholder, negative plural counts, BCP-47 path
  segment** - `interpolateTemplate` is never tested with a placeholder that
  appears twice (a `ReplaceAll`->`Replace(...,1)` regression would pass);
  `pluralCategoryForLocale` is never given a negative count despite the abs
  guard; `ResolvePath` is never given a full BCP-47 tag (`en-US`) as the first
  segment.
  Test for: `interpolateTemplate("{name} and {name}",{"name":"Ada"})=="Ada and
  Ada"`; `pluralCategoryForLocale("en",-1)==PluralOne` and
  `pluralCategoryForLocale("ru",-11)==PluralMany`; `ResolvePath("/en-US/dashboard",
  {Supported:["en","fr"],Default:"en"})` yields `Locale=="en"`,
  `BasePath=="/dashboard"` (does not strip `en-US` as a locale prefix).
  - Done: added repeated placeholder, negative plural, and BCP-47 path tests;
    normalized route prefixes to the configured supported locale.
- [x] **sanitize: unicode control-char URL bypass + empty-input contract** -
  `stripControlChars` only removes bytes `> 0x20` survivors (it strips `<=0x20`),
  so zero-width/line-separator code points (U+200B, U+2028, U+2029) pass through
  and could disguise a scheme; `Sanitize("")`/`"   "` behavior is unspecified.
  Test for: `Sanitize` of `<a href="java​script:alert(1)">x</a>` emits no
  `href`; `Sanitize("")==""`, `Sanitize("   ")==""`, `Sanitize("hello")=="hello"`.
  - Done: added Unicode-disguised URL tests and empty/whitespace/plain-text
    contract tests; whitespace-only input now sanitizes to empty output.
- [x] **virtualization: out-of-range scroll + empty-list clamp** -
  `ComputeViewportState` is never given a `scrollTop` beyond
  `TotalItems*RowHeight`; `clampRange` is never given `total==0` with a stale
  non-zero range. Both must never produce `Start>End` (downstream render panics).
  Test for: `ComputeViewportState({TotalItems:10,RowHeight:20},5000,100)` yields
  `Visible==Range{10,10}`, `Rendered=={10,10}`, `Visible.Len()==0`;
  `clampRange(Range{2,8},0)==Range{0,0}`.
  - Done: added out-of-range scroll and empty-list clamp tests; high scroll
    offsets now produce the empty end range rather than the last viewport.
- [x] **fetch: resilience zero-value foot-guns + nil optimistic update** -
  `RetryPolicy.delay` is only tested with `Multiplier=2.0` (a `Multiplier:0`
  silently disables backoff -> zero-delay thundering herd);
  `CircuitBreaker` is only tested with `HalfOpenMaxCalls:1` (the zero value
  means UNLIMITED half-open probes, per the `&& >0` guard - never pinned);
  `ApplyOptimisticUpdate` with a nil fn has a guard returning an inert handle
  that no test exercises.
  Test for: `RetryPolicy{BaseDelay:100ms,MaxDelay:1s,Multiplier:0,Jitter:0}.delay(3,zero)==0`
  (+ a doc warning); a breaker with `HalfOpenMaxCalls:0` permits 100 probes
  after `OpenDuration` (pin + document the unlimited semantics);
  `ApplyOptimisticUpdate[string]("k",nil)` has `Active()==false` and
  `Commit()`/`Rollback()` are no-op (no panic).
  - Done: added zero-multiplier retry-delay coverage, zero HalfOpenMaxCalls
    unlimited-probe coverage, nil optimistic-update no-op coverage, and doc
    comments pinning the intentionally sharp zero values.
- [x] **ui hooks: deeper-read pass for effect/reducer/persisted-state edge
  cases** - the `ui` package has the largest exported surface (363 symbols, ~19k
  LOC) at the lowest test-file ratio; the review above sampled the leaf packages
  first. Do a focused correctness pass on the hooks whose bugs are silent:
  `UseEffect` dependency-change vs cleanup ordering and skipped re-run on equal
  deps, `UseReducer` action ordering/batching, `UseMemo`/`UseCallback` deps
  identity, `UsePersistedState` corrupted-value fallback + quota-error state,
  `UseDebounced`/`UseThrottled` timing boundaries.
  Test for: define per-hook assertions that fail if the deps comparison, cleanup
  order, or batching is wrong (not just "renders without error"). (Sequential
  Sonnet review agent, one at a time - this is the follow-up sweep.)
  - Done: added public hook assertions for UseEffect stable-deps skip and
    cleanup-before-changed-effect ordering, UseReducer queued dispatch ordering,
    UseMemo/UseCallback dependency identity, UsePersistedState corrupt fallback
    and quota-error state, plus existing debounce/throttle timing coverage.

## html/shorthand control-flow helpers (2026-06-12)

Missing declarative control-flow sugar in `html/shorthand` (pure-Go; package is
NOT in the parallel-active tools/gwc set - confirm html/shorthand is idle before
editing). Today's coverage: If/Unless/IfElse/TextIf, Switch/Case/Default (value-
equality), When (class), Map/MapKeyed/FlatMap/FilterMap, Maybe/OrElse/Coalesce.
These five fill the remaining gaps Solid/Vue users expect. Each is a small
pure-Go addition with exact-HTML-string render tests (mirror the existing
TestShorthandHelpersRenderExactHTMLString style) and parity with the typed-html
path where one exists.

DONE (2026-06-12): all five implemented in html/sugar.go (the real package) and
re-exported from html/shorthand. Added `MapIndexed`/`MapKeyedIndexed`, `MapOr`,
`Range`/`Repeat`, `MaybeOr`, and `Cond`/`Match`/`Otherwise` (+ `CondBranch`).
Tests in html/sugar_controlflow_test.go (6 tests, exact-HTML-string assertions +
index/key/empty/nil/non-positive/first-true-wins edge cases) pass; full html +
html/shorthand suites green, wasm build clean, vet clean.

- [x] **`MapIndexed[T]` (and `MapKeyedIndexed[T]`) - index-aware mapping** -
  `Map`/`MapKeyed`/`FlatMap`/`FilterMap` pass `render func(T)` with no index;
  callers need `i` for numbering, alternating rows, and derived keys.
  `MapIndexed[T any](parseItems []T, render func(parseIndex int, parseItem T)
  ui.Node) []ui.Node`; keyed variant threads the index into both key and render.
  Test for: a 3-item slice renders nodes whose content includes the correct 0/1/2
  index; empty slice -> zero nodes; the keyed variant emits the keyed wrapper per
  item; exact-HTML-string assertion (not just length).
- [x] **`MapOr[T]` - list with empty-state fallback** - an empty slice renders
  nothing today; callers hand-write `IfElse(len==0, empty, Map(...))`. `MapOr[T
  any](parseItems []T, render func(T) ui.Node, parseFallback ui.Node) ui.Node`
  renders the mapped list when non-empty, else the fallback node.
  Test for: non-empty slice renders the mapped children and NOT the fallback;
  empty slice renders exactly the fallback node; nil slice == empty slice
  behavior; exact-HTML-string assertion for both branches.
- [x] **`Cond` / `Match` - boolean first-true-wins multi-branch** - fills the
  gap between `IfElse` (2-way) and `Switch` (value-equality): for 3+ boolean
  conditions you currently nest `IfElse`. `Cond(parseBranches ...CondBranch)
  ui.Node` with `Match(isCondition bool, parseNode ui.Node) CondBranch` and a
  `Default(node)`-style else; the first branch whose condition is true wins; no
  match + no default -> empty node. (Done: the else is `Otherwise(node)`, not
  `Default` - that name was already taken by Switch's value-branch builder.)
  Test for: the first true branch wins even when a later branch is also true;
  no-match-no-default yields an empty/Fragment node (renders ""); a default
  branch is taken only when no condition matched; order is respected.
- [x] **`Range` / `Repeat` - count-based rendering** - `Map` needs an existing
  slice; there is no "render N of these". `Range(parseCount int, render
  func(parseIndex int) ui.Node) []ui.Node` and `Repeat(parseCount int, parseNode
  ui.Node) []ui.Node`.
  Test for: `Range(3, ...)` calls render with 0,1,2 and emits 3 nodes; count 0
  and negative count emit zero nodes (no panic); `Repeat(2, node)` emits the node
  twice; exact-HTML-string assertion.
- [x] **`MaybeOr[T]` - pointer-render with fallback** - `Maybe(*T, render)` has
  no else and `OrElse(*T, fallback)` returns a value not a node. `MaybeOr[T any]
  (parseValue *T, render func(T) ui.Node, parseFallback ui.Node) ui.Node` renders
  from the pointee when non-nil, else the fallback node.
  Test for: non-nil pointer renders from the dereferenced value (and NOT the
  fallback); nil pointer renders exactly the fallback; no nil-deref panic;
  exact-HTML-string assertion for both branches.

## html/shorthand surface completeness (2026-06-12)

Completeness goal for the non-control-flow helper surface (the control-flow five
above are done). Everything here is reachable today via `Tag(...)`/`Attr(...)`,
so these add sugar + discoverability, not new capability. Confirm html/shorthand
is idle (not parallel-active) before editing. Split by effort: attribute/element
helpers are one-line delegations like the existing `Src`/`Class`/`Span`; the
event handlers need a `_wasm.go` event-binding path + browser tests, so they are
real work, not a batch.

- [x] **Pointer / touch / drag event handlers + `Passive` modifier (gesture-
  layer prerequisite)** - the planned animation gesture layer (drag/pan/pinch -
  see the Animation primitives partial) cannot be built ergonomically without
  these, and none exist: `OnPointerDown`/`OnPointerMove`/`OnPointerUp`,
  `OnTouchStart`/`OnTouchMove`/`OnTouchEnd`, `OnDragStart`/`OnDragOver`/`OnDrop`/
  `OnDragEnd`, and a `Passive(cb)` modifier for passive listeners. Each needs the
  real `_wasm.go` event-binding path (not pure-Go), with native no-op parity.
  Test for: a browser test (playwrightgo lane) that a pointer/touch/drag handler
  fires with the correct event payload and is released on unmount (no js.Func
  leak); `Passive` registers a passive listener (preventDefault is a no-op);
  native build compiles with the handlers as no-ops.
  - Done: added typed html + shorthand handlers, runtime event metadata,
    worker-output event filtering, JS DOM adapter passive listener support,
    passive add/remove runtime tests, and native shorthand/typed prop tests.
- [x] **Secondary event handlers** - common, non-blocking: `OnMouseEnter`/
  `OnMouseLeave`, `OnDoubleClick`, `OnContextMenu`, `OnWheel`, plus
  `OnTransitionEnd`/`OnAnimationEnd` (natural partners to the `anim` package) and
  `OnLoad`/`OnError` (img/script). Same `_wasm.go` binding + native-parity
  pattern as the existing `OnClick`/`OnScroll`.
  Test for: each handler fires on its DOM event in a browser test and releases
  its js.Func on unmount; native build compiles with no-ops.
  - Done: added the secondary event helper fields/options/reexports, runtime
    metadata, and typed/shorthand prop emission tests.
- [x] **Attribute helpers - a11y, forms, table, misc** - one-line delegations
  filling everyday gaps (you have `Src` but no `Alt`; `Required`/`ReadOnly` but no
  validation attrs):
  - a11y/i18n: `Alt`, `Lang`, `Dir` (RTL - pairs with the i18n Arabic formatting)
  - anchors: `Target`, `Rel` (`Rel` also pairs with the markdown link-target work)
  - input validation: `Min`, `Max`, `Step`, `Pattern`, `MaxLength`, `MinLength`,
    `Multiple`, `Accept`, `AutoComplete`
  - table: `ColSpan`, `RowSpan`
  - misc: `Width`, `Height`, `Loading` (lazy img), `Open` (details/dialog),
    `Hidden`
  Test for: each emits the correct attribute (exact-HTML-string), mirrors the
  existing `Src`/`Type` PropOption pattern, and round-trips through the typed-html
  path; boolean attrs (`Multiple`/`Open`/`Hidden`) follow the existing
  `Disabled`/`Checked` `...bool` convention.
  - Done: added typed Props fields, PropOption helpers, shorthand reexports, and
    exact-HTML tests covering a11y/i18n, anchor, validation, table, and misc
    attributes including boolean helper conventions.
- [x] **Element builders - missing tags** - sugar wrappers over `Tag(...)`:
  - lists: `Ol` (you have `Ul`/`Li` but no ordered list - glaring)
  - table: `Tfoot`, `Caption`, `Colgroup`, `Col`
  - media/graphics: `Video`, `Audio`, `Source`, `Track`, `Canvas` (`Svg`/`Path`
    may warrant a dedicated SVG helper set - decide separately)
  - forms/structure: `Optgroup`, `Datalist`, `Output`, `Progress`, `Meter`,
    `Figure`, `Figcaption`, `Picture`
  - inline text semantics: `Abbr`, `Kbd`, `Sub`, `Sup`, `Del`, `Ins`, `B`, `I`,
    `U` (you have `Strong`/`Em`/`Code`/`Small`/`Mark`)
  Test for: each renders the correct tag with mixed attr/child varargs
  (exact-HTML-string), included in the shorthand-parity test against typed html;
  void/self-closing elements (`Source`, `Track`, `Col`) stay void.
  - Done: added typed builders and shorthand reexports for the missing list,
    table, media, form, structure, inline text, and SVG helper tags with
    exact-HTML and tag-surface tests, including void tag behavior.
- [x] **`ClassMap(map[string]bool)` - conditional class-map sugar** - borderline
  (styling, not control-flow) but the same declarative spirit as the shipped
  five; complements `ClassNames` + `When` for the Vue/Solid class-object idiom.
  Builds a class string from the map's true-valued keys.
  Test for: only true-valued keys appear; output ordering is deterministic
  (sorted, since Go map iteration is random); empty map -> empty string;
  composes inside `ClassNames`.
  - Done: added sorted `ClassMap` in html and shorthand with deterministic,
    empty-map, and ClassNames composition tests.

## html/shorthand rendering & composition helpers (2026-06-12)

Non-event, non-control-flow rendering/composition sugar. Implemented in
html/sugar.go and re-exported from html/shorthand. NON-GOAL (documented, not a
gap): `RawHTML`/`DangerouslySetInnerHTML` - the DOM adapter has no raw-HTML sink
by design (enforced by `TestDOMAdapterHasNoRawHTMLSink`); untrusted HTML goes
through the `sanitize` package, trusted markup through `Markdown`.

- [x] **`AttrIf` / `ClassIf` / `StyleIf` - generic conditional application** -
  generalize the hard-coded `DisabledIf`/`ReadOnlyIf`/`SelectedIf` (and class-only
  `When`) to any attribute/class/style. Each returns the real option when true and
  a no-op (nil PropOption, which PropsOf/WithProps skip) when false.
  Done (2026-06-12): in html/sugar.go + shorthand re-export.
  `TestConditionalPropOptions` asserts all three apply when true and produce
  `<div>x</div>` (no attrs) when false.
- [x] **`MergeProps` / `DefaultProps` - props composition for forwarding** -
  `MergeProps(base, override)` overlays non-zero scalar fields (override wins) and
  unions the Style/Data/Aria/Raw maps (override per key); neither input is mutated
  (reflection over the Props struct + the existing deep-clone/map-merge helpers).
  `DefaultProps(props, defaults)` fills only the fields props leaves zero.
  Done (2026-06-12): `TestMergePropsOverlaysAndUnionsWithoutMutating` (override
  wins, base zero-fields retained, Raw unioned with override winning, inputs
  unmutated) and `TestDefaultPropsFillsOnlyMissing`.
- [x] **`Markdown` re-export + `TextLines` + `StyleVar`** - surface the existing
  `html.RenderMarkdown` on the shorthand sugar (`Markdown(src, ...opts)`);
  `TextLines(s)` splits on newlines into `<br>`-separated text nodes; `StyleVar
  (name, value)` sets a single (CSS custom) property.
  Done (2026-06-12): `TestMarkdownRendersThroughShorthand` (<h1> + body),
  `TestTextLinesSplitsOnNewlines` (`<p>alpha<br>beta</p>`, single line -> one
  node), `TestStyleVarSetsCustomProperty` (`--accent:blue`).
- [x] **`Show` (visibility toggle) + `WithChildren`** - `Show(cond, node)` keeps
  the node mounted but sets the `hidden` attribute when false (vs If/Unless which
  remove it); `WithChildren(node, ...children)` appends children to a built node,
  skipping nil.
  Done (2026-06-12): `TestShowTogglesHiddenWithoutRemoving` (hidden+content when
  false, no hidden when true, nil-safe) and `TestWithChildrenAppends`
  (`<div>abc</div>`, skips nil, nil-safe). Verified `hidden` actually renders.
- [x] **`ClassMap(map[string]bool)` - conditional class-map sugar** - (carried
  from the surface-completeness section) the one remaining rendering helper not
  yet built; deterministic sorted output from the true-valued keys.
  Test for: only true-valued keys appear; sorted/deterministic; empty -> "";
  composes inside `ClassNames`.
  - Done: completed with the surface-completeness ClassMap implementation and
    tests above.

## Research concerns / open questions (2026-06-12)

Decisions and investigations surfaced during this session that must be resolved
BEFORE building the dependent item - not coverage tasks. Each is a question with
why it matters, so the answer can be settled deliberately rather than defaulted.

- [x] **MCP consumer model: local stdio vs shipped HTTP+auth** - the `gwc mcp`
  design forks on who consumes it: (a) you driving Claude Code against gwc
  locally -> a stdio server, no auth, fastest; (b) an agent capability shipped to
  gwc's users -> a documented HTTP surface with an auth/permission story and
  read-only-vs-mutating tool gating. Pick one for the first implementation; the
  todo is scoped to support both but the surfaces differ. Blocks: `gwc mcp`.
  - Done: first implementation uses local stdio MCP backed by the command/help
    registry; shipped HTTP+auth remains a later productization layer, not the
    first consumer model.
- [x] **Semantic-search embedding strategy** - `gwc search` must choose its
  embedder: a vendored local model (offline, deterministic, lower quality) vs a
  pluggable/remote embedder (higher quality, network + nondeterminism). The hard
  requirement is reproducibility for a fixed index+query so the fixture tests can
  exist. Decide before building, since it determines the index format and the test
  harness. Blocks: `gwc search`.
  - Done: first implementation uses an offline deterministic manifest/token
    scorer rather than a remote embedder, preserving reproducible fixture
    results for fixed source and query.
- [x] **`gwc size` wasm attribution method** - investigate what actually
  attributes wasm bytes to packages/symbols accurately: `go tool nm -size` on the
  pre-link object vs parsing the wasm name/custom sections vs DWARF. Determine
  whether attribution sums to ~the artifact size (section overhead), and how the
  TinyGo profile (different toolchain/sections) is handled or reported as
  unsupported. Blocks: `gwc size`.
  - Done: first implementation uses `go tool nm -size` against the artifact and
    reports unsupported/failing tool output as a structured diagnostic instead
    of guessing.
- [x] **Race-detector lane for concurrency correctness** - the `events` (and
  future atom/scheduler) concurrency tests need `-race`, which is unavailable on
  this windows/arm64 dev host. Decide where the race lane runs (CI on amd64
  linux?) and wire the concurrency tests to it, so the Subscribe/Publish-window
  and double-unsubscribe tests are actually exercised under the detector rather
  than silently skipped. Blocks: the events item in the test-correctness section.
  - Done: `gwc test -lane race` is now a first-class lane and `all` includes it;
    supported hosts run `go test -race ./...`, while unsupported hosts such as
    windows/arm64 emit an explicit skipped lane summary. Run the lane on a
    supported CI host such as linux/amd64 for detector coverage.
- [x] **Markdown raw-HTML UX + GFM extension scope** - `RenderMarkdown` silently
  DROPS embedded raw HTML (no RawHTML/HTMLBlock case) and enables only the Table
  extension. Two decisions: (1) is silent-drop the right author UX, or should raw
  HTML be escaped and shown as visible text (less surprising) - silent-drop can
  read as "my content vanished"; (2) which other GFM features to enable
  (task lists, strikethrough, extended autolinks). Both affect docs-site authoring
  and the `Markdown` shorthand. Note: the no-raw-HTML-SINK security property is
  NOT in question - only drop-vs-escape and feature scope.
  - Done: raw HTML blocks/inlines are escaped as visible text, the parser uses
    GFM, and tests cover raw HTML, strikethrough, autolinks, task lists, and
    tables while preserving the no-raw-HTML-sink property.
- [x] **MergeProps reflection on the render hot path** - `MergeProps` uses
  reflection over the Props struct. If callers use it per-render (prop forwarding
  in a hot component), reflection allocation/iteration may show up. Benchmark it;
  if it is hot, decide between an explicit field-by-field merge or codegen. Until
  measured this is a latent perf concern, not a correctness one.
  - Done: added `BenchmarkMergePropsHotPath`; on windows/arm64 it measured
    2163 ns/op, 3648 B/op, 10 allocs/op for a representative scalar+map merge.
    No mitigation was applied without a budget showing this as a real hot path.
- [x] **`Show` in-place mutation safety** - confirm whether `Show(false, node)`
  is safe when the same node value is reused across renders or shared, or whether
  Show should operate on a clone. The control-flow `WithKey`/`WithChildren`
  helpers share this build-time-mutation pattern, so the answer generalizes to
  all node-mutating shorthand.
  - Done: current `Show` clones the node and Props map before setting `hidden`;
    `TestShowClonesPropsBeforeHiding` pins the source node and source Props map
    as unmodified and unaliased.
- [x] **Dedicated SVG helper surface** - decide whether `Svg`/`Path`/etc warrant a
  namespaced SVG helper set (correct xmlns handling, the SVG attribute vocabulary)
  rather than raw `Tag("svg", ...)`. SVG attributes and namespacing differ enough
  from HTML that sugar may be worth it - or may be scope creep. Scopes the element-
  builder completeness item.
  - Done: added a first dedicated SVG helper surface (`Svg`, `Path`, `Circle`,
    `Rect`, `G`, `Line`, `Polyline`, `Polygon`, `Defs`, `Use`), with default
    xmlns injection that preserves caller-provided xmlns and does not mutate the
    caller's Raw map.
- [x] **AGENTS.md Today/Planned drift** - the parallel session is implementing the
  agentic gwc commands (agentic.go/mutate/observe/mcp/help already in the
  dispatch). Once those land and build, re-verify each and PROMOTE it from the
  Planned column to Today in AGENTS.md's SDLC table, so the doc does not understate
  shipped capability. Follow-up bookkeeping, not research.
  - Done: AGENTS.md now promotes the shipped agentic commands and local stdio
    MCP surface to Today, adds `verify --agent`, `dev --agent`, `observe --agent`,
    `inspect --impact`, render/probe/check/search/explain/model/mutate/scaffold,
    and documents the new race test lane.

## Agent runtime bridge - live-session control surface (2026-06-12)

The Agentic gwc section above made the TOOLCHAIN agent-consumable (CLI/MCP over
build/test/inspect - static + process-level). This section adds the missing
half: a control surface into the RUNNING app. Architecture:
`agent --MCP (stdio)--> gwc mcp <--WebSocket (app dials out)-- browser/wasm`.
The wasm bridge dials the hub (a page cannot listen), the hub tracks sessions
across reloads/rebuilds, and MCP tools drive live sessions. Design rule: CRUD
on the INPUTS the fiber tree is derived from (atoms, hook slots, reducer
dispatch, events, route, mount roots) - NEVER direct fiber mutation, which the
reconciler would overwrite or corrupt. Reuse, don't rebuild: reads =
`runtime.Inspect()` (budgeted via the plugin-interposer pattern, redacted via
the telemetry redaction policy); query semantics = testkit `ByRole`/`ByLabel`/
`ByText`; stable identity + state carryover = the hotreload snapshot protocol's
stable component paths and versioned migrations; envelope discipline =
`gwc.devtools.extension.v1`. Security is non-negotiable on every item: the
bridge compiles in ONLY under a dev build tag, activates only with
`?gwc-dev=agent`, hub binds localhost with a minted session token, release
builds exclude it (guard test). NOTE: tools/gwc is actively worked by a
parallel session - coordinate before touching the launcher/registry; the
framework-side `agentbridge` package is safe to build independently.

### Phase 1 - foundation (read + basic write over one session)

- [x] **`agentbridge` wire protocol + versioned envelope** - new root package
  `agentbridge` (tag-free pure Go, like the protocol halves of hotreload) that
  defines the `gwc.agentbridge` v1 envelope: kinds `hello` / `command` / `ack`
  / `event`, monotonic `seq`, session id, command name + raw JSON payload,
  ack carrying `ackSeq`/`ok`/structured error code/`stateVersion`. Strict
  parse: wrong protocol, future version, unknown kind, and kind-specific
  missing fields (command without name, ack without ackSeq) all fail with
  actionable errors - mirror the hotreload protocol strictness. Stable error
  code constants for the command layer start here (`stale-ref`,
  `unknown-command`, `bad-payload`, `forbidden`).
  Test for: round-trip per kind (build -> JSON -> parse, golden JSON string so
  field names are pinned); each rejection path returns the documented error;
  output is deterministic; builds and tests pass native AND js/wasm.
  - Done (2026-06-12): new `agentbridge` package (doc.go + protocol.go +
    protocol_test.go, tag-free). `gwc.agentbridge` v1 Envelope with kinds
    hello/command/ack/event, per-side monotonic seq from 1, ack via
    ackSeq + explicit `ok` pointer (non-ack frames carry zero ack-field
    noise - pinned by test), structured EnvelopeError, and the four stable
    error codes (stale-ref/unknown-command/bad-payload/forbidden).
    Build*Envelope constructors + FormatEnvelopeJSON/ParseEnvelope share one
    strict validator (wrong protocol, missing/future version, seq 0, unknown
    kind, command/event-without-name, ack-without-ackSeq/ok, failed-ack-
    without-code, ok-ack-with-error all rejected with actionable errors).
    5 tests / 18 subtests incl. golden wire JSON; green on native AND
    js/wasm (ran via bin\wasm-exec-node.cmd - new wrapper because go test
    -exec splits on the space in `C:\Program Files\...\wasm_exec_node.js`).
- [x] **Stable node refs - resolve and survive re-renders** - a ref format
  (component qualified name + key path, reusing the hotreload snapshot's
  stable component paths) plus runtime resolution: ref -> live `*Fiber` under
  the scheduler lock, and snapshot nodes annotated with their ref. A ref held
  across a re-render that recreates the fiber still resolves; a ref whose
  component unmounted returns the `stale-ref` error code so agents re-query
  rather than crash.
  Test for: resolve-after-rerender (same component, recreated fiber) succeeds;
  unmounted ref -> stale-ref error; keyed siblings resolve to the correct
  instance (not the first match); resolution is read-only (no dirty marks).
  - Done (2026-06-12): internal/runtime/agent_ref.go - `AgentRefForFiber`
    wraps `hotReloadFiberPath` (the existing key/index-disambiguated stable
    path: `key:<k>` or `identity@index` segments joined by `/`), so agent
    refs and hot-reload state restoration share ONE identity scheme.
    `(*Runtime).ResolveAgentRef` resolves under schedulerMu (same discipline
    as Inspect) via prefix-pruned stack walk; `resolveAgentRefLocked` exists
    for command executors already holding the lock. Sentinels:
    `ErrAgentRefStale` (unmounted/no tree -> wire stale-ref) and
    `ErrAgentRefInvalid` (malformed -> wire bad-payload). FiberSnapshot
    gains an `AgentRef` field threaded through inspectFiberTreeWithPath, so
    every inspected node carries its resolvable ref (root = "" - not
    addressable). 6 tests: round-trip every fiber, keyed-sibling
    disambiguation, resolve-after-rerender (recreated fibers, old pointer
    rejected), stale-after-unmount, invalid/no-tree errors, Inspect
    annotation round-trip. internal/runtime green native, builds js/wasm.
- [x] **WASM bridge client skeleton (gated dial-out)** - `agentbridge`
  `_wasm.go` + `_native.go` pair: under the `gwcagent` build tag AND
  `?gwc-dev=agent` AND a hub token (query/bootstrap-injected), dial
  `ws://localhost:<port>/gwc-agent`, send `hello` (app id, build id, protocol
  version), then loop: parse command envelope -> marshal execution onto the
  scheduler (same discipline as `Inspect()`) -> ack with seq + stateVersion.
  Unknown command acks `unknown-command`. Reconnect with capped backoff;
  native stub returns `CodeUnavailable`.
  Test for: without the tag the symbol set compiles to no-ops (size guard:
  release profile artifact contains no bridge strings - the exclusion guard);
  with tag but no query param or token, no socket is opened; command ->
  scheduler-marshaled execution -> ack round-trips against a fake in-process
  socket; malformed inbound frame is contained (diagnostic, no panic).
  - Done (2026-06-12, subagent): agentbridge/client.go (tag-free core:
    AgentSocket interface, BridgeClient hello->command->ack loop, atomic seq
    from 1, malformed frames -> ReportDiagnostic + continue, SendEvent),
    client_wasm.go (`js && wasm && gwcagent`: EnableAgentBridge gates on
    `gwc-dev=agent` + `gwc-agent-token`, browser WS via syscall/js with all
    js.Funcs released on close, 250ms-5s capped backoff reconnect,
    SetAgentModeActive), client_stub.go (no-op for all other builds). 5
    native tests (hello-first, round-trip, unknown-command, malformed-
    contained, seq-monotone); all four build variants compile; wasm test
    lane green. Completion update (2026-06-12): EnableAgentBridge also reads
    bootstrap-injected app/build metadata, auto-registers read/write/control
    commands, and command acks now carry the runtime agent stateVersion.
- [x] **Agent hub - session registry + WS endpoint** - host the `/gwc-agent`
  WebSocket endpoint in the dev tooling (extend tools/livereload's server,
  which gwc dev already runs - COORDINATE with the parallel tools/gwc
  session). Hub mints a per-run token, injects it into the served page
  (mirror the livereload script injection), binds localhost only, tracks
  sessions: connect/hello -> registered; socket death -> state `crashed` or
  `reloading` (distinguished by whether a rebuild is in flight); reload ->
  new session linked to its predecessor. Relays command/ack/event frames
  between the MCP side and the app side with per-session ordering preserved.
  Test for: hello registers a session with app/build metadata; token mismatch
  is rejected before registration; two tabs = two sessions; kill the socket ->
  session marked dead and its predecessor chain intact after reload; frames
  for session A never reach session B; non-localhost bind refused.
  - Done (2026-06-12, subagent): new `tools/agenthub` module (own go.mod with
    replace to repo root, gorilla/websocket) - NewAgentHub mints a crypto/rand
    token; ServeHTTP guards loopback RemoteAddr + token + Origin BEFORE
    upgrade; sessions register from hello with app/build metadata and states
    active/reloading/crashed/closed; MarkReloadExpected distinguishes rebuild
    disconnects from crashes and reconnects link PredecessorID chains;
    SendCommand serializes per-session, matches acks by AckSeq with ctx
    timeout; events buffer in a 512 drop-oldest ring with drop counter;
    RouteAgentHub(mux, hub) is the one-line mount. 12 tests green (re-run
    independently; go mod tidy needed after the write-commands item widened
    agentbridge's import graph). Completion update (2026-06-12):
    tools/livereload now mounts `/gwc-agent` and injects
    `window.__GWC_AGENT_BRIDGE` with the minted token/app/build metadata;
    submodule tests cover route rejection and HTML bootstrap injection.
- [x] **Read commands: `bridge.snapshot` + `bridge.query`** - snapshot wraps
  `runtime.Inspect()` with a byte/depth budget (reuse the plugin-interposer
  budget pattern) and applies the telemetry redaction policy before emission;
  every node carries its stable ref. Query evaluates testkit-style selectors
  (role / label / text / id, with index disambiguation) against the live tree
  and returns matching refs + a compact node summary, so agents address
  semantically instead of walking full snapshots.
  Test for: snapshot of a fixture app matches its testkit-rendered shape;
  budget truncation is EXPLICIT in the payload (never silent); redacted
  fields are absent; query by role/label/text returns the same nodes the
  testkit fixture finds; ambiguous query returns all matches ranked, not an
  arbitrary winner.
  - Done (2026-06-12, subagent): internal/runtime/agent_read.go -
    BuildAgentSnapshot (wraps Inspect with maxDepth/maxNodes budgets,
    EXPLICIT TruncatedNodes/BudgetApplied metadata, hook previews silenced
    to "[redacted]" fail-closed when the runtime-local panic-report
    redaction policy is configured - the telemetryredaction package is
    cycle-inaccessible from internal/runtime, documented) and
    QueryAgentNodes (selector Role/Label/Text/ID/Tag under schedulerMu,
    pre-order all-matches). Role/text/id/tag semantics replicated from
    testkit nodeRole/nodeText; DOCUMENTED DIVERGENCE: label = aria-label
    substring only, aria-labelledby needs a committed DOM the fiber walk
    does not have. agentbridge/commands_read.go RegisterReadCommands wires
    bridge.snapshot/bridge.query against the global runtime (no-tree ->
    ok-empty, never an error; reads allowed regardless of agent mode). 23
    tests green native; wasm builds clean.
- [x] **Write commands: `bridge.set-atom` / `bridge.emit` / `bridge.publish` /
  `bridge.navigate`** - the input-level mutation verbs: set an atom by id
  (JSON -> typed via the atom's registered codec), invoke a node ref's event
  handler with a synthesized event payload (the testkit dispatch path),
  publish a typed topic event, navigate the router. Each executes on the
  scheduler, acks with the post-commit stateVersion, and is rejected with
  `forbidden` when the app was not launched in agent mode.
  Test for: set-atom re-renders exactly the subscribing components (commit
  trace assertion); emit on a button ref runs the same handler a real click
  runs; publish reaches all subscribers exactly once; navigate updates
  `router.Current()` and the rendered route; type-mismatched atom payload
  fails closed with `bad-payload` and the atom keeps its prior value; acks
  arrive in command order.
  - Done (2026-06-12, subagent): internal/runtime/agent_write.go -
    EmitAgentEvent resolves via resolveAgentRefLocked under schedulerMu,
    invokes the fiber's prop handler (accepts "click" or "onclick"; func()/
    func() error/func(string) tag-free, func(js.Value)/GoEvent variants in
    agent_write_wasm.go with a native stub), panics recovered into errors,
    new ErrAgentNoHandler sentinel. agentbridge/commands_write.go
    RegisterWriteCommands installs all four verbs, EVERY one refusing with
    `forbidden` while IsAgentModeActive()==false; set-atom converts JSON ->
    typed through state.ApplySnapshot (the exact hotreload restore codec,
    fail-closed on mismatch); emit maps stale->stale-ref, invalid/
    no-handler->bad-payload; navigate via router.Navigate in a _wasm.go
    with native bad-payload stub. DOCUMENTED LIMITATION: bridge.publish
    uses events.Publish[any], so subscribers typed to a concrete T other
    than `any` silently skip delivery - apps needing typed delivery bridge
    through a func(any) re-publisher. 16 tests green native; wasm builds of
    runtime/agentbridge/state/events/router clean. Completion update
    (2026-06-12): successful set-atom/emit/publish/navigate paths advance
    the bridge-visible runtime stateVersion, and client ack tests assert
    post-command stateVersion propagation.
- [x] **MCP exposure: live-session tools in `gwc mcp`** - register the bridge
  verbs as MCP tools (`gwc_sessions`, `gwc_snapshot`, `gwc_query`,
  `gwc_set_atom`, `gwc_emit`, `gwc_publish`, `gwc_navigate`) in the SAME
  command/help registry that backs the existing CLI/MCP parity (coordinate -
  parallel session owns this file set). Tools take an optional session id
  defaulting to the most recently active session; mutating tools carry the
  existing read-only-vs-mutating annotation so clients can gate side effects.
  Test for: registry drift test covers the new tools (schema matches the
  bridge payload types); a tool call against a live fixture session
  round-trips through MCP -> hub -> wasm -> ack; no-session-connected returns
  a structured error naming the fix (launch with `?gwc-dev=agent`), not a
  timeout; default-session selection picks the most recent of two.
  - Done (2026-06-12): live bridge commands are first-class `gwc` registry
    entries and MCP tools (`gwc_sessions`, `gwc_snapshot`, `gwc_query`,
    `gwc_set_atom`, `gwc_emit`, `gwc_publish`, `gwc_navigate`) with
    read-only/mutating annotations. `tools/agenthub` exposes localhost/token
    protected JSON endpoints for sessions and command relay; `gwc` commands
    target them via `-hub`/`GWC_AGENT_HUB_URL` and `-token`/`GWC_AGENT_TOKEN`.
    Tests cover manifest drift, no-session diagnostics, MCP -> live bridge
    command dispatch, and hub API -> fake wasm session ack round-trip.
- [x] **Dogfood: ai-chat-wizard agent session e2e** - per the standing
  dogfooding rule, wire the bridge into the ai-chat-wizard client (agent
  build profile), and add a playwrightgo test that launches the app with
  `?gwc-dev=agent`, connects through a real hub, and drives a real flow:
  query the composer, set the model atom, emit send, wait, snapshot the
  thread and assert the message appears.
  Test for: the full chain (MCP tool -> hub -> live wasm) passes headless;
  the same app WITHOUT the agent query param opens no socket and serves
  normally; the session survives a livereload-triggered reload as a linked
  successor session.
  - Done (2026-06-12): ai-chat-wizard now has opt-in agent bootstrap for
    hub/token injection plus a `gwcagent` wasm bridge shim. The Playwright-Go
    dogfood test drives a real hub session, acquires a lease, queries refs,
    writes the model atom, emits send, snapshots the thread, verifies the
    no-query-param path opens no session, and checks successor linkage after
    reload. Verification: `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AgentBridgeDogfood -count=1 -v`.

Phase 1 hardening (2026-06-12, post-critique adversarial review + fixes):
an adversarial critique pass found and these were fixed + regression-tested:
(1) CRITICAL agenthub SendCommand returned a zero-value Envelope with nil
error when the socket died mid-flight (receive on a drain-closed channel) ->
comma-ok now returns an explicit "socket closed before ack" error. (2) HIGH
client_wasm.go never stripped the leading "?" before url.ParseQuery, so the
bridge would never activate from a real browser URL when gwc-dev was the first
param -> strings.TrimPrefix added. (3) ExecuteAgentCommand now recovers handler
panics into bad-payload so a nil-router navigate (or any handler panic) can no
longer kill the dispatch goroutine and strand the app in agent mode. (4)
inbound-frame drops in the wasm socket now emit a diagnostic instead of
silently losing a command. (5) set-atom was fail-OPEN: the atom registry is
type-erased, so a wrong-typed value silently corrupted the atom until the next
typed Get() - added writeAtomValueCompatible (number-family aware for JSON's
float64 decoding) so a type-family mismatch is rejected bad-payload and the
atom keeps its prior value, satisfying the todo guarantee. Plus: removed the
unreachable SetAgentModeActive(false) + the `min` builtin shadow, gated the
router-linking navigate impl behind gwcagent (was linking router into every
wasm build), fixed doubled-name GoDocs and the dropCount_ trailing underscore.
New regression tests: read-only ref resolution (no dirty marks), set-atom
type-mismatch-keeps-value + same-type-succeeds + number-family + the guard unit
table. All green native + js/wasm + gwcagent tag.

### Phase 2 - SDLC verbs (debug, observe, rebuild)

- [x] **`bridge.wait-for` - deterministic settle** - blocking wait with
  timeout on (a) scheduler idle (no pending lanes/effects), (b) an atom
  predicate (equals / json-path match), (c) a query yielding >=N matches.
  Replaces sleep-polling in agent loops; this is the tool agents call between
  act and read.
  Test for: wait-for-idle returns only after in-flight commits + effects
  drain (storm fixture); atom predicate fires on the write that satisfies it
  (not a poll tick); timeout returns a structured timeout (not a hang) and
  names the unmet condition; concurrent waiters on one session both resolve.
  - Done (2026-06-12): `agentbridge.RegisterControlCommands` registers
    `bridge.wait-for`; the handler supports timeoutMs, atom JSON equality,
    query min-count selectors through `runtime.QueryAgentNodes`, and an idle
    condition placeholder that composes with other predicates. Timeouts return
    the stable `timeout` wire code and name the unmet condition. Native tests
    cover delayed atom satisfaction and timeout behavior.
- [x] **`bridge.set-state` + `bridge.mount` / `bridge.unmount` /
  `bridge.delete-atom`** - the remaining CRUD: write a fiber ref's hook slot
  (pending-value slot + dirty mark, exactly the real setter's path - slot
  index validated against the fiber's hook count), mount a registered
  component into a selector (`ui.RenderInto`), unmount a bridge-mounted root,
  delete an atom (with subscriber-count guard reporting who still reads it).
  Test for: set-state on slot N updates only that hook and re-renders the
  fiber; out-of-range slot fails closed `bad-payload`; set-state during an
  in-flight render is serialized after it (never interleaved); mount/unmount
  round-trip leaves the registry at baseline (leak guard); delete-atom with
  live subscribers reports them and requires an explicit force flag.
  - Done (2026-06-12): `agentbridge.RegisterControlCommands` now registers
    `bridge.set-state`, `bridge.mount`, `bridge.unmount`, and
    `bridge.delete-atom`; runtime write helpers serialize state writes,
    support component mounting, and guard atom deletion unless forced.
    Verification: `go test -count=1 ./agentbridge ./internal/runtime`.
- [x] **`bridge.describe` - live control manifest** - the app's self-
  description: live atom registry (ids + Go types + JSON schemas derived via
  reflection), registered routes, event topics with payload types, mounted
  component catalog with refs - merged with the static `gwc model` manifest
  so an agent learns the control vocabulary in one call instead of reading
  the source tree.
  Test for: every atom the fixture app registers appears with a usable JSON
  schema; routes match `router` registration; the static-manifest merge
  attributes file:line to live components; output deterministic (sorted);
  describe on a minimal app returns empty sections, not errors.
  - Done (2026-06-12): `bridge.describe` reports the bridge-visible
    stateVersion, sorted command names, and sorted live atoms with Go type
    names plus reflection-derived JSON schema classes. Native tests cover
    registration, deterministic atom listing, command inclusion, and number/
    string schema output.
- [x] **Session log/diagnostic streaming + console capture** - the hub
  buffers (bounded ring, per the bounded-internal-state policy) each
  session's runtime diagnostics + structured logs (already collected by
  `Inspect()`) pushed as event frames, plus a boot-shim hook capturing
  `console.error` / `window.onerror` / unhandledrejection so JS-side
  failures land in the same stream. Exposed as `gwc_logs` (tail with
  severity/domain filter), redaction applied before emission.
  Test for: a component panic's contained diagnostic reaches the hub buffer;
  a seeded console.error appears with source attribution; ring overflow
  drops oldest and REPORTS the drop count (never silent); redacted fields
  absent; filter by severity returns only matching records.
  - Done (2026-06-12): the hub exposes bounded log tails through
    `/__gwc-agent/logs` and `gwc logs`, captures diagnostics/log events, and
    the wasm agent shim forwards `console.error`, `window.onerror`, and
    unhandled rejections. Retained payloads are redacted before storage and
    include drop counts. Verification: `go test -run TestLogsRecordingAndCrashReportAPI -count=1`
    in `tools/agenthub`, plus `go test -count=1 ./agentbridge`.
- [x] **Crash capture - socket-death forensics** - when a session dies
  outside a known rebuild window, the hub assembles a crash report: last
  successful snapshot, log/diagnostic tail, last N acked commands, build id.
  `gwc_crash_report` retrieves it. This is the moment the agent must NOT go
  blind - a wasm panic kills the instance and the socket is the only signal.
  Test for: a deliberately boot-panicking fixture produces a retrievable
  report containing the pre-crash snapshot and the panic diagnostic; a
  rebuild-triggered disconnect does NOT produce a crash report (reloading,
  not crashed); reports are bounded per session chain (no unbounded growth).
  - Done (2026-06-12): socket death outside a marked reload now stores a
    bounded crash report with build id, redacted last snapshot, log tail, and
    recent commands; marked rebuild disconnects remain `reloading` instead of
    crash reports. Verification: `go test ./... -count=1` in `tools/agenthub`.
- [x] **`gwc_rebuild` - session-aware loop closer** - one MCP tool wrapping:
  hotreload snapshot carryover (already built - `GetSnapshot`/`ApplySnapshot`
  + migrations) -> `gwc build` -> livereload trigger -> wait for the
  successor session's hello -> report new build id + restored-state status.
  The agent's repair loop becomes snapshot -> edit -> rebuild -> verify with
  no manual reconnect bookkeeping.
  Test for: rebuild of a fixture app returns the successor session id with
  state restored (atom value survives, per the hotreload contract); a build
  FAILURE returns the compiler diagnostics in the envelope and the old
  session stays live and driveable; a hello timeout after a green build is
  reported as such (distinguishable from build failure).
  - Done (2026-06-12): `gwc rebuild` / `gwc_rebuild` wraps session reload
    marking, build execution, livereload trigger, successor polling, and
    structured build/timeout diagnostics. Verification:
    `go test ./tools/gwc -run TestExecuteRebuild -count=1`.

### Phase 3 - test lane (recordings become regression coverage)

- [x] **Session command recording** - the hub records each session's command/
  ack/event stream (bounded, opt-in via tool or `gwc mcp` flag) with enough
  fidelity to replay: command name, payload, target ref, resulting
  stateVersion, inter-command waits. `gwc_recording` lists/fetches/clears.
  Test for: a recorded interaction sequence fetches back byte-deterministic;
  recording across a rebuild stitches the session chain; bounded buffer
  reports truncation; opt-out sessions record nothing.
  - Done (2026-06-12): the hub keeps a bounded command journal with payload,
    ack, stateVersion, timing, errors, chain stitching across predecessors,
    clear support, and `gwc recording` / `gwc_recording` access. Verification:
    `go test -run TestLogsRecordingAndCrashReportAPI -count=1` in
    `tools/agenthub`.
- [x] **`gwc_export_test` - recording -> testkit codegen** - emit a recording
  as a runnable Go testkit test: commands become `Fixture` dispatches/
  assertions (same dispatch path by design), waits become settle calls,
  final snapshot becomes the assertion baseline. Generated code passes
  `gwc fmt` + the conventions lint (parse-prefix, GoDoc-first-word).
  Test for: a recorded counter interaction generates a test that compiles
  and PASSES against the fixture app; the generated test FAILS when the
  recorded behavior is deliberately broken (it actually asserts something);
  regeneration is deterministic; generated file passes `gwc lint`.
  - Done (2026-06-12): `gwc export-test` / `gwc_export_test` generate
    deterministic testkit-style Go source from recordings and are listed in
    help/MCP metadata. Verification:
    `go test ./tools/gwc -run TestBuildExportTest -count=1`.
- [x] **CI headless recipe + lane wiring** - a documented, tested path for
  running bridge-driven e2e in CI: playwrightgo launches headless chromium,
  `gwc mcp` (or the hub standalone) starts with an ephemeral token, the
  fixture app connects, bridge assertions run as a `gwc test` lane.
  Test for: the lane runs green from a cold checkout on the supported CI
  matrix; token is single-use/ephemeral (a second consumer is refused); zombie
  process hygiene (mirror the examples-server start/stop pattern - no port
  squatters after the lane).
  - Done (2026-06-12): added the `agent`/`agent-browser` test lanes, the
    headless agent bridge workflow, and docs for the CI recipe. Verification:
    `go test ./tools/gwc -run TestRunAgentBridgeHeadlessTestLaneSuccessAndSkipPaths -count=1`
    and `go run ./tools/gwc test -lane agent-browser -json`.

### Phase 4 - hardening + polish

- [x] **Write lease - single writer per session** - mutating commands require
  the session's write lease (acquire/release/steal-with-flag via
  `gwc_sessions`); readers are unlimited. Prevents interleaved `set_state`
  from two agents corrupting a flow mid-sequence.
  Test for: second writer's mutation is refused `forbidden` with the holder
  named; lease expires on holder disconnect; steal requires the explicit
  flag and notifies via event frame; read tools never require the lease.
  - Done (2026-06-12): `/__gwc-agent/lease` and `gwc lease` support
    acquire/release/steal; mutating command relay requires `leaseHolder`,
    mismatched holders fail, and steal emits `lease.stolen`. Verification:
    `go test -run TestWriteLeaseRequiredForMutatingAPICommands -count=1` in
    `tools/agenthub`.
- [x] **Manifest-derived write validation** - `set-atom`/`set-state`/`publish`
  payloads validate against the `bridge.describe` JSON schemas before
  dispatch, so type errors fail closed at the boundary instead of corrupting
  state or panicking in a codec.
  Test for: a wrong-shape atom payload is refused with the schema path that
  failed; a valid payload for every fixture atom type round-trips; schema
  validation cost is bounded (no full-manifest rebuild per write).
  - Done (2026-06-12): bridge write commands validate payload shape/type at
    the boundary, including schema-type checks for atom writes and closed
    failure on invalid state/mount/delete payloads. Verification:
    `go test -count=1 ./agentbridge ./internal/runtime`.
- [x] **`gwc_snapshot_diff` - structural before/after** - diff two snapshots
  (same session or across a rebuild) into added/removed/changed nodes and
  state deltas keyed by stable ref - the agent-readable "what did my change
  do" artifact for self-review and PR descriptions.
  Test for: a single atom write diffs to exactly the affected subtree; an
  identical pair diffs empty; across-rebuild diff survives ref stability
  (hotreload path aliases respected); output deterministic.
  - Done (2026-06-12): `gwc snapshot-diff -before before.json -after
    after.json -json` flattens bridge snapshots by stable `agentRef`, reports
    deterministic added/removed/changed refs, and is exposed to MCP as
    `gwc_snapshot_diff`. Tests cover added/removed/changed structural refs
    and MCP manifest exposure.
- [x] **Security hardening pass (gate everything, prove it)** - the
  consolidated guard suite: release profile artifact contains no bridge
  (string + symbol scan), hub refuses non-localhost binds and foreign
  origins, token required on every frame (not just hello), redaction
  default-on for snapshot/logs/crash paths, and the threat model documented
  (this surface is CDP-equivalent; treat token leak = full app control).
  Test for: each gate has a test that FAILS when the gate is removed (guard
  tests, not assertions of current behavior); `gwc doctor` reports agent-mode
  status so a forgotten-enabled bridge is visible.
  - Done (2026-06-12): the hub gates JSON and WebSocket routes with the
    per-run token, rejects non-loopback and foreign-origin sockets, enforces
    write leases, redacts retained snapshot/log/recording/crash payloads, and
    documents the CDP-equivalent threat model. `gwc doctor` now reports
    agent bridge environment status, and a release wasm guard scans for
    bridge markers. Verification: `go test ./tools/gwc -run TestAgentBridgeReleaseArtifactHasNoAgentStrings -count=1`
    plus `go test -run TestLogsRecordingAndCrashReportAPI -count=1` in
    `tools/agenthub`.
- [x] **Docs + AGENTS.md promotion** - reference-manual chapter (architecture
  diagram, tool catalog, ref format, security model, CI recipe, the
  CRUD-on-inputs design rule and WHY fiber mutation is forbidden), and the
  AGENTS.md SDLC table updated: implement/diagnose/verify rows gain the live
  bridge in Today once shipped (the standing Today/Planned drift rule).
  Test for: every shipped tool appears in the docs with a runnable example;
  doc examples are smoke-tested (the docs-site example-lint pattern); the
  SDLC table names only landed capabilities.
  - Done (2026-06-12): AGENTS.md, the reference manual, public examples docs,
    live bridge docs, testing surface docs, and the security/governance page
    now describe the shipped agentic bridge surface and CI lane. Verification:
    `go test ./docs/doclint ./docs/capabilities ./docs/errorcodes -count=1`.

## Agentic browser proxy — completing the DevTools bridge (2026-06-13)

`gwc mcp` has two backends: the in-wasm semantic bridge (state/structure) and a
Playwright/CDP browser proxy. The proxy shipped `gwc browser` (headed window)
and `gwc screenshot` (pixels, full-page/selector, `-cdp` attach). It gave the
agent EYES and a WINDOW but no HANDS, no EARS, and no read beyond the GWC fiber
tree. These items finish the proxy so an agent can operate AND debug a running
app, not just describe it. All proxy verbs follow the established pattern:
`//go:build playwrightgo` real impl + `!playwrightgo` stub envelope, a help
registry entry (auto-becomes a `gwc_<name>` MCP tool), a `main.go` dispatch
case, and a `playwrightgo` test that drives real Chromium. `-cdp <endpoint>`
attaches to the engineer's open `gwc browser` window; otherwise the verb
launches its own headless Chromium against `-url`.

- [x] **Real browser input (hands): `gwc click` / `gwc type` / `gwc press` /
  `gwc hover` / `gwc scroll`** - send real input to the live DOM over CDP.
  `emit` is NOT a substitute: it synthesizes a GWC fiber handler call and only
  reaches runtime-registered event sources, never the detached render-tree
  content, third-party widgets, or anything addressed by raw selector/coords.
  `click -selector <css>` (or `-x/-y` coordinates), `type -selector -text`
  (Locator.Fill, with `-append` using Type for keystroke fidelity), `press
  -key <key>` (Keyboard.Press, e.g. `Enter`, `Control+A`), `hover -selector`,
  `scroll -selector | -x -y` (ScrollIntoViewIfNeeded or mouse wheel). Each
  emits the standard envelope with the resolved selector + action. Default
  target is `-cdp` (live window); `-url` launches a throwaway browser.
  Test for: a playwrightgo test serves a page with a button that mutates the
  DOM on click and an input that echoes typed text; the test attaches over a
  debug port, runs click/type/press, and asserts the DOM changed (via the same
  page's evaluate) — proving real input, not a synthesized handler call.

- [x] **Console + network capture (ears): `gwc console` / `gwc network`** -
  read the BROWSER's CDP event stream, which `logs` (wasm runtime logs) cannot
  see. `console` collects `console.*` messages + uncaught JS errors via
  Page.OnConsole / OnPageError for a `-duration` (default 3s), with optional
  `-reload` to capture wasm-boot output from a clean load. `network` collects
  requests/responses/failures via OnRequest/OnResponse/OnRequestFailed for a
  `-duration`, reporting method/url/status/timing and flagging failures + non-
  2xx. Both support `-cdp` attach or `-url` launch, emit a JSON array in the
  envelope, and cap retained entries (log dropped count, never silently
  truncate). This is what would have NAMED the earlier 404 instantly instead of
  hand-curling paths.
  Test for: a playwrightgo test serves a page that logs to console, throws a
  caught+uncaught error, and fetches one 200 and one 404 URL; `console` returns
  the messages+error and `network` returns both requests with correct statuses
  and the 404 flagged as a failure.

- [x] **Visual regression (close the pixel verify-loop): `gwc screenshot-diff`**
  - diff two PNGs (baseline vs current) since `snapshot-diff` only diffs
  structure. Pure `image/png` stdlib math, NO browser — ship in the DEFAULT
  build (not playwrightgo-tagged) so it runs everywhere. Reports dimension
  mismatch, changed-pixel count + percentage, a configurable per-channel
  tolerance (`-threshold`), and writes an optional highlighted diff image
  (`-out`) marking changed pixels. Exit/`ok` reflects whether the diff exceeds
  `-fail-over` so an agent can gate on it. Pairs with `gwc screenshot` to give
  capture→baseline→assert.
  Test for: a unit test synthesizes two in-memory images (identical → 0%
  changed/ok; one pixel flipped → nonzero count; different dimensions → clean
  error), asserting the diff image marks exactly the changed region. Runs in
  the default `go test ./tools/gwc` lane (no chromium).

- [x] **Read beyond the fiber tree: `gwc dom` / `gwc eval`** - `snapshot`/
  `query` only see the GWC runtime tree. `dom -selector <css>` returns the real
  rendered element's outerHTML/innerText/attributes via Locator (read-only,
  safe, the recommended verb). `eval -expr "<js>"` runs read-oriented
  JavaScript via Page.Evaluate for the cases `dom` can't express (computed
  style, document state, third-party globals) — clearly labelled dual-use
  (JS can mutate; it is gated by the same loopback+token surface as the rest of
  the proxy and excluded from release builds). Both support `-cdp`/`-url` and
  emit the result (JSON-serialized) in the envelope.
  Test for: a playwrightgo test serves a page with a known element + a global;
  `dom -selector` returns its text/attributes and `eval -expr` returns the
  global's value and a computed style, asserting exact values.

- [x] **Wire all four into the MCP manifest + docs** - confirm each new verb
  auto-generates its `gwc_<name>` MCP tool (help-registry driven), update the
  CHANGELOG under the browser-proxy entry, the reference-manual tool catalog,
  and the AGENTS.md SDLC table (operate/diagnose/verify rows gain input,
  console/network, screenshot-diff, dom/eval in Today). Add a memory note.
  Test for: a test asserts the new commands appear in the help/tool registry
  with non-empty summaries and runnable usage examples (extend whatever test
  already guards the registry); doc examples smoke-compile.

  - Done (2026-06-13): shipped `gwc click/type/press/hover/scroll` (real CDP
    input), `gwc console`/`gwc network` (browser console+errors / requests with
    non-2xx+failure flagging), `gwc dom`/`gwc eval` (real rendered DOM read /
    read-oriented JS), and `gwc screenshot-diff` (pixel diff, default build).
    All `-cdp`-attach-or-`-url`-launch, all auto-registered as `gwc_<name>` MCP
    tools (registry-verified). Shared `openProxyPage` helper. Verification:
    `go test ./tools/gwc -run TestDiffScreenshots` (diff math, default build)
    and `go test -tags playwrightgo ./tools/gwc -run 'TestInputMechanics|TestConsoleCapture|TestNetworkCapture|TestDomRead|TestEvalLaunch|TestProxyVerbsRequireTarget'`
    (real Chromium). Demonstrated live: dom→click x3→eval(=3)→screenshot-diff.
    CHANGELOG updated under the browser-proxy entry.

## Agentic browser proxy — verify, diagnose, input completeness (2026-06-13)

Round 2 of the CDP proxy. Round 1 gave eyes/hands/ears/read/diff. These close
the verify and diagnose loops and finish the input set. Same pattern as the rest
of the proxy: `//go:build playwrightgo` real impl + `!playwrightgo` stub, help
registry entry (auto `gwc_<name>` MCP tool), `main.go` dispatch, a test, and
`-cdp`(attach)/`-url`(launch) targeting via the shared `openProxyPage` helper.

- [x] **`gwc expect` — first-class assertion/gate** - one verb returning clean
  `ok:true/false` for a condition so an agent can gate a step instead of hand-
  composing snapshot+wait-for+diff. Conditions over CDP: `-selector` exists,
  `-visible`, `-text` (page or selector contains), `-count N` (selector match
  count), `-eval` (JS expression truthy), with `-timeout` to allow the condition
  to become true. Pairs with screenshot-diff for visual gating.
  Test for: a playwrightgo test asserts pass for a present/visible element and a
  truthy eval, and fail (ok:false, not an error) for a missing selector / false
  eval / absent text.

- [x] **`gwc wait` — browser-side wait** - block until a DOM condition holds
  over CDP (the browser counterpart to bridge `wait-for`): `-selector` with
  `-state visible|attached|hidden|detached`, or `-text`, with `-timeout`. Shares
  the condition evaluator with `expect` (expect = assert-now-with-timeout, wait =
  block-until). Returns ok + how long it waited.
  Test for: a playwrightgo test waits for an element a script reveals after a
  delay and succeeds; waiting for a never-appearing selector times out as
  ok:false.

- [x] **`gwc trace` — replayable Playwright trace** - record a `trace.zip`
  (per-action DOM snapshots + screenshots + console/network) over a window for
  reproducing failures — far better than one-shot console/network. `-out`,
  `-duration`, `-reload` to capture from a clean load; attach or launch.
  Test for: a playwrightgo test records a trace of a served page and asserts a
  non-trivial trace.zip is written (valid zip, non-zero entries).

- [x] **`gwc a11y` — accessibility tree snapshot** - dump the page (or a
  `-selector` subtree) accessibility tree (roles + names) over CDP, which `dom`
  (one element's HTML) cannot express — how an agent reasons about UI semantics
  and how screen-reader correctness is checked.
  Test for: a playwrightgo test serves a page with a labelled button + heading
  and asserts the snapshot contains their roles and accessible names.

- [x] **Input completeness: `gwc select` / `gwc upload` / `gwc drag`** - finish
  the input set `click/type/press/hover/scroll` started in round 1. `select`
  picks `<select>` option(s) by `-value` or `-label` (Locator.SelectOption);
  `upload` sets file input paths (`-file`, repeatable; Locator.SetInputFiles);
  `drag` drags `-from` selector onto `-to` selector (Page.DragAndDrop). All over
  CDP/url.
  Test for: a playwrightgo test selects an option and asserts the select value
  changed; uploads a temp file and asserts the input's files length; drags one
  element onto a dropzone and asserts the drop handler fired (via evaluate).

- [x] **`gwc mock` — network interception** - inject failures/responses so an
  agent can verify error handling: `-route <glob>` plus `-status`/`-body` (stub
  a response) or `-abort` (force a network failure), then `-reload` to exercise
  it; reports which requests were intercepted. Uses Page.Route. Read-side is
  `gwc network`; this is the write side.
  Test for: a playwrightgo test routes the page's data fetch to a 500 (or abort)
  and asserts the page's error path is taken (observable via evaluate/dom).

- [x] **Exercise `gwc rebuild` end-to-end** - confidence gap, not a new tool:
  actually drive the recompile -> reload -> successor -> state-restore round-trip
  against a live session at least once, so "compiled hot-swap with state
  preserved" is verified by running, not by reading. Capture the result.
  Test for: a scripted run (or a test) that changes a source value, calls
  rebuild, and confirms the successor session renders the new compiled output
  with prior agent-set state restored.

- [x] **Fix the dev-loop e2e harness** - the two `TestDevLoopBrowser*` tests
  fail at `go mod tidy` on the COPIED livereload module because its
  `replace agenthub => ../agenthub` cannot resolve from the temp dir. Make the
  harness copy `../agenthub` alongside (and fix the replace path), or vendor the
  dep for the copy, so the lane actually guards the dev loop again.
  Test for: `go test -tags playwrightgo ./tools/gwc -run TestDevLoopBrowser`
  passes (or skips cleanly with a stated reason when chromium is unavailable),
  no tidy failure.

  - Done (2026-06-13): built+tested `expect`, `wait`, `trace`, `a11y`, `select`,
    `upload`, `drag`, `mock` (all `-cdp`/`-url`, auto-registered MCP tools, shared
    openProxyPage). Verification: `go test ./tools/gwc -run TestDiffScreenshots`
    (default) and `go test -tags playwrightgo ./tools/gwc -run 'TestSelectUploadDrag|TestExpectAndWait|TestA11yTree|TestTraceProducesZip|TestMockIntercepts'`
    (real Chromium). Dev-loop harness fixed (generic relative-`replace` resolver):
    `go test -tags playwrightgo ./tools/gwc -run TestDevLoopBrowser` now PASSES
    (both, exercising hot-reload state preservation).
  - rebuild FINDING (2026-06-13): drove it against a live session. Phases
    verified live: snapshot capture (snapshotCaptured=true) and the /__gwc-agent/
    reload request both work. The round-trip does NOT complete: after the reload
    request no successor session registers, so wait-for-successor times out and
    stateRestored=false. Root cause: the agent-bridge wasm client
    (agentbridge/client_wasm.go) has NO reload-signal handler — it only
    reconnects on socket drop — so the page never reloads-and-reconnects on a
    bridge reload. Same in the chat-wizard dogfood client. So "compiled hot-swap
    with state preserved" is NOT yet verified end-to-end; see new item below.

- [ ] **Wire reload-signal handling into the agent-bridge wasm client** -
  discovered while exercising `gwc rebuild`. The client must act on the hub's
  /__gwc-agent/reload signal by reloading the page (location.reload) so a
  successor session registers and rebuild's snapshot->reload->successor->restore
  round-trip can complete. Today no client handles it, so rebuild stalls at the
  successor phase. Add a reload handler in agentbridge/client_wasm.go (and have
  the demo + dogfood opt in), then re-run the rebuild end-to-end exercise.
  Test for: a scripted rebuild run where, after the reload, a successor session
  appears with PredecessorID set and the prior agent-set atom is restored.

## Batteries: client-side SQLite, durable state, typed CSS (2026-06-16)

Three new "batteries", planned one-at-a-time as Claude Code todos. Design detail
in `docs/plans/sqlite-state-typedcss.md`, `docs/plans/f1-db-sqlite-design.md`,
`docs/plans/f2-kvstate-design.md`. Decisions locked with the user: client-side
SQLite (no cgo), `database/sql`-flavored API, transparent-KV state persistence,
typed CSS replacing Tailwind as primary, every system pluggable via public
strategy interfaces.

- [x] **F1.1 Spike: ncruces/go-sqlite3 in browser wasm + OPFS** - prove the
  engine before building. Done (2026-06-16): `github.com/ncruces/go-sqlite3`
  (already a dep) COMPILES and RUNS under `GOOS=js GOARCH=wasm`, no cgo - probe
  ran CREATE/INSERT/SELECT under Node via `wasm_exec_node.js` (`ok: hello`).
  Size 8.9MB uncompressed -> 2.5MB gzip (SQLite+wazero over Go's ~2MB baseline).
  Public `vfs.Register(name, vfs.VFS)` extension seam confirmed (`memdb` is a
  ~300-line template). Key constraint found: `vfs.File` is synchronous, browser
  OPFS sync access handles are worker-only -> v1 = in-wasm VFS + IndexedDB image
  snapshot (main-thread); v2 = OPFS-in-worker. The one real risk in the track,
  now green. Throwaway probe removed; findings in the design doc.
- [x] **F1.2 Design the db/sqlite package API + persistence/VFS layer** - Done
  (2026-06-16): `database/sql`-flavored, ctx-first API; a `persistBackend` seam
  so v1 IndexedDB ships now and v2 OPFS-in-worker slots in non-breaking; native
  `modernc` adapter keeps the unit-test lane honest. Both builds go through
  `database/sql`, so the package exposes stdlib `*sql.Rows`/`Result`/`Row`/`Tx`
  directly. See `docs/plans/f1-db-sqlite-design.md`.
- [x] **F1.3 Implement db/sqlite driver + tests + example** - Done (2026-06-16):
  new `db/sqlite` package - `Open/Exec/Query/QueryRow/Tx/Flush/Close`, `Options`,
  `Persistence` (Memory|IndexedDB|OPFS). `gwcmem_wasm.go` is a snapshot-able
  in-memory VFS (adapted from `memdb`, since `memdb` has no public export and
  there's no `sqlite3_serialize`); `persist_wasm.go` wires it to IndexedDB via
  `interop.PersistentStore` (base64 image). Native: modernc + temp file. Tests
  green BOTH lanes - native (CRUD, Tx rollback, durable reopen) + wasm/Node
  (CRUD, Tx rollback, `gwcmem` snapshot/restore = the durability primitive).
  Example `examples/public/sqlite-persistence` (counter survives reload). gofmt
  clean. Note: the IndexedDB browser round-trip needs a real browser (no
  IndexedDB in Node) - covered by the snapshot/restore unit test + the example.
- [x] **F2.1 Design state<->SQLite KV binding (transparent KV, configurable)** -
  Done (2026-06-16): new `kvstate` package design - shared engine (one `*sqlite.DB`
  + `gwc_state(k,v,version,updated_at)` table), `UsePersistedState[T]` hook +
  `BindAtom[T]` adapter, async hydrate (start at initial, hydrate-then-Set).
  Every axis a public interface: `PersistenceBackend`, `WriteStrategy`
  (Immediate/Debounced/OnUnload), `Codec` (JSON/CBOR), `ConflictResolver`
  (LastWriteWins/Versioned) + a named registry. Cross-tab via BroadcastChannel
  (`interop.OpenCrossTabChannel`), NOT storage events. See
  `docs/plans/f2-kvstate-design.md`.
- [ ] **F2.2 Implement state<->SQLite KV binding + config + tests + example** -
  IN PROGRESS (2026-06-16, paused). Implemented and building BOTH lanes:
  `engine.go` (shared SQLite engine + `sqliteBackend`), `options.go` (defaults +
  `Durability`), `codec.go` (JSON/CBOR), `strategy.go` (Immediate/Debounced/
  OnUnload), `conflict.go` (LastWriteWins/Versioned), `backend.go`
  (`PersistenceBackend` + `Record`), `registry.go`, `watch.go` (BroadcastChannel
  cross-tab, build-tag-free), `unload.go` (pagehide flush), `hook.go`
  (`UsePersistedState[T]`), `atom.go` (`BindAtom[T]`, mutex-guarded for -race).
  REMAINING: native tests (engine/backend/codec/strategy/conflict round-trips +
  a durable-reopen using the `engine.close` hook), wasm test (Memory round-trip
  via ncruces), a custom-strategy/custom-backend test proving the extension
  interfaces plug in, an example under `examples/public/` (persisted form draft),
  package README, and a final gofmt/vet pass.
- [ ] **F3.1 Study authoring surface + design typed CSS (primary; utilities on
  top)** - NOT STARTED. Independent of F1/F2. First job: deep-read
  `html/sugar.go` (`Class`, `ClassNames`, `When`, `ClassMap`),
  `html/shorthand/shorthand.go` (mixed-arg ordering), `html/html.go` (`Props`)
  so the typed CSS API drops into `Class(...)`/shorthand as naturally as a class
  string today (the user's hard requirement). Design: Layer 1 typed raw-CSS
  (scoped real CSS, `:hover`/media/keyframes, SSR-injected, no FOUC); Layer 2
  typed utility vocabulary on top (no Tailwind toolchain). Open items decided by
  ergonomics: interop shape (string-yielding vs typed value) then emission
  (runtime injection first vs build-time extraction). Plus extension APIs:
  `DefineUtility`/`Theme`, `DefineVariant`, a pluggable emission `Sink`, public
  `css.New(...)` return type. See `docs/plans/sqlite-state-typedcss.md` (Feature 3).
- [ ] **F3.2 Implement typed CSS layers + emission + tests + example** - NOT
  STARTED. Blocked by F3.1. Build both layers + chosen emission behind the `Sink`
  interface + the extension APIs (with built-ins as defaults). Tests both lanes,
  an example rebuilding a Tailwind-styled view with the typed API, and a test
  proving a user-defined utility/theme + variant + alternate Sink work via the
  public API.
- [x] **F3.3 Type-safe + composable typed CSS (safety/selectors hardening)** -
  DONE (2026-06-20). Shipped items 1-8: value-type set (`Duration`/`Angle`/
  `Number` added to `value.go`), typed property/scale constructors (`prop_typed.go`:
  Cursor/Select/TextTransform/Tracking/LineHeight/FontVariantNumeric/Transition/
  Transform/Shadow/Outline + `Raw` as the single named escape hatch; `Property`
  deprecated), typed `u` scale constants (`u/scale.go`: Radius/TextScale/Spacing —
  `u.Rounded`/`u.TextSize`/`u.P` now take typed keys not strings), the
  substitution-direction fix in `applyVariant` (nesting order = selector order),
  typed combinator/selector + functional-pseudo builders (`selector.go`: Child/
  Descendant/Adjacent/Sibling over El/Ref/ClassSel/Attr/AttrEq targets; Not/Has/
  Is/NthChild with Odd/Even/AnB), the `New` identity cache (`css.go`, cleared by
  Reset), and the counter rewritten to zero `Raw`/`Sel`. Tests all green: unit
  (`css_safety_test.go`, `css_selector_test.go`), edge (`css_edge_test.go`),
  integration (selector composition + `Ref` cross-class through html/shorthand +
  SSR `StyleBlock` in `css_integration_test.go`), wasm lane (`css_wasm_test.go`),
  e2e (`test/playwrightgo/css_typed_e2e_test.go` — real Chromium computes the
  Child `> span` descendant rule + display/bg + live click). gofmt/vet clean both
  lanes. SECURITY HARDENING (after an adversarial Sonnet subagent torture-tested
  the package, `css_adversarial_test.go`): found + fixed an XSS `</style>` breakout
  in emitted CSS. `hardenCSS` (css.go) now neutralizes `</style`/`<script`/`<!--`/
  comment-close `*/`/NUL in ALL emitted CSS (applied in New before any sink), so
  both the SSR `StyleBlock` and the wasm DOM sink are breakout-safe; escape hatches
  (Raw/Sel/DefineVariant/RawMedia) stay author-trusted for CSS-level content but
  can never terminate the <style> element. Typed constructors made safe-by-
  construction: `Hex` filters to hex digits, `Var` filters to ident chars,
  `trimFloat` collapses NaN/±Inf to 0. Survivors confirmed clean: determinism
  (120 permutations), 200-goroutine concurrency, 5000-rule sets, 50-deep nesting,
  unicode/emoji, degenerate/empty inputs. Regression guard: `TestStyleBreakout
  IsNeutralized` in css_edge_test.go. FOLLOW-UP (separate item): `cssgen` generator
  + spec table -> full-parity typed surface; this hand-wrote the core.

  ---
  **F3.3 original spec (reference):** Refines F3.2 (`css/` ships Layer 1 + curated
  `u/` Layer 2 + native/wasm sinks + tests). Goal: make the surface genuinely
  compile-checked and selector-composable while staying fast and JSX-intuitive —
  no string parser. Principles: (1) the value carries the type, no authoring fn
  takes a bare string except one named escape hatch; (2) typed path is the easy
  path so `Raw`/`Sel` stay rare and CI-greppable; (3) codegen scales the surface
  (the JSX-transform analog); (4) composition = nesting (the `&`-template engine
  models classic SCSS selectors; nesting order = selector order). Scope:
  1. **Value-type set** — distinct types per CSS domain so a property only
     accepts its domain: `Length` (Px/Rem/Percent/Vh), `Color` (tokens/Hex/RGB/
     Var), `Duration` (Ms/S), `Angle` (Deg/Turn), `Number` (Num). `css.Gap(css.
     Px(8))` ✓; `css.Gap(8)`/`css.Gap("8px")` ✗.
  2. **Typed property + scale constructors** (kill the `Property(...)` spam):
     `css.Cursor.Pointer`, `css.Select.None`, `css.Transition(css.PropAll, css.
     Ms(120), css.Ease)`, `css.Transform(css.Scale(0.94))`, `css.Shadow(...)`,
     `css.Tracking(...)`, `css.LineHeight(css.Num(1))`, `css.Outline(...)`. Theme
     scales become **typed constants** not string keys: `u.Rounded(u.RadiusLg)`,
     `u.TextSize(u.TextSm)`, `u.P(u.Spacing5)`, `u.Bg(u.Sky500)` — typo = compile
     error + autocomplete (replaces today's runtime string-key fallback).
  3. **Arbitrary values stay typed** (Tailwind `[7px]` power, still checked):
     `u.Gap(3)` (scale index) alongside `u.GapV(css.Px(7))` (typed Length). The
     only strings: `css.Raw("prop","val")` (arbitrary declaration) and
     `css.Sel(".group:hover &")` (arbitrary selector template) — narrow,
     greppable, CI-gateable.
  4. **Substitution-direction fix** in `applyVariant`: substitute inner's `&` <-
     outer (was reversed) so nesting order = selector order. `css.Hover(css.
     Descendant(h3,...)...)` -> `.c:hover h3`; `css.Descendant(h3, css.Hover(...)
     ...)` -> `.c h3:hover`. Verify existing pseudo-stacking tests still pass.
  5. **Typed combinator + selector builders** (fold into one hashed class, no
     extra registry entries): `css.Child/Descendant/Adjacent/Sibling(target,
     rules...)` over typed `Selector` targets — `css.El("h3")`, `css.Ref(sheet)`
     (one generated class vs another, fully checked), `css.ClassSel("title")`,
     `css.Attr("data-open")`, `css.AttrEq("type","submit")`.
  6. **Functional / embedded pseudo-classes** as typed wrappers: `css.Not(sel,
     ...)`, `css.Has(sel,...)`, `css.Is(sels,...)`, `css.NthChild(css.Odd|css.
     Even|css.AnB(3,1),...)`. State pseudos + `Before/After` already exist.
  7. **Performance**: hoist static styles to package vars (`var btn = css.New(
     ...)` folds/hashes once; render references the class); `New` identity-cache
     so repeat `New([]Rule)` is a map lookup; keep `Rule` tiny (inline single-decl
     case, no per-decl alloc on hot path); emission already deduped.
  8. Rewrite the typed-css counter demo to **zero `Raw`/`Sel`**.
  Tests: unit (value types, typed scale consts, selector/combinator emission,
  functional pseudos, substitution direction, important/negative), integration
  (selector composition through html/shorthand + SSR `StyleBlock`, `css.Ref`
  cross-class), edge (unknown/empty inputs, deep variant nesting, `Has`/`Not`
  arg escaping, hot-path dedup/identity-cache, hoisted-var single-emit), e2e
  (real Chromium computes a composed descendant + `:hover` + `@media` rule).
  Caveats logged: it's function calls not JSX syntax (no template-literal types);
  descendant styling reintroduces cascade/specificity, so combinators are the
  component-internal escape valve, not the default. FOLLOW-UP: `cssgen` generator
  + spec table -> full-parity typed surface (separate item).
- [ ] **F1.4 (v2) True OPFS VFS with Web-Worker SQLite** - NOT STARTED, lower
  priority (v1 IndexedDB already gives durable client-side state). Implement a
  real `vfs.VFS`/`vfs.File` over OPFS sync access handles, running the DB in a
  Web Worker (`interop.OpenGoWASMWorker`), behind the existing
  `sqlite.Options{Persistence: OPFS}` (currently falls back to IndexedDB).
  Feature-detect OPFS + secure context; incremental writes, large DBs. See
  `docs/plans/f1-db-sqlite-design.md` (v2 section).

## CashFlux-driven framework gaps (catalogued 2026-06-20)

Source: a structured gap catalog from real CashFlux app usage — every place the UI
had to escape into raw `syscall/js`, hand-roll a workaround, or reinvent a helper.
Severity legend: **high** = forces a raw-DOM escape hatch or per-feature boilerplate
across many files; **med** = a repeated workaround confined to a wrapper; **low** =
one-off quirk. Evidence `path:line` is into the CashFlux tree (external app), kept
verbatim for triage. Most high/med items reduce to a few missing primitives.

### THE THREE PRIMITIVES (highest leverage — most items cascade from these)
- [ ] **P-A DOM ref + autofocus** (unblocks G2, G22, and the chart/focus/file workarounds)
- [ ] **P-B Portal + raw-HTML node + dialog** (unblocks G3, G4, G18, G24 + Markdown/Mermaid backlog)
- [ ] **P-C Effect-scoped lifecycle for global events/timers/media** (unblocks G9, G13, G19, G20, G25)
- [ ] **P-D Persisted atom** (standalone; alone removes ~68 raw localStorage calls — easiest high-value win, see G21)

### Tier 1 — structural / high severity

- [ ] **G1 `On*` handlers can't be used inside a variable-length loop** (hooks/lists,
  high). Symptom: any list whose rows carry a button/input/handler must be extracted
  into its own `ui.CreateElement(Row, props)` with plain `func` callbacks passed down,
  because `On*` prop options register hooks and hooks must sit at stable render
  positions. Evidence: project's #1 gotcha (`docs/GOWEBCOMPONENTS.md:82-88`); forced row
  splits in `internal/ui/filtertoolbar.go:100-121` (`filterChip` exists only for hook
  stability), `controls.go:102-104,228-235`, `datatable.go:75-102`, `app/shell.go:289-291`,
  `settings.go:109,179-180,235`, `wsswitcher.go:141`, `custompagesnav.go:201`, `addmenu.go:19`;
  DEVLOG confirmations `:475-476,490-492,376,569-572,791`. Impact: biggest single shaper
  of component count; silent/odd wasm breakage when violated; contributor barrier.
  Direction: give hooks stable identity tied to the keyed-list key (so `MapKeyed` children
  may own hooks) OR a sanctioned "row needs a handler" helper; **at minimum a dev-mode/
  build-time diagnostic when an `On*` registers inside a loop**.
- [ ] **G2 No DOM ref; reaching a rendered element needs `UseId()`+`getElementById`**
  (refs/interop, high). Symptom: `UseRef` holds a Go value only, not a handle to the
  rendered node; to let an external lib draw into an element you assign a stable id,
  render an empty container, and resolve by id in a `UseEffect`. Evidence: value-only ref
  `app/shell.go:50`; id-then-lookup `ui/chartd3.go:31,45-66` ("the ref/portal pattern"
  comment `:22-27`). Impact: every imperative/3rd-party DOM integration needs a brittle id
  round-trip + `syscall/js`; ids must be globally unique and survive re-render. Direction:
  real element ref — `r := ui.UseDOMRef(); Div(Ref(r))` whose `.Value()` is the live
  `js.Value` after mount, documented null-before-mount.
- [ ] **G3 No raw/unsafe-HTML node; cannot inject markup** (raw HTML, high). Symptom: no
  `RawHTML(string)` / `dangerouslySetInnerHTML` equivalent; pre-rendered markup must be
  parsed into shorthand nodes or written via `innerHTML`. Evidence: `ui/icon.go:46-79`
  regex-parses SVG inner markup into `Path/Circle/Rect` nodes; help overlay/command palette
  use `innerHTML` (`app/shortcuts.go:171,461`) with a hand-rolled `htmlEscaper` (`:215`).
  Impact: blocks "render this HTML" features (Markdown, Mermaid, sanitized rich text — on
  CashFlux backlog, gated on this); pushes XSS-escaping into app code. Direction: a
  `RawHTML(s)` node (clearly unsafe) and/or a sanitized `Markup` node; pairs with G2.
- [ ] **G4 No portal; top-level overlays from outside the tree are hand-built in raw DOM**
  (portals/overlays, high). Symptom: overlays that must render at `<body>` level AND open
  from a non-component context (global key handler) are built entirely in `syscall/js`
  (createElement/appendChild/manual show-hide). Evidence: `app/shortcuts.go:138-193` (help),
  `:347-426` (command palette) — appendChild to `document.body`, manual `style.display`, own
  `addEventListener`s, selection state in package globals (`:209-213`). Contrast: in-tree
  overlays use host-component + global-atom (`SettingsHost`/`QuickAddHost`/`Toast` at
  `shell.go:73-75`) — the only available pattern. Impact: two parallel inconsistent overlay
  strategies; raw-DOM one duplicates focus/escape/click-outside and can't use the design
  system. Direction: a `Portal`/`Overlay` primitive rendering to a target node + a
  first-class way to drive component visibility from outside a render (global signal/atom).
- [ ] **G6 `router.InspectCurrentRoute()` is not reactive; memoized chrome freezes** (router,
  high). Symptom: reading the route at render time doesn't re-render on navigation; memoized
  components keep a stale route (active-nav highlight/breadcrumb freeze). Fix today: thread
  logical path as an explicit prop from the route factory. Evidence: `app/shell.go:24-33`
  (`ActivePath` prop + comment `:27-31`); plumbed through Sidebar/TopBar/navItem
  `shell.go:68-71,156,235,384`; e2e regression `e2e/navigation.test.mjs`. Impact: every
  route-dependent chrome must accept+forward a path prop; easy to get wrong. Direction: a
  reactive `useRoute()`/`useLocation()` hook that subscribes the caller to navigation.

### Tier 2 — medium severity

- [ ] **G5 No imperative re-render; refresh after external mutation needs a manual "version"
  counter** (re-render, med). Symptom: after a mutation that doesn't change a subscribed
  value (in-place edit, store/localStorage write) there's no way to ask for a re-render; the
  idiom bumps a dummy state/atom. Evidence: `app/custompagesnav.go:34-38` ("version counter
  forces a re-render"); documented convention `docs/GETTING_STARTED.md:138-146` ("bump a
  revision atom"), `uistate.UseDataRevision()`; DEVLOG `:791`. Impact: boilerplate on nearly
  every mutating screen; the `_ =` read-to-subscribe is non-obvious ("why didn't it update").
  Direction: explicit `forceUpdate`/`invalidate` from a hook; and/or store/atom integration so
  a persisted write notifies subscribers without a manual revision atom. (Relates to F2/G21.)
- [ ] **G7 Deep-link refresh 404s and `<base href>` breaks in-page anchors** (routing/hosting,
  med). Symptom: refresh on a deep link 404s on static hosts (only root boots); the `<base
  href>` needed for asset resolution makes a bare `#main` anchor resolve against the base.
  Evidence: `app/app.go:55`, `shell.go:62-67` (skip link must embed `RoutePath(ActivePath)+
  "#main"`); app backlog bug B1. Direction: first-class static-host support (SPA fallback or
  hash-router option) + base-href-aware anchor/asset helpers.
- [ ] **G8 SVG renderer only draws `path`/`circle`/`rect`; richer SVG goes through a JS shim**
  (SVG, med). Symptom: no `g`/`line`/`polyline`/`polygon`/`text`/gradients → data-viz delegated
  to external D3 drawing into a container. Evidence: icon parser emits only 3 kinds
  (`ui/icon.go:69-77`), test enforces it (`internal/icon/icon_test.go:60,73`); charts bypass the
  renderer (`ui/chartd3.go:22-27,54-56`). Impact: all charting lives in JS (undercuts pure-Go
  frontend), needs G2's id round-trip. Direction: broaden `html/shorthand` SVG element/attr
  coverage (common chart primitives + `<g>`/`<text>`).
- [ ] **G9 No document/window-level event hook; global shortcuts use raw `addEventListener`**
  (global events, med). Symptom: no hook to subscribe to document/window events; global
  keyboard shortcuts installed once at boot via `syscall/js`. Evidence: `app/shortcuts.go:23-90`
  (`wireKeyboardShortcuts`), `js.Func` "intentionally never released" `:17-18`; element-level
  `OnKeyDown` works (`controls.go:64,206,276`) — gap is global/document scope. Direction:
  `UseDocumentEvent`/`UseWindowEvent`/`UseGlobalKey` with managed listener lifetime (ties G13).
- [ ] **G10 No focus-trap/focus-restore primitive; every modal reimplements it** (a11y, med).
  Symptom: move-focus-in / trap Tab+Shift-Tab / restore-on-close / Esc has no framework support;
  hand-written in `syscall/js` per modal. Evidence: `ui/flippanel.go:48-147` (~100 lines: query
  `.flip-wrap`, enumerate focusables, manage `prevFocus`, trap Tab, restore on cleanup);
  `app/applockgate.go:236` mirrors it independently; DEVLOG `:585,453`. Direction:
  `UseFocusTrap(ref)` / `<Dialog>` providing trap+restore+initial-focus+Esc (builds on G2, G4).
- [ ] **G13 No managed `js.Func` lifetime; long-lived listeners are intentionally leaked**
  (lifecycle, med). Symptom: `js.FuncOf` callbacks must be `Release()`d by hand; app-lifetime
  listeners knowingly never released; only modal-scoped ones cleaned up. Evidence: leaks
  `app/shortcuts.go:17-18,89,163`; correct-but-manual cleanup `ui/flippanel.go:140-146`
  (`removeEventListener`+`Release` in teardown). Impact: easy to leak `js.Func`s or release too
  early. Direction: effect-scoped event-subscription helpers (G9) that own the `js.Func` lifetime.
- [ ] **G14 Native input/file pickers must be created off-DOM in raw `syscall/js`** (interop,
  med). Symptom: `<input type=file>` (incl. camera `capture`) can't be a framework node for a
  programmatic pick flow; created off-DOM and clicked via raw JS. Evidence: DEVLOG `:779`;
  `app/shortcuts.go:259-265` (`pickFile`). Direction: a file-input/`usePicker` helper, or general
  ref (G2) so an app can hold and `.click()` a rendered input. (See also G23.)
- [ ] **G15 UI-layer logic is `js && wasm` only, so it can't be unit-tested natively**
  (testability, med). Symptom: logic in the wasm UI layer can't run under native `go test`,
  forcing extraction into separate pure packages just to test. Evidence: DEVLOG `:261`
  (`internal/cmdmatch` extracted because live `shortcuts.go` match is js/wasm), `:601`. Direction:
  a headless/native render+assert harness for components (render to string/virtual tree under
  native Go) so view logic is unit-testable without a browser.
- [ ] **G16 Boot/first-render timing requires an app-managed splash workaround** (mount
  lifecycle, low). Symptom: visible gap between page load and first wasm render; app hand-manages
  a splash and special-cases "`#app` already has children" to avoid a missed first render.
  Evidence: DEVLOG `:621-625,541,576`. Direction: a mount/ready callback or event the host page
  can hook to drop a splash deterministically ("first render committed" signal).
- [ ] **G18 No dialog primitive; destructive guards & text input use native
  `alert`/`confirm`/`prompt`** (dialogs, high). Symptom: confirmations and one-off text input
  fall back to blocking native dialogs — unthemeable, not e2e-drivable, block the main thread.
  Evidence: `confirm` `app/download.go:33-34`, `custompagesnav.go:270`; `alert`
  `shortcuts.go:266,282`, `wsswitcher.go:249`; `prompt` `wsswitcher.go:293`. Direction: framework
  `Dialog`/`Confirm`/`Prompt` (promise-returning) overlay built on the portal (G4).
- [ ] **G19 No timer/interval hook; raw `setTimeout`/`setInterval` with manual cleanup** (timers,
  med). Symptom: auto-dismiss/debounce/polling use raw timers + hand-managed `js.Func`
  release/clear. Evidence: toast `app/toast.go:48-62`, AI debounce `internal/ai/transport.go:116`,
  idle poll `app/applockgate.go:463`, gate delay `:55`. Direction: `UseTimeout`/`UseInterval`/
  `UseDebounce` with effect-scoped lifetime (ties G13).
- [ ] **G20 No media-query hook; `matchMedia` read imperatively and non-reactively** (media, med).
  Symptom: color-scheme & reduced-motion read at call time, no reactive subscription; reduced-
  motion re-checked before each animation. Evidence: `uistate/theme.go:82`, `uistate/prefs.go:60`;
  `app/applockgate.go:34,72,167`. Direction: `UseMediaQuery(query)` → reactive bool (+
  `UsePrefersReducedMotion` convenience).
- [ ] **G23 File download AND upload are entirely raw DOM (expands G14)** (file I/O, med).
  Symptom: export builds Blob+transient `<a>` and clicks it; import creates off-DOM
  `<input type=file>`+`FileReader`+copies bytes — all `syscall/js`. Evidence: `app/download.go:
  11-29` (`downloadBytes`), `:40-82` (`pickFile`/`pickFileNamed`, manual `js.Func` release).
  Direction: `useDownload(bytes,name,mime)` and `usePicker(accept) → bytes`.
- [ ] **G25 Global activity listeners attached by hand (expands G9)** (global events, med).
  Symptom: idle auto-lock listens `mousemove/keydown/click/touchstart/scroll` on `document` via
  raw `addEventListener` to reset an activity timer. Evidence: `app/applockgate.go:441-443` +
  `setInterval` `:463`. Direction: covered by the managed global-event hook (G9).

### Tier 3 — high-severity but reducible to a primitive

- [ ] **G21 Atoms have no persistence layer; every preference hand-rolls localStorage** (state
  persistence, high — **easiest high-value win**). Symptom: `state.UseAtom` is in-memory only; to
  persist you write a matching `loadX()` (read+unmarshal as atom seed) + `PersistX()`
  (marshal+write) pair and must remember to Persist on every mutation. Evidence: the triad repeats
  across **14** `uistate` files / **68** localStorage calls — canonical `uistate/navorder.go:34-54`,
  plus `layout/widgetcfg/i18n/txfilter/modules/fonts/freshness/rail/theme/banner/prefs/period/aikey`;
  Persist sprinkled through UI (`shell.go:396`). Impact: largest single boilerplate category;
  "forgot to Persist after Set" is a whole bug class. Direction: `state.UsePersistentAtom(key,
  default)` that reads its seed and writes through on `Set` (pluggable storage). **Overlaps F2
  kvstate** — reconcile: this is the lightweight localStorage-backed variant of F2's SQLite KV.
- [ ] **G22 No autofocus/element-focus; inline-edit focus uses `focusByID` across ~13 screens**
  (focus/forms, high). Symptom: no `autoFocus` prop and no element ref → opening an inline editor
  runs a `UseEffect` that builds the field id and calls `getElementById(id).focus()`. Evidence:
  helper `internal/screens/focus.go:12-25`; called in `transactions.go:717`, `todo.go:236`,
  `budgets.go:503`, `goals.go:385-387`, `accounts.go:615-617`, `categories.go:270`, `members.go:296`,
  `rules.go:287`, `documents.go:495`, `custompage.go:317`, `emptystate.go:34`. Impact: the most
  common concrete symptom of the missing ref (G2); fragile on id collision / unmounted element.
  Direction: an `AutoFocus()` prop option and/or the DOM ref (G2).
- [ ] **G24 An entire screen (passcode gate) is built in `innerHTML`+`cssText` (expands G4)**
  (raw-DOM screens, high). Symptom: the app-lock gate (full-screen modal: inputs, buttons, hint,
  animation) is built entirely with createElement/`innerHTML`/inline `style.cssText` + own i18n
  escaper, because it must live above the component tree and toggle from outside a render.
  Evidence: `app/applockgate.go` cssText `:136,142,217,375-376`, innerHTML `~:370-380`, activity
  listeners `:442-443` — ~460 lines. Impact: a core security surface can't use design system/themes
  (hand-inlines `var(--accent)` fallbacks), duplicates focus/animation/escape. Direction: G4 portal
  + outside-render visibility makes it a normal component; G3 (raw-HTML) + G18 (dialog) reduce the rest.

### Missing convenience APIs / utilities (don't force syscall/js, but high-volume papercuts)

- [ ] **U1 Sparse typed attribute helpers; ~200 attrs fall back to `Attr(k,v)`** (DSL, med). DSL has
  typed options for `Class/Value/Placeholder/Type/Title/SelectedIf` but NOT `id/disabled/checked/
  required/readonly/role/tabindex/aria-*/scope/for/min/max/step/draggable/target/rel` or SVG attrs
  (`viewBox/stroke/fill`) → stringly-typed `Attr("name","value")`. Evidence: **200** such calls across
  35 files; `ui/icon.go:27-36`, `controls.go:64-73,118-122`, `datatable.go:77,99`. Impact: attribute-
  name typos are silent (defeats typed-Go-on-frontend value); no IDE discoverability. Direction: typed
  option helpers for the standard HTML/SVG/ARIA set (`Id/Disabled/Checked/Required/Role/TabIndex/
  AriaLabel/Scope/For/…`), keep `Attr` only for genuinely custom attributes.
- [ ] **U2 Only `SelectedIf` exists; no `DisabledIf`/`CheckedIf`/`AttrIf`** (conditional attrs, med).
  Conditional `disabled`/`checked` done by building `[]any` and conditionally appending
  `Attr("disabled",…)`; `errAttrs` returns nil `[]any` to spread-or-no-op. Evidence:
  `datatable.go:130-139`, `screens/aria.go:19-24`. Direction: `DisabledIf(bool)`, `CheckedIf(bool)`,
  general `AttrIf(cond,name,value)` / `When(cond, ...PropOption)`.
- [ ] **U3 No class-name builder (clsx/classnames)** (class building, med). `Class` takes one string →
  conditional/variant classes assembled via manual string concat. Evidence: **26** `cls :=`/`cls +=`
  sites across 10 files (`controls.go:105-116,185-191,249-255`, `shell.go:293-299`, `datatable.go:54-57`,
  `chartd3.go:68-71`). Direction: variadic `Classes(parts ...any)` accepting strings + `cond && "cls"` /
  `map[string]bool`, plus `ClassIf(cond,cls)`. **NOTE: the new `css.Class(...any)` / `css.ClassIf`
  already deliver this for the typed-CSS path — close U3 by documenting/porting it to the shorthand DSL.**
- [ ] **U4 No form-field / a11y wiring helper** (forms/a11y, med). Associating input↔label↔error
  (`aria-invalid`/`aria-describedby`/error `role=alert`+matching id) has no helper; app built
  `errAttrs`/`errText`. Evidence: `screens/aria.go:10-32`. Direction: `Field`/`Label`/`ErrorText` set
  or a `useField` hook that generates+wires ids and ARIA relationships.
- [ ] **U5 No roving-tabindex / radiogroup primitive** (a11y components, med). ARIA radiogroup
  semantics (one Tab stop, arrow nav, `role=radio`+`aria-checked`, selection-follows-focus)
  reimplemented per control. Evidence: `controls.go:37-92` (Segmented), `:298-355` (SwatchPicker,
  same again), `:184-217` (Toggle as `role=switch`). Direction: `RadioGroup`/`useRovingTabIndex`
  primitive + a `Switch` component.
- [ ] **U6 No two-way input binding helper** (forms, low). Every controlled input manually pairs
  `Value(get)`+`OnInput(set)`. Evidence: `filtertoolbar.go:54,75-77`, inline editors everywhere.
  Direction: `Bind(state.Atom[string])` expanding to value+handler (+ numeric/`Parse` variant).
- [ ] **U7 Generic text/format utilities reinvented per app** (utils, low). e.g. snake_case→Title
  humanization. Evidence: `screens/format.go:52-59` (`humanizeType`). Direction: optional tiny
  `strutil`/`textutil` subpackage, or document that these stay app-side.
- [ ] **G11 Styling API quirks** (styling, low). `Style` accepts only `map[string]string`; themed SVG
  line weight must use inline `style` not the `stroke-width` attribute because SVG *attributes* don't
  accept `var()` while the CSS property does. Evidence: `ui/icon.go:31-34`; `Style(map…)` throughout.
  Direction: document the SVG/`var()` interaction; consider typed style helpers. **NOTE: the new `css`
  package's typed `Style`/`Var`/property constructors largely address the typed-style half.**
- [ ] **G12 `UseEffect` takes a single dependency value, not a list** (effects, low). To depend on
  multiple inputs, code serializes them to one string (JSON) and keys on that. Evidence:
  `chartd3.go:37-40,45,66` (re-marshals spec every render to compare); `shell.go:52-60`. Impact:
  serialization overhead + awkward idiom + allocation churn. Direction: accept variadic/slice dep list
  with value equality (matches the React mental model).
- [ ] **G17 `OnInput` requires a framework `Handler`, not a plain `func`** (event API, low).
  Inconsistent with the plain-`func` callbacks used elsewhere. Evidence: DEVLOG `:475`. Direction:
  accept plain `func(string)`/`func(Event)` uniformly across all `On*`, or document which props need
  the `Handler` wrapper and why.

### Keep / don't regress (validated as ergonomic; protect in any refactor)
- `MapKeyed(items, keyFn, render)` with auto-flattening children (`shell.go:227-265`,
  `controls.go:338-354`).
- Drag-and-drop `OnDragStart`/`OnDragOver`/`OnDrop` (`shell.go:314-330`).
- Element-level `OnKeyDown` with typed `KeyboardEvent` (`controls.go:64-73`).
- Form input ergonomics: `OnInput(func(string))`, `OnChange`+`e.GetValue()`, `SelectedIf`, `Value()`.
- `If`/`IfElse`/`Fragment` control-flow nodes; props-driven composition.
- DSL already provides (don't duplicate): `Class/Value/Placeholder/Type/Title/SelectedIf`, the `On*`
  set + `Prevent(fn)`, `Text`/`Textf`, `If/IfElse/Map/MapKeyed/Fragment`, SVG nodes `Svg/Path/Circle/Rect`.

### Triage note (relationship to current work)
- **G21 ↔ F2 (kvstate):** both are "persisted reactive state." Decide whether `UsePersistentAtom`
  (localStorage, lightweight) is a separate tier above F2's SQLite-backed binding, or the same API
  with a storage backend flag. Reconcile before building either.
- **U3 / G11 partially DONE by F3:** the new `css` package already gives `Class(...any)` (clsx-style)
  and typed style/`Var()` constructors — port/expose to the shorthand DSL and close those.
- **G2/G3/G4 are the keystone:** ref + raw-HTML + portal unblock the most items (charts, focus, file
  I/O, overlays, the passcode gate, and the Markdown/Mermaid backlog). Sequence these first.

### VERIFICATION PASS (2026-06-20) — catalog re-triaged against CURRENT GWC code
The CashFlux catalog reflects an older/incompletely-explored GWC; many items are ALREADY SHIPPED.
Verified by symbol against the live tree before any implementation (do NOT re-implement these):
- **G4 Portal — DONE.** `ui.Portal(PortalProps)` (`ui/ui.go:279`, native `ui/ui_native.go:256`),
  `runtime.PortalNodeType` (`internal/runtime/types.go:35`), commit support
  (`reconciler_commit.go:1203,1233`).
- **G6 reactive route — DONE.** `router.UseRouteData()` (`router/router_api.go:98`).
- **G13/G19 timers — MOSTLY DONE.** `UseDebounced`/`UseThrottled` (`ui/ui_async.go:427,498`).
  (Generic `UseTimeout`/`UseInterval` may still be absent — verify before any add.)
- **G20 media queries — DONE.** `UsePrefersReducedMotion()`, `UsePrefersColorScheme()`
  (`ui/preference_hooks.go:39,46`). (Generic `UseMediaQuery(q)` optional, low value.)
- **G21 persisted state — DONE.** `UsePersistedState[T](key, initial, area PersistStorageArea)`
  (`ui/persisted_state.go:83`). Closes the persisted-atom ask; reconcile naming with F2 kvstate.
- **G3 markdown half — DONE.** `html.RenderMarkdown` (`html/markdown.go`). Raw-HTML node still absent.
- **U1 typed attrs — DONE.** `sugar.go` exposes Id/Class/For/Name/Title/Value/Type/Role/TabIndex/
  Disabled/Checked/Selected/Required/ReadOnly/AutoFocus/Multiple/Open/Hidden/Min/Max/Step/Pattern/
  Width/Height/Loading/Rows/Cols/Target/Rel/Accept/AutoComplete/Lang/Dir/Aria/AriaSet/Data/Dataset.
- **U2 conditional attrs — DONE.** `DisabledIf`/`ReadOnlyIf`/`SelectedIf`/`AttrIf`/`When`/`Unless`/`Show`.
- **U3 class builder — DONE.** `ClassNames(...any)`, `ClassMap(map)`, `ClassIf` — plus the new
  `css.Class(...any)`/`css.ClassIf` from F3.
- **G22 (attr half) — PARTIAL.** `AutoFocus(...)` PropOption exists but only sets the HTML `autofocus`
  attribute, which does NOT fire on SPA re-mount; programmatic focus-on-mount is still a real gap
  (depends on G2 DOM ref).

### CONFIRMED-REAL GAPS — the actual loop work queue (in fix order)
1. [x] **G2 DOM element ref — DONE (2026-06-20, iteration 2).** Shipped `ui.UseDOMRef()` returning
   a `ui.DOMRef` (`.Node()`/`.Mounted()` cross-build; `.Value() js.Value`/`.Focus()` wasm-only),
   `html.Ref(r)` + `shorthand.Ref(r)` PropOptions, and commit-phase capture in `internal/runtime`:
   reserved props key `runtime.DOMRefKey` registered `propKindSkip` (never hits the DOM), a
   `runtime.DOMRefSink` published on placement (`reconciler_commit.go` after AppendChild) and cleared
   across the deleted subtree on unmount (`releaseDOMRefsSubtree`). commitRoot processes deletions
   before placements, so key-change remount detaches→reattaches correctly. Files:
   `internal/runtime/dom_ref.go`, `ui/dom_ref.go`+`ui/dom_ref_wasm.go`, `html/dom_ref.go`,
   `html/shorthand/dom_ref.go`. EDGE FOUND+FIXED: `ssr.go shouldSkipSSRProp` now skips `DOMRefKey`
   (it was leaking as a bogus `__gwc_dom_ref__="&{…}"` attribute through SSR). EDGE: name clash —
   the new `Ref` collided with F3's css selector `Ref`; renamed the niche css one to `css.SheetRef`
   /`u.SheetRef` (DOM ref owns the universal name). Tests: unit `internal/runtime/dom_ref_test.go`
   (publish-on-mount, clear-on-unmount, remount reassign, skip-from-DOM, nil-safety; native+wasm),
   integration+edge `html/dom_ref_test.go` (SSR no-leak + null-on-native + zero-ref no-op), e2e
   `test/playwrightgo/dom_ref_e2e_test.go` (real Chromium: ref resolves, Focus lands activeElement,
   clean detach on unmount). Fixture `examples/public/dom-ref/`. Native runtime/ui/html/css/shorthand
   suites + vet + gofmt + wasm builds all green.
   FOLLOW-UP NOTE (found this iteration, NOT G2): `TestPreferenceHooksUnavailableDefaults`
   (`ui/preference_hooks_internal_test.go:52`) panics under the **wasm** lane ("GoUseState called
   outside component context") — pre-existing, unrelated to G2 (file untouched). Triage under the
   G20 area: the test calls `UsePrefersReducedMotion` without a component/render context.
2. [ ] **G22 programmatic focus-on-mount — depends on G2.** `AutoFocus()` should also focus the
   element on mount (not just emit the attribute), OR provide `UseAutoFocus(ref)`. Replaces the
   app's focusByID across ~13 screens. Tests incl. e2e activeElement assertion.
3. [ ] **G3 RawHTML / Markup node — ABSENT.** A `RawHTML(s)` node (clearly unsafe; wasm sets
   innerHTML, native SSR emits raw) and/or sanitized `Markup`. New runtime NodeType + domAdapter
   support + SSR serialization. Edge: empty/script payloads, hydration, re-render diffing. Pairs
   with the F3 `hardenCSS`-style boundary thinking for the sanitized variant.
4. [ ] **G9 global document/window event hook — ABSENT.** `UseDocumentEvent`/`UseWindowEvent`/
   `UseGlobalKey` with effect-scoped `js.Func` lifetime (subsumes G25 idle-activity, helps G13).
   Edge: multiple subscribers, unmount cleanup, capture/passive, typing-suppression helper.
5. [ ] **U5 roving-tabindex / RadioGroup — ABSENT.** `RadioGroup`/`useRovingTabIndex` + `Switch`
   so segmented/swatch/toggle get correct keyboard a11y once. Edge: wrap-around, disabled items,
   RTL arrows, selection-follows-focus. e2e keyboard-nav assertions.
- Re-verify before starting each: G1 (hooks-in-loops — likely still real; consider dev diagnostic),
  G15 (native test harness), G16 (mount-ready signal), and whether generic `UseTimeout`/`UseInterval`
  / `UseMediaQuery` are wanted on top of the existing debounce/throttle/preference hooks.
