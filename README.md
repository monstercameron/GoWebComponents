<p align="center">
  <img src="hero.jpg" alt="GoWebComponents Hero Image" width="900">
</p>

# GoWebComponents

[![CI + Release](https://github.com/monstercameron/GoWebComponents/actions/workflows/release.yml/badge.svg?branch=master)](https://github.com/monstercameron/GoWebComponents/actions/workflows/release.yml)
[![Deploy Examples To Pages](https://github.com/monstercameron/GoWebComponents/actions/workflows/pages.yml/badge.svg?branch=master)](https://github.com/monstercameron/GoWebComponents/actions/workflows/pages.yml)
[![Release Version](https://img.shields.io/github/v/release/monstercameron/GoWebComponents)](https://github.com/monstercameron/GoWebComponents/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/monstercameron/GoWebComponents)](https://goreportcard.com/report/github.com/monstercameron/GoWebComponents)

GoWebComponents is a Go + WebAssembly UI framework with a React-style component model, hooks, a fiber-based runtime, and browser-side rendering through `syscall/js`.

## Status

Current repo state as of 2026-03-14:

- Core runtime lives in `internal/runtime/`
- Preferred public packages are `ui`, `html`, `state`, `fetch`, `router`, and `devtools`
- Remaining examples and test fixtures now build through local ui/html bridge helpers instead of legacy compatibility packages
- Native `internal/runtime` statement coverage is `100%`
- Native runtime tests pass with `go test ./internal/runtime`
- Browser component, integration, and deep state stress suites pass under Playwright
- Separate `js/wasm` tests and benchmarks exist for wasm-only runtime and adapter paths

## Install

```bash
go get github.com/monstercameron/GoWebComponents@latest
```

Import packages from the module path exactly as declared in `go.mod`:

```go
import (
  "github.com/monstercameron/GoWebComponents/fetch"
  "github.com/monstercameron/GoWebComponents/html"
  "github.com/monstercameron/GoWebComponents/router"
  "github.com/monstercameron/GoWebComponents/state"
  "github.com/monstercameron/GoWebComponents/ui"
)
```

The repository root is the module boundary, not a directly importable package. Consumers should import one or more public subpackages such as `ui` and `html`.

Requirements:

- Go 1.25+
- A browser that supports WebAssembly
- Node.js only for the example dev server and Playwright-based browser tests

## Features

Project-wide capabilities currently included in GoWebComponents:

- React-style function components with a fiber-based runtime and browser-side rendering through `syscall/js`
- Typed HTML element builders in `html` so UI trees can be written without raw string templates
- Core hooks in `ui`, including `UseState`, `UseReducer`, `UseEffect`, `UseMemo`, `UseCallback`, `UseRef`, `UsePrevious`, and `UseId`
- Event and async helpers in `ui`, including `UseEvent`, `UseChannel`, `UseTask`, `UseLazyNode`, `UseDebounced`, `UseThrottled`, and `UseForm`
- Context API support through `CreateContext`, provider components, and `UseContext`
- Shared atom-based state in `state` with subscriptions, derived atoms, computed values, snapshot export/import, and optional browser-storage persistence
- Fetch helpers in `fetch`, including low-level `UseFetch`, typed `UseResource[T]`, and imperative `Fetch(...)`
- Client-side routing in `router` with hash routers, browser/history routers, params, query helpers, redirects, metadata, guards, loaders, manual revalidation, nested layout routes, and `router.Outlet()`
- Server-side rendering through `ui.RenderToString(...)` on native targets
- Browser hydration through `ui.Hydrate(...)`, including bootstrap payload restore, matching DOM reuse, subtree fallback on structural mismatch, mismatch diagnostics, atom snapshot restore, and deterministic `UseId` resume via transferred ID seed
- SSR bootstrap transport helpers for inline JSON, sidecar JSON, and optional CBOR payload encoding/decoding
- In-browser inspection via the public `devtools` package, including component tree snapshots, hook inspection, route inspection, profiling hotspots, and structured diagnostics
- Native Go tests, js/wasm tests, Playwright browser suites, and native plus browser benchmark coverage
- A broad examples suite covering local state, forms, async work, atoms, routing, devtools, SSR hydration, request-time SSR, and nested layout routes

## Minimal Example

```go
package main

import (
    "fmt"

    "github.com/monstercameron/GoWebComponents/html"
    "github.com/monstercameron/GoWebComponents/ui"
)

type CounterProps struct {
    Initial int
}

func Counter(props CounterProps) ui.Node {
    count := ui.UseState(props.Initial)
    increment := ui.UseEvent(func() {
        count.Update(func(prev int) int { return prev + 1 })
    })

    return html.Div(html.Props{},
        html.H1(html.Props{}, html.Text("Counter")),
        html.P(html.Props{}, html.Text(fmt.Sprintf("Count: %d", count.Get()))),
        html.Button(html.Props{OnClick: increment}, html.Text("Increment")),
    )
}

func main() {
    ui.Render(ui.CreateElement(Counter, CounterProps{Initial: 0}), "#app")
    select {}
}
```

Example HTML:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>GoWebComponents</title>
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

## Core Packages

- `ui`: component composition, hooks, render entrypoint, and typed event wrappers
- `html`: typed HTML builders and DOM prop metadata
- `state`: shared atom-based state
- `fetch`: browser fetch helpers layered on top of the runtime hook/fetch APIs
- `router`: browser/hash routing helpers
- `devtools`: embeddable runtime inspection and diagnostics panel

For new projects imported with `go get`, use `ui`, `html`, `state`, `fetch`, and `router` as the main public surface. Add `devtools` when you want in-app inspection during development.

## Context API

The `ui` package now includes a minimal Context API for subtree-scoped values.
Use `CreateContext` to define a context, `UseContext` to read the nearest
provider value, and `context.Provider` with `ui.CreateElement(...)` to set a
value for a subtree. Missing providers resolve to the context default value.

```go
type ThemeProps struct{}

var themeContext = ui.CreateContext("light")

func ThemeLabel(props ThemeProps) ui.Node {
  theme := ui.UseContext(themeContext)
  return html.P(html.Props{}, html.Text("Theme: "+theme))
}

func App() ui.Node {
  return ui.CreateElement(themeContext.Provider, ui.ContextProviderProps[string]{
    Value: "dark",
    Child: ui.CreateElement(ThemeLabel, ThemeProps{}),
  })
}
```

Recommended use cases are theme, auth/session state, configuration, and
service-like helpers that should not be threaded through many intermediate
component props.

## Runtime Layout

The implementation center of gravity is `internal/runtime/`:

- `types.go`: core data structures
- `reconciler.go`: render tree diffing and commit preparation
- `scheduler.go`: update scheduling and work loop entrypoints
- `hooks.go`: hook state/effect/memo/ref logic
- `state.go`: atom registry and subscriptions
- `runtime.go`: runtime bootstrap and global runtime wiring

## Development Server

The repo now uses a Node/Express example server.

Start it from the repo root:

```powershell
npm run dev:examples
```

Default URLs:

- `http://127.0.0.1:8090/examples`
- `http://127.0.0.1:8090/examples/01-counter/counter.html`
- `http://127.0.0.1:8090/healthz`

The `/examples` index is generated from the actual filesystem and is not hard-coded.

## Examples

Primary example entry points:

- Styled showcase: `http://127.0.0.1:8090/examples/static/index.html`
- Filesystem index: `http://127.0.0.1:8090/examples`

Current examples under `examples/`:

- `01-counter`: counter state basics
- `02-text-input`: controlled text input
- `03-toggle`: boolean state and conditional UI
- `04-form`: struct-backed form state
- `05-todo-basic`: basic array CRUD example
- `06-todo-advanced`: richer todo app with filters and categories
- `07-goroutines`: background tasks and timers
- `08-fetch`: async fetch patterns and loading states
- `09-atoms`: shared global state via atoms
- `10-advanced-form`: validation and async form patterns
- `11-blog`: blog landing page composition example
- `12-portfolio-site`: full multi-section portfolio app
- `13-browser-compiler`: browser-side compiler tooling example
- `14-omi`: OMI integration example
- `15-calculator`: animated scientific calculator demo
- `16-devtools`: standalone devtools panel and diagnostics example
- `17-ssr-routing`: static SSR shell plus bootstrap-driven hydration into advanced routed client flows
- `18-ssr-server-routing`: request-time server rendering with route-aware bootstrap hydration over real URLs
- `19-nested-routes`: nested layout routes, outlets, and multi-level route trees

Examples currently featured on the styled showcase page include the interactive demos from `01` through `12`, plus `15-calculator`. The repo also includes the newer devtools, SSR, and nested-route demos listed above.

## Testing

Runtime tests:

```bash
go test ./internal/runtime
```

Wasm-only runtime tests on Windows:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/runtime
```

Browser tests:

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

Browser React-vs-Go comparison benchmark:

```powershell
cd tests
npm install
npx playwright install chromium
npm run bench
```

Latest browser comparison run on 2026-03-14:

- Render: `GoWebComponents 117 ms`, `React 49 ms`
- Update: `GoWebComponents 62 ms`, `React 48 ms`
- Clear: `GoWebComponents 59 ms`, `React 55 ms`
- Deep tree: `GoWebComponents 165 ms`, `React 63 ms`
- Hooks: `GoWebComponents 145 ms`, `React 77 ms`

Notes:

- The browser benchmark currently compares render, update, clear, deep-tree, and many-hooks scenarios from `test/specs/performance_benchmark.spec.ts`.
- The old compute-primes scenario is intentionally excluded from the main comparison run because it is not yet a stable benchmark case for this repo.

Recent measured wins from the current optimization pass include:

- `DivWithComponents4`: `1654 ns/op` -> `736.9 ns/op`
- `WithComponentsGeneric4`: `1820 ns/op` -> `829.0 ns/op`
- `CleanupAtomSubscriptions8`: `4477 ns/op` -> `3629 ns/op`

## Documentation

- [CHANGELOG.md](CHANGELOG.md)
- [docs/README.md](docs/README.md)
- [examples/README.md](examples/README.md)
- [test/README.md](test/README.md)
- [tools/README.md](tools/README.md)

## Notes

- The old `fiber/` path references in older docs are obsolete; the runtime now lives under `internal/runtime/`.
- The Express dev server replaced the older doc paths that referenced `scripts/` or ad hoc static serving.
- The browser-compiler example may generate large local package archives under `examples/13-browser-compiler/static/pkg/`, but those artifacts are now ignored and should not be committed.
