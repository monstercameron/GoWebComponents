<p align="center">
    <img src="docs/assets/chatgpt.png" alt="GoWebComponents ChatGPT Hero Image" width="900">
</p>

# GoWebComponents

[![CI + Release](https://github.com/monstercameron/GoWebComponents/actions/workflows/release.yml/badge.svg?branch=master)](https://github.com/monstercameron/GoWebComponents/actions/workflows/release.yml)
[![Deploy Examples To Pages](https://github.com/monstercameron/GoWebComponents/actions/workflows/pages.yml/badge.svg?branch=master)](https://github.com/monstercameron/GoWebComponents/actions/workflows/pages.yml)
[![Release Version](https://img.shields.io/github/v/release/monstercameron/GoWebComponents)](https://github.com/monstercameron/GoWebComponents/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/monstercameron/GoWebComponents)](https://goreportcard.com/report/github.com/monstercameron/GoWebComponents)

GoWebComponents is a Go + WebAssembly UI framework with a React-style component model, hooks, a fiber-based runtime, typed HTML builders, shorthand authoring helpers, a typed compile-checked CSS engine, client-side routing, and shared state. It also ships streaming SSR with real async suspension, hydration and static islands, crash containment by default, realtime data hooks, feature flags, i18n, accessibility primitives, and PWA/offline support.

It is **batteries-included**: rendering, hooks, routing, shared state, data fetching, SSR/hydration, i18n, accessibility, PWA/offline, feature flags, and devtools all ship in the same Go module, so there is no separate JavaScript build, bundler config, or npm dependency tree to assemble. The trade-offs that come with that — wasm runtime cost and bundle size — are described under [Performance and Trade-offs](#performance-and-trade-offs); read that section before adopting so expectations are set honestly.

It is aimed at teams that want to build browser UI in Go without dropping into a separate JavaScript application stack for rendering, state, routing, and browser lifecycle management.

## Why GoWebComponents

- Write browser UI in Go instead of splitting application logic across Go backends and JavaScript frontends.
- Use a familiar component and hook model for local state, effects, async work, and composition.
- Build DOM trees with typed helpers in `html` or the mixed-argument sugar surface in `html/shorthand` instead of raw string templates.
- Add routing, shared state, fetch helpers, SSR, hydration, and devtools from the same module.
- Validate behavior with native Go tests, js/wasm tests, browser suites, and benchmark coverage already used in this repo.

## Quick Start

### 30-second golden path

Clone the repo and run a real example with rebuild-on-save:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc dev -app .\examples\public\counter\main.go
```

`gwc dev` builds the app to wasm, serves it, and live-reloads on every save.
To scaffold your own app, use `go run ./tools/gwc start` (interactive); the
starter gallery is documented in [docs/STARTERS.md](docs/STARTERS.md). Browse
the example catalog with `go run ./tools/gwc examples`.

### Add to an existing module

Install the module:

```bash
go get github.com/monstercameron/GoWebComponents@latest
```

Import public packages from the module path exactly as declared in `go.mod`:

```go
import (
  "github.com/monstercameron/GoWebComponents/css"
  "github.com/monstercameron/GoWebComponents/css/u"
  "github.com/monstercameron/GoWebComponents/fetch"
  "github.com/monstercameron/GoWebComponents/flags"
  "github.com/monstercameron/GoWebComponents/hotreload"
  "github.com/monstercameron/GoWebComponents/html"
    . "github.com/monstercameron/GoWebComponents/html/shorthand"
  "github.com/monstercameron/GoWebComponents/router"
  "github.com/monstercameron/GoWebComponents/state"
  "github.com/monstercameron/GoWebComponents/ui"
)
```

Requirements:

- Go 1.26+ (matches the `go` directive in `go.mod`)
- A browser with WebAssembly support

The repository root is the module boundary, not a directly importable package. Application code should import public subpackages such as `ui`, `html`, `html/shorthand`, `css`, `css/u`, `state`, `fetch`, `flags`, `router`, `devtools`, and `hotreload`.

The repo-standard workflow uses the `gwc` runner under `tools/gwc`. See [docs/REFERENCE_MANUAL/02-gwc-workflows.md](docs/REFERENCE_MANUAL/02-gwc-workflows.md) for the canonical launcher guide.

Useful entrypoints:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc examples
go run ./tools/gwc dev -app .\examples\public\counter\main.go
go run ./tools/gwc build -app .\examples\public\counter\main.go -profile development
go run ./tools/gwc build -app .\examples\public\counter\main.go -profile debug -out .\bin\debug\counter.wasm
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc verify -app .\examples\public\counter\main.go -root .\examples\public\counter
```

For standalone wasm apps that want state-preserving reload, enable `hotreload.Enable()` in your app and use `gwc dev`.

## Choose Your Path

Use these entry docs instead of wandering the tree blindly:

- Library user: [docs/REFERENCE_MANUAL/README.md](docs/REFERENCE_MANUAL/README.md) and [docs/REFERENCE_MANUAL/01-getting-started.md](docs/REFERENCE_MANUAL/01-getting-started.md)
- Package author working in public APIs: [ui/README.md](ui/README.md), [html/README.md](html/README.md), [state/README.md](state/README.md), [fetch/README.md](fetch/README.md), [flags/README.md](flags/README.md), [router/README.md](router/README.md)
- Framework contributor: [internal/README.md](internal/README.md), [internal/runtime/README.md](internal/runtime/README.md), [internal/platform/README.md](internal/platform/README.md), [internal/runtime2/README.md](internal/runtime2/README.md)
- Tooling contributor: [tools/README.md](tools/README.md), [docs/REFERENCE_MANUAL/02-gwc-workflows.md](docs/REFERENCE_MANUAL/02-gwc-workflows.md), [tools/gwc/docs/README.md](tools/gwc/docs/README.md)
- Example explorer: [examples/README.md](examples/README.md)
- Test or validation work: [test/README.md](test/README.md) and [docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md](docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md)

## Repository Layout

If you are contributing to the repo rather than just consuming the module, use this map first:

- `ui/`, `html/`, `html/shorthand/`, `css/`, `css/u/`, `state/`, `fetch/`, `flags/`, `router/`: primary public library packages
- `devtools/`, `head/`, `hotreload/`, `i18n/`, `logging/`, `plugin/`, `prerender/`, `pwa/`, `virtualization/`: companion public packages
- `internal/platform/`, `internal/runtime/`, `internal/runtime2/`: platform adapters and runtime internals
- `testkit/`: reusable consumer-facing test helpers
- `test/`: repo-owned validation suites, fixtures, and browser coverage
- `tools/gwc/`, `tools/livereload/`, `tools/runnerconfig/`: developer tooling and repo automation
- `docs/`: prose documentation and backlog tracking
- `examples/`: examples, showcase apps, and validation fixtures
- `agents/`, `scripts/`: local agent instructions and repository helper scripts
- `third_party/`: pinned external dependencies and cached tool payloads used by this repo
- `bin/`, `log/`: ignored local runtime outputs and local log sinks

For the fuller human-oriented docs entrypoint, use [docs/REFERENCE_MANUAL/README.md](docs/REFERENCE_MANUAL/README.md).
Local package maps live in the directory READMEs for the larger implementation areas, especially `internal/platform/`, `internal/runtime/`, `internal/runtime2/`, and `tools/runnerconfig/`.

## Starter App Example

`main.go`:

```go
package main

import (
    "fmt"

    "github.com/monstercameron/GoWebComponents/css"
    "github.com/monstercameron/GoWebComponents/css/u"
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
    OnIncrement   ui.Handler
}

func CounterPanel(props CounterPanelProps) ui.Node {
    return Div(
        // Layer 2: Tailwind-shaped typed utilities from css/u — autocompleted and
        // compile-checked, no class-string parsing.
        u.Class(
            u.Flex, u.FlexCol, u.Gap(u.Spacing3),
            u.Rounded(u.RadiusXl), u.Border(u.Slate200), u.Bg(u.White), u.Pad(u.Spacing5),
        ),
        P(u.Class(u.TextSize(u.TextSm), u.Fg(u.Slate600)), Textf("Hello, %s.", props.Name)),
        P(u.Class(u.TextSize(u.TextLg), u.FontSemibold, u.Fg(u.Slate900)), Textf("Count: %d", props.Count)),
        P(u.Class(u.TextSize(u.TextSm), u.Fg(u.Slate500)), Textf("Previous count: %s", props.PreviousCount)),
        Button(
            Type("button"),
            OnClick(props.OnIncrement),
            // Layer 1: typed raw CSS with a hover variant — folds into one hashed
            // class. css.Class is an html.PropOption, so it drops into any tag.
            css.Class(
                css.Rounded(css.Rem(0.75)),
                css.PaddingX(css.Rem(1)), css.PaddingY(css.Rem(0.5)),
                css.Bg(css.Slate900), css.TextColor(css.White),
                css.Hover(css.Bg(css.Slate700)),
            ),
            "Increment",
        ),
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
        u.Class(u.Block, u.Bg(u.Slate100), u.PadX(u.Spacing6), u.PadY(u.Spacing12), u.Fg(u.Slate900)),
        Div(
            u.Class(u.Flex, u.FlexCol, u.Gap(u.Spacing6)),
            H1(u.Class(u.TextSize(u.Text4xl), u.FontBold), props.Title),
            P(u.Class(u.TextSize(u.TextSm), u.Fg(u.Slate600)), "A small starter that uses dot-imported shorthand tags with typed css/u utilities for state, events, composition, and reactive text."),
            Input(
                Type("text"),
                Value(name.Get()),
                OnInput(updateName),
                Placeholder("Who is using the app?"),
                u.Class(u.WFull, u.Rounded(u.RadiusXl), u.Border(u.Slate300), u.Bg(u.White), u.PadX(u.Spacing4), u.PadY(u.Spacing3), u.TextSize(u.TextSm)),
            ),
            If(name.Get() == "", P(u.Class(u.TextSize(u.TextSm), u.Fg(u.Amber500)), "Tip: enter a name to personalize the panel.")),
        ),
        ui.CreateElement(CounterPanel, CounterPanelProps{
            Name:          name.Get(),
            Count:         count.Get(),
            PreviousCount: previousLabel,
            OnIncrement:   increment,
        }),
        P(u.Class(u.TextSize(u.TextXs), u.Fg(u.Slate500)), Textf("Current count is %d", count.Get())),
    )
}

func main() {
    // ui.Run builds the component, mounts it at "#app", and keeps the wasm
    // program alive — the one-line equivalent of ui.Render + utils.WaitForever.
    ui.Run("#app", StarterApp, StarterAppProps{
        Title:        "Starter App",
        InitialCount: 0,
    })
}
```

This version stays small, but it shows the normal flow most React users expect:

- `UseState` for local UI state
- `UseEvent` for typed event handlers
- Component composition with a child `CounterPanel`
- `UsePrevious` for render-time comparisons
- Dot-imported `html/shorthand` tags and helpers such as `Div`, `Button`, `Textf`, and `If`
- Typed, compile-checked styling: Tailwind-shaped utilities from `css/u` (`u.Flex`, `u.Gap(u.Spacing3)`, `u.Bg(u.White)`, responsive/state variants like `u.Md(...)` / `u.Hover(...)`) layered over raw typed CSS from `css` (`css.Bg(css.Slate900)`, `css.Hover(...)`), which folds into a single hashed class. No Tailwind toolchain or `class` string parsing — `u.Class(...)` / `css.Class(...)` are `html.PropOption`s that drop into any tag, and the wasm sink auto-injects the generated `<style>`.

Mounting the app does not require a global stylesheet: the styles above are emitted by the `css` engine at render time. The host HTML stays minimal.

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

- Components return `ui.Node` and are mounted with `ui.Run(...)` (the one-line entrypoint) or, when you need to do work after mount, with `ui.Render(...)` plus `utils.WaitForever()`.
- The `html` package provides the stable typed DOM builders such as `html.Div`, `html.Button`, `html.Input`, and `html.Tag`.
- The `html/shorthand` package provides ergonomic mixed-argument sugar for dot-imported tags and helper funcs such as `Div`, `Button`, `Class`, `If`, `Text`, and `Textf`.
- The `css` package provides typed, compile-checked styling: raw typed CSS (`css.New` / `css.Class` with properties, values, and pseudo/at-rule variants) plus the Tailwind-shaped utility layer in `css/u` (`u.Flex`, `u.Gap(u.Spacing3)`, `u.Hover(...)`, `u.Md(...)`). Styles fold into hashed classes emitted through a Sink that auto-injects in the browser and serializes to a `<style>` block for SSR.
- Local component behavior lives in `ui` hooks such as `UseState`, `UseEffect`, `UseReducer`, `UseRef`, and `UseEvent`.
- Shared application state lives in `state`, with atoms, derived values, computed values, and snapshot helpers.
- Routing, fetch helpers, SSR, hydration, and diagnostics are layered on top of the same runtime rather than split into unrelated packages.

## Public Packages

The preferred public surface is:

- `ui`: component composition, hooks, rendering, hydration, async boundaries, events, portals, and form helpers
- `html`: stable typed HTML builders and DOM prop metadata
- `html/shorthand`: mixed-argument authoring sugar, helper funcs, and dot-import-friendly host tags layered on `html`
- `css`: typed, compile-checked CSS — hashed atomic classes from typed properties/values, pseudo and at-rule variants, theme tokens, dynamic custom-property values, and SSR `<style>` serialization with hydration seeding
- `css/u`: Tailwind-shaped typed utility layer over `css` (`u.Flex`, `u.Gap`, `u.Bg`, `u.Rounded`, responsive `u.Md`/`u.Lg`, state `u.Hover`/`u.Focus`, `u.Dark`) resolved against the active theme
- `state`: atom-based shared state, derived state, computed values, and snapshot helpers
- `fetch`: browser fetch helpers, typed resources, tag-aware query cache, realtime `UseWebSocket` / `UseEventSource` hooks, and imperative fetch flows
- `flags`: browser-visible feature flag and deterministic experiment helpers
- `router`: hash routing, browser routing, params, query helpers, redirects, loaders, guards, metadata, nested layouts, and hydration-aware mount helpers
- `i18n`: locale provider, pluralization, number/date formatting, and SSR-aligned locale bootstrap
- `interop`: typed browser bridges for storage, clipboard, custom events, observers, cross-tab/multi-window channels, and lazy module loading
- `pwa`: service-worker registration, cache-storage plans, installability observation, and offline mutation replay
- `virtualization`: windowed list rendering with viewport diagnostics and scroll restoration for long feeds
- `devtools`: embeddable inspection, diagnostics, profiling hints, and snapshots
- `head`: optional companion SSR head composition helpers for router metadata, social tags, robots tags, JSON-LD, alternate locale links, and resource hints
- `plugin`: supported companion host for explicit plugin manifests, capability-checked registration, and subsystem hook contributions layered on public APIs
- `hotreload`: state-preserving development reload bridge and snapshot helpers for standalone wasm apps

## Feature Overview

### Rendering and Hooks

- React-style function components with a fiber-based runtime and browser-side rendering through `syscall/js`
- Hooks including `UseState`, `UseReducer`, `UseEffect`, `UseRef`, `UsePrevious`, `UseId`, `UseDeferredValue`, `UseTransition`, and context support
- Event and async helpers including `UseEvent`, `UseChannel`, `UseTask`, `UseDebounced`, `UseThrottled`, `UseLazyNode`, `AsyncBoundary`, and `UseForm`
- Crash containment by default: panics in render, events, effects, cleanup, and async work (`ui.SafeGo(...)`) are caught at every boundary and reported as structured, agent-readable console diagnostics instead of killing the page (see [docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md](docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md))

### Styling

- Typed, compile-checked CSS in the `css` package: `css.New(...)` / `css.Class(...)` fold typed properties and values (`css.Display.Flex`, `css.Gap(css.Px(8))`, `css.Bg(css.Slate900)`) plus pseudo and at-rule variants (`css.Hover(...)`, `css.Media(css.MinW(768), ...)`) into a single hashed atomic class
- Tailwind-shaped utility layer in `css/u`, resolved against the active `css.Theme`: `u.Flex`, `u.Gap(u.Spacing3)`, `u.Rounded(u.RadiusXl)`, responsive `u.Md(...)` / `u.Lg(...)`, and state `u.Hover(...)` / `u.Focus(...)` / `u.Dark(...)` — all typed symbols, so a typo is a compile error, not a silent no-op
- `css.Class(...)` / `u.Class(...)` are `html.PropOption`s that drop into any `html` or `shorthand` tag with no edits to those packages; the wasm sink auto-injects a managed `<style>`, and `css.StyleBlock()` + `css.SeedFromDocument()` carry the same rules through SSR and hydration
- Dynamic runtime values stay on a stable class via `css.Dynamic*` (a CSS custom property set inline), avoiding a new class per distinct value

### State and Data

- Shared atom-based state with subscriptions, derived atoms, computed values, snapshot export or import, and optional browser-storage restore flows
- Fetch helpers including low-level `UseFetch`, typed `UseResource[T]`, realtime `UseWebSocket` / `UseEventSource`, and imperative `Fetch(...)`

### Routing

- Hash router and browser/history router support
- Params, query helpers, redirects, route metadata, guards, loaders, manual revalidation, nested layout routes, and `router.Outlet()`

### SSR and Hydration

- Native server-side rendering through `ui.RenderToString(...)`
- Streaming SSR through `ui.RenderToStream(...)` / `ui.RenderToStreamObserved(...)`: a shell chunk flushes before unresolved `AsyncBoundary` content, then out-of-order replacement chunks flush as suspensions resolve
- Real async suspension via render-time `ui.SuspendUntil(...)` / `ui.Await(...)` with fallback capture and retry on resolve
- Browser hydration through `ui.Hydrate(...)`, plus multi-root selective activation (static islands) through `ui.HydrateInto(...)`
- Bootstrap transport helpers for inline JSON, sidecar JSON, and optional CBOR payload encoding or decoding

### Tooling and Validation

- Public `devtools` package for in-app inspection and diagnostics
- `go run ./tools/gwc import -src .\path\to\layout.html -out .\bin\converter\layout\main.go` converts static `.html`, `.htm`, `.jsx`, or `.tsx` files into an inspectable GWC `main.go` built from current public `html` builders
- `go run ./tools/gwc examples` serves the example catalog through the repo-standard Go runner
- `go run ./tools/gwc dev -app .\path\to\main.go` starts the standalone wasm inner loop with rebuild-on-save and hotreload support
- `gwc build` / `gwc release` support a guarded `tinygo` profile (`-target=wasm -opt=z`) for smaller leaf-app binaries and a `source-debug` profile (`-gcflags=all=-N -l`, untrimmed paths) for browser stack correlation
- `go run ./tools/gwc test -lane unit -lane wasm -lane browser` runs the supported launcher-owned validation lanes
- The documentation site under `examples/site` is itself a pure GoWebComponents application: every page, style, and behavior is Go compiled to wasm, with the single boot shell generated by `tools/sitegen` (no authored `.html`/`.js`/`.css`)
- Launcher-owned temporary artifacts now resolve under `bin/tmp/` beneath the relevant project root instead of the OS temp directory
- Native Go tests, js/wasm tests, browser suites, and benchmark coverage
- Large example suite spanning local state, forms, routing, async work, SSR, hydration, nested routes, and diagnostics

## SSR and Hydration

GoWebComponents supports both request-time server rendering and browser hydration.

Current repo examples cover:

- `ui.RenderToString(...)` for server-rendered HTML generation on native Go targets
- `ui.Hydrate(...)` for resuming matching DOM in the browser
- bootstrap payload helpers for transferring route data, atoms, IDs, and initialization data
- request-time SSR with route-aware hydration under `examples/server/server-side-rendering-routing`
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
- Example entrypoint: `http://127.0.0.1:8090/examples/public-examples-site/`
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
go run ./tools/gwc verify -app .\examples\public\counter\main.go -root .\examples\public\counter
```

Wasm-only runtime tests on Windows:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/runtime
```

For the broader browser harness and focused browser flows, use [test/README.md](test/README.md) as the authoritative reference.

## Performance and Trade-offs

A frank scorecard — the good and the bad — so you can decide with eyes open.

**What it's genuinely strong at:**

- **One language, one toolchain.** The entire UI — rendering, state, routing,
  fetch, SSR, hydration — is Go compiled to wasm. No JavaScript build, bundler
  config, or npm dependency tree, and no context-switching between a Go backend
  and a separate JS frontend.
- **Shared types end-to-end.** The same Go structs and validation logic drive the
  server and the browser, so route data, request/response shapes, and form models
  can't silently drift across a language boundary.
- **Batteries included.** Routing, shared state, data fetching, SSR with real
  async suspension, hydration/static islands, i18n, accessibility primitives,
  PWA/offline, feature flags, and devtools all ship in one module — you assemble
  far less to reach a complete app.
- **Crash containment by default.** Panics in render, events, effects, cleanup,
  and async work are caught at every boundary and surfaced as structured,
  agent-readable diagnostics instead of killing the page.
- **SSR-first friendly.** Server-render real HTML, stream a shell before async
  content resolves, then hydrate — the wasm payload resumes already-painted markup
  rather than blocking first paint on download.
- **Ships compressed automatically.** Release tooling precompresses and serves
  brotli/gzip, so users download ~20% of the raw artifact (details below).

**What it costs (be honest about these):**

- **Not native speed, and not faster than React.** GoWebComponents renders in
  WebAssembly through `syscall/js`, so it is not a native-JavaScript framework and
  makes no "native" or "near-native" DOM-speed claim. In the local
  render-benchmark, a native-JS React 19.2.4 baseline typically leads on raw
  DOM-ready latency; GoWebComponents' runtime-1 path is competitive mainly at the
  paint-proxy (one-frame) boundary. wasm DOM calls cross the `syscall/js` bridge,
  which is inherently slower than JavaScript touching the DOM directly. The value
  is writing the whole UI in Go, not out-rendering hand-tuned JavaScript. See
  [Benchmarks](#benchmarks) to reproduce both sides.
- **Bundle size is larger than JS, but compresses dramatically.** Standard
  Go→wasm binaries embed the Go runtime, so the raw artifact is big — a minimal
  `counter` app is ~6.3 MB raw. But wasm is highly compressible, and the raw size
  is *not* what ships: the same app is **1.70 MB gzipped (3.7x) and 1.24 MB under
  brotli (5.1x — only ~20% of raw)**. GoWebComponents' `gwc release` tooling
  precompresses artifacts and serves `Content-Encoding: br`/`gzip` automatically,
  so users download the compressed payload, not the raw `.wasm`. Shrink further
  with the guarded `tinygo` build profile (`gwc build -profile tinygo`,
  `-target=wasm -opt=z`) for TinyGo-compatible leaf apps, and measure raw plus
  compressed sizes for any package with
  `go run ./tools/gwc wasm measure -package .\path\to\app`. Even compressed, the
  first-load payload is still larger than a comparable native-JS bundle —
  GoWebComponents trades that for a single Go codebase and toolchain.
- **Maturing in places.** The framework is broad and the core runtime is well
  tested, but some newer surfaces (parts of SSR router-data, certain companion
  packages) are still settling. Pin versions and read the
  [API stability policy](docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md)
  before depending on the experimental edge.
- **Smaller community than React/Vue.** Fewer third-party components, examples,
  and StackOverflow answers, and a smaller hiring pool — you trade ecosystem mass
  for a single-language stack.

**Best fit — reach for GoWebComponents when:**

- You are a Go-centric team that wants to stop building and maintaining a parallel
  JavaScript frontend stack.
- You are building internal tools, dashboards, admin panels, or line-of-business
  apps where Go-end-to-end and type safety matter more than shaving the last
  milliseconds of render latency.
- You already share substantial Go types and logic between server and client and
  want them to stay in sync by construction.
- You want SSR-first delivery where wasm hydrates already-rendered HTML, so first
  paint does not wait on the bundle.

**Reach for something else when:**

- The hard requirement is the smallest possible payload to anonymous, first-time,
  mobile visitors — a native-JS framework still ships smaller initial bundles.
- You need to beat hand-tuned React/Svelte on raw client-side render latency.
- You depend on the breadth of the npm/React ecosystem, its component libraries,
  or a large established hiring pool today.

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

Release-style wasm comparisons and benchmark reporting are driven through [docs/REFERENCE_MANUAL/02-gwc-workflows.md](docs/REFERENCE_MANUAL/02-gwc-workflows.md), [docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md](docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md), [docs/REFERENCE_MANUAL/13-assets-deployment-and-pwa.md](docs/REFERENCE_MANUAL/13-assets-deployment-and-pwa.md), and [tools/README.md](tools/README.md), especially `gwc build`, `gwc release`, and `gwc bench`.

Following the no-drift policy in [docs/BENCHMARKS.md](docs/BENCHMARKS.md), this
README does not hard-code result figures: the canonical numbers live in the
checked-in JSON reports and the generated browser report, which cannot silently
drift from prose. Reproduce the whole-program native + wasm sweep with:

```bash
go run ./tools/gwc bench            # -> docs/benchmarks/latest.json
```

Browser render comparison (Playwright vs a vendored React 19.2.4 baseline):

```bash
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample201BrowserBenchmarkReport -v
```

That test measures per-scenario DOM-ready and paint-proxy latency and writes a
full report to `bin/test-results/example-201-browser-benchmark/` (a local
artifact); the checked-in scoring reference is
`examples/testing/render-benchmark/score-reference.json`. In the latest local run
(`2026-06-11`, chromium, React 19.2.4), and as expected for Go/wasm versus
native-JS React, React leads on raw DOM-ready logic time while the runtime-1 path
is frequently ahead at the paint-proxy (one-frame) boundary.

Recent verified native-runtime improvements (same-machine before/after on
`go1.26.3`, `windows/arm64`; see
[docs/benchmarks/MICROBENCH_REPORT_arm64.md](docs/benchmarks/MICROBENCH_REPORT_arm64.md)
for the full sweep of 500 benchmarks):

- Inspector snapshot (`collectFlamegraphFrames`): `36,150 B/op` -> `5,348 B/op`
  (-85%), `~6,900 ns/op` -> `~3,600 ns/op` (~1.9x faster)
- Telemetry redaction array path (`redactValue`): `4,175 ns/op` -> `2,945 ns/op`
  (-29%), `65` -> `48 allocs/op`
- SSR attribute serialization (`writeSSRProps`): `-100 allocs/op` per 100-row
  server render

Earlier element-construction and subscription passes (historical, prior machine):

- `DivWithComponents4`: `1654 ns/op` -> `736.9 ns/op`
- `WithComponentsGeneric4`: `1820 ns/op` -> `829.0 ns/op`
- `CleanupAtomSubscriptions8`: `4477 ns/op` -> `3629 ns/op`

## Development Workflow

Use the `gwc` runner as the repo-standard entrypoint for local workflows.

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc examples
go run ./tools/gwc dev -app .\examples\public\counter\main.go
go run ./tools/gwc build -app .\examples\public\counter\main.go -profile ci
go run ./tools/gwc build -app .\examples\public\counter\main.go -profile tinygo
go run ./tools/gwc release -app .\examples\public\counter\main.go -out-dir .\bin\gwc-release
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
- Preferred public packages are `ui`, `html`, `html/shorthand`, `css`, `css/u`, `state`, `fetch`, `flags`, `router`, `devtools`, and `hotreload`
- Example and test fixture code now builds through current `ui`/`html` bridge helpers and shorthand sugar instead of older compatibility layers
- Native `internal/runtime` tests pass and report ≈88% statement coverage (reproduce with `go test ./internal/runtime -cover`)
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
- [docs/STARTERS.md](docs/STARTERS.md)
- [docs/REFERENCE_MANUAL/README.md](docs/REFERENCE_MANUAL/README.md)
- [docs/REFERENCE_MANUAL/01-getting-started.md](docs/REFERENCE_MANUAL/01-getting-started.md)
- [docs/REFERENCE_MANUAL/02-gwc-workflows.md](docs/REFERENCE_MANUAL/02-gwc-workflows.md)
- [docs/REFERENCE_MANUAL/03-app-shapes.md](docs/REFERENCE_MANUAL/03-app-shapes.md)
- [docs/REFERENCE_MANUAL/09-ssr-and-hydration.md](docs/REFERENCE_MANUAL/09-ssr-and-hydration.md)
- [docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md](docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md)
- [docs/REFERENCE_MANUAL/13-assets-deployment-and-pwa.md](docs/REFERENCE_MANUAL/13-assets-deployment-and-pwa.md)
- [docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md](docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md)
- [examples/README.md](examples/README.md)
- [test/README.md](test/README.md)
- [tools/README.md](tools/README.md)

## Notes

- Older references to `fiber/` are obsolete; the runtime now lives under `internal/runtime/`.
- The repo-standard workflow now goes through `go run ./tools/gwc ...` instead of ad hoc local launcher scripts.
- The browser-compiler example may generate large local package archives under `examples/public/browser-compiler/static/pkg/`; those artifacts should remain ignored.

