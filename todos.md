# Feature Backlog

Gap analysis against React/Next/Solid ecosystems (2026-06-11). Ordered by
impact; exactly three active items carry the next-work marker.

## High impact

- [ ] [next] **Selective / progressive hydration (islands)** - hydration is
  whole-tree with per-subtree mismatch fallback. Add a first-class API for
  hydrate-on-visible / hydrate-on-interaction islands, plus budget-driven
  validation around the static-islands example seed. Attacks the measured
  wasm-startup gap vs React directly.
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
- [ ] [next] **Route-level code splitting** - Go wasm ships as one binary.
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
- [x] **Go-source debugging workflow** - `gwc build` / `gwc release` now accept
  a first-class `debug` / `source-debug` profile that records `gcflags`,
  builds Go `js/wasm` artifacts with untrimmed paths and `-gcflags=all=-N -l`,
  exposes `gcflags` through `pwa.WasmReleaseFlags`, documents the browser
  stack-correlation workflow, and clearly states the current Go-toolchain
  boundary around source maps and DWARF sections.

## Lower impact / ecosystem

- [ ] **Headless a11y component kit** - package the overlay/focus primitives
  into a menu/combobox/listbox/datepicker/table set instead of examples.
- [~] **Animation primitives** - transition hooks exist; add spring physics,
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
  not a jump. Remaining: the gesture layer (drag/pan/pinch).
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
- [ ] [next] **Strict mode** - strict hydration exists, but no general dev-time
  strict mode: double-invoke renders to flush impure components, warn on
  setState-during-render, detect asymmetric effect cleanups.
- [ ] **Deterministic replay** - capture/replay of an update stream for
  reproducing production bugs; profiling already records events
  internally, but nothing exports or replays them.
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
- [ ] **Long-session memory hygiene** - no GC-pressure monitoring or leak
  diagnostics for week-long dashboard sessions, which is where wasm apps
  fail quietly.

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
- [~] **Offline mutation replay hardening** - `fetch.MutationQueue` exists;
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
  confirm 2 queued writes survive into IndexedDB before reload). Honestly
  not drivable from this example without a dedicated fixture: a
  failing-executor control (the example only has a resolving conflict
  handler) and a full offline reload-and-reboot (needs the cache warmed
  first so the wasm can re-boot offline) - both noted in the test.
- [~] **Installability flow e2e** - `pwa.ObserveInstallability` exists but
  has no browser test.
  Test for: beforeinstallprompt capture, prompt() round trip, and state
  cleanup on dismissal (chromium supports faking the event).
  Done (2026-06-12, real browser): `TestPWAInstallabilityFlow` boots the
  installability example, asserts the manifest-validity / install-reasons
  / service-worker-lifecycle diagnostics render (not blank), that "Refresh
  installability" reflects `ObserveInstallability()`, and that "Prompt
  install" follows the documented GRACEFUL-REFUSAL path with a structured
  "install prompt is not currently available" message (headless Chromium
  does not fire a real beforeinstallprompt). Remaining: faking a real
  beforeinstallprompt + prompt() round-trip needs CDP event injection
  beyond standard Playwright.

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
- [ ] **Snapshot schema versioning** (promotes the enterprise contracts
  item) - add a version field + migration hook to
  `state.SaveSnapshot`/`SavePersistentSnapshot` payloads.
  Test for: v(N) snapshot restores through a registered v(N-1)->v(N)
  migration; unknown future version is rejected loudly, not silently
  dropped; missing-version legacy payloads still restore (compat path);
  partial migration failure restores nothing (atomicity).
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
- [~] **Cross-root eventing guidance + test** - components in different GWC
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
  subscription). No uncaught/panic console errors. Remaining: the
  plugin-host-panel cross-tree round-trip and an explicit js.Func
  cleanup-on-unmount assertion + a short guidance doc.
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

