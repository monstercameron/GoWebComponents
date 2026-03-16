# README Proposal

This file is a proposed replacement for `README.md`.
It is intentionally separate so the current README stays unchanged until you approve the rewrite.

---

<p align="center">
  <img src="hero.jpg" alt="GoWebComponents Hero Image" width="900">
</p>

# GoWebComponents

[![CI + Release](https://github.com/monstercameron/GoWebComponents/actions/workflows/release.yml/badge.svg?branch=master)](https://github.com/monstercameron/GoWebComponents/actions/workflows/release.yml)
[![Deploy Examples To Pages](https://github.com/monstercameron/GoWebComponents/actions/workflows/pages.yml/badge.svg?branch=master)](https://github.com/monstercameron/GoWebComponents/actions/workflows/pages.yml)
[![Release Version](https://img.shields.io/github/v/release/monstercameron/GoWebComponents)](https://github.com/monstercameron/GoWebComponents/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/monstercameron/GoWebComponents)](https://goreportcard.com/report/github.com/monstercameron/GoWebComponents)

GoWebComponents is a Go + WebAssembly UI framework with a React-style component model, hooks, a fiber-based runtime, typed HTML builders, client-side routing, shared state, and SSR or hydration support.

It is aimed at teams that want to build browser UI in Go without dropping into a separate JavaScript application stack for rendering, state, routing, and browser lifecycle management.

## Why GoWebComponents

- Write browser UI in Go instead of splitting application logic across Go backends and JavaScript frontends.
- Use a familiar component and hook model for local state, effects, async work, and composition.
- Build DOM trees with typed helpers in `html` instead of raw string templates.
- Add routing, shared state, fetch helpers, SSR, hydration, and devtools from the same module.
- Validate behavior with native Go tests, js/wasm tests, Playwright suites, and benchmark coverage already used in this repo.

## Quick Start

Install the module:

```bash
go get github.com/monstercameron/GoWebComponents@latest
```

Import public packages from the module path exactly as declared in `go.mod`:

```go
import (
  "github.com/monstercameron/GoWebComponents/fetch"
  "github.com/monstercameron/GoWebComponents/html"
  "github.com/monstercameron/GoWebComponents/router"
  "github.com/monstercameron/GoWebComponents/state"
  "github.com/monstercameron/GoWebComponents/ui"
)
```

Requirements:

- Go 1.25+
- A browser with WebAssembly support
- Node.js only when you want the example dev server or Playwright browser suites

The repository root is the module boundary, not a directly importable package. Application code should import public subpackages such as `ui`, `html`, `state`, `fetch`, `router`, and `devtools`.

## Starter App Example

`main.go`:

```go
package main

import (
    "fmt"
    "syscall/js"

    "github.com/monstercameron/GoWebComponents/html"
    "github.com/monstercameron/GoWebComponents/ui"
)

type StarterAppProps struct {
    Title        string
    InitialCount int
}

type CounterPanelProps struct {
    Name          string
    Count         int
    PreviousCount string
    OnIncrement   func()
}

func CounterPanel(props CounterPanelProps) ui.Node {
    return html.Div(html.Props{},
        html.P(html.Props{}, html.Text(fmt.Sprintf("Hello, %s.", props.Name))),
        html.P(html.Props{}, html.Text(fmt.Sprintf("Count: %d", props.Count))),
        html.P(html.Props{}, html.Text(fmt.Sprintf("Previous count: %s", props.PreviousCount))),
        html.Button(html.Props{OnClick: props.OnIncrement}, html.Text("Increment")),
    )
}

func StarterApp(props StarterAppProps) ui.Node {
    count := ui.UseState(props.InitialCount)
    name := ui.UseState("Go developer")
    previousCount := ui.UsePrevious(count.Get())

    increment := ui.UseEvent(func() {
        count.Update(func(previous int) int { return previous + 1 })
    })
    updateName := ui.UseEvent(func(event ui.InputEvent) {
        name.Set(event.GetValue())
    })

    ui.UseEffect(func() func() {
        document := js.Global().Get("document")
        if document.Truthy() {
            document.Set("title", fmt.Sprintf("%s (%d)", props.Title, count.Get()))
        }
        return nil
    }, props.Title, count.Get())

    previousLabel := "none yet"
    if previousCount.Ok() {
        previousLabel = fmt.Sprintf("%d", previousCount.Get())
    }

    return html.Div(html.Props{},
        html.H1(html.Props{}, html.Text(props.Title)),
        html.P(html.Props{}, html.Text("A small React-like component with local state, typed events, composition, a previous value, and an effect.")),
        html.Input(html.Props{
            Value:       name.Get(),
            OnInput:     updateName,
            Placeholder: "Who is using the app?",
        }),
        ui.CreateElement(CounterPanel, CounterPanelProps{
            Name:          name.Get(),
            Count:         count.Get(),
            PreviousCount: previousLabel,
            OnIncrement:   increment,
        }),
    )
}

func main() {
    ui.Render(ui.CreateElement(StarterApp, StarterAppProps{
        Title:        "Starter App",
        InitialCount: 0,
    }), "#app")
    select {}
}
```

This version stays small, but it shows the normal flow most React users expect:

- `UseState` for local UI state
- `UseEvent` for typed event handlers
- Component composition with a child `CounterPanel`
- `UsePrevious` for render-time comparisons
- `UseEffect` for browser-side side effects

Host HTML:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Starter App</title>
    <script src="./wasm_exec.js"></script>
    <script>
      const go = new Go();
      WebAssembly.instantiateStreaming(fetch("./bin/main.wasm"), go.importObject)
        .then((result) => go.run(result.instance));
    </script>
  </head>
  <body>
    <div id="app"></div>
  </body>
</html>
```

Build manually:

```bash
GOOS=js GOARCH=wasm go build -o static/bin/main.wasm main.go
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" static/
```

On Windows PowerShell:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go build -o static/bin/main.wasm main.go
Copy-Item "$(go env GOROOT)\lib\wasm\wasm_exec.js" static\wasm_exec.js
```

## Core Concepts

- Components return `ui.Node` and are mounted with `ui.Render(...)`.
- The `html` package provides typed DOM builders such as `Div`, `Button`, `Input`, `Section`, and `Tag`.
- Local component behavior lives in `ui` hooks such as `UseState`, `UseEffect`, `UseReducer`, `UseRef`, and `UseEvent`.
- Shared application state lives in `state`, with atoms, derived values, computed values, and snapshot helpers.
- Routing, fetch helpers, SSR, hydration, and diagnostics are layered on top of the same runtime rather than split into unrelated packages.

## Public Packages

The preferred public surface is:

- `ui`: component composition, hooks, rendering, hydration, async boundaries, events, portals, and form helpers
- `html`: typed HTML builders and DOM prop metadata
- `state`: atom-based shared state, derived state, computed values, and snapshot helpers
- `fetch`: browser fetch helpers, typed resources, and imperative fetch flows
- `router`: hash routing, browser routing, params, query helpers, redirects, loaders, guards, metadata, nested layouts, and hydration-aware mount helpers
- `devtools`: embeddable inspection, diagnostics, profiling hints, and snapshots

## Feature Overview

### Rendering and Hooks

- React-style function components with a fiber-based runtime and browser-side rendering through `syscall/js`
- Hooks including `UseState`, `UseReducer`, `UseEffect`, `UseRef`, `UsePrevious`, `UseId`, `UseDeferredValue`, `UseTransition`, and context support
- Event and async helpers including `UseEvent`, `UseChannel`, `UseTask`, `UseDebounced`, `UseThrottled`, `UseLazyNode`, `AsyncBoundary`, and `UseForm`

### State and Data

- Shared atom-based state with subscriptions, derived atoms, computed values, snapshot export or import, and optional browser-storage restore flows
- Fetch helpers including low-level `UseFetch`, typed `UseResource[T]`, and imperative `Fetch(...)`

### Routing

- Hash router and browser/history router support
- Params, query helpers, redirects, route metadata, guards, loaders, manual revalidation, nested layout routes, and `router.Outlet()`

### SSR and Hydration

- Native server-side rendering through `ui.RenderToString(...)`
- Browser hydration through `ui.Hydrate(...)`
- Bootstrap transport helpers for inline JSON, sidecar JSON, and optional CBOR payload encoding or decoding

### Tooling and Validation

- Public `devtools` package for in-app inspection and diagnostics
- Native Go tests, js/wasm tests, Playwright browser suites, and benchmark coverage
- Large example suite spanning local state, forms, routing, async work, SSR, hydration, nested routes, and diagnostics

## SSR and Hydration

GoWebComponents supports both request-time server rendering and browser hydration.

Current repo examples cover:

- `ui.RenderToString(...)` for server-rendered HTML generation on native Go targets
- `ui.Hydrate(...)` for resuming matching DOM in the browser
- bootstrap payload helpers for transferring route data, atoms, IDs, and initialization data
- request-time SSR with route-aware hydration under `examples/18-ssr-server-routing`
- SSR bootstrap and route-data reuse demos under the numbered example catalog

The hydration model in this repo already includes DOM reuse, transferred bootstrap state, mismatch diagnostics, and subtree fallback on structural mismatch.

## Examples

The repository ships both larger integrated demos and feature-isolated catalog pages.

From the repo root, start the example server with:

```powershell
npm run dev:examples
```

Primary URLs:

- Styled showcase: `http://127.0.0.1:8090/examples`
- Raw filesystem listing: `http://127.0.0.1:8090/examples/list`
- Example entrypoint: `http://127.0.0.1:8090/examples/01-counter/counter.html`
- Health check: `http://127.0.0.1:8090/healthz`

Use the showcase when you want to browse by feature. Use the filesystem listing when you want direct diagnostics against the real folder structure.

For the full example inventory and API-to-example mapping, see [examples/README.md](examples/README.md).

## Testing

Run the full Go suite:

```bash
go test ./...
```

Native runtime tests only:

```bash
go test ./internal/runtime
```

Wasm-only runtime tests on Windows:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/runtime
```

Example-oriented browser tests:

```powershell
cd examples
npm install
npx playwright test
```

Main browser regression suites:

```powershell
cd test
npm install
npm run install:browsers
npm test
```

Focused browser suites:

```powershell
npm run test:components
npm run test:integration
npm run test:state
```

## Benchmarks

Native runtime microbenchmarks:

```bash
go test ./internal/runtime -run ^$ -bench . -benchmem
```

Wasm adapter microbenchmarks:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/platform/jsdom -run ^$ -bench . -benchmem
```

Browser comparison benchmark:

```powershell
cd test
npm install
npx playwright install chromium
npm run bench
```

Latest browser comparison run on 2026-03-16:

- Core render: `GoWebComponents 59 ms`, `React 42 ms`
- Core update: `GoWebComponents 67 ms`, `React 33 ms`
- Content render: `GoWebComponents 65 ms`, `React 35 ms`
- Content update: `GoWebComponents 67 ms`, `React 32 ms`
- Clear: `GoWebComponents 66 ms`, `React 33 ms`
- Deep tree: `GoWebComponents 69 ms`, `React 34 ms`
- Hooks: `GoWebComponents 62 ms`, `React 33 ms`

Recent measured native runtime improvements include:

- `DivWithComponents4`: `1654 ns/op` -> `736.9 ns/op`
- `WithComponentsGeneric4`: `1820 ns/op` -> `829.0 ns/op`
- `CleanupAtomSubscriptions8`: `4477 ns/op` -> `3629 ns/op`

## Development Workflow

The repo uses a Node/Express example server for local example development.

```powershell
npm run dev:examples
```

Relevant directories:

- `examples/`: example apps and feature-isolated catalog pages
- `examples/static/`: shared example assets
- `examples/static/bin/`: generated wasm binaries for example entrypoints
- `tools/dev-server/`: local example server implementation
- `test/`: main Playwright-based browser regression suites

Generated wasm binaries and local browser-compiler package archives should stay out of git unless there is a deliberate release reason to commit them.

## Status

Current repo state as reflected in the codebase:

- Core runtime lives in `internal/runtime/`
- Preferred public packages are `ui`, `html`, `state`, `fetch`, `router`, and `devtools`
- Example and test fixture code now builds through current ui/html bridge helpers instead of older compatibility layers
- Native `internal/runtime` statement coverage is `100%`
- Native runtime tests pass with `go test ./internal/runtime`
- Browser component, integration, and deep-state suites exist under Playwright
- Separate js/wasm tests and benchmarks exist for wasm-only runtime and adapter behavior

## Internal Architecture

The implementation center of gravity is `internal/runtime/`:

- `types.go`: core data structures
- `reconciler.go`: render tree diffing and commit preparation
- `scheduler.go`: update scheduling and work-loop entrypoints
- `hooks.go`: hook state, effect, memo, and ref behavior
- `state.go`: atom registry and subscriptions
- `runtime.go`: runtime bootstrap and global runtime wiring

## Documentation

- [CHANGELOG.md](CHANGELOG.md)
- [docs/README.md](docs/README.md)
- [docs/API_POLICY.md](docs/API_POLICY.md)
- [docs/MIGRATIONS.md](docs/MIGRATIONS.md)
- [docs/TODO.md](docs/TODO.md)
- [examples/README.md](examples/README.md)
- [test/README.md](test/README.md)
- [tools/README.md](tools/README.md)

## Notes

- Older references to `fiber/` are obsolete; the runtime now lives under `internal/runtime/`.
- The Express example server replaced older docs that referred to `scripts/` or ad hoc static serving.
- The browser-compiler example may generate large local package archives under `examples/13-browser-compiler/static/pkg/`; those artifacts should remain ignored.