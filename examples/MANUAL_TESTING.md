# Example Manual Testing Guide

Location: `examples/`

This document is the developer-facing manual verification guide for the numbered example catalog.

Use it when you are:

- changing public APIs and need to confirm examples still demonstrate the intended behavior
- updating example styling, layout, hydration, or routing behavior
- deciding whether a regression belongs in a dedicated Playwright spec or only in the broad smoke suite
- doing release verification before demoing the examples catalog

## Current Browser Baseline

Captured on `2026-03-16`.

- `npx playwright test tests/catalog-smoke.spec.ts` passed for `75` example pages.
- The smoke suite covers every buildable numbered example page that is served directly from the examples catalog.
- `18-ssr-server-routing` is not part of the generic catalog smoke because the real behavior requires its standalone Go server.
- Dedicated example specs currently cover `01`, `02`, `05`, `06`, `07`, `08`, `10`, `12`, `13`, `16`, `17`, `18`, `19`, and `20`.
- The last dedicated-spec baseline produced `21` passes and `6` failures.

## Automation Delta Worth Knowing

- `07-goroutines` has stale dedicated assertions around intermediate status text and overly broad `.font-mono` selectors; the example page itself still loads and the smoke suite passes.
- `08-fetch` has stale dedicated assertions that collide with duplicate visible text and a detail-error path that is no longer deterministic under the current UI.
- `10-advanced-form` has a stale dedicated assertion expecting the exact text `Running async validation`.
- `13-browser-compiler` passes its dedicated spec, but it may use the mock compiler fallback if the real compiler asset path is unavailable.

Treat those four examples as manual-priority pages until their focused specs are refreshed.

## Environment

From the repo root:

```powershell
npm run dev:examples
```

Main catalog URLs:

- `http://127.0.0.1:8090/examples`
- `http://127.0.0.1:8090/examples/01-counter/counter.html`

Standalone SSR server for `18-ssr-server-routing`:

```powershell
Set-Location .\examples
.\build.ps1 -Example "18-ssr-server-routing"
Set-Location ..
go run ./examples/18-ssr-server-routing
```

Standalone SSR server URLs:

- `http://127.0.0.1:8079/`
- `http://127.0.0.1:8079/docs/ssr`
- `http://127.0.0.1:8079/search?q=routing`
- `http://127.0.0.1:8079/secure`
- `http://127.0.0.1:8079/secure?auth=true&role=maintainer`

## Manual Verification Rules

- Confirm the page shell is dark mode and the primary heading renders without console or page errors.
- Exercise at least one state change, route change, async transition, or overlay action on every page.
- Verify the visible output changes in a way that matches the example's teaching goal.
- If the example demonstrates routing or hydration, also verify refresh and direct navigation behavior where applicable.
- If an example exposes stats, badges, or status copy, verify those values change together with the primary interaction.

## Manual Checklist

### Integrated Apps

- `01-counter`: Click increment twice, decrement once, and reset. Expected: the count changes `0 -> 2 -> 1 -> 0` and the stat card stays in sync.
- `02-text-input`: Type text, wait for the debounce window, then clear it. Expected: live preview updates immediately, debounced preview catches up, and both counts return to zero on clear.
- `03-toggle`: Toggle on and off several times. Expected: the visual state and any boolean label remain synchronized with each click.
- `04-form`: Fill the sample fields and submit. Expected: controlled inputs hold their values and the form output or summary reflects the submitted data.
- `05-todo-basic`: Add several todos, remove one, then clear all. Expected: the list count, rendered items, and empty state remain consistent.
- `06-todo-advanced`: Add a todo with priority and category, mark one completed, then switch between `All`, `Active`, and `Completed`. Expected: badges render correctly and filters show the right subset.
- `07-goroutines`: Start the background task, cancel another run midway, then start and reset the timer. Expected: progress reaches completion, cancel stops progress, and timer controls update the clock display correctly.
- `08-fetch`: Let the initial async directory load, switch detail views, use reload and cancel controls, and retry any surfaced error. Expected: list content, detail panel, and async status boundaries update cleanly without blank states.
- `09-atoms`: Trigger shared state updates from more than one control. Expected: every bound view updates together and the page stays mounted after interaction.
- `10-advanced-form`: Submit invalid data first, then fix it and submit valid data. Expected: validation feedback appears on the relevant fields and the success state only appears after valid submission.
- `11-blog`: Navigate the blog landing experience and interact with any article or CTA affordances. Expected: the content layout remains readable and the route or selection state changes visibly.
- `12-portfolio-site`: Verify hero navigation, docs navigation, and one interactive mini-app. Expected: section links, route transitions, and embedded interactions all stay responsive.
- `13-browser-compiler`: Open the browser compiler UI, wait for the compilation pipeline to finish, and run the demo. Expected: the terminal output contains the browser-compiler success copy and the UI does not hang during compile or fallback.
- `14-omi`: Load the OMI example and exercise its primary interaction path. Expected: the larger shell renders correctly in dark mode and the embedded UI updates without console errors.
- `15-calculator`: Enter several expressions, switch between `Graphite` and `Midnight`, and verify the result and memory state update. Expected: evaluation output is correct and no light theme is available anymore.
- `16-devtools`: Open the embedded devtools panel and inspect diagnostics or profiling output. Expected: tree, hook, or diagnostic surfaces render and update with the app state.
- `17-ssr-routing`: Load the static SSR page, confirm prerendered docs content appears before interaction, then navigate through docs, search, secure, legacy redirect, and sign-in flows. Expected: hydration resumes the shell cleanly and route features continue working after startup.
- `18-ssr-server-routing`: Run the standalone server and test direct navigation, refresh, search query rendering, secure redirect behavior, and authenticated secure access. Expected: each direct URL returns real HTML first and then hydrates without replacing the whole shell unnecessarily.
- `19-nested-routes`: Jump directly into dashboard settings, switch between `Profile` and `Team`, then move into docs and report routes. Expected: parent layout shells stay mounted while only the outlet content changes.
- `20-portals`: Open the modal, confirm it renders under `#portal-root`, then toggle tooltip and popover. Expected: overlays exist only in the portal container and clean up correctly on close or dismiss.