- [~] **Message extraction + locale completeness tooling** - nothing scans
  code for T(namespace, key) usage to scaffold catalogs or diff locales;
  incomplete translations ship silently (pairs with the enterprise
  missing-translation enforcement item).
  Test for: extraction finds every T() call across build tags (wasm and
  native files); diff reports keys missing per locale and stale keys no
  longer referenced; a gwc lane fails CI when a non-default locale is
  incomplete; dynamic/computed keys are reported as unverifiable rather
  than silently skipped.
  Partial (2026-06-11): new i18n/extract package - `ExtractFromSource`/
  `ExtractFromDir` (go/ast; scans build-tagged _wasm.go AND _native.go,
  dedups), records non-literal ns/key as `DynamicUsage` (unverifiable,
  not skipped), `DiffLocale` (Missing incl. empty-string + Stale),
  `IsComplete`, `CheckLocales` (all-locale gate, sorted, false if any
  incomplete). 17 tests green. Remaining: wire `CheckLocales` into a
  `gwc` lane that fails CI on an incomplete non-default locale.
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

- [ ] **CSP nonce threading** - neither RenderToString nor the new
  RenderToStream emits or threads script nonces, and there is no
  documented wasm-unsafe-eval guidance; strict CSPs block deployment.
  Test for: a nonce supplied per request appears on every emitted
  script tag across both SSR paths including out-of-order streamed
  chunks; hydration succeeds under a strict CSP (no inline-eval
  violations in the browser console); generated boot shells accept an
  injected nonce; a CSP-violation fixture page proves the test setup
  actually enforces the policy.
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
- [~] **Root-module security scanning + SBOM + SECURITY.md** - gosec and
  govulncheck run only for the bridge submodule; releases ship no SBOM;
  no security disclosure policy exists.
  Test for: scanners run green on the root module in CI with a
  documented suppression file; release workflow attaches a CycloneDX
  SBOM whose package list matches go.mod; SECURITY.md present with a
  disclosure contact.
  Partial (2026-06-11): SECURITY.md added (private GitHub advisory
  disclosure, scope, supported versions, defensive posture). govulncheck
  wired into release.yml as an informational (continue-on-error) step on
  the root module, AND its findings acted on: golang.org/x/net bumped to
  v0.55.0 cleared 5 of 7 findings (see the HTML-sanitizer entry); the 2
  remaining are Go-stdlib toolchain advisories. SBOM done (2026-06-12):
  new tools/sbom package generates a CycloneDX 1.5 SBOM from the resolved
  module graph (`go list -m -json all`, 171 components with golang purls),
  with a CLI (cmd/sbom) wired into release.yml that writes
  bin/sbom.cyclonedx.json. Tests: fixture decode (main + version-less
  excluded, sorted, purl/type) + a real-graph check (contains x/net,
  >=50 components, valid JSON). Remaining: promote govulncheck to blocking
  with a documented suppression file + a CI Go-toolchain patch bump.
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

- [ ] **PII redaction hooks for telemetry** - crash reports, structured
  logs, and devtools snapshots carry paths, props, and state values
  with no redaction policy (support bundles have a sanitize step;
  console crash reports and future transports do not).
  Test for: a registered redaction policy scrubs configured fields from
  crash-report payloads, log attributes, and devtools snapshots before
  emission; redaction failures fail closed (drop the field, keep the
  event); policy application is covered for both the console path and
  the OnReport hook path.
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
- [ ] **Version-skew refresh flow** - wire formats are versioned now, but
  detection is not acted on: stale cached wasm against a redeployed
  server should trigger a controlled refresh, not an error.
  Test for: a sidecar/wasm version mismatch triggers exactly one forced
  reload with cache bypass (no reload loop on persistent mismatch -
  loop guard verified); user state is snapshotted before the reload and
  restored after when versions allow migration.
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

