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

- [ ] **Fetch circuit breakers / retry policy** - the cache and mutation
  queue exist, but there's no declarative retry/backoff/circuit-breaker
  policy for flaky enterprise networks.
- [ ] **Long-session memory hygiene** - no GC-pressure monitoring or leak
  diagnostics for week-long dashboard sessions, which is where wasm apps
  fail quietly.

## Capability reviews (2026-06-11) - build items + what to test

### PWA / offline mode

- [ ] **Generated service worker + manifest for the docs site** - sitegen
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
- [ ] **Offline mutation replay hardening** - `fetch.MutationQueue` exists;
  prove it under adversarial conditions.
  Test for: mutations enqueued offline replay exactly once after
  reconnect (no dupes on rapid online/offline flaps); replay order
  preserved; executor failure leaves the entry queued, not dropped;
  queue survives a page reload mid-outage (IndexedDB persistence);
  `pwa.InspectDiagnostics` queue counts match reality.
- [ ] **Installability flow e2e** - `pwa.ObserveInstallability` exists but
  has no browser test.
  Test for: beforeinstallprompt capture, prompt() round trip, and state
  cleanup on dismissal (chromium supports faking the event).

### Session / long-term web storage

- [ ] **`UsePersistedState[T](key, initial, area)` hook** - bind a UseState
  to localStorage/sessionStorage/IndexedDB with write-through and
  cross-tab change subscription.
  Test for: state survives unmount/remount and full reload; storage
  `storage`-event from a second tab updates the first tab's component
  (two playwright pages, one context); JSON round-trip of non-trivial T
  (structs, slices); corrupted stored value falls back to initial
  instead of panicking (crash containment must catch decode panics);
  quota-exceeded write surfaces an error state, does not wedge renders.
- [ ] **`RequestPersistentStorage` helper** - wrap
  `navigator.storage.persist()`; diagnostics already read the flag.
  Test for: persisted flag flips after grant (headless chromium grants
  silently); denial path returns false without error; native build
  returns the unavailable stub.
- [ ] **Snapshot schema versioning** (promotes the enterprise contracts
  item) - add a version field + migration hook to
  `state.SaveSnapshot`/`SavePersistentSnapshot` payloads.
  Test for: v(N) snapshot restores through a registered v(N-1)->v(N)
  migration; unknown future version is rejected loudly, not silently
  dropped; missing-version legacy payloads still restore (compat path);
  partial migration failure restores nothing (atomicity).
- [ ] **Typed cookie helper** - first-class document.cookie access for
  session-adjacent apps (read/write/expire, SameSite/Secure attrs).
  Test for: attribute round-trips, expiry honored, and unavailability
  on native builds.

### Cross-component eventing

- [ ] **`events.UseTopic[T](topic)` fan-out bus** - typed in-app pub/sub
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
- [ ] **Cross-root eventing guidance + test** - components in different GWC
  roots / exported custom elements communicating via
  `interop.GetDocumentEvents()` CustomEvents.
  Test for: typed detail payload round-trip through Dispatch/Subscribe;
  subscription cleanup releases the underlying js.Func (no released-
  function warnings); events cross from a GWC tree into a plugin-host
  panel and back.
- [ ] **Cross-tab eventing soak** - `SubscribeDecodedCrossTab[T]` works in
  the example; pin it with a test.
  Test for: typed envelope round-trip between two pages in one browser
  context; decode error of a malformed envelope surfaces via the error
  callback, not a contained panic; channel close mid-flight does not
  crash either tab.

### Accessibility (visually impaired)

- [ ] **Automated a11y audit in the browser suites** - the primitives
  (UseAnnouncer, UseFocusTrap, UseCompositeNavigation, AccessibleOverlay)
  exist but no axe-core-style audit runs in CI, so contrast/label/name
  regressions ship silently.
  Test for: every public example page passes an automated audit at the
  serious/critical level; the audit runs inside the existing playwright
  lanes; intentional violations in a fixture page are detected (the
  audit itself is tested, not just wired); docs-site routes included.
