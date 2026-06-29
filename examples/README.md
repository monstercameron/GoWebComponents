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
- `http://127.0.0.1:8090/examples/public-examples-site/`

The `/examples` route redirects into the public examples site. If you need the raw auto-generated folder listing for diagnostics, use `/examples/list`.

## Build Public Site WASM

To refresh the public examples site shell and the first embedded example binary locally:

```powershell
go run ./tools/gwc examples build-public-site
```

That command builds:

- `public-examples-site.wasm` into the launcher examples wasm directory
- `counter.wasm` into the launcher examples wasm directory
- mirrored copies into `examples/static/bin/` for direct static preview tools such as VS Code Live Server
- a staged embedded copy at `examples/public-examples-site/assets/bins/counter.wasm`
- a refreshed source mirror of `examples/public/` under `examples/public-examples-site/assets/code/`

## Example Inventory

The example set is now organized as a catalog. Use the grouped landing page at `/examples` for browsing, and use the mapping below when you want the quickest reference for a specific public API.

Every feature-isolated catalog page is expected to answer three questions clearly: what the tool is for, what the user-facing behavior looks like, and how the implementation is wired. If an example stops doing one of those jobs, it should be treated as documentation debt rather than just a cosmetic issue.

## Current Boundary

- Shipped: a browsable example catalog, dedicated example browser coverage via Playwright-Go wrappers, and integrated examples that act as reference surfaces for larger app shapes.
- Not shipped: a promise that every example is a production starter template or that every example is kept runnable through every possible serving path.
- Use the examples to understand supported APIs and composed flows; use the package docs and workflow docs for policy boundaries.

### Integrated Apps

Larger, multi-feature reference surfaces. These combine several primitives to show how they compose in realistic apps:

- `examples/public/single-shell-auth`: one running WASM shell that keeps marketing, sign-in, and a guarded workspace in one router tree with bounded return-to recovery and no full-page auth detour.
- `examples/server/server-side-rendering-routing`: the metadata-heavy SSR routing reference — request-time HTML, route bootstrap reuse, `head.Render(...)` output, and the ownership boundary between first-paint SSR metadata and later hydrated navigation.
- `examples/server/atlas-commerce-os`: the production-shaped SSR reference server — shared UI, SSR, hydration, server-owned routes, mutations, static assets, and reviewer-facing docs.
- `examples/server/server-side-rendering-secure-forms`: request-time rendered forms with CSRF validation, multipart uploads, validation round-trips, `303` redirects, and the progressive half of the server-action contract.

Use the integrated apps to see how multiple primitives compose in one realistic surface. Use the feature-isolated pages when you want the clearest statement of a single API or tool's purpose.

### Feature-Isolated Catalog

Every public API has a dedicated, slug-named example under `examples/public/<slug>/` (for example `examples/public/use-state`, `examples/public/route-loaders`, `examples/public/async-boundary`). The authoritative, generated catalog — title, API surface, tags, and source links for every example — is `examples/public-examples-site/assets/data/catalog.json`, browsable via:

```powershell
go run ./tools/gwc examples
```

The catalog is generated from the example sources, so it never drifts from the on-disk set. Prefer it over a hand-maintained list: an enumerated inventory here would silently rot the next time an example is renamed — the failure this section used to demonstrate when it still listed the removed numbered (`NN-slug`) layout.

### Hot Reload Workflow

For a single app surface, enable the public `hotreload` package in that app and run the dev server:

```powershell
go run ./tools/gwc dev -app .\examples\public\hot-reload\main.go -root .\examples\public\hot-reload
```

Use `gwc examples` when you want to browse many examples. Use `gwc dev` when you want rebuild-on-save and state-preserving reload for one specific app.

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
- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample201BrowserBenchmarkReport -v`

From the repo root, `go run ./tools/gwc test -lane browser` now runs the main browser lane.

For the developer-facing manual verification checklist that covers the example catalog, see `examples/MANUAL_TESTING.md`.

## Notes

- Older docs referenced ad hoc live reload commands as the primary example workflow. The primary documented path is now `gwc examples`.
- The browser-compiler example may generate a large local package archive tree under `examples/public/browser-compiler/static/pkg/`. That output is ignored and should stay out of git.
