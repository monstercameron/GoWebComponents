<p align="center">
    <img src="docs/assets/hero.jpg" alt="GoWebComponents Hero Image" width="900">
</p>

# GoWebComponents

[![CI + Release](https://github.com/monstercameron/GoWebComponents/actions/workflows/release.yml/badge.svg?branch=master)](https://github.com/monstercameron/GoWebComponents/actions/workflows/release.yml)
[![Deploy Examples To Pages](https://github.com/monstercameron/GoWebComponents/actions/workflows/pages.yml/badge.svg?branch=master)](https://github.com/monstercameron/GoWebComponents/actions/workflows/pages.yml)
[![Release Version](https://img.shields.io/github/v/release/monstercameron/GoWebComponents)](https://github.com/monstercameron/GoWebComponents/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/monstercameron/GoWebComponents)](https://goreportcard.com/report/github.com/monstercameron/GoWebComponents)

GoWebComponents is a Go + WebAssembly UI framework with a React-style component model, hooks, a fiber-based runtime, typed HTML builders, shorthand authoring helpers, client-side routing, shared state, and SSR or hydration support.

It is aimed at teams that want to build browser UI in Go without dropping into a separate JavaScript application stack for rendering, state, routing, and browser lifecycle management.

## Why GoWebComponents

- Write browser UI in Go instead of splitting application logic across Go backends and JavaScript frontends.
- Use a familiar component and hook model for local state, effects, async work, and composition.
- Build DOM trees with typed helpers in `html` or the mixed-argument sugar surface in `html/shorthand` instead of raw string templates.
- Add routing, shared state, fetch helpers, SSR, hydration, and devtools from the same module.
- Validate behavior with native Go tests, js/wasm tests, browser suites, and benchmark coverage already used in this repo.

## Quick Start

Install the module:

```bash
go get github.com/monstercameron/GoWebComponents@latest
```

Import public packages from the module path exactly as declared in `go.mod`:

```go
import (
  "github.com/monstercameron/GoWebComponents/fetch"
  "github.com/monstercameron/GoWebComponents/hotreload"
  "github.com/monstercameron/GoWebComponents/html"
    . "github.com/monstercameron/GoWebComponents/html/shorthand"
  "github.com/monstercameron/GoWebComponents/router"
  "github.com/monstercameron/GoWebComponents/state"
  "github.com/monstercameron/GoWebComponents/ui"
)
```

Requirements:

- Go 1.25+
- A browser with WebAssembly support

The repository root is the module boundary, not a directly importable package. Application code should import public subpackages such as `ui`, `html`, `state`, `fetch`, `router`, `devtools`, and `hotreload`.

The repo-standard workflow uses the `gwc` runner under `tools/gwc`. See [docs/GWC.md](docs/GWC.md) for the canonical launcher guide.

Useful entrypoints:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc examples
go run ./tools/gwc dev -app .\examples\01-counter\main.go
go run ./tools/gwc build -app .\examples\01-counter\main.go -profile development
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc verify -app .\examples\01-counter\main.go -root .\examples\01-counter
```

For standalone wasm apps that want state-preserving reload, enable `hotreload.Enable()` in your app and use `gwc dev`.

## Starter App Example

`main.go`:

```go
package main

import (
    "fmt"

    . "github.com/monstercameron/GoWebComponents/html/shorthand"
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
    return Div(
        Class("space-y-3 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"),
        P(Class("text-sm text-slate-600"), Textf("Hello, %s.", props.Name)),
        P(Class("text-lg font-semibold text-slate-900"), Textf("Count: %d", props.Count)),
        P(Class("text-sm text-slate-500"), Textf("Previous count: %s", props.PreviousCount)),
        Button(Type("button"), OnClick(props.OnIncrement), Class("rounded-xl bg-slate-900 px-4 py-2 text-sm font-medium text-white"), "Increment"),
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

    previousLabel := "none yet"
    if previousCount.Ok() {
        previousLabel = fmt.Sprintf("%d", previousCount.Get())
    }

    return Main(
        Class("min-h-screen bg-slate-50 px-6 py-12 text-slate-900"),
        Div(
            Class("mx-auto max-w-2xl space-y-6"),
            H1(Class("text-4xl font-black tracking-tight"), props.Title),
            P(Class("max-w-xl text-sm leading-7 text-slate-600"), "A small starter that uses dot-imported shorthand tags and helper functions for state, events, composition, and reactive text."),
            Input(
                Type("text"),
                Value(name.Get()),
                OnInput(updateName),
                Placeholder("Who is using the app?"),
                Class("w-full rounded-2xl border border-slate-300 bg-white px-4 py-3 text-sm shadow-sm"),
            ),
            If(name.Get() == "", P(Class("text-sm text-amber-700"), "Tip: enter a name to personalize the panel.")),
        ),
        ui.CreateElement(CounterPanel, CounterPanelProps{
            Name:          name.Get(),
            Count:         count.Get(),
            PreviousCount: previousLabel,
            OnIncrement:   increment,
        }),
        P(Class("mx-auto mt-6 max-w-2xl text-xs uppercase tracking-[0.18em] text-slate-500"), Textf("Current count is %d", count.Get())),
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
- Dot-imported `html/shorthand` tags and helpers such as `Div`, `Button`, `Class`, `Textf`, and `If`

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

Build with the repo runner:

```powershell
go run ./tools/gwc build -app .\main.go -profile development
```

## Core Concepts

- Components return `ui.Node` and are mounted with `ui.Render(...)`.
- The `html` package provides the stable typed DOM builders such as `html.Div`, `html.Button`, `html.Input`, and `html.Tag`.
- The `html/shorthand` package provides ergonomic mixed-argument sugar for dot-imported tags and helper funcs such as `Div`, `Button`, `Class`, `If`, `Text`, and `Textf`.
- Local component behavior lives in `ui` hooks such as `UseState`, `UseEffect`, `UseReducer`, `UseRef`, and `UseEvent`.
- Shared application state lives in `state`, with atoms, derived values, computed values, and snapshot helpers.
- Routing, fetch helpers, SSR, hydration, and diagnostics are layered on top of the same runtime rather than split into unrelated packages.

## Public Packages

The preferred public surface is:

- `ui`: component composition, hooks, rendering, hydration, async boundaries, events, portals, and form helpers
- `html`: stable typed HTML builders and DOM prop metadata
- `html/shorthand`: mixed-argument authoring sugar, helper funcs, and dot-import-friendly host tags layered on `html`
- `state`: atom-based shared state, derived state, computed values, and snapshot helpers
- `fetch`: browser fetch helpers, typed resources, and imperative fetch flows
- `router`: hash routing, browser routing, params, query helpers, redirects, loaders, guards, metadata, nested layouts, and hydration-aware mount helpers
- `devtools`: embeddable inspection, diagnostics, profiling hints, and snapshots
- `head`: optional companion SSR head composition helpers for router metadata, social tags, robots tags, JSON-LD, alternate locale links, and resource hints
- `plugin`: experimental companion host for explicit plugin manifests, capability-checked registration, and subsystem hook contributions layered on public APIs
- `hotreload`: state-preserving development reload bridge and snapshot helpers for standalone wasm apps

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
- `go run ./tools/gwc import -src .\path\to\layout.html -out .\bin\converter\layout\main.go` converts static `.html`, `.htm`, `.jsx`, or `.tsx` files into an inspectable GWC `main.go` built from current public `html` builders
- `go run ./tools/gwc examples` serves the example catalog through the repo-standard Go runner
- `go run ./tools/gwc dev -app .\path\to\main.go` starts the standalone wasm inner loop with rebuild-on-save and hotreload support
- `go run ./tools/gwc test -lane unit -lane wasm -lane browser` runs the supported launcher-owned validation lanes
- Launcher-owned temporary artifacts now resolve under `bin/tmp/` beneath the relevant project root instead of the OS temp directory
- Native Go tests, js/wasm tests, browser suites, and benchmark coverage
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

From the repo root, start the example catalog with:

```powershell
go run ./tools/gwc examples
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

Runner-owned validation lanes:

```powershell
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc test -lane browser
go run ./tools/gwc verify -app .\examples\01-counter\main.go -root .\examples\01-counter
```

Wasm-only runtime tests on Windows:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/runtime
```

For the broader browser harness and focused browser flows, use [test/README.md](test/README.md) as the authoritative reference.

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

Release-style wasm comparisons and benchmark reporting are driven through [docs/GWC.md](docs/GWC.md), [tools/README.md](tools/README.md), and [docs/PERFORMANCE.md](docs/PERFORMANCE.md), especially `gwc build`, `gwc release`, and `gwc bench`.

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

Use the `gwc` runner as the repo-standard entrypoint for local workflows.

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc examples
go run ./tools/gwc dev -app .\examples\01-counter\main.go
go run ./tools/gwc build -app .\examples\01-counter\main.go -profile ci
go run ./tools/gwc release -app .\examples\01-counter\main.go -out-dir .\bin\gwc-release
```

Relevant directories:

- `examples/`: example apps and feature-isolated catalog pages
- `examples/static/`: shared example assets
- `bin/examples/`: generated wasm binaries for example entrypoints served at `/static/bin/...` by the local example servers
- `bin/converter/`: ignored local converter outputs for inspectable imported layouts and screenshot comparisons
- `tools/gwc/`: canonical repo runner and launcher commands
- `test/`: main browser regression suites

Generated wasm binaries and local browser-compiler package archives should stay out of git unless there is a deliberate release reason to commit them.

## Status

Current repo state as reflected in the codebase:

- Core runtime lives in `internal/runtime/`
- Preferred public packages are `ui`, `html`, `html/shorthand`, `state`, `fetch`, `router`, `devtools`, and `hotreload`
- Example and test fixture code now builds through current `ui`/`html` bridge helpers and shorthand sugar instead of older compatibility layers
- Native `internal/runtime` statement coverage is `100%`
- Native runtime tests pass with `go test ./internal/runtime`
- Browser component, integration, and deep-state suites exist under the launcher-owned browser harness
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
- [docs/START_HERE.md](docs/START_HERE.md)
- [docs/WORKFLOWS.md](docs/WORKFLOWS.md)
- [docs/WALKTHROUGHS.md](docs/WALKTHROUGHS.md)
- [docs/REFERENCE_MAP.md](docs/REFERENCE_MAP.md)
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- [docs/FORMS.md](docs/FORMS.md)
- [docs/API_POLICY.md](docs/API_POLICY.md)
- [docs/ACCESSIBILITY.md](docs/ACCESSIBILITY.md)
- [docs/HEAD_MANAGEMENT.md](docs/HEAD_MANAGEMENT.md)
- [docs/MIGRATIONS.md](docs/MIGRATIONS.md)
- [docs/TODO.md](docs/TODO.md)
- [examples/README.md](examples/README.md)
- [test/README.md](test/README.md)
- [tools/README.md](tools/README.md)

## Notes

- Older references to `fiber/` are obsolete; the runtime now lives under `internal/runtime/`.
- The repo-standard workflow now goes through `go run ./tools/gwc ...` instead of ad hoc local launcher scripts.
- The browser-compiler example may generate large local package archives under `examples/13-browser-compiler/static/pkg/`; those artifacts should remain ignored.
