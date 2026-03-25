# Examples

Location: `examples/`

This directory contains the framework examples and shared static assets.

## Current Status

- The example catalog is now a first-class documented surface rather than a loose folder listing.
- `go run ./tools/gwc examples` is the primary current way to browse the catalog from the repo root.

## Serve Examples

From the repo root:

```powershell
go run ./tools/gwc examples
```

Then open:

- `http://127.0.0.1:8090/examples`
- `http://127.0.0.1:8090/examples/01-counter/counter.html`

The `/examples` page opens the styled showcase catalog. If you need the raw auto-generated folder listing for diagnostics, use `/examples/list`.

## Example Inventory

The example set is now organized as a catalog. Use the grouped landing page at `/examples` for browsing, and use the mapping below when you want the quickest reference for a specific public API.

Every feature-isolated catalog page is expected to answer three questions clearly: what the tool is for, what the user-facing behavior looks like, and how the implementation is wired. If an example stops doing one of those jobs, it should be treated as documentation debt rather than just a cosmetic issue.

## Current Boundary

- Shipped: a browsable example catalog, dedicated example browser coverage via Playwright-Go wrappers, and integrated examples that act as reference surfaces for larger app shapes.
- Not shipped: a promise that every example is a production starter template or that every example is kept runnable through every possible serving path.
- Use the examples to understand supported APIs and composed flows; use the package docs and workflow docs for policy boundaries.

### Integrated Apps

- `01-counter` through `20-portals`
- These combine multiple features and are useful for seeing how the primitives fit together in larger apps.
- `18-ssr-server-routing`: the metadata-heavy SSR routing reference, combining request-time HTML, route bootstrap reuse, `head.Render(...)` output, and the ownership boundary between first-paint SSR metadata and later hydrated navigation
- `86-atlas-commerce-os`: the current production-shaped SSR reference server, combining shared UI, SSR, hydration, server-owned routes, mutations, static assets, and reviewer-facing docs
- `87-ssr-secure-forms`: request-time rendered forms with CSRF validation, multipart uploads, validation round-trips, `303` redirects, and the progressive half of the server-action contract
- `106-single-shell-auth`: one running WASM shell that keeps marketing, sign-in, and a guarded workspace in one router tree with bounded return-to recovery and no full-page auth detour

Use the integrated apps when you want to understand how multiple primitives compose in one realistic surface. Use the feature-isolated pages when you want the clearest statement of a single API or tool's purpose.

### ui Package

- `76-use-effect`: `ui.UseEffect`
- `77-accessible-overlay`: `ui.AccessibleOverlay`, `ui.UseFocusManager`, `ui.UseFocusTrap`
- `78-composite-navigation`: `ui.UseCompositeNavigation`
- `79-form-accessibility`: `ui.UseAnnouncer`, validation announcements, focus-to-error behavior
- `81-overlay-stack`: `ui.Overlay`, `ui.UseOverlayStack`, nested dialog and popover coordination
- `82-overlay-anchor`: `ui.Overlay` with anchored menu and tooltip retargeting
- `21-ui-render`: `ui.Render`
- `22-create-element`: `ui.CreateElement`
- `23-fragment`: `ui.Fragment`
- `24-use-ref`: `ui.UseRef`
- `25-use-previous`: `ui.UsePrevious`
- `26-use-deferred-value`: `ui.UseDeferredValue`
- `27-transition-hooks`: `ui.StartTransition`, `ui.UseTransition`, typeahead filtering, tab swaps, and route-style section transitions
- `28-use-reducer`: `ui.UseReducer`
- `28-scaling-local-state-with-use-reducer`: `ui.UseReducer` wrapped in an app-specific hook for larger local workflows
- `29-use-debounced`: `ui.UseDebounced`
- `30-use-throttled`: `ui.UseThrottled`
- `31-context-api`: `ui.CreateContext`, `ui.UseContext`
- `32-async-boundary`: `ui.AsyncBoundary`
- `33-lazy`: `ui.Lazy`
- `34-error-boundary`: `ui.ErrorBoundary`
- `35-use-id`: `ui.UseId`
- `36-typed-events`: `ui.UseEvent`, typed event aliases
- `46-raw-handler`: `ui.WrapHandler`
- `47-portal-selector`: `ui.Portal` with selector target
- `48-portal-target`: `ui.PortalTarget` with explicit node target
- `49-use-channel`: `ui.UseChannel`
- `50-use-task`: `ui.UseTask`
- `51-use-form`: `ui.UseForm`
- `91-worker-text-index`: `ui.UseWorkerTask`, worker progress, and CPU-heavy text indexing off the main thread
- `70-render-to-string`: `ui.RenderToString` plus prerender-style `head.Render(...)` composition for robots tags, social tags, and JSON-LD
- `71-hydrate`: `ui.Hydrate`
- `73-ssr-bootstrap`: `ui.RenderBootstrapScript`, `ui.ReadBootstrapScript`
- `107-staged-rollout-config`: typed `ui.ReadBootstrapPayload(...)`, public-config and rollout-flag bootstrap restore, and hydration-aligned route gating for environment-aware apps
- `101-static-islands`: explicit multi-root `ui.Hydrate(...)` selective activation with in-page startup, hydration, and interaction budgets
- `102-static-export-site`: `prerender.Export(...)`, manifest-backed hashed asset URLs, route-scoped preload and prefetch hints, responsive images, and lazy media for static-hosted marketing or docs pages
- `104-use-callback`: `ui.UseCallback`, dependency-driven callback identity, and the split between memoized callbacks, `ui.UseEvent`, and ordinary inline handlers
- `105-use-lazy-node`: `ui.UseLazyNode`, manual `Reload()` and `Cancel()` control, and explicit `ui.AsyncBoundary(...)` composition
- `87-ssr-secure-forms`: SSR HTML forms, `html.HiddenInput`, typed `html.Props{EncType: ...}`, and secure form-post conventions