- [ ] **Docs-site dogfood: a11y primitives in the search modal** - the new
  pure-GWC site's search modal lacks UseFocusTrap/UseAnnouncer and the
  gallery filters lack composite keyboard navigation.
  Test for: focus is trapped while the modal is open and restored to the
  Search button on close (mirror TestAccessibleOverlayBrowserE2E);
  result-count changes are announced politely; Escape closes from any
  focused element inside the modal; filter chips are arrow-key navigable.
- [ ] **Reduced-motion / contrast preference hooks** - interop exposes
  GetMediaQuery but there is no UsePrefersReducedMotion /
  UsePrefersColorScheme hook pair, so apps re-derive them.
  Test for: hook reflects the media query at mount, updates live when the
  emulated preference flips (playwright EmulateMedia), and unsubscribes
  on unmount without leaking js.Func handles.

### Internationalization

- [ ] **Message extraction + locale completeness tooling** - nothing scans
  code for T(namespace, key) usage to scaffold catalogs or diff locales;
  incomplete translations ship silently (pairs with the enterprise
  missing-translation enforcement item).
  Test for: extraction finds every T() call across build tags (wasm and
  native files); diff reports keys missing per locale and stale keys no
  longer referenced; a gwc lane fails CI when a non-default locale is
  incomplete; dynamic/computed keys are reported as unverifiable rather
  than silently skipped.
- [ ] **Relative-time and list formatting** - FormatNumber/FormatDate exist
  but there is no FormatRelativeTime ("3 days ago") or FormatList
  ("a, b, and c"), the two most-requested formatters after dates.
  Test for: CLDR-correct output across at least en/fr/ar/ja including an
  RTL locale; plural-category interaction (1 day vs 2 days vs 0 days);
  boundary rounding (59s vs 1m, 23h vs 1d); native and wasm parity.
- [ ] **Browser Intl bridge** - the i18n formatters are framework
  implementations; expose an opt-in interop path to the browser's full
  ICU (Intl.NumberFormat/DateTimeFormat) for locales/options the Go
  implementation does not cover.
  Test for: bridge output matches browser Intl for sampled locale/option
  matrices; graceful fallback to the Go formatter when Intl or the
  requested locale is unavailable; no js.Func leaks across repeated
  formats (formatter instances cached and released).

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
- [ ] **SRI emission** - gwc release manifests already carry SHA-256 per
  asset but generated shells do not emit integrity attributes.
  Test for: integrity hash on wasm/script/style references matches the
  release manifest; a tampered asset is refused by the browser (fixture
  flips one byte); dev-server mode omits SRI so hot reload still works.
- [ ] **HTML sanitizer for untrusted content** - no DOMPurify-equivalent
  exists for rendering user-supplied HTML.
  Test for: an XSS corpus (script tags, event handlers, javascript:
  URLs, SVG payloads, mXSS nesting cases) is neutralized; allowlist
  configuration round-trips; sanitized output is stable across native
  and wasm builds; benchmark guard so sanitizing large documents stays
  off the render hot path.
- [ ] **Root-module security scanning + SBOM + SECURITY.md** - gosec and
  govulncheck run only for the bridge submodule; releases ship no SBOM;
  no security disclosure policy exists.
  Test for: scanners run green on the root module in CI with a
  documented suppression file; release workflow attaches a CycloneDX
  SBOM whose package list matches go.mod; SECURITY.md present with a
  disclosure contact.
- [ ] **Reproducible-build verification** - trimpath is set but nothing
  verifies two builds of one commit are bit-identical, which the
  provenance attestation implicitly promises.
  Test for: a CI job builds the release wasm twice in clean dirs and
  compares SHA-256; intentional nondeterminism (embedded timestamp
  fixture) is caught by the check.

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
- [ ] **WebCrypto bridge + encrypted persistent storage** - no
  crypto.subtle interop exists; persisted snapshots and IndexedDB
  caches store plaintext.
  Test for: encrypt/decrypt round-trip via WebCrypto from Go (AES-GCM
  with a non-extractable key); an EncryptedPersistentStore wrapper
  round-trips JSON and rejects tampered ciphertext; key unavailability
  degrades to an explicit error, never silent plaintext; native builds
  return the unavailable stub.

## Enterprise tier - operational resilience