### ui Package

- `21-ui-render`: Load the page and use the primary controls. Expected: the rendered subtree mounts into `#app` and updates visibly after interaction.
- `22-create-element`: Exercise the example's main control path. Expected: the page demonstrates `ui.CreateElement` composition and the rendered result changes without remount glitches.
- `23-fragment`: Trigger the rendered fragment output. Expected: sibling content updates without an extra wrapper node affecting layout.
- `24-use-ref`: Interact with the ref-driven control, usually a focus or imperative read. Expected: the DOM element responds through the ref-backed action.
- `25-use-previous`: Change the tracked value more than once. Expected: the page shows both the current and previous value correctly.
- `26-use-deferred-value`: Type or change input rapidly. Expected: the immediate value changes first and the deferred value lags behind before settling.
- `27-transition-hooks`: Trigger the transition workload. Expected: pending UI appears during the transition and the final state commits when work completes.
- `28-use-reducer`: Dispatch multiple actions. Expected: reducer-driven state changes are deterministic and all derived stats match the action history.
- `29-use-debounced`: Type quickly, then stop. Expected: debounced output changes only after the delay window elapses.
- `30-use-throttled`: Trigger repeated updates rapidly. Expected: throttled output updates at the expected cadence rather than on every event.
- `31-context-api`: Change provider state and verify consumers update. Expected: every consumer reflects the same contextual value.
- `32-async-boundary`: Trigger the async boundary's loading and completion states. Expected: fallback content appears first and resolves into the final content without a full-page flash.
- `33-lazy`: Activate the lazy-loaded content. Expected: the loading state is visible briefly and the lazy component mounts once resolved.
- `34-error-boundary`: Trigger the deliberate error path and then recover if the example supports it. Expected: the fallback UI catches the error instead of breaking the full page.
- `35-use-id`: Refresh and interact with the page. Expected: generated IDs remain stable within a render and wire labels or inputs correctly.
- `36-typed-events`: Use the exposed controls that rely on typed events. Expected: event payload handling updates the page correctly without raw event casting errors.
- `46-raw-handler`: Use the prebuilt raw handler interaction path. Expected: the raw event wiring works through the forwarded handler value and does not panic.
- `47-portal-selector`: Open the selector-targeted portal content. Expected: overlay content appears in the selected DOM target and disappears cleanly.
- `48-portal-target`: Trigger the explicit target-node portal flow. Expected: content renders into the provided node target rather than the normal component subtree.
- `49-use-channel`: Start the channel-backed interaction. Expected: produced messages or values arrive in order and the UI stays responsive while receiving them.
- `50-use-task`: Trigger the background task. Expected: task state moves through idle, running, and completion or cancellation states visibly.
- `51-use-form`: Fill the form, submit, and reset if available. Expected: form state, validation, and submit result all stay synchronized.
- `70-render-to-string`: Load the instruction page, then run the standalone server variant if you need the full request-time render flow. Expected: the instructional shell renders in dark mode and the server-backed version shows exact HTML string output beside the preview.
- `71-hydrate`: Verify prerendered markup is visible immediately, then click the buttons after wasm starts. Expected: hydration resumes the existing DOM and later updates stay interactive.
- `73-ssr-bootstrap`: Load the static bootstrap page and verify the inline bootstrap content resumes into the hydrated UI. Expected: inline JSON data is reused and the prerendered content survives startup.
- `76-use-effect`: Trigger dependency changes and cleanup behavior. Expected: effect-run and cleanup counters move in the expected order and document-side effects stay in sync.