### virtualization Package

- `103-virtualized-feed`: `virtualization.List`, `ViewportDiagnostics`, fixed-height row-window diagnostics, row-state ownership guidance, and scroll-restoration behavior for long feeds

### i18n Package

- `83-locale-switcher`: `i18n.UseLocale`, `i18n.Provider`, `i18n.UseI18n`, pluralization, and number or date formatting
- `84-ssr-i18n-bootstrap`: `ui.SSRBootstrap.I18n`, `i18n.BundleFromSSRBootstrap`, and hydration-aligned locale restore
- `85-locale-routing`: `i18n.PrefixPath`, `i18n.ResolvePath`, and locale-aware route-loader content selection

### state Package

- `75-use-state`: `ui.UseState`
- `37-use-atom`: `state.UseAtom`
- `38-use-computed`: `state.UseComputed`
- `39-use-derived`: `state.UseDerived`
- `40-snapshot-export-import`: `state.ExportSnapshot`, `state.ImportSnapshot`, JSON round-tripping
- `41-snapshot-storage`: `state.SaveSnapshot`, `state.LoadSnapshot`, `state.RestoreSnapshot`

### fetch Package

- `42-use-fetch`: `fetch.UseFetch`
- `43-use-resource`: `fetch.UseResource`
- `44-use-cached-resource`: `fetch.UseCachedResource`
- `45-fetch-imperative`: `fetch.Fetch`
- `93-ssr-cache-bootstrap`: SSR-seeded shared cache restore, `fetch.RestoreCacheBootstrap`, and resume policies

### html Package

- `52-semantic-html`: semantic layout helpers
- `53-html-forms`: typed form controls via `html.Props`
- `54-html-tag`: `html.Tag`
- `88-web-components`: `html.CustomElement`, `html.Props{Slot: ...}`, and `interop.SubscribeDecoded[T]`
- `89-exported-custom-element`: experimental export-side custom-element wrapper via `ui.RenderInto`

### interop Package

- `90-browser-interop`: storage, clipboard, grouped DOM lookup, resize or media observation, typed custom events, and lazy module loading through `interop`
- `94-cross-tab-sync`: `interop.OpenCrossTabChannel`, typed cross-tab messages, transport diagnostics, and opt-in state hint broadcasting
- `95-multi-window-console`: `interop.OpenSecondaryWindowChannel`, `interop.WindowOpenerChannel`, and typed shared session or route coordination across popup surfaces
- `97-multi-client-presence`: `interop.PublishClientHello`, `interop.PublishClientQuery`, `interop.PublishClientResult`, `interop.SubscribeClientMessages`, and `interop.SubscribeClientWindowMessages` for late join discovery and targeted popup replies
- `97-multi-client-binary`: `interop.PublishClientBinaryCrossTab`, `interop.PublishClientBinaryWindow`, JSON control-plane handshake, binary preview payloads on supported transports, and JSON fallback when cross-tab transport cannot carry bytes

### pwa Package