- [ ] **Crash-loop safe mode** - containment keeps a running page alive,
  but a panic during boot reloads into the same crash forever.
  Test for: N consecutive failed boots (tracked in storage) switch the
  generated shell to a minimal diagnostics view with a cache-purge
  action instead of re-running the wasm; a successful boot resets the
  counter; the diagnostics view itself needs no wasm; e2e drives a
  deliberately boot-panicking module through the full loop.
- [ ] **Version-skew refresh flow** - wire formats are versioned now, but
  detection is not acted on: stale cached wasm against a redeployed
  server should trigger a controlled refresh, not an error.
  Test for: a sidecar/wasm version mismatch triggers exactly one forced
  reload with cache bypass (no reload loop on persistent mismatch -
  loop guard verified); user state is snapshotted before the reload and
  restored after when versions allow migration.
- [ ] **Remote flag provider + kill switch** - the new flags package is
  build/boot-time; no remote-config provider (poll/SSE) or kill-switch
  semantics exist.
  Test for: a flag flip on a mock remote provider reaches subscribed
  components within the polling interval; provider outage retains last
  known values with staleness surfaced; kill-switch flag disables a
  feature subtree without reload; misbehaving provider payloads are
  contained, never crash the app.

## Enterprise tier - release engineering

- [ ] **Canary / gradual wasm rollout** - no mechanism serves two wasm
  versions side-by-side with percentage routing and instant rollback.
  Test for: deterministic cohort assignment (same client stays on its
  version across reloads); rollback flips 100% within one cache TTL;
  both versions report their build id through artifact metadata so
  crash reports distinguish cohorts.
- [ ] **Perf-budget CI gate** - route startup budgets exist in profiling
  and gwc bench measures, but nothing fails a build on regression.
  Test for: a gwc lane fails when wasm size or measured route-startup
  exceeds the checked-in budget by the configured tolerance; budgets
  update through an explicit ratchet command, not silently; the gate
  output names the offending route and delta.
- [ ] **Visual regression lane** - playwright is wired everywhere but no
  screenshot-diff lane protects the examples or docs site.
  Test for: baseline capture + pixel-diff with anti-flake masking
  (timestamps, spinners); an intentional 1px style change in a fixture
  is caught; per-page thresholds configurable; lane runs on the docs
  site routes and a representative example subset.

## Enterprise tier - isolation & conformance

- [ ] **Shadow-DOM style isolation for exported custom elements** - no
  shadow-root helpers exist; embedded GWC widgets leak styles both ways.
  Test for: a GWC custom element mounted in a hostile host page (global
  CSS resets, conflicting class names) renders identically to its
  isolated baseline; host styles do not bleed in and widget styles do
  not bleed out; events and portals still work across the shadow
  boundary; focus trap and announcer behave inside shadow roots.
- [ ] **Cross-browser conformance matrix** - webkit/firefox run only in
  the Atlas smoke; the framework behavior suite is chromium-only.
  Test for: the core browser suite (events, hydration, router, storage,
  overlay focus) passes on chromium, firefox, and webkit in CI; known
  per-engine differences are encoded as explicit skips with linked
  issues, not silent passes.
- [ ] **Plugin API conformance suite** - kernel-backed plugins feed
  devtools but third parties have no test kit proving they meet the
  contract.
  Test for: a published conformance package a plugin author can run
  against their plugin (lifecycle, section contribution, diagnostics,
  teardown); the built-in kernel plugin passes it; a deliberately
  non-conforming fixture plugin fails with actionable messages.

## Security findings (2026-06-11 code review) - fix

- [ ] **Markdown URL-scheme allowlist (latent stored-XSS in a public API)** -
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
- [ ] **Livereload WebSocket origin validation (dev-server CSWSH)** -
  `tools/livereload/livereload.go` sets `CheckOrigin` to always return
  true, so while the dev server runs any visited website can open
  ws://127.0.0.1:<port> and receive the reload stream / trigger reloads.
  Fix: validate the request Origin against the configured dev host:port
  (and localhost variants), reject mismatches.
  Test for: same-origin upgrade succeeds; a foreign Origin is rejected
  with 403; missing Origin handled per policy; an explicit opt-out flag
  exists for tunnel/LAN dev with a logged warning.
