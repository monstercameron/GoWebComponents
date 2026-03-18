# Examples

Location: `examples/`

This directory contains the framework examples and shared static assets.

## Serve Examples

From the repo root:

```powershell
npm run dev:examples
```

Then open:

- `http://127.0.0.1:8090/examples`
- `http://127.0.0.1:8090/examples/01-counter/counter.html`

The `/examples` page opens the styled showcase catalog. If you need the raw auto-generated folder listing for diagnostics, use `/examples/list`.

## Example Inventory

The example set is now organized as a catalog. Use the grouped landing page at `/examples` for browsing, and use the mapping below when you want the quickest reference for a specific public API.

Every feature-isolated catalog page is expected to answer three questions clearly: what the tool is for, what the user-facing behavior looks like, and how the implementation is wired. If an example stops doing one of those jobs, it should be treated as documentation debt rather than just a cosmetic issue.

### Integrated Apps

- `01-counter` through `20-portals`
- These combine multiple features and are useful for seeing how the primitives fit together in larger apps.
- `87-ssr-secure-forms`: request-time rendered forms with CSRF validation, multipart uploads, validation round-trips, and `303` redirects

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
- `27-transition-hooks`: `ui.StartTransition`, `ui.UseTransition`
- `28-use-reducer`: `ui.UseReducer`
- `29-use-debounced`: `ui.UseDebounced`
- `30-use-throttled`: `ui.UseThrottled`
- `31-context-api`: `ui.CreateContext`, `ui.UseContext`
- `32-async-boundary`: `ui.AsyncBoundary`
- `33-lazy`: `ui.Lazy`
- `34-error-boundary`: `ui.ErrorBoundary`
- `35-use-id`: `ui.UseId`
- `36-typed-events`: `ui.UseEvent`, typed event aliases
- `46-raw-handler`: `ui.RawHandler`
- `47-portal-selector`: `ui.Portal` with selector target
- `48-portal-target`: `ui.PortalTarget` with explicit node target
- `49-use-channel`: `ui.UseChannel`
- `50-use-task`: `ui.UseTask`
- `51-use-form`: `ui.UseForm`
- `70-render-to-string`: `ui.RenderToString`
- `71-hydrate`: `ui.Hydrate`
- `73-ssr-bootstrap`: `ui.RenderBootstrapScript`, `ui.ReadBootstrapScript`
- `87-ssr-secure-forms`: SSR HTML forms, `html.HiddenInput`, typed `html.Props{EncType: ...}`, and secure form-post conventions

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

### html Package

- `52-semantic-html`: semantic layout helpers
- `53-html-forms`: typed form controls via `html.Props`
- `54-html-tag`: `html.Tag`

### router Package

- `55-hash-router`: `router.NewHashRouter`, `Register`, `Mount`
- `56-browser-router`: `router.NewRouter`
- `57-use-navigate`: `router.UseNavigate`
- `58-route-params`: `router.UseParams`
- `59-route-query`: `router.UseQuery`, `router.UseSearchParams`
- `60-route-loaders`: `router.Options.Loader`, route loading and error states
- `61-use-revalidator`: `router.UseRevalidator`
- `62-router-redirects`: route redirects
- `63-router-metadata`: route-managed title and metadata
- `64-nested-layout-routes`: layout routes, `router.Outlet`
- `65-router-guards`: `BeforeEnter`, `BeforeLeave`
- `80-routed-accessibility`: route-change announcements and heading focus after navigation
- `72-router-hydrate-mount`: `router.HydrateMount`
- `74-ssr-route-data-reuse`: route-loader bootstrap reuse during hydration

### devtools Package

- `66-devtools-panel`: `devtools.Panel`
- `67-use-snapshot`: `devtools.UseSnapshot`
- `68-snapshot-now`: `devtools.SnapshotNow`
- `69-devtools-diagnostics`: devtools diagnostics payloads

## Shared Static Assets

`examples/static/` contains assets shared across example pages, including:

- `script/wasm_exec.js`
- shared CSS
- shared images
- common HTML entrypoints

Generated wasm binaries for examples belong under `examples/static/bin/` and should not be committed unless there is a deliberate reason to do so.

## Running Example Browser Tests

The `examples/` directory has its own Playwright setup for example-oriented testing.

If you are working on the main framework regression suites, use `test/` instead. If you are specifically validating example pages, use the example-local Playwright config.

Useful commands from `examples/`:

- `npm test`: full example-local Playwright suite that runs against the static catalog server
- `npm run test:catalog`: catalog-only link and smoke coverage against the dev server
- `npm run test:atlas-ssr`: Atlas native server SSR and mutation-flow coverage

For the developer-facing manual verification checklist that covers every numbered example, see `examples/MANUAL_TESTING.md`.

## Notes

- Older docs referenced ad hoc live reload commands as the primary example workflow. The primary documented path is now the Express server in `tools/dev-server/`.
- The browser-compiler example may generate a large local package archive tree under `examples/13-browser-compiler/static/pkg/`. That output is ignored and should stay out of git.