- [ ] **Canary / gradual wasm rollout** - no mechanism serves two wasm
  versions side-by-side with percentage routing and instant rollback.
  Test for: deterministic cohort assignment (same client stays on its
  version across reloads); rollback flips 100% within one cache TTL;
  both versions report their build id through artifact metadata so
  crash reports distinguish cohorts.
- [~] **Perf-budget CI gate** - route startup budgets exist in profiling
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
  Remaining: a `gwc` lane wrapper + an explicit `-update-budgets` ratchet
  command (the JSON is hand-maintained for now).
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

- [~] **Shadow-DOM style isolation for exported custom elements** - no
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
  flagged informational, not a defect.) Remaining: portals-across-boundary
  and focus-trap/announcer-inside-shadow assertions.
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

- [~] **Auto-run doctor on first build/dev failure** - `gwc doctor` is
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
  warn-only=silent). Remaining: wire the same `diagnoseEnvironmentOnFailure`
  helper into `gwc build` (runBuild lives in release_build.go, currently
  edited by the concurrent session - deferred to avoid a collision).
- [~] **Browser build-status indicator in `gwc dev`** - the 304ms rebuild
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
  a release build). Remaining: a dedicated badge-transition browser e2e -
  belongs in the dev-loop browser harness (tools/gwc/
  dev_loop_browser_e2e_test.go) that drives the real gwc dev server.
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
- [~] **Disciplined CHANGELOG + release notes** - CHANGELOG.md exists but is
  not tied to the release flow; v-tags ship auto-generated notes only.
  Test for: the release workflow fails if CHANGELOG has no entry for the
  tag being cut; entries follow Keep-a-Changelog sections; the docs site
  surfaces the latest release notes.
  Partial (2026-06-12): new tools/changelogcheck package - `HasEntry`
  (whole-token version match, v-normalized, bracket/date tolerant),
  `LatestEntry` (first section for docs-site surfacing), `CheckFile`, +
  a CLI (cmd/changelogcheck) that exits non-zero on a missing entry (7
  tests). Added the Keep-a-Changelog `## [Unreleased]` block to the top of
  CHANGELOG.md. Wired an informational (continue-on-error) CHANGELOG-entry
  step into release.yml. Remaining: flip the gate to blocking once
  releases adopt version headers (the historical log is date-based), and
  surface LatestEntry on the docs site.
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