- [ ] **Remove or fence the unused SetInnerHTML adapter sink** -
  `internal/runtime/interfaces.go` exposes `SetInnerHTML` on the DOM
  adapter; it is implemented (jsdom/mockdom) but never called from the
  render/commit path - an unprotected raw-HTML sink on the public
  interface. Fix: remove it, or if retained, document it as
  trusted-input-only and add a render-path assertion that it stays
  unreachable.
  Test for: a grep/analyzer guard fails if SetInnerHTML gains a
  render-path caller; if kept, a doc note plus an example of safe use.
- [ ] **Document the global cache-key namespace sharp edge** -
  `fetch.UseCachedResource` keys live in one app-global `sync.Map`
  (fetch/cache.go), so unrelated components choosing the same key string
  share state (type conflicts warn, so this is by-design, not a crash).
  Fix: document the global-namespace contract and recommend a
  module-prefix convention; consider an optional scoping helper.
  Test for: doc example of the collision and the prefix convention; the
  existing conflicting-type warning remains covered.

## Developer experience (2026-06-11 review) - build items + what to test

### Sharpest three (do first)

- [ ] **Commit editor support + document the gopls wasm-tag setting** -
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
- [ ] **Wire devtools.ErrorOverlay into `gwc dev` by default** - runtime
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
- [ ] **Write CONTRIBUTING.md for framework contributors** - AGENTS.md and
  the getting-started chapter target users; nothing captures how to work
  ON the framework: the `GOOS=js GOARCH=wasm go test -c -o x.wasm` + node
  runner dance, wasm coverage flags, playwright build-tag/package
  conventions, the Windows BOM/CRLF hazards, and the run-tests-sequentially
  gotchas.
  Test for: a fresh contributor can run the wasm unit suite, a playwright
  lane, and the coverage report by following the doc only; commands in
  the doc are copy-pasteable and verified in CI (doc-command lint).

### Onboarding & inner loop

- [ ] **Auto-run doctor on first build/dev failure** - `gwc doctor` is
  opt-in, so a fresh machine with the wrong Go version or missing
  wasm_exec hits a cryptic deep build error instead of an upfront
  diagnosis.
  Test for: a simulated missing-prerequisite environment triggers the
  doctor summary automatically on `gwc dev`/`gwc build` failure with a
  fix hint; a healthy environment never shows it; opt-out flag respected.
- [ ] **Browser build-status indicator in `gwc dev`** - the 304ms rebuild
  is only visible as terminal text; developers watching the browser get
  no signal. Add a small corner badge (building / ready / error) and an
  optional rebuild-failure notification when the terminal is unfocused.
  Test for: badge reflects building->ready transitions over the livereload
  channel in an e2e; error state shows on a failed rebuild and clears on
  the next success; badge is dev-only and never ships to production.
- [ ] **`gwc init` time-to-first-pixel guarantee** - scaffolding works but
  no CI test proves `gwc init` output builds and renders on a clean
  machine across the offered presets.
  Test for: each preset scaffold runs `go mod tidy` + `gwc build` clean
  and mounts a non-empty tree in a headless browser; a broken preset
  fails the lane with the offending preset named.

### Discoverability & polish

- [ ] **Capability matrix ("what's in the box")** - storage, PWA, i18n,
  a11y, flags, realtime, and snapshots all exist but are hard to
  discover by reading; surface a single matrix on the docs site mapping
  capability -> package -> example -> manual chapter.
  Test for: the matrix is generated from the catalog (no hand-maintained
  drift); every row links to a real example and chapter; a CI check fails
  if a listed capability's example or chapter link 404s.
- [ ] **Mark legacy dev flag aliases as deprecated in help** - `gwc dev`
  carries `-main`/`-index`/`-output` legacy aliases beside
  `-app`/`-html`/`-wasm`; help does not flag them, widening the surface a
  new user must navigate.
  Test for: help output labels each legacy alias as deprecated and points
  to the canonical flag; the aliases still function (no behavior change).
- [ ] **Web playground / shareable snippet runner** - the examples catalog
  is the closest thing to an evaluation surface; there is no edit-and-run
  snippet experience for quick evaluation/sharing.
  Test for: a snippet compiles and renders in the browser sandbox; a
  shared URL round-trips the snippet source; compile errors surface in the
  sandbox with the structured diagnostic, not a blank frame.

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