- `97-pwa-installability`: `pwa.ObserveInstallability`, `pwa.RegisterServiceWorker`, manifest validation, and install-prompt diagnostics
- `97-pwa-offline-cache`: `pwa.BuildCacheStoragePlan`, `pwa.OpenCacheStorageManager`, `pwa.InspectDiagnostics`, `pwa.ServiceWorkerRegistration.RegisterSync`, and conflict-aware offline mutation replay inspection
- `97-pwa-multi-client`: `pwa.RegisterServiceWorker`, `pwa.BuildCacheStoragePlan`, `pwa.InspectDiagnostics`, `interop.OpenCrossTabChannel`, and typed multi-client hello or sync or invalidation messaging across wasm tabs

These three focused examples form the current offline-first reference slice for the repo: installability plus service-worker ownership, shell caching plus durable replay, and multi-tab coordination plus invalidation under one PWA-shaped surface area.

### router Package

- `55-hash-router`: `router.NewHashRouter`, `Register`, `Mount`
- `56-browser-router`: `router.NewRouter`
- `57-use-navigate`: `router.UseNavigate`
- `58-route-params`: `router.UseParams`
- `59-route-query`: `router.UseQuery`, `router.UseSearchParams`
- `60-route-loaders`: `router.Options.Loader`, route loading and error states
- `61-use-revalidator`: `router.UseRevalidator`
- `62-router-redirects`: route redirects
- `63-router-metadata`: client-managed title, description, and canonical metadata during browser navigation
- `64-nested-layout-routes`: layout routes, `router.Outlet`
- `65-router-guards`: `BeforeEnter`, `BeforeLeave`
- `92-protected-routes`: guarded redirects, safe `return_to`, manual authorizing UI, and `fetch.LoadCached`
- `106-single-shell-auth`: one-shell marketing, sign-in, and protected workspace routing with `BeforeEnter`, `router.PreserveReturnTo(...)`, `router.ReadReturnTo(...)`, and `router.RouteContract`
- `96-code-splitting`: route shells, `ui.Lazy` nested panels, and route-family shell preservation
- `80-routed-accessibility`: route-change announcements and heading focus after navigation
- `72-router-hydrate-mount`: `router.HydrateMount`
- `74-ssr-route-data-reuse`: route-loader bootstrap reuse during hydration

### devtools Package

- `66-devtools-panel`: `devtools.Panel`
- `67-use-snapshot`: `devtools.UseSnapshot`
- `68-snapshot-now`: `devtools.SnapshotNow`
- `69-devtools-diagnostics`: devtools diagnostics payloads
- `98-hot-reload`: `hotreload.Enable()`, `tools/dev.ps1`, and state-preserving rebuilds across edits

### plugin Package

- `99-plugin-host`: `plugin.Host`, `plugin.Plugin`, capability-checked registration, route and async-data hooks, SSR head contributions, and form-validation hooks

### Hot Reload Workflow

For a single app surface, use the standalone dev server and enable the public `hotreload` package in that app:

```powershell
.\tools\dev.ps1 -App .\examples\98-hot-reload\main.go -Root .\examples\98-hot-reload -Html .\examples\98-hot-reload\hot-reload.html
```

Use `gwc examples` when you want to browse many examples. Use `tools/dev.ps1` or `tools/dev.sh` when you want rebuild-on-save and state-preserving reload for one specific app.

## Shared Static Assets

`examples/static/` contains assets shared across example pages, including:

- `script/wasm_exec.js`
- shared CSS
- shared images
- common HTML entrypoints

Generated wasm binaries for examples belong under `bin/examples/` and should not be committed unless there is a deliberate reason to do so. Example pages still load them via `/static/bin/...` through the local example servers.

## Running Example Browser Tests

The `examples/` directory has dedicated browser smoke suites for example-oriented testing, powered by Go `playwright-go` wrappers under `test/playwrightgo/examples`.

If you are working on the main framework regression suites, use `test/` instead. If you are specifically validating example pages, use these commands from the repo root:

- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExamplesAll -v`
- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestCatalog -v`
- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestSSRServerRouting -v`
- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestAtlasSSR -v`
- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestStartup -v`
- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestAtlasStartup -v`
- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestChatWizard -v`

From the repo root, `go run ./tools/gwc test -lane browser` now runs the main browser lane.

For the developer-facing manual verification checklist that covers every numbered example, see `examples/MANUAL_TESTING.md`.

## Notes

- Older docs referenced ad hoc live reload commands as the primary example workflow. The primary documented path is now `gwc examples`.
- The browser-compiler example may generate a large local package archive tree under `examples/13-browser-compiler/static/pkg/`. That output is ignored and should stay out of git.