### state Package

- `75-use-state`: Click the controls that call `Set` and `Update`. Expected: counter and message state both change exactly as described by the stat cards.
- `37-use-atom`: Change atom state from the provided controls. Expected: all subscribers reflect the same shared value immediately.
- `38-use-computed`: Change the source inputs. Expected: computed output recalculates from the atom inputs and never requires manual refresh.
- `39-use-derived`: Update the parent value. Expected: derived state follows the source and the displayed dependency chain remains coherent.
- `40-snapshot-export-import`: Export a snapshot, mutate state, then import or restore the snapshot. Expected: the state graph returns to the saved version accurately.
- `41-snapshot-storage`: Save a snapshot to storage, change state, and reload or restore it. Expected: persisted state survives the reload and restores into the expected view.

### fetch Package

- `42-use-fetch`: Trigger the managed fetch flow. Expected: loading, success, and error states transition through the hook-managed UI correctly.
- `43-use-resource`: Load resource-backed content and retry if needed. Expected: the resource cache and fallback behavior behave predictably across rerenders.
- `44-use-cached-resource`: Revisit or retrigger the same fetch path. Expected: cached content is reused instead of showing the cold-loading path every time.
- `45-fetch-imperative`: Invoke the imperative fetch action repeatedly. Expected: responses update the UI on demand and do not require hook remounting.

### html Package

- `52-semantic-html`: Verify the semantic layout renders correct headings, landmarks, and content grouping. Expected: the DOM structure is semantically meaningful and visually intact.
- `53-html-forms`: Use the typed form controls. Expected: inputs, selects, and labels stay wired correctly through `html.Props`.
- `54-html-tag`: Inspect the dynamic tag output. Expected: the requested tag renders with the right children and attributes.

### router Package

- `55-hash-router`: Navigate through the hash-based routes using links and browser history. Expected: the hash changes and the rendered view follows the route.
- `56-browser-router`: Navigate with the history router. Expected: path-based route changes update the UI without full page reloads.
- `57-use-navigate`: Trigger programmatic navigation. Expected: the route changes through `router.UseNavigate` and the destination view renders immediately.
- `58-route-params`: Navigate between parameterized routes. Expected: the page reads the current route params and displays the correct value.
- `59-route-query`: Change query string inputs or route links. Expected: query-aware output updates when the search params change.
- `60-route-loaders`: Visit routes with loader-backed content. Expected: loading, loaded, and error states reflect the route loader lifecycle.
- `61-use-revalidator`: Trigger a revalidation after the initial route load. Expected: the route refreshes its data without losing surrounding route state.
- `62-router-redirects`: Open redirecting routes directly. Expected: the app lands on the redirected destination and the visible content matches the final route.
- `63-router-metadata`: Navigate across routes that set metadata. Expected: document title or related metadata changes with the current route.
- `64-nested-layout-routes`: Switch between layout children. Expected: the parent layout remains mounted while only the outlet subtree changes.
- `65-router-guards`: Attempt guarded navigation and leave flows. Expected: allowed routes proceed and guarded routes block or redirect according to the rule.
- `72-router-hydrate-mount`: Load the prerendered route first and then navigate after hydration. Expected: the initial route is preserved during attach and later route changes work normally.
- `74-ssr-route-data-reuse`: Load the prerendered products route, then revalidate. Expected: the first render uses bootstrap data and later revalidation falls back to the live client loader path.

### devtools Package

- `66-devtools-panel`: Open the panel and inspect tree or hook state. Expected: the public panel surface mounts and tracks the example state.
- `67-use-snapshot`: Trigger snapshot capture from the devtools hook. Expected: the current state snapshot becomes visible or exportable without interrupting the app.
- `68-snapshot-now`: Capture an immediate snapshot. Expected: the snapshot reflects current state at the moment of capture rather than after a deferred tick.
- `69-devtools-diagnostics`: Open the diagnostics view and trigger the example warnings or counters. Expected: the diagnostics payload surfaces the intended runtime information.

## Suggested Regression Workflow

- Run the full smoke suite first to catch blank pages, panics, or failed asset loads.
- Run the focused specs next for examples that already have deeper coverage.
- Use the checklist above for examples in the area you changed, plus adjacent examples that share the same package surface.
- If a manual-only regression repeats twice, promote it into a dedicated example Playwright spec.
