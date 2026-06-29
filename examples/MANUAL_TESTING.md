# Example Manual Testing Guide

Location: `examples/`

This document is the developer-facing manual verification guide for the example catalog.

Use it when you are:

- changing public APIs and need to confirm examples still demonstrate the intended behavior
- updating example styling, layout, hydration, or routing behavior
- deciding whether a regression belongs in a dedicated Playwright spec or only in the broad smoke suite
- doing release verification before demoing the examples catalog

## Canonical Automated Baseline

The authoritative example coverage is the Playwright-Go browser suite, not a hand-kept list:

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExamplesAll -v
```

Focused suites: `TestCatalog`, `TestLinks`, `TestSSRServerRouting`, `TestAtlasSSR`, `TestStartup`, `TestVirtualization`, `TestAtlasStartup`, `TestBrowserCompat`, and `TestChatWizard`.

The example set itself is the generated catalog at `examples/public-examples-site/assets/data/catalog.json` (browse with `go run ./tools/gwc examples`). Feature-isolated examples live at `examples/public/<slug>/`; the larger SSR servers live under `examples/server/<slug>/`. There is intentionally no numbered (`NN-slug`) checklist here anymore: it drifted every time an example was renamed. Verify against the slugs the catalog reports.

## Environment

From the repo root:

```powershell
go run ./tools/gwc examples
```

Main catalog URLs:

- `http://127.0.0.1:8090/examples`
- `http://127.0.0.1:8090/examples/public-examples-site/`

Standalone SSR routing server (`examples/server/server-side-rendering-routing`):

```powershell
go run ./examples/server/server-side-rendering-routing
```

Standalone SSR server URLs:

- `http://127.0.0.1:8079/`
- `http://127.0.0.1:8079/docs/ssr`
- `http://127.0.0.1:8079/search?q=routing`
- `http://127.0.0.1:8079/secure`
- `http://127.0.0.1:8079/secure?auth=true&role=maintainer`

Standalone Atlas commerce-OS SSR server (`examples/server/atlas-commerce-os`):

```powershell
go run ./examples/server/atlas-commerce-os/server
```

Standalone Atlas SSR URLs:

- `http://127.0.0.1:8096/`
- `http://127.0.0.1:8096/shop`
- `http://127.0.0.1:8096/shop/frame-desk`
- `http://127.0.0.1:8096/app/dashboard`
- `http://127.0.0.1:8096/app/inventory`

## Manual Verification Rules

Apply these to whichever examples cover the surface you changed (find them in the catalog by API or tag):

- Confirm the page shell is dark mode and the primary heading renders without console or page errors.
- Exercise at least one state change, route change, async transition, or overlay action on every page.
- Verify the visible output changes in a way that matches the example's teaching goal.
- If the example demonstrates routing or hydration, also verify refresh and direct navigation behavior where applicable.
- If an example exposes stats, badges, or status copy, verify those values change together with the primary interaction.
- For overlay and accessibility examples, verify focus trap, Escape dismissal, focus restoration, and live-region announcements.
- For SSR and hydration examples, verify the first response is real HTML and hydration resumes the existing DOM without a full-shell replacement.
- For PWA and offline examples, verify cache warm-up, offline mutation queue and replay, and that capability gaps surface a structured message rather than a silent failure.

## Suggested Regression Workflow

- Run the full smoke suite first to catch blank pages, panics, or failed asset loads.
- Run the focused specs next for examples that already have deeper coverage.
- Use the rules above for examples in the area you changed, plus adjacent examples that share the same package surface.
- If a manual-only regression repeats twice, promote it into a dedicated example Playwright spec.

## Mobile Safari Coverage

Use this smaller pass before calling a workflow broadly browser-compatible (verify the current slug in the catalog):

- `hydration`: confirm prerendered markup is visible before wasm, hydration attaches, and the resumed buttons still work with touch input.
- `server-side-rendering-cache-bootstrap`: confirm inline bootstrap data restores correctly and the hydrated UI does not restart from empty client state.
- `static-islands`: confirm the static shell remains readable before startup, only the island roots activate, and the budget pills populate on-device.
- `server-side-rendering-secure-forms` (server app): confirm keyboard, focus, validation, multipart upload, and redirect flows behave correctly under Mobile Safari input constraints.
- `progressive-web-app-offline-cache` or `progressive-web-app-multi-client`: confirm storage, service-worker, cache, and reduced-capability fallback behavior when offline features matter to the release.
- the AI chat wizard server app: confirm wasm startup, long-lived input, reconnect behavior, and scrolling remain stable on a constrained device before describing the app shape as production-ready.

Treat this as the minimum device-family gate for wasm-heavy, hydration-heavy, storage-heavy, or form-heavy releases.
