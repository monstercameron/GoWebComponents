# 16 API Browser

Use this chapter when you want a dense package-by-package browser for the public GoWebComponents surface.

It is intentionally higher density and less tutorial-driven than the earlier chapters. The goal is fast lookup:

- which package owns a capability
- which top-level functions start the workflow
- which parameter or props structs shape the call
- which returned handle or object you work with next

Package GoDoc and source remain the final authority for exact fields and methods. This chapter is the map.

## How To Read This Chapter

- start with the package index if you only know the feature area
- use the `Call shape` column to remember the canonical one-line usage
- use the `Parameter objects / handles` column when you need the props, options, or returned object types
- jump back to the narrative chapters when you need architecture guidance instead of symbol lookup

## Package Index

| Package | Kind | Use it for | Main manual chapter |
| --- | --- | --- | --- |
| `ui` | core | components, hooks, SSR, hydration, overlays, forms, async helpers | [04](04-ui-rendering-and-hooks.md), [09](09-ssr-and-hydration.md), [11](11-forms-accessibility-and-i18n.md) |
| `html` | core | typed DOM builders, prop options, markdown helpers, resource hints | [05](05-html-authoring.md) |
| `html/shorthand` | core companion | mixed-argument host tag wrappers over `html` | [05](05-html-authoring.md) |
| `state` | core | shared atoms, selectors, derived values, snapshots | [06](06-state-and-reactivity.md) |
| `fetch` | core | typed resources, shared cache, mutation queue, uploads | [07](07-data-loading-and-mutations.md) |
| `router` | core | routers, route contracts, params, guards, metadata | [08](08-routing.md) |
| `interop` | core | browser APIs, workers, storage, channels, JS bridges | [10](10-browser-interop-and-workers.md) |
| `i18n` | companion | locale state, translation catalogs, formatting, locale routing | [11](11-forms-accessibility-and-i18n.md) |
| `devtools` | supported companion | in-app inspection, snapshots, capture bundles, replay | [12](12-devtools-testing-and-observability.md) |
| `head` | supported companion | SSR head composition over router metadata | [09](09-ssr-and-hydration.md), [15](15-design-notes-and-boundaries.md) |
| `pwa` | supported companion | manifest, service worker, installability, cache storage diagnostics | [13](13-assets-deployment-and-pwa.md) |
| `plugin` | supported companion | explicit plugin host, guards, panels, head/bootstrap providers | [15](15-design-notes-and-boundaries.md) |
| `prerender` | support | static export helpers | [13](13-assets-deployment-and-pwa.md) |
| `virtualization` | support | fixed-height virtualized list rendering and viewport math | [10](10-browser-interop-and-workers.md) |
| `diagnostics` | support | structured diagnostic reports and HTTP error emission | [12](12-devtools-testing-and-observability.md) |
| `hotreload` | support | state-preserving development hot reload bridge | [12](12-devtools-testing-and-observability.md) |
| `logging` | support | structured browser logging | [12](12-devtools-testing-and-observability.md) |
| `utils` | support | js/wasm helpers such as `WaitForever`, debug toggles, hot reload bridge | [01](01-getting-started.md), [12](12-devtools-testing-and-observability.md) |
| `testkit/render` | test support | js/wasm component fixture harness | [12](12-devtools-testing-and-observability.md) |
| `testkit/hooks` | test support | hook harness on top of `testkit/render` | [12](12-devtools-testing-and-observability.md) |
| `testkit/router` | test support | router fixture harness and guard helpers | [12](12-devtools-testing-and-observability.md) |
| `testkit/ssr` | test support | SSR snapshots and hydration smoke helpers | [09](09-ssr-and-hydration.md), [12](12-devtools-testing-and-observability.md) |

## ui

Import with `import "github.com/monstercameron/GoWebComponents/ui"`.

Source anchors:

- [ui/doc.go](../../ui/doc.go)
- [ui/ui.go](../../ui/ui.go)
- [ui/ui_async.go](../../ui/ui_async.go)
- [ui/context_shared.go](../../ui/context_shared.go)
- [ui/accessibility_shared.go](../../ui/accessibility_shared.go)
- [ui/form.go](../../ui/form.go)
- [ui/overlay.go](../../ui/overlay.go)
- [ui/parallel_region.go](../../ui/parallel_region.go)
- [ui/ssr_bootstrap.go](../../ui/ssr_bootstrap.go)
- [ui/ssr_transfer.go](../../ui/ssr_transfer.go)
- [ui/ssr_observability.go](../../ui/ssr_observability.go)
- [ui/worker_wasm.go](../../ui/worker_wasm.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `CreateElement`, `Component`, `Fragment`, `Text` | core node composition | [`Node`](../../ui/ui.go), [`Element`](../../ui/ui.go) | `ui.CreateElement(renderApp, appProps{})` |
| `Render`, `RenderInto`, `RenderToString`, `RenderToStringObserved`, `Hydrate`, `HydrateInto` | browser mount, SSR HTML, hydration attach | [`HydrationOptions`](../../ui/hydration.go), [`SSRObservabilityOptions`](../../ui/ssr_observability.go), [`SSRBootstrap`](../../ui/ssr_bootstrap.go) | `ui.Render(ui.CreateElement(renderApp, nil), "#app")` |
| `UseState`, `UseReducer`, `UseRef`, `UsePrevious`, `UseId`, `UseEffect`, `UseMemo`, `UseCallback`, `UseEvent`, `WrapHandler` | local state and event wiring | [`State[T]`](../../ui/ui.go), [`Reducer[S,A]`](../../ui/ui.go), [`Ref[T]`](../../ui/ui.go), [`Previous[T]`](../../ui/ui.go), [`Handler`](../../ui/ui.go) | `storeCount := ui.UseState(0)` |
| `StartTransition`, `UseTransition`, `UseDeferredValue`, `UseDebounced`, `UseThrottled`, `UseChannel`, `UseTask`, `UseWorkerTask` | scheduling, rate limiting, channel/task consumption, worker-backed jobs | [`Transition`](../../ui/ui.go), [`Debounced[T]`](../../ui/ui.go), [`Throttled[T]`](../../ui/ui.go), [`Channel[T]`](../../ui/ui.go), [`Task[T]`](../../ui/ui.go), [`WorkerTask[Request,Progress,Result]`](../../ui/worker_wasm.go) | `searchTransition := ui.UseTransition()` |
| `AsyncBoundary`, `UseLazyNode`, `Lazy`, `ErrorBoundary` | async subtree fallbacks and panic recovery | [`AsyncBoundaryProps`](../../ui/ui.go), [`LazyProps`](../../ui/ui.go), [`LazyNode`](../../ui/ui.go), [`ErrorBoundaryProps`](../../ui/ui.go) | `return ui.AsyncBoundary(ui.AsyncBoundaryProps{Pending: state.Loading, Content: body})` |
| `CreateContext`, `UseContext`, `Portal`, `ReactiveRegion`, `If`, `Match` | subtree-scoped values, portals, fine-grained regions, conditional composition | [`Context[T]`](../../ui/context_shared.go), [`ContextProviderProps[T]`](../../ui/context_shared.go), [`PortalProps`](../../ui/ui.go), [`ReactiveSource`](../../ui/ui.go), [`MatchBuilder`](../../ui/branching.go) | `session := ui.UseContext(appSessionContext)` |
| `UseFocusManager`, `UseFocusTrap`, `UseCompositeNavigation`, `UseAnnouncer`, `AccessibleOverlay`, `UseOverlayStack`, `Overlay` | accessibility, focus management, roving tabindex, live regions, layered surfaces | [`FocusOptions`](../../ui/accessibility_shared.go), [`FocusTrapOptions`](../../ui/accessibility_shared.go), [`CompositeItem`](../../ui/accessibility_shared.go), [`CompositeNavigationOptions`](../../ui/accessibility_shared.go), [`AccessibleOverlayProps`](../../ui/accessibility_shared.go), [`OverlayStackOptions`](../../ui/overlay.go), [`OverlayProps`](../../ui/overlay.go), [`OverlayStack`](../../ui/overlay.go), [`Announcer`](../../ui/accessibility_shared.go) | `dialog := ui.Overlay(ui.OverlayProps{Open: isOpen, OnDismiss: handleClose})` |
| `UseForm`, `NewCSRFToken`, `GetFiles` | typed local form state, server-action result mapping, file input extraction | [`Form[T]`](../../ui/form.go), [`CSRFToken`](../../ui/form.go), [`ServerActionResult`](../../ui/form.go), [`ServerFormErrors`](../../ui/form.go), [`FieldStatus`](../../ui/form.go), [`File`](../../ui/file_wasm.go) | `profileForm := ui.UseForm(profileDraft{})` |
| `MarshalSSRBootstrap`, `MarshalSSRBootstrapObserved`, `MarshalSSRBootstrapBinary`, `MarshalSSRBootstrapBinaryObserved`, `UnmarshalSSRBootstrap`, `UnmarshalSSRBootstrapBinary`, `RenderBootstrapScript`, `RenderBootstrapScriptObserved`, `RenderBootstrapReferenceScript`, `ReadBootstrapScript`, `ReadBootstrapReferenceScript`, `ReadBootstrapReference`, `UnmarshalSSRBootstrapReference` | SSR bootstrap encoding and script emission | [`SSRBootstrap`](../../ui/ssr_bootstrap.go), [`SSRBootstrapReference`](../../ui/ssr_bootstrap.go), [`SSRI18nBootstrap`](../../ui/ssr_bootstrap.go), [`SSRRouteBootstrap`](../../ui/ssr_bootstrap.go) | `scriptHTML, err := ui.RenderBootstrapScript(bootstrap, ui.DefaultBootstrapScriptID)` |
| `RegisterBootstrapPayload`, `ReadBootstrapPayload`, `RegisterRouteBootstrapData`, `ReadRouteBootstrapData`, `RegisterFormBootstrapDefaults`, `ReadFormBootstrapDefaults`, `RegisterCacheBootstrapSeed`, `ReadCacheBootstrapSeed`, `RegisterSessionBootstrapHint`, `ReadSessionBootstrapHint`, `InspectBootstrapPayloads`, `RegisterStateUpdatePayload`, `MarshalSSRStateUpdateText`, `UnmarshalSSRStateUpdateText`, `MarshalSSRStateUpdateBinary`, `UnmarshalSSRStateUpdateBinary`, `ApplySSRStateUpdate`, `NewSSRBootstrapBudget`, `InspectSSRBootstrapSize` | typed payload registration, scoped bootstrap reads, state update envelopes, size budgeting | [`SSRPayloadOptions`](../../ui/ssr_transfer.go), [`SSRPayloadValue[T]`](../../ui/ssr_transfer.go), [`SSRPayloadFilter`](../../ui/ssr_transfer.go), [`SSRPayloadMetadata`](../../ui/ssr_transfer.go), [`SSRStateUpdate`](../../ui/ssr_transfer.go), [`SSRBootstrapBudget`](../../ui/ssr_transfer.go), [`SSRBootstrapSizeReport`](../../ui/ssr_transfer.go) | `err := ui.RegisterBootstrapPayload(&bootstrap, "session", sessionValue)` |
| `RegisterSSRObserver`, `SSRObservabilityOptionsFromContext`, `WithCorrelationID`, `CorrelationIDFromContext`, `WithW3CTraceContext`, `TraceContextFromContext`, `ConfigureBootstrapCorrelation`, `GetSSRObservationAttributes`, `WrapSSRCorrelation` | SSR and hydration correlation, tracing, observability subscriptions | [`SSRObservation`](../../ui/ssr_observability.go), [`SSRObserverSubscription`](../../ui/ssr_observability.go), [`W3CTraceContext`](../../ui/correlation_native.go) | `subscription := ui.RegisterSSRObserver(func(observation ui.SSRObservation) {})` |
| `RegisterParallelRegion`, `ParallelRegion`, `BuildParallelRegionSourceIDs`, `GetParallelRegionRuntimeStatus`, `UnsupportedOnServer` | parallel region registration, runtime bridge status, explicit server-only guardrails | [`ParallelRegionSpec[Props]`](../../ui/parallel_region.go), [`ParallelRegionStatus`](../../ui/parallel_region.go) | `return ui.ParallelRegion(ui.ParallelRegionSpec[props]{RendererID: "search-grid", Props: props})` |

Key `ui` handles and objects:

- [`State[T]`](../../ui/ui.go): `Get`, `Set`, `Update`
- [`Reducer[S,A]`](../../ui/ui.go): `Get`, `Dispatch`
- [`Ref[T]`](../../ui/ui.go): `Get`, `Set`
- [`Transition`](../../ui/ui.go): `Pending`, `Start`
- [`Previous[T]`](../../ui/ui.go): `Get`, `Ok`
- [`Channel[T]`](../../ui/ui.go): `Get`, `Ok`, `Closed`
- [`Task[T]`](../../ui/ui.go): `Get`, `Start`, `Cancel`
- [`WorkerTask[Request,Progress,Result]`](../../ui/worker_wasm.go): `Get`, `Start`, `Cancel`
- [`Form[T]`](../../ui/form.go): `Get`, `Set`, `Update`, `SetField`, `Validate`, `ValidateAsync`, `Submit`, `SubmitWithIntent`, `ApplyServerErrors`, `ApplyServerActionResult`, `Reset`
- [`Announcer`](../../ui/accessibility_shared.go): `Region`, `Polite`, `Assertive`, `Announce`, `Clear`

Constants and sentinels you will search for:

- `DefaultBootstrapScriptID`, `DefaultBootstrapReferenceScriptID`, `CurrentSSRBootstrapVersion`, `CurrentSSRStateUpdateVersion`
- `DefaultRouteBootstrapPayloadKey`, `SSRBootstrapFormatJSON`, `SSRBootstrapFormatBinary`
- `DefaultCSRFHeaderName`, `DefaultCSRFFormFieldName`
- `ErrorBoundary`

## html

Import with `import html "github.com/monstercameron/GoWebComponents/html"`.

Source anchors:

- [html/doc.go](../../html/doc.go)
- [html/html.go](../../html/html.go)
- [html/sugar.go](../../html/sugar.go)
- [html/markdown.go](../../html/markdown.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `Html`, `Head`, `Body`, `Meta`, `Link`, `Script`, `NoScript` | document and head markup | [`Props`](../../html/html.go) | `html.Head(html.Props{}, html.Meta(html.Props{Name: "description", Content: "Store"}))` |
| `Main`, `Div`, `Section`, `Article`, `Aside`, `Header`, `Footer`, `Nav`, `P`, `Span`, `Strong`, `Em`, `Small`, `Mark`, `Code`, `Pre`, `Blockquote`, `Time` | general content structure | [`Props`](../../html/html.go) | `html.Section(html.Props{Class: "panel"}, html.H2(html.Props{}, html.Text("Orders")))` |
| `H1`, `H2`, `H3`, `H4`, `H5`, `H6` | headings | [`Props`](../../html/html.go) | `html.H1(html.Props{}, html.Text("Dashboard"))` |
| `Form`, `Fieldset`, `Legend`, `Label`, `Input`, `Textarea`, `Select`, `Option`, `Button`, `Details`, `Summary`, `Dialog`, `HiddenInput` | forms and interactive controls | [`Props`](../../html/html.go), [`CustomElementProps`](../../html/html.go) | `html.Input(html.Props{Name: "email", Type: "email"})` |
| `Ul`, `Li`, `Table`, `Thead`, `Tbody`, `Tr`, `Th`, `Td` | lists and tables | [`Props`](../../html/html.go) | `html.Tr(html.Props{}, html.Td(html.Props{}, html.Text("A-102")))` |
| `Img`, `Br`, `Hr`, `Tag`, `Fragment`, `CustomElement` | media, void tags, generic/custom elements | [`Props`](../../html/html.go), [`CustomElementProps`](../../html/html.go) | `html.CustomElement("x-chart", html.CustomElementProps{}, ui.Text("ready"))` |
| `Text`, `Textf`, `TextIf`, `Children`, `WithKey`, `If`, `IfElse`, `Unless`, `When`, `ClassNames`, `Map`, `MapKeyed`, `FlatMap`, `FilterMap`, `Join`, `Maybe`, `OrElse`, `Coalesce`, `Switch`, `Case`, `Default` | additive sugar on top of explicit typed builders | [`SwitchBranch`](../../html/sugar.go) | `html.Textf("Count: %d", count)` |
| `PropsOf`, `WithProps`, `Class`, `ID`, `Name`, `For`, `Title`, `Value`, `Placeholder`, `Type`, `Href`, `Src`, `Role`, `Rows`, `TabIndex`, `Disabled`, `Checked`, `Selected`, `Required`, `ReadOnly`, `AutoFocus`, `DisabledIf`, `ReadOnlyIf`, `SelectedIf`, `Style`, `Data`, `Dataset`, `Aria`, `AriaSet`, `Attr`, `Attrs`, `OnBlur`, `OnChange`, `OnClick`, `OnClickParallel`, `OnFocus`, `OnInput`, `OnKeyDown`, `OnKeyUp`, `OnMouseDown`, `OnMouseUp`, `OnScroll`, `OnSubmit` | prop assembly and event option helpers | [`Props`](../../html/html.go), [`PropOption`](../../html/html.go) | `buttonProps := html.PropsOf(html.Class("btn"), html.OnClick(handleSave))` |
| `Prevent`, `Stop`, `Debounce`, `Throttle` | event wrapper helpers | event callbacks, [`PropOption`](../../html/html.go) | `html.Button(html.Props{OnClick: html.Debounce(250*time.Millisecond, handleSave)}, html.Text("Save"))` |
| `RenderMarkdown`, `ResolveMarkdownHref`, `DNSPrefetch`, `Preconnect`, `Prefetch`, `Preload`, `ModulePreload` | markdown rendering and resource hints | [`MarkdownRenderOptions`](../../html/markdown.go), [`MarkdownClasses`](../../html/markdown.go) | `nodes := html.RenderMarkdown(readmeText)` |

Important parameter objects:

- [`Props`](../../html/html.go)
- [`PropOption`](../../html/html.go)
- [`CustomElementProps`](../../html/html.go)
- [`MarkdownRenderOptions`](../../html/markdown.go)
- [`MarkdownClasses`](../../html/markdown.go)

## html/shorthand

Import with `import h "github.com/monstercameron/GoWebComponents/html/shorthand"`.

Source anchors:

- [html/shorthand/doc.go](../../html/shorthand/doc.go)
- [html/shorthand/shorthand.go](../../html/shorthand/shorthand.go)

`html/shorthand` mirrors the `html` builder surface, but each host-tag wrapper accepts one mixed argument list of prop options and children.

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `A`, `Article`, `Body`, `Br`, `Button`, `Code`, `Details`, `Div`, `Form`, `H1`, `H2`, `H3`, `Head`, `Header`, `Hr`, `Html`, `Img`, `Input`, `Label`, `Li`, `Main`, `Mark`, `Meta`, `NoScript`, `Option`, `P`, `Pre`, `Script`, `Section`, `Select`, `Span`, `Summary`, `Table`, `Tbody`, `Td`, `Th`, `Thead`, `Tr`, `Ul`, `Tag`, `Fragment` | mixed-argument host tag wrappers | [`Props`](../../html/shorthand/shorthand.go), [`PropOption`](../../html/shorthand/shorthand.go) | `h.Div(h.Class("panel"), h.H2("Counter"), h.Button(h.OnClick(handleAdd), "Add"))` |
| `FromProps`, `PropsOf`, `WithProps` | mix fully built props into shorthand argument lists | [`Props`](../../html/shorthand/shorthand.go), [`PropOption`](../../html/shorthand/shorthand.go) | `h.Input(h.FromProps(baseProps), h.Placeholder("Email"))` |
| `Text`, `Textf`, `TextIf`, `Children`, `If`, `IfElse`, `Unless`, `When`, `ClassNames`, `Map`, `MapKeyed`, `FlatMap`, `FilterMap`, `Join`, `Maybe`, `OrElse`, `Coalesce`, `Switch`, `Case`, `Default`, `WithKey`, `Prevent`, `Stop`, `Debounce`, `Throttle` | the same additive authoring sugar as `html` | [`SwitchBranch`](../../html/shorthand/shorthand.go) | `h.Ul(h.Map(items, renderItem)...)` |
| `Class`, `ID`, `Name`, `For`, `Title`, `Value`, `Placeholder`, `Type`, `Href`, `Src`, `Role`, `Rows`, `TabIndex`, `Disabled`, `Checked`, `Selected`, `Required`, `ReadOnly`, `AutoFocus`, `DisabledIf`, `ReadOnlyIf`, `SelectedIf`, `Style`, `Data`, `Dataset`, `Aria`, `AriaSet`, `Attr`, `Attrs`, `OnBlur`, `OnChange`, `OnClick`, `OnClickParallel`, `OnFocus`, `OnInput`, `OnKeyDown`, `OnKeyUp`, `OnMouseDown`, `OnMouseUp`, `OnScroll`, `OnSubmit` | shorthand re-exports of `html` prop helpers | [`PropOption`](../../html/shorthand/shorthand.go) | `h.Button(h.Class("btn"), h.Prevent(handleSubmit), "Submit")` |

## state

Import with `import "github.com/monstercameron/GoWebComponents/state"`.

Source anchors:

- [state/doc.go](../../state/doc.go)
- [state/state.go](../../state/state.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `UseAtom` | shared read/write state keyed by ID | [`Atom[T]`](../../state/state.go) | `sessionAtom := state.UseAtom("session", sessionState{})` |
| `UseComputed` | render-local computed value based on hooks, atoms, or props | [`Computed[T]`](../../state/state.go) | `fullName := state.UseComputed(func() string { return first + " " + last })` |
| `UseDerived`, `UseSelector`, `Select` | shared read-only derived atoms and projected selectors | [`Derived[T]`](../../state/state.go) | `selectedUser := state.Select("selected-user", usersAtom, projectUser)` |
| `ExportSnapshot`, `GetSnapshot`, `ApplySnapshot`, `ImportSnapshot`, `MarshalSnapshotJSON`, `UnmarshalSnapshotJSON` | in-memory snapshot export/import and JSON serialization | [`Snapshot`](../../state/state.go) | `snapshotJSON, err := state.MarshalSnapshotJSON(snapshot)` |
| `SaveSnapshot`, `LoadSnapshot`, `RestoreSnapshot`, `SavePersistentSnapshot`, `LoadPersistentSnapshot`, `RestorePersistentSnapshot` | browser storage restore flows for atom snapshots | [`StorageArea`](../../state/state.go), [`PersistentSnapshotOptions`](../../state/state.go), [`Snapshot`](../../state/state.go) | `_, err := state.RestorePersistentSnapshot(ctx, "drafts")` |

Key state objects:

- [`Atom[T]`](../../state/state.go): `Get`, `Set`, `Update`, `ReactiveRegionSourceIDs`, `Text`
- [`Computed[T]`](../../state/state.go): `Get`
- [`Derived[T]`](../../state/state.go): `Get`, `ReactiveRegionSourceIDs`, `Text`
- [`Snapshot`](../../state/state.go): `Select`

Storage constants:

- `LocalStorage`
- `SessionStorage`

## fetch

Import with `import "github.com/monstercameron/GoWebComponents/fetch"`.

Source anchors:

- [fetch/doc.go](../../fetch/doc.go)
- [fetch/fetch.go](../../fetch/fetch.go)
- [fetch/cache.go](../../fetch/cache.go)
- [fetch/cache_persistent.go](../../fetch/cache_persistent.go)
- [fetch/mutation_queue.go](../../fetch/mutation_queue.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `UseFetch` | low-level raw fetch state by URL | [`Options`](../../fetch/fetch.go), [`Resource`](../../fetch/fetch.go), [`State`](../../fetch/fetch.go) | `resource := fetch.UseFetch("/api/orders")` |
| `UseResource` | typed async loader with cancellation and dependency reloads | [`AsyncResource[T]`](../../fetch/fetch.go), [`ResourceState[T]`](../../fetch/fetch.go) | `userResource := fetch.UseResource(loadUser, userID)` |
| `UseCachedResource`, `InvalidateResource`, `DisposeResource`, `InspectCachedResources`, `SweepCachedResources`, `LoadCached`, `ConfigurePersistentCache`, `RestoreCacheBootstrap` | shared cached async state, invalidation, resume and inspection | [`CachedResource[T]`](../../fetch/cache.go), [`CachedResourceState[T]`](../../fetch/cache.go), [`CacheOptions`](../../fetch/cache.go), [`PersistentCacheOptions`](../../fetch/cache_persistent.go), [`CacheBootstrap`](../../fetch/cache_persistent.go) | `orders := fetch.UseCachedResource("orders:list", loadOrders)` |
| `Fetch`, `Upload`, `ReturnChannel` | low-level request and upload primitives | [`Options`](../../fetch/fetch.go), [`Result`](../../fetch/fetch.go), [`UploadUpdate`](../../fetch/fetch.go), [`MultipartBody`](../../fetch/fetch.go), [`MultipartFile`](../../fetch/fetch.go) | `resultCh := fetch.Fetch("/api/export", fetch.Options{Method: "POST"})` |
| `OpenMutationQueue`, `NewMutationConflict`, `GetMutationConflict`, `AsMutationConflictError`, `IsMutationConflict` | durable offline write replay and conflict handling | [`MutationQueue`](../../fetch/mutation_queue.go), [`MutationQueueOptions`](../../fetch/mutation_queue.go), [`MutationDraft`](../../fetch/mutation_queue.go), [`QueuedMutation`](../../fetch/mutation_queue.go), [`MutationReplayOptions`](../../fetch/mutation_queue.go), [`MutationConflict`](../../fetch/mutation_queue.go), [`MutationConflictResolution`](../../fetch/mutation_queue.go) | `queue, err := fetch.OpenMutationQueue()` |

Key fetch objects:

- [`Resource`](../../fetch/fetch.go): `Get`, `Refetch`
- [`AsyncResource[T]`](../../fetch/fetch.go): `Get`, `Reload`, `Cancel`
- [`CachedResource[T]`](../../fetch/cache.go): `Get`, `Reload`, `Cancel`, `Invalidate`, `Dispose`, `Set`, `Update`
- [`MutationQueue`](../../fetch/mutation_queue.go): `Enqueue`, `List`, `Remove`, `Clear`, `Replay`, `ReplayWithOptions`

Constants and keys:

- `CacheBootstrapDataKey`

## router

Import with `import "github.com/monstercameron/GoWebComponents/router"`.

Source anchors:

- [router/doc.go](../../router/doc.go)
- [router/router.go](../../router/router.go)
- [router/router_api.go](../../router/router_api.go)
- [router/router_state.go](../../router/router_state.go)
- [router/contracts.go](../../router/contracts.go)
- [router/metadata.go](../../router/metadata.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `NewHashRouter`, `NewHistoryRouter` | create one router instance with hash or history semantics | [`Router`](../../router/router.go), [`RouterOptions`](../../router/router.go) | `appRouter := router.NewHistoryRouter()` |
| `(*Router).Register`, `RegisterRoute`, `(*Router).Mount`, `(*Router).HydrateMount`, `(*Router).MountElement`, `(*Router).HydrateMountElement`, `(*Router).Current`, `(*Router).IsLoading`, `(*Router).Navigate`, `(*Router).NavigateReplace`, `(*Router).Revalidate`, `(*Router).GetCurrentRouterPath` | route registration, mount, navigation, route rendering, loader refresh | [`Options`](../../router/router.go), [`Attrs`](../../router/router.go), [`RouteContext`](../../router/router.go) | `appRouter.Register("/inventory/:sku", renderInventoryPage, router.Options{Loader: loadInventory})` |
| `Navigate`, `NavigateReplace`, `Revalidate`, `GetCurrentPath`, `UseNavigate`, `UseRevalidator` | global navigation and loader refresh helpers inside components | [`Navigator`](../../router/router.go), [`Revalidator`](../../router/router.go) | `router.UseNavigate().Navigate("/checkout")` |
| `UseQuery`, `UseSearchParams`, `UseParams`, `UseRouteData`, `GetOutlet`, `GetRoute`, `GetRouter`, `InspectCurrentRoute`, `RegisterElementRoute` | access current route state, outlets, params, query, loader attrs, and route inspection | [`Query`](../../router/router.go), [`SearchParams`](../../router/router.go), [`Params`](../../router/router.go), [`RouteInspection`](../../router/router.go) | `sku := router.UseParams().Get("sku")` |
| `DefineRoute`, `MustDefineRoute` | validate patterns and build typed reverse-routing contracts | [`RouteContract`](../../router/contracts.go), [`RouteParamsProvider`](../../router/contracts.go), [`RouteQueryProvider`](../../router/contracts.go) | `inventoryRoute := router.MustDefineRoute("/inventory/:sku")` |
| `BuildMetadataNode` | render narrow head metadata from route metadata | [`Metadata`](../../router/metadata.go) | `headNode := router.BuildMetadataNode(metadata)` |
| `PreserveReturnTo`, `ReadReturnTo`, `AllowNavigation`, `BlockNavigation`, `RedirectNavigation` | return-to preservation and guard result helpers | [`GuardResult`](../../router/router.go), [`GuardDecision`](../../router/router.go), [`GuardFunc`](../../router/router.go), [`AsyncGuardFunc`](../../router/router.go), [`LeaveGuardFunc`](../../router/router.go), [`AsyncLeaveGuardFunc`](../../router/router.go) | `return router.BlockNavigation("unsaved changes")` |

Key router handles:

- [`Navigator`](../../router/router.go): `Navigate`, `Replace`
- [`Revalidator`](../../router/router.go): `Revalidate`, `Loading`
- [`Query`](../../router/router.go): `Get`, `Has`, `Values`, `Encode`
- [`SearchParams`](../../router/router.go): `Get`, `Has`, `Values`, `Encode`, `Set`, `Delete`, `Replace`, `Navigate`, `ReplaceAll`
- [`Params`](../../router/router.go): `Get`, `Has`, `Values`, `Int`, `Bool`
- [`RouteContract`](../../router/contracts.go): `Pattern`, `ParamNames`, `Path`, `MustPath`, `Href`, `MustHref`, `PathFor`, `MustPathFor`, `HrefFor`, `MustHrefFor`

## interop

Import with `import "github.com/monstercameron/GoWebComponents/interop"`.

Source anchors:

- [interop/doc.go](../../interop/doc.go)
- [interop/interop.go](../../interop/interop.go)
- [interop/interop_channels.go](../../interop/interop_channels.go)
- [interop/interop_client_protocol.go](../../interop/interop_client_protocol.go)
- [interop/interop_module.go](../../interop/interop_module.go)
- [interop/worker_pool.go](../../interop/worker_pool.go)
- [interop/persistence_wasm.go](../../interop/persistence_wasm.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `GlobalThis`, `GetGlobalThis`, `GetDocument`, `CurrentDocument`, `GetWindowEnv`, `SharedWindowEnv`, `GetWindowLocation`, `GetWindowHistory`, `GetDocumentEvents`, `GetWindowEvents`, `GetMediaQuery` | raw-but-typed browser handle access without dropping into `syscall/js` | [`Value`](../../interop/value_wasm.go), [`Document`](../../interop/interop.go), [`WindowEnv`](../../interop/interop.go), [`Location`](../../interop/interop.go), [`History`](../../interop/interop.go), [`EventTarget`](../../interop/interop.go), [`MediaQueryList`](../../interop/interop.go) | `windowEnv, err := interop.GetWindowEnv()` |
| `GetLocalStorage`, `LocalStorage`, `GetSessionStorage`, `SessionStorage`, `OpenPersistentStore`, `LoadPersistentJSON` | storage and durable browser persistence | [`Storage`](../../interop/interop.go), [`PersistentStore`](../../interop/interop.go), [`PersistentStoreOptions`](../../interop/interop.go) | `store, err := interop.OpenPersistentStore(ctx, interop.PersistentStoreOptions{Name: "gwc-cache"})` |
| `GetClipboard`, `NavigatorClipboard`, `ScheduleTimeout`, `SetTimeout`, `ScheduleInterval`, `SetInterval` | clipboard and timer helpers | [`Clipboard`](../../interop/interop.go), [`Timer`](../../interop/interop.go) | `timer, err := interop.ScheduleTimeout(250*time.Millisecond, handleFlush)` |
| `ImportModule`, `OpenWorker`, `OpenGoWASMWorker`, `NewGoWASMWorker`, `OpenWorkerPool`, `RequestWorkerDecoded`, `OpenMessageChannel`, `OpenSharedBuffer`, `GetSharedMemorySupport`, `GetWorkerScope` | dynamic modules, raw workers, worker pools, message ports, shared buffers | [`Module`](../../interop/interop.go), [`Worker`](../../interop/interop.go), [`WorkerOptions`](../../interop/interop.go), [`GoWASMWorkerOptions`](../../interop/interop.go), [`WorkerPool`](../../interop/interop.go), [`WorkerPoolOptions`](../../interop/worker_pool.go), [`MessageChannel`](../../interop/interop.go), [`SharedBuffer`](../../interop/interop.go), [`SharedMemorySupport`](../../interop/interop.go), [`WorkerScope`](../../interop/interop.go) | `worker, err := interop.OpenWorker(ctx, interop.WorkerOptions{URL: "/worker.js"})` |
| `OpenCrossTabChannel`, `OpenSecondaryWindowChannel`, `OpenWindowOpenerChannel`, `SubscribeDecoded`, `SubscribeDecodedCrossTab`, `SubscribeDecodedMessagePort`, `SubscribeDecodedWindow`, `SubscribeDecodedWorker`, `SubscribeClientMessages`, `SubscribeClientWindowMessages`, `SubscribeSurfaceSignals` | cross-tab, cross-window, message-port, and worker messaging subscriptions | [`CrossTabChannel`](../../interop/interop.go), [`CrossTabChannelOptions`](../../interop/interop.go), [`WindowChannel`](../../interop/interop.go), [`WindowChannelOptions`](../../interop/interop.go), [`Subscription`](../../interop/interop.go) | `channel, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{Name: "gwc-sync"})` |
| `PublishClientHello`, `PublishClientHelloWithCapabilities`, `PublishClientHelloWindow`, `PublishClientHelloWindowWithCapabilities`, `PublishClientGoodbye`, `PublishClientGoodbyeWindow`, `PublishClientMessage`, `PublishClientWindowMessage`, `PublishClientEvent`, `PublishClientIntent`, `PublishClientQuery`, `PublishClientResult`, `PublishClientInvalidation`, `PublishClientBinaryCrossTab`, `PublishClientBinaryWindow`, `PublishSurfaceSignal`, `PublishIntent`, `PublishLogout`, `PublishRouteFocus`, `PublishSelection`, `PublishSessionExpired` | public client-protocol and multi-surface signaling | [`ClientIdentity`](../../interop/interop.go), [`ClientCapabilities`](../../interop/interop.go), [`ClientMessage`](../../interop/interop.go), [`SurfaceSignal`](../../interop/interop.go), [`SurfaceIntentSignal`](../../interop/interop.go) | `err = interop.PublishSelection(windowChannel, "catalog", sku, "table")` |
| `Decode`, `DecodeClientMessage`, `DecodeCrossTabEnvelope`, `DecodeCustomEvent`, `DecodeMessagePortMessage`, `DecodeWindowEnvelope`, `DecodeSurfaceSignal`, `DecodeWorkerMessage`, `AsError`, `CodeOf`, `IsCode`, `ClientProtocolCompatible`, `ClientSupportsEncoding`, `ClientSupportsTopic`, `ClientCanExchange` | payload decoding, protocol compatibility, and typed error inspection | [`Error`](../../interop/interop.go), [`ErrorCode`](../../interop/interop.go), [`DecodedWindowEnvelope[T]`](../../interop/interop.go), [`DecodedWorkerMessage[T]`](../../interop/interop.go), [`DecodedCrossTabEnvelope[T]`](../../interop/interop.go) | `if interop.IsCode(err, interop.CodeUnavailable) { return }` |

Important public protocol types:

- [`ClientIdentity`](../../interop/interop.go), [`ClientCapabilities`](../../interop/interop.go), [`ClientMessage`](../../interop/interop.go), [`ClientMessageKind`](../../interop/interop.go), [`ClientPayloadEncoding`](../../interop/interop.go)
- [`SurfaceSignal`](../../interop/interop.go), [`SurfaceSignalKind`](../../interop/interop.go), [`SurfaceRouteSignal`](../../interop/interop.go), [`SurfaceSelectionSignal`](../../interop/interop.go), [`SurfaceSessionSignal`](../../interop/interop.go)
- [`WindowEnvelope`](../../interop/interop.go), [`CrossTabEnvelope`](../../interop/interop.go), [`WorkerMessage`](../../interop/interop.go), [`MessagePortMessage`](../../interop/interop.go)
- `ClientPresenceTopic`

## i18n

Import with `import "github.com/monstercameron/GoWebComponents/i18n"`.

Source anchors:

- [i18n/doc.go](../../i18n/doc.go)
- [i18n/runtime.go](../../i18n/runtime.go)
- [i18n/bundle.go](../../i18n/bundle.go)
- [i18n/provider.go](../../i18n/provider.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `UseLocale`, `UseI18n`, `Provider` | locale state and translation runtime inside component trees | [`LocaleOptions`](../../i18n/runtime.go), [`LocaleState`](../../i18n/runtime.go), [`Runtime`](../../i18n/runtime.go), [`ProviderProps`](../../i18n/provider.go) | `runtime := i18n.UseI18n()` |
| `NewBundle`, `BundleFromSSRBootstrap` | create or rehydrate translation catalogs | [`Bundle`](../../i18n/bundle.go), [`BundleOptions`](../../i18n/bundle.go), [`SSRBootstrapOptions`](../../i18n/bundle.go) | `bundle := i18n.NewBundle()` |
| `FormatDate`, `FormatNumber`, `NormalizeLocale`, `DirectionForLocale` | locale formatting and normalization helpers outside a runtime instance | [`DateOptions`](../../i18n/runtime.go), [`NumberOptions`](../../i18n/runtime.go), [`Direction`](../../i18n/runtime.go) | `label := i18n.FormatDate(locale, createdAt)` |
| `ResolvePath`, `PrefixPath` | locale-aware route/path shaping | [`RouteOptions`](../../i18n/runtime.go), [`ResolvedPath`](../../i18n/runtime.go) | `href := i18n.PrefixPath(locale, "/pricing", i18n.RouteOptions{})` |

Key i18n objects:

- [`Bundle`](../../i18n/bundle.go): `Register`, `RegisterNamespace`, `Translate`, `Locales`, `DefaultLocale`, `FallbackLocale`, `ToSSRBootstrap`
- [`LocaleState`](../../i18n/runtime.go): `Get`, `Set`, `Direction`, `FallbackLocale`, `SupportedLocales`
- [`Runtime`](../../i18n/runtime.go): `T`, `Locale`, `SetLocale`, `Direction`, `FormatDate`, `FormatNumber`, `PrefixPath`

## devtools

Import with `import "github.com/monstercameron/GoWebComponents/devtools"`.

Source anchors:

- [devtools/doc.go](../../devtools/doc.go)
- [devtools/types.go](../../devtools/types.go)
- [devtools/devtools_wasm.go](../../devtools/devtools_wasm.go)
- [devtools/bug_capture.go](../../devtools/bug_capture.go)
- [devtools/trace_capture.go](../../devtools/trace_capture.go)
- [devtools/support_bundle.go](../../devtools/support_bundle.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `Panel`, `ErrorOverlay` | embeddable devtools overlay UI | [`PanelProps`](../../devtools/types.go), [`ErrorOverlayProps`](../../devtools/types.go) | `return devtools.Panel(devtools.PanelProps{})` |
| `SnapshotNow`, `UseSnapshot`, `CompareSnapshots` | current inspection snapshots and diffing | [`Snapshot`](../../devtools/types.go), [`SnapshotComparison`](../../devtools/types.go) | `snapshot := devtools.SnapshotNow()` |
| `CaptureBugBundle`, `ImportBugCaptureBundleJSON`, `ExportBugCaptureBundleJSON`, `ReplayBugCaptureBundle` | bug-capture bundle capture, export, import, replay | [`BugCaptureBundle`](../../devtools/bug_capture.go) | `bundle := devtools.CaptureBugBundle("checkout-crash")` |
| `CaptureSupportDiagnosticBundle`, `ImportSupportDiagnosticBundleJSON`, `ExportSupportDiagnosticBundleJSON`, `SanitizeBugCaptureBundleForSupport` | support-safe diagnostic bundle flows | [`SupportDiagnosticBundle`](../../devtools/support_bundle.go) | `supportBundle := devtools.CaptureSupportDiagnosticBundle("sync-failure")` |
| `CaptureTrace`, `ImportTraceCaptureJSON`, `ExportTraceCaptureJSON`, `SetTraceReplay`, `CurrentTraceReplay`, `ClearTraceReplay` | trace capture and replay | [`TraceCapture`](../../devtools/trace_capture.go) | `trace := devtools.CaptureTrace("cold-start")` |
| `InspectCoordination`, `SetCoordinationInspection`, `ResetCoordinationInspection`, `InspectMultiClient`, `SetMultiClientInspection`, `ResetMultiClientInspection`, `InspectSerializationBoundaries`, `SetSerializationBoundaryInspection`, `ResetSerializationBoundaryInspection`, `InspectExtensionSections`, `SetExtensionSections`, `ResetExtensionSections`, `InspectErrorOverlayActions`, `SetErrorOverlayActions`, `ResetErrorOverlayActions`, `InspectBootstrapBoundaries` | explicit inspection state injection and reset hooks for diagnostics surfaces | [`Coordination`](../../devtools/coordination_state.go), [`MultiClient`](../../devtools/multi_client_state.go), [`BoundaryInspection`](../../devtools/serialization_boundaries.go), [`ExtensionSection`](../../devtools/extension_sections.go), [`ErrorOverlayAction`](../../devtools/error_overlay_actions.go) | `devtools.SetMultiClientInspection(state)` |

Key devtools objects:

- [`Snapshot`](../../devtools/types.go)
- [`BugCaptureBundle`](../../devtools/bug_capture.go)
- [`SupportDiagnosticBundle`](../../devtools/support_bundle.go)
- [`TraceCapture`](../../devtools/trace_capture.go)
- [`BoundaryInspection`](../../devtools/serialization_boundaries.go)

## head

Import with `import "github.com/monstercameron/GoWebComponents/head"`.

Source anchors:

- [head/doc.go](../../head/doc.go)
- [head/head.go](../../head/head.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `Compose`, `MetaName`, `MetaProperty`, `LinkRel`, `Hreflang`, `Robots`, `OpenGraph`, `Twitter`, `SocialTags`, `AlternateLinks`, `ResourceHints` | SSR head markup layered on top of router metadata | [`Document`](../../head/head.go), [`SocialMetadata`](../../head/head.go), [`AlternateLink`](../../head/head.go), [`ResourceHint`](../../head/head.go) | `headNode := head.Compose(routeMetadata, head.Robots("index,follow"))` |
| `RenderJSONLD`, `RenderToString`, `Merge`, `Resolve` | JSON-LD emission, head string rendering, layered document resolution | [`Document`](../../head/head.go), [`JSONLDBlock`](../../head/head.go), [`MergeOptions`](../../head/head.go), [`RouteLayer`](../../head/head.go) | `document := head.Resolve(layoutLayer, pageLayer)` |

## pwa

Import with `import "github.com/monstercameron/GoWebComponents/pwa"`.

Source anchors:

- [pwa/doc.go](../../pwa/doc.go)
- [pwa/manifest.go](../../pwa/manifest.go)
- [pwa/service_worker.go](../../pwa/service_worker.go)
- [pwa/installability.go](../../pwa/installability.go)
- [pwa/cache_storage.go](../../pwa/cache_storage.go)
- [pwa/release_manifest.go](../../pwa/release_manifest.go)
- [pwa/diagnostics.go](../../pwa/diagnostics.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `MarshalManifestJSON`, `MarshalManifestJSONIndented` | normalized and validated manifest JSON emission | [`Manifest`](../../pwa/manifest.go), [`ManifestImage`](../../pwa/manifest.go), [`ManifestShortcut`](../../pwa/manifest.go), [`RelatedApplication`](../../pwa/manifest.go) | `manifestJSON, err := pwa.MarshalManifestJSON(manifest)` |
| `RegisterServiceWorker`, `BuildServiceWorkerAssetPlan`, `ParseWasmReleaseManifestJSON` | service worker registration and release-manifest planning | [`ServiceWorkerOptions`](../../pwa/service_worker.go), [`ServiceWorkerRegistration`](../../pwa/service_worker.go), [`ServiceWorkerAssetPlan`](../../pwa/release_manifest.go), [`ServiceWorkerAssetPlanOptions`](../../pwa/release_manifest.go), [`WasmReleaseManifest`](../../pwa/release_manifest.go) | `registration, err := pwa.RegisterServiceWorker(ctx, options)` |
| `OpenCacheStorageManager`, `BuildCacheStoragePlan` | cache storage inspection and asset planning | [`CacheStorageManager`](../../pwa/cache_storage.go), [`CacheStoragePlan`](../../pwa/cache_storage.go), [`CacheStoragePlanOptions`](../../pwa/cache_storage.go) | `manager, err := pwa.OpenCacheStorageManager()` |
| `ObserveInstallability` | install-prompt state and prompt handling | [`InstallabilityManager`](../../pwa/installability.go), [`InstallabilityOptions`](../../pwa/installability.go), [`InstallabilityState`](../../pwa/installability.go) | `installManager, err := pwa.ObserveInstallability(options)` |
| `InspectDiagnostics` | PWA diagnostics snapshotting | [`DiagnosticsOptions`](../../pwa/diagnostics.go), [`DiagnosticsSnapshot`](../../pwa/diagnostics.go) | `diagnostics, err := pwa.InspectDiagnostics(ctx)` |

Key PWA objects:

- [`Manifest`](../../pwa/manifest.go): `Normalized`, `Validate`
- [`ServiceWorkerRegistration`](../../pwa/service_worker.go): `Snapshot`, `BackgroundSyncCapabilities`, `RegisterSync`, `Update`, `Unregister`, `SkipWaiting`, `SubscribeLifecycle`, `ReloadOnControllerChange`
- [`InstallabilityManager`](../../pwa/installability.go): `State`, `Prompt`, `Subscribe`

## plugin

Import with `import "github.com/monstercameron/GoWebComponents/plugin"`.

Source anchors:

- [plugin/doc.go](../../plugin/doc.go)
- [plugin/plugin.go](../../plugin/plugin.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `Define`, `NewHost` | define plugins and create an application-owned host | [`Manifest`](../../plugin/plugin.go), [`HostOptions`](../../plugin/plugin.go), [`Plugin`](../../plugin/plugin.go), [`Host`](../../plugin/plugin.go) | `host := plugin.NewHost(plugin.HostOptions{Capabilities: []plugin.Capability{plugin.CapabilityRouter}})` |
| `Allow`, `Block`, `Redirect` | route-guard decisions | [`GuardDecision`](../../plugin/plugin.go), [`GuardOutcome`](../../plugin/plugin.go), [`RouteRequest`](../../plugin/plugin.go), [`RouteGuard`](../../plugin/plugin.go) | `return plugin.Redirect("/login", "session expired")` |
| `(*Host).Register`, `Close`, `Capabilities`, `Plugins`, `SetValue`, `Value` | plugin lifecycle and shared host values | [`Host`](../../plugin/plugin.go) | `err := host.Register(searchPlugin)` |
| `(*Host).AddRouteGuard`, `EvaluateRoute`, `AddNavigationObserver`, `NotifyNavigation` | route and navigation extension points | [`NavigationEvent`](../../plugin/plugin.go), [`NavigationObserver`](../../plugin/plugin.go) | `decision := host.EvaluateRoute(plugin.RouteRequest{Path: "/admin"})` |
| `(*Host).AddCacheKeyDecorator`, `DecorateCacheKey`, `AddRequestObserver`, `NotifyRequest` | cache and request extension points | [`CacheKeyDecorator`](../../plugin/plugin.go), [`RequestEvent`](../../plugin/plugin.go), [`RequestObserver`](../../plugin/plugin.go) | `decoratedKey := host.DecorateCacheKey("orders:list")` |
| `(*Host).AddPanelProvider`, `Panels`, `AddHeadProvider`, `HeadNodes`, `AddBootstrapProvider`, `BootstrapData`, `AddFormValidator`, `ValidateForm`, `AddSubmitObserver`, `NotifySubmit` | devtools panels, head nodes, bootstrap payloads, and form validators | [`Panel`](../../plugin/plugin.go), [`PanelProvider`](../../plugin/plugin.go), [`HeadProvider`](../../plugin/plugin.go), [`BootstrapPayload`](../../plugin/plugin.go), [`BootstrapProvider`](../../plugin/plugin.go), [`FormSubmission`](../../plugin/plugin.go), [`ValidationIssue`](../../plugin/plugin.go), [`FormValidator`](../../plugin/plugin.go), [`SubmitObserver`](../../plugin/plugin.go) | `issues := host.ValidateForm(plugin.FormSubmission{ID: "checkout"})` |

Plugin enums and compatibility markers:

- `TierStable`, `TierSupportedCompanion`, `TierExperimental`, `TierInternal`
- `CapabilityRouter`, `CapabilityAsyncData`, `CapabilityDevtools`, `CapabilitySSR`, `CapabilityForms`

## virtualization

Import with `import "github.com/monstercameron/GoWebComponents/virtualization"`.

Source anchors:

- [virtualization/list.go](../../virtualization/list.go)
- [virtualization/viewport.go](../../virtualization/viewport.go)
- [virtualization/observe_wasm.go](../../virtualization/observe_wasm.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `List` | fixed-height virtualized lists with an owned scroll container | [`ListProps[T]`](../../virtualization/list.go), [`RowRenderProps[T]`](../../virtualization/list.go) | `return virtualization.List(virtualization.ListProps[item]{ID: "orders", Items: items, Height: 480, RowHeight: 40, ItemKey: itemKey, RenderRow: renderRow})` |
| `ComputeViewportState`, `ObserveOwnedViewport` | viewport math and scroll ownership observation | [`ViewportConfig`](../../virtualization/viewport.go), [`ViewportState`](../../virtualization/viewport.go), [`ViewportDiagnostics`](../../virtualization/viewport.go), [`Subscription`](../../virtualization/observe_wasm.go), [`Range`](../../virtualization/viewport.go) | `viewportState, err := virtualization.ComputeViewportState(config, scrollTop, height)` |

Key virtualization objects:

- [`ViewportState`](../../virtualization/viewport.go): `Diagnostics`
- [`ListProps[T]`](../../virtualization/list.go): `ID`, `Items`, `Height`, `RowHeight`, `Overscan`, `ItemKey`, `RenderRow`

## prerender

Import with `import "github.com/monstercameron/GoWebComponents/prerender"`.

Source anchors:

- [prerender/export.go](../../prerender/export.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `Export` | static HTML and bootstrap export to an output directory | [`Route`](../../prerender/export.go), [`Target`](../../prerender/export.go), [`RouteOutput`](../../prerender/export.go), [`ExportSummary`](../../prerender/export.go) | `summary, err := prerender.Export("./dist", routes)` |

## diagnostics

Import with `import "github.com/monstercameron/GoWebComponents/diagnostics"`.

Source anchors:

- [diagnostics/diagnostics.go](../../diagnostics/diagnostics.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `NewReport`, `Emit`, `WriteHTTPError` | structured diagnostics report construction and HTTP emission | [`Options`](../../diagnostics/diagnostics.go), [`Report`](../../diagnostics/diagnostics.go) | `diagnostics.WriteHTTPError(w, 500, diagnostics.NewReport(options))` |

## hotreload

Import with `import "github.com/monstercameron/GoWebComponents/hotreload"`.

Source anchors:

- [hotreload/doc.go](../../hotreload/doc.go)
- [hotreload/hotreload.go](../../hotreload/hotreload.go)
- [hotreload/hotreload_wasm.go](../../hotreload/hotreload_wasm.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `Enable`, `Configure`, `Disable`, `Enabled`, `IsEnabled`, `Prepare`, `GetSnapshot`, `ApplySnapshot` | development-state hot reload bridge | [`Config`](../../hotreload/hotreload.go) | `hotreload.Configure(hotreload.Config{AtomIDs: []string{"session"}, ResetKey: "layout-v2"})` |

## logging

Import with `import "github.com/monstercameron/GoWebComponents/logging"`.

Source anchors:

- [logging/doc.go](../../logging/doc.go)
- [logging/logging.go](../../logging/logging.go)
- [logging/browser_console_wasm.go](../../logging/browser_console_wasm.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `New`, `Log`, `AttachBrowserConsole` | structured logging and browser-console attachment | [`Logger`](../../logging/logging.go), [`Fields`](../../logging/logging.go), [`BrowserConsoleOptions`](../../logging/browser_console_wasm.go) | `logger := logging.New("checkout")` |

Key logging object:

- [`Logger`](../../logging/logging.go): `Scope`, `Log`, `Debug`, `Info`, `Warn`, `Error`

## utils

Import with `import "github.com/monstercameron/GoWebComponents/utils"` in js/wasm builds.

Source anchors:

- [utils/utils.go](../../utils/utils.go)
- [utils/utils_production_mode.go](../../utils/utils_production_mode.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `WaitForever` | keep a browser `main()` alive after mount | package-level helper | `utils.WaitForever()` |
| `EnableDebug`, `DisableDebug`, `ConfigureDebugNamespace`, `ConfigureDebugNamespaces`, `ConfigureDebugNamespacesExclusive`, `EnableAllDebug`, `DisableAllDebug`, `GetDebugStatus`, `ConfigureMemStatsSampleRate`, `GetMemStatsSampleRate` | development debug and memory-stat toggles | package-level helper state | `utils.ConfigureDebugNamespace("router", true)` |
| `EnableHotReload`, `IsHotReloadEnabled`, `InstallHotReloadBridge` | js/wasm hot-reload bridge convenience wrappers | package-level helper state | `utils.InstallHotReloadBridge("session", "draft")` |
| `EnableGoroutineMonitoring`, `DisableGoroutineMonitoring`, `ConfigureGoroutineThreshold`, `GetGoroutineStats`, `ResetGoroutineBaseline` | browser-side goroutine leak monitoring | package-level helper state | `utils.EnableGoroutineMonitoring()` |
| `WriteConsole`, `WriteConsoleStructured`, `ResolveDocumentURL` | browser-console output and document-relative URL resolution | package-level helper state | `utils.WriteConsoleStructured("info", "bootstrap", "mounted", nil)` |

## testkit/render

Import with `import render "github.com/monstercameron/GoWebComponents/testkit/render"`.

Source anchors:

- [testkit/render/doc.go](../../testkit/render/doc.go)
- [testkit/render/render_wasm.go](../../testkit/render/render_wasm.go)
- [testkit/render/async.go](../../testkit/render/async.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `New`, `WithQueuedScheduler` | create a js/wasm component fixture | [`Fixture`](../../testkit/render/render_wasm.go), [`Option`](../../testkit/render/render_wasm.go) | `fixture := render.New(t, render.WithQueuedScheduler())` |
| `BuildFailureError`, `BuildHydrationMismatchError`, `BuildLoaderFailureError`, `BuildRouteGuardFailureError`, `BuildCacheConflictError`, `BuildOfflineReplayError`, `ParallelSafetyContract` | failure helpers and test-lane contract messages | [`FailureError`](../../testkit/render/async.go) | `err := render.BuildLoaderFailureError("/users/42", "timeout")` |
| `NewResourceController` | deterministic async resource orchestration in tests | [`ResourceController[T]`](../../testkit/render/async.go), [`ResourceAttempt`](../../testkit/render/async.go) | `controller := render.NewResourceController[user]()` |

Important fixture surface:

- [`Fixture`](../../testkit/render/render_wasm.go): `Render`, `Rerender`, `Flush`, `FlushTimers`, `Stabilize`, `Cleanup`, query helpers such as `ByID`, `ByText`, `ByRole`, `ByLabel`, `ByDescription`, `ByLiveRegion`, interaction helpers such as `ClickByID`, `InputByID`, `ChangeByID`, `SubmitByID`, and diagnostics helpers such as `BuildDiagnostics`, `BuildLogs`, `BuildRenderCounts`

## testkit/hooks

Import with `import hooks "github.com/monstercameron/GoWebComponents/testkit/hooks"`.

Source anchors:

- [testkit/hooks/doc.go](../../testkit/hooks/doc.go)
- [testkit/hooks/hooks_wasm.go](../../testkit/hooks/hooks_wasm.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `RenderHook` | hook harness built on `testkit/render` | [`Harness[T]`](../../testkit/hooks/hooks_wasm.go) | `harness := hooks.RenderHook(t, func() int { return storeCount.Get() })` |

Key hook harness object:

- [`Harness[T]`](../../testkit/hooks/hooks_wasm.go): `Current`, `Rerender`, `Flush`, `Act`, `Cleanup`

## testkit/router

Import with `import routertest "github.com/monstercameron/GoWebComponents/testkit/router"`.

Source anchors:

- [testkit/router/doc.go](../../testkit/router/doc.go)
- [testkit/router/router_wasm.go](../../testkit/router/router_wasm.go)
- [testkit/router/guard_failures.go](../../testkit/router/guard_failures.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `NewHash`, `NewHistory` | router fixture creation | [`Fixture`](../../testkit/router/router_wasm.go) | `fixture := routertest.NewHistory(t)` |
| `BuildGuardBlocked`, `BuildGuardRedirect`, `BuildAsyncGuardDenied`, `NewLoaderController` | guard and loader test scaffolding | [`LoaderController`](../../testkit/router/loader_controller.go), [`LoaderAttempt`](../../testkit/router/loader_controller.go) | `guard := routertest.BuildGuardBlocked("blocked in test")` |

Important router fixture surface:

- [`Fixture`](../../testkit/router/router_wasm.go): `Register`, `SetPath`, `Render`, `Navigate`, `Replace`, `Inspect`, `Path`, `Query`, `Params`, `Router`, query helpers, interaction helpers, `Text`, `Cleanup`

## testkit/ssr

Import with `import ssr "github.com/monstercameron/GoWebComponents/testkit/ssr"`.

Source anchors:

- [testkit/ssr/doc.go](../../testkit/ssr/doc.go)
- [testkit/ssr/ssr.go](../../testkit/ssr/ssr.go)
- [testkit/ssr/hydrate_wasm.go](../../testkit/ssr/hydrate_wasm.go)

| Surface | Use it for | Parameter objects / handles | Call shape |
| --- | --- | --- | --- |
| `Render`, `RequirePayload`, `LoadStaticExport` | SSR snapshots, payload assertions, static export inspection | [`Snapshot`](../../testkit/ssr/ssr.go), [`StaticExport`](../../testkit/ssr/ssr.go), [`ExportedRoute`](../../testkit/ssr/ssr.go), [`StructuredSnapshot`](../../testkit/ssr/ssr.go) | `snapshot := ssr.Render(t, ui.CreateElement(renderApp, nil))` |
| `SmokeHydrate`, `RoundTripHydrate`, `RoundTripHydrateMismatch` | hydration smoke and mismatch harnesses | [`HydrationHarness`](../../testkit/ssr/hydrate_wasm.go), [`HydrationOptions`](../../testkit/ssr/hydrate_wasm.go) | `harness := ssr.RoundTripHydrate(t, ui.CreateElement(renderApp, nil))` |

Key SSR test objects:

- [`Snapshot`](../../testkit/ssr/ssr.go): `Contains`, `Structured`
- [`HydrationHarness`](../../testkit/ssr/hydrate_wasm.go): `ByID`, `ByText`, `Text`, `Cleanup`

## Validation

Use quick package-level checks rather than a full suite when you are validating this browser chapter:

```powershell
go doc github.com/monstercameron/GoWebComponents/ui
go doc github.com/monstercameron/GoWebComponents/html
go doc github.com/monstercameron/GoWebComponents/fetch
go doc github.com/monstercameron/GoWebComponents/state
go doc github.com/monstercameron/GoWebComponents/router
go doc github.com/monstercameron/GoWebComponents/interop
```

For test support and companion packages:

```powershell
go doc github.com/monstercameron/GoWebComponents/devtools
go doc github.com/monstercameron/GoWebComponents/head
go doc github.com/monstercameron/GoWebComponents/pwa
go doc github.com/monstercameron/GoWebComponents/plugin
go doc github.com/monstercameron/GoWebComponents/i18n
go doc github.com/monstercameron/GoWebComponents/testkit/render
```

## Topic Pagination

Topic 16 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.

- Previous topic: [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [01 Getting Started](01-getting-started.md)
