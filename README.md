# GoWebComponents

GoWebComponents is a Go + WebAssembly UI framework with a React-style component model, hooks, a fiber-based runtime, and browser-side rendering through `syscall/js`.

## Status

Current repo state as of 2026-03-14:

- Core runtime lives in `internal/runtime/`
- Public browser-facing packages are `dom`, `hooks`, `render`, `state`, `fetch`, and `router`
- Native `internal/runtime` statement coverage is `100%`
- Native runtime tests pass with `go test ./internal/runtime`
- Browser component, integration, and deep state stress suites pass under Playwright
- Separate `js/wasm` tests and benchmarks exist for wasm-only runtime and adapter paths

## Install

```bash
go get github.com/monstercameron/GoWebComponents@latest
```

Requirements:

- Go 1.22+
- A browser that supports WebAssembly
- Node.js only for the example dev server and Playwright-based browser tests

## Minimal Example

```go
package main

import (
    "fmt"

    "github.com/monstercameron/GoWebComponents/dom"
    "github.com/monstercameron/GoWebComponents/hooks"
    "github.com/monstercameron/GoWebComponents/render"
)

func Counter(props dom.Attrs) *render.Element {
    count, setCount := hooks.UseState(0)

    increment := hooks.GoUseFunc(func() {
        setCount(func(prev int) int { return prev + 1 })
    })

    return dom.Div(nil,
        dom.H1(nil, dom.Text("Counter")),
        dom.P(nil, dom.Text(fmt.Sprintf("Count: %d", count()))),
        dom.Button(dom.Attrs{"onclick": increment}, dom.Text("Increment")),
    )
}

func main() {
    render.To(dom.CreateElement(Counter, nil), "#app")
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

- `dom`: element constructors and DOM attribute helpers
- `hooks`: local state, effects, memoization, refs, IDs, event wrappers, fetch hook
- `render`: browser runtime bootstrap and mounting helpers
- `state`: shared atom-based state
- `fetch`: browser fetch helpers layered on top of the runtime hook/fetch APIs
- `router`: browser/hash routing helpers

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

- Render: `GoWebComponents 104 ms`, `React 47 ms`
- Update: `GoWebComponents 48 ms`, `React 44 ms`
- Clear: `GoWebComponents 49 ms`, `React 41 ms`
- Deep tree: `GoWebComponents 69 ms`, `React 62 ms`
- Hooks: `GoWebComponents 83 ms`, `React 79 ms`

Notes:

- The browser benchmark currently compares render, update, clear, deep-tree, and many-hooks scenarios from `tests/performance.spec.ts`.
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