- [ ] **`--json` on every endpoint + a stable result envelope** - `--json` is
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
- [ ] **`gwc mcp` - serve the same surface over MCP (CLI and/or MCP, dev
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

### Representations (read surfaces - the agent's world-model)

- [ ] **`gwc model --json` - component manifest** - go/packages static
  analysis emitting every component, its props struct, hooks used, atoms
  read/written, events emitted, and file:line, with stable IDs and
  deterministic ordering. The map an agent navigates instead of grepping. Pure
  static; complements the existing `gwc inspect`.
  Test for: a fixture app's manifest lists every component and its hook/atom
  usage with no false negatives; output is byte-stable across runs (sorted,
  no map nondeterminism); scans both `_wasm.go` and `_native.go` build tags
  (mirror i18n/extract); a renamed prop changes the manifest deterministically.
- [ ] **`gwc render <component> --props=<json> --json` - headless SSR oracle**
  - render a component to its DOM tree via `ui.RenderToString`, returning the
  serialized tree plus any render diagnostics, no browser. The millisecond
  inner loop: edit -> render -> assert. Highest-leverage single tool.
  Test for: a known component renders to the expected tree shape; a panicking
  component returns a contained structured error (crash containment), not a
  process crash; invalid props JSON fails with an actionable message; output
  is deterministic for deterministic components.
- [ ] **`gwc probe <example> --json` - browser oracle** - drive the existing
  playwright harness for one example and return DOM + console + axe a11y +
  perf-budget status as one blob. The "did my change actually work in a
  browser" check, reusing the test/playwrightgo harness helpers.
  Test for: a clean example reports ok with zero serious/critical a11y and
  within-budget perf; an example with a seeded console error / a11y violation
  is reported, not swallowed; webkit flakiness degrades to a documented skip
  (mirror the cross-browser conformance policy), never a false pass.
- [ ] **Hydration-diff + commit-trace representations** - structured SSR-vs-
  client-first-render delta (the framework's classic silent bug) and an NDJSON
  commit log (which atom/state changed -> which components committed, with
  counts) exposed via `--json`. Detects hydration mismatches and needless
  re-renders an agent otherwise can't see.
  Test for: an intentional hydration mismatch fixture is reported with the
  offending node path; a clean app reports zero mismatches; the commit trace
  attributes a re-render to the state/atom write that caused it; a render
  storm (N writes -> N commits) is visible as such.
- [ ] **`gwc inspect --impact <symbol> --json` - blast-radius query (Plan
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

### Tools (act surfaces - verbs designed for agents)
- [ ] **`gwc mutate <op> --json` - structured edit / codemod (Implement
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

- [ ] **`gwc scaffold <kind> --json --no-input` - non-interactive generation**
  - component/hook/example scaffolding with no TTY prompts, emitting the JSON
  envelope (files written + next steps). Closes the long-standing
  no-non-interactive-scaffold-path gap so an agent can create surfaces.
  Test for: each kind generates files that build native+wasm and pass the
  conventions lint (parse-prefix, godoc-first-word) with zero prompts; an
  existing-file collision fails safely (no clobber) with a structured error;
  `--dry-run` lists planned files without writing.
- [ ] **`gwc check --json` - agent-shaped diagnostics** - typecheck +
  lint-zero + conventions as structured diagnostics with machine-applicable
  fix suggestions (not human prose), so an agent can apply fixes and re-check.
  Aggregates existing `gwc lint`/vet/build into one agent-consumable result.
  Test for: a file with a known convention violation yields a diagnostic with
  code + file:line + suggested edit; a clean tree yields an empty diagnostic
  set with ok=true; suggested edits, when applied, make the diagnostic
  disappear (round-trip).
- [ ] **`gwc explain <errorcode|capability> --json`** - surface
  `docs/errorcodes` and `docs/capabilities` as a queryable endpoint so an
  agent resolves a code to cause/fix and checks capability availability
  without reading docs prose.
  Test for: every code in docs/errorcodes resolves; an unknown code returns a
  structured not-found (not empty success); capability queries report the
  native/wasm availability matrix.

- [ ] **`--help` everywhere - self-documenting CLI for humans AND agents** -
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
- [ ] **`gwc search <query> --json` - semantic API search** - an agent (or
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

### Agent-native dev loop

- [ ] **`gwc dev --agent` - structured event stream** - emit dev-loop events
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

### Closing the loop (acceptance + observe)

- [ ] **`gwc verify --agent` - single acceptance gate / definition-of-done
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
- [ ] **`gwc observe --agent` - close the loop from runtime back to Plan
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

### Toolchain hygiene & analysis (untracked gaps - 2026-06-12)

Commands the toolchain lacks today (verified absent from the `gwc` dispatch);
distinct from the agent-surface items above. Note: coverage already exists as a
test lane (`gwc test -lane coverage`), so it is NOT listed here.

- [ ] **`gwc fmt` - convention-aware formatter (the missing half of `gwc
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
- [ ] **`gwc clean` - remove build artifacts and caches** - no command wipes
  `bin/`, generated wasm/`wasm_exec.js` outputs, tailwind build output, release
  packages, and tool caches. Agents accumulate `./bin/*.test`/`*.exe` (the
  workflow rules route ad-hoc binaries there) with no sweep.
  Test for: clean removes the known artifact set and leaves source/tracked
  files untouched; `--dry-run` lists what would be removed without deleting;
  selective targets (`-artifacts`, `-cache`, `-bin`) work; cleaning an
  already-clean tree is a no-op success; never deletes outside the repo root.
- [ ] **`gwc test --watch` / `gwc watch` - re-run a lane on change (TDD inner
  loop)** - `gwc dev` livereloads the APP but nothing re-runs the relevant
  TEST lane on save. Provide a watch that re-runs the selected lane(s) on file
  change with debounced rebuilds, surfacing pass/fail. Pairs with the planned
  `gwc dev --agent` NDJSON stream (emit `test-passed`/`test-failed` events).
  Test for: editing a source file triggers exactly one debounced re-run (rapid
  saves coalesce, not N runs); a failing test is reported and a subsequent fix
  flips it green without restart; the watched lane set is configurable; the
  watcher exits cleanly on signal and leaks no processes (mirror the
  zombie-server hygiene the http tests needed).
- [ ] **`gwc size` - wasm bundle size attribution** - `gwc wasm measure` gives
  the size NUMBER but not the BREAKDOWN. Attribute wasm bytes to packages /
  symbols (parse the Go wasm section / `go tool nm` size data) so "why is the
  binary 6MB" is answerable. Directly unblocks the open binary-size /
  route-splitting / server-component items, which currently fly blind.
  Test for: the report attributes bytes to packages and sums to ~the artifact
  size (within section overhead); the largest contributors are ranked; a JSON
  mode feeds a budget/ratchet; building the same commit twice yields the same
  attribution (deterministic); a TinyGo-profile artifact is handled or clearly
  reported as unsupported, not silently wrong.
- [ ] **`gwc docs` - generate / serve project API docs** - no godoc-style
  surface for the project's own packages; the planned `gwc explain` covers
  errorcodes/capabilities only, not the API. Generate a browsable/JSON API
  index (exported symbols + GoDoc, per package) and optionally serve it. Pairs
  with the planned `gwc model` + `gwc search` (shared symbol index).
  Test for: every exported symbol in a fixture package appears with its GoDoc
  first sentence; the index covers `_wasm.go` AND `_native.go` symbols; JSON
  output validates against a schema and is deterministic; an undocumented
  exported symbol is reported (doc-coverage gate), not silently omitted.
- [ ] **`gwc deadcode` - unused component / export detection** - nothing finds
  exported symbols or components that nothing references, so refactors guess at
  what is safe to delete. Falls out of the `gwc model` graph + `inspect
  --impact` work (a symbol with zero dependents across app + tests + docs).
  Test for: a deliberately unreferenced component/export is reported; a symbol
  referenced only from a test is NOT flagged as dead (or is flagged distinctly
  as test-only); reflection/registry-based references (router registration,
  plugin capability tables) are accounted for or reported as unverifiable
  rather than falsely dead; output is deterministic.
- [ ] **`gwc deps` / `gwc update` - dependency + framework version report and
  bump** - `gwc upgrade` only migrates the `gwc-start.json` schema; nothing
  reports or bumps `go.mod` dependencies or the framework version. Provide a
  report (current vs latest, with the known-vuln overlay from govulncheck) and
  a guarded bump that re-runs build+verify before keeping the change. Pairs
  with the SBOM / root-security items.
  Test for: the report lists outdated modules and flags any with govulncheck
  advisories; a bump that breaks the build is rolled back (the tree is left
  building); `--dry-run` reports the planned bumps without writing go.mod;
  the framework's own version is distinguished from third-party deps.

## Test correctness gaps (2026-06-12 review) - add tests

Specific weak/missing CORRECTNESS tests found by reviewing source vs `_test.go`
in the public framework packages (verified against the code, not coverage-for-
coverage's-sake). Each is a happy-path-only or absent assertion where a real
bug would slip through. Scope excluded tools/gwc (parallel-active) and the
ai-chat-wizard example. Pure-Go logic only - host-testable.

- [ ] **anim: easing interior values + settle/interpolate semantics** - the
  easing tests only assert clamping (t<0 -> 0, t>1 -> 1), so a sign error in
  the interior expansion passes; `Interpolate` is only ever tested with
  `Linear`; `IsSettled` is only exercised as a loop-exit (its `&&` epsilon
  logic is never pinned at the boundary).
  Test for: `EaseOutCubic(0.5)==0.875`, `EaseInCubic(0.5)==0.125`,
  `EaseInOutCubic(0.25)==0.0625`; `Interpolate(0,10,0.5,EaseInQuad)==2.5` (not
  5.0); a spring with `position=1.0001,target=1.0,velocity=0.0001,eps=0.001`
  reports `IsSettled()==false`, false again with zero velocity but non-zero
  position error, true only when both are within epsilon.
- [ ] **events: concurrency window + unsubscribe idempotency** - the
  concurrency test only uses pre-registered subscribers (never races
  `Subscribe` against `Publish`), and no test calls the unsubscribe closure
  twice.
  Test for: race `Subscribe("t",h)` against `Publish("t",1)` under `-race` with
  no data race and no dropped delivery; `unsub();unsub()` does not panic and the
  subscriber count returns to zero. (Note: `-race` is unavailable on the
  windows/arm64 dev host - gate or run this lane where the race detector exists.)
- [ ] **state: snapshot edge values (NaN/Inf, nil, select)** - `normalizeSnapshot`
  routes floats by `math.Trunc(v)==v`; NaN/Inf behavior is unspecified by any
  test (NaN must stay `float64`, never become `int`); `Snapshot.Select` is only
  tested with string/bool, never a nil value.
  Test for: `normalizeSnapshot(math.NaN())` and `normalizeSnapshot(math.Inf(1))`
  return `float64` without panic; `Snapshot{"k":nil}.Select("k")` returns
  `len==1` with `["k"]==nil` and `ApplySnapshot` of it does not panic.
- [ ] **flags: empty-value fallback, bucket stability, pre-cancelled Poll** -
  `GetValue` returns the fallback when a present+enabled flag has `Value:""`
  (untested, and a latent trap - document the intent); `getBucket`'s FNV32a
  output is never pinned, so a hash change would silently re-assign A/B cohorts;
  `RemoteProvider.Poll` is never given an already-cancelled context.
  Test for: enabled flag with `Value:""` yields the fallback (asserted +
  documented); `getBucket("pricing","v1","customer-123",100)` equals a pinned
  integer; `Poll` with a pre-cancelled ctx returns `context.Canceled` and calls
  `Refresh` at most once.
- [ ] **i18n: repeated placeholder, negative plural counts, BCP-47 path
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
- [ ] **sanitize: unicode control-char URL bypass + empty-input contract** -
  `stripControlChars` only removes bytes `> 0x20` survivors (it strips `<=0x20`),
  so zero-width/line-separator code points (U+200B, U+2028, U+2029) pass through
  and could disguise a scheme; `Sanitize("")`/`"   "` behavior is unspecified.
  Test for: `Sanitize` of `<a href="java​script:alert(1)">x</a>` emits no
  `href`; `Sanitize("")==""`, `Sanitize("   ")==""`, `Sanitize("hello")=="hello"`.
- [ ] **virtualization: out-of-range scroll + empty-list clamp** -
  `ComputeViewportState` is never given a `scrollTop` beyond
  `TotalItems*RowHeight`; `clampRange` is never given `total==0` with a stale
  non-zero range. Both must never produce `Start>End` (downstream render panics).
  Test for: `ComputeViewportState({TotalItems:10,RowHeight:20},5000,100)` yields
  `Visible==Range{10,10}`, `Rendered=={10,10}`, `Visible.Len()==0`;
  `clampRange(Range{2,8},0)==Range{0,0}`.
- [ ] **fetch: resilience zero-value foot-guns + nil optimistic update** -
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
- [ ] **ui hooks: deeper-read pass for effect/reducer/persisted-state edge
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
