# 02 GWC Workflows

Use this chapter when you want the supported command-line workflow for GoWebComponents instead of assembling ad hoc shell scripts from commit history.

It is the right chapter for:

- environment checks
- project bootstrap and scaffold setup
- example browsing
- standalone app dev loops
- builds, release packaging, and validation
- runner config ownership
- project inspection, file inventory, and seed or benchmark tasks

Use another chapter instead when:

- you need the first app shape rather than the command surface: go to [01 Getting Started](01-getting-started.md)
- you need product architecture choices such as SPA versus SSR: go to [03 App Shapes](03-app-shapes.md)
- you need static export, deployment targets, or PWA boundaries in detail: go to [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)

## Overview

`gwc` is the canonical tooling surface for this repo and for applications built on top of it.

The current public launcher flow is:

- check the machine with `doctor`
- bootstrap or scaffold with `bootstrap` and `start`
- explore or run examples with `examples`
- use `dev` for the inner loop
- use `build` and `release` for artifacts
- use `test`, `verify`, and `lint` for validation
- use `files`, `import`, `bench`, `wasm`, `dashboard`, and `seed` for focused operational tasks

Use the launcher from the repo root:

```powershell
go -C tools run ./gwc <command> [flags]
```

Use `go -C tools run ./gwc <command> -h` for exact flag details. This chapter explains when to use each command and how the commands fit together.

## Stability Note

The `gwc` workflow is the documented launcher contract for this repo.

Practical rule:

- prefer `gwc` when a documented command already owns the workflow
- prefer direct low-level `go build` or `go test` only when you intentionally need compiler-level control outside the launcher contract

The exact flag surface can grow over time, but the high-level workflow categories documented here are the supported path for app authors.

## Minimal Workflow

If you only need the shortest path from repo checkout to a running example, use this sequence:

```powershell
go -C tools run ./gwc doctor
go -C tools run ./gwc examples
go -C tools run ./gwc dev -app .\examples\public\counter\main.go
go -C tools run ./gwc verify -app .\examples\public\counter\main.go -root .\examples\public\counter
```

What each command proves:

- `doctor` checks the local machine and runtime prerequisites
- `examples` gives you the fastest entrypoint into the example catalog
- `dev` runs the active inner loop for one app
- `verify` proves the app can pass launcher-owned validation and a CI-style wasm build

## Production-Shaped Workflow

For a real app, the workflow usually becomes: scaffold, configure, run, validate, release.

`main.go` with hot reload opt-in:

```go gwc:build
package main

import (
	"github.com/monstercameron/GoWebComponents/v7/hotreload"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// renderRootApp returns the root application node for the standalone dev loop.
func renderRootApp() ui.Node {
	return ui.Text("hello from gwc dev")
}

// main enables state-preserving reload for gwc dev and mounts the app.
func main() {
	hotreload.Enable()
	ui.Run("#app", renderRootApp)
}
```

`gwc-runner.json`:

```jsonc
{
  "paths": {
    // Keep launcher-owned outputs under one predictable root for the workspace.
    "artifactRoot": "bin",
    "workspaceBuildRoot": "bin",
    "browserWorkspace": "test",
    "livereloadWorkspace": "tools/livereload"
  }
}
```

Typical command sequence:

```powershell
go -C tools run ./gwc doctor
go -C tools run ./gwc start
go -C tools run ./gwc dev -app .\main.go
go -C tools run ./gwc test -lane unit -lane wasm
go -C tools run ./gwc verify -app .\main.go -root .
go -C tools run ./gwc release -app .\main.go -out-dir .\bin\release -compression gzip+brotli
```

Why this is the normal production-shaped path:

- the app stays on the documented launcher contract
- the dev loop, validation, and release artifacts all resolve through one tool surface
- runner config captures stable workspace rules while short-lived overrides remain on flags
- hot-reload WebSocket messages use `gwc.livereload.ws` v1 and the app bridge uses `gwc.hotreload.snapshot` v1; older unversioned payloads are accepted, but future mismatches are diagnosed and ignored

## Scale-Up Workflow

For a larger team or CI pipeline, treat `gwc-runner.json` as a stable workspace policy layer and keep temporary operator intent on CLI flags.

Example policy-shaped runner config:

```jsonc
{
  "paths": {
    "generatedProjectRoot": "generated-projects",
    "artifactRoot": "enterprise-artifacts",
    "workspaceBuildRoot": "bin",
    "wasmExecJS": "vendor/wasm_exec.js",
    "goWasmExec": "tools/go_js_wasm_exec.bat",
    "browserWorkspace": "test",
    "livereloadWorkspace": "tools/livereload"
  }
}
```

One practical team sequence is:

```powershell
go -C tools run ./gwc doctor -audit
go -C tools run ./gwc test -lane unit -lane wasm -lane browser
go -C tools run ./gwc verify -app .\main.go -root . -audit
go -C tools run ./gwc release -app .\main.go -out-dir .\bin\release -compression gzip+brotli -validate-smoke
go -C tools run ./gwc bench -root .
go -C tools run ./gwc wasm measure -package .\examples\public\ui-render
```

Why this scales:

- `doctor -audit` and `verify -audit` give the team one policy-aware health gate
- `test` keeps lane selection explicit instead of burying it in custom scripts
- `release` centralizes manifest, compression, and smoke validation
- `bench` and `wasm measure` provide repeatable measurement paths when performance matters

## Command Reference

### Bootstrap And Setup

| Command | Use it when | Representative call |
| --- | --- | --- |
| `doctor` | you need prerequisite checks, runtime-asset checks, or port/audit signals | `go -C tools run ./gwc doctor -audit` |
| `bootstrap` | you want prerequisite checks and then either scaffold or examples bootstrap flow | `go -C tools run ./gwc bootstrap -examples` |
| `start` | you want the scaffold TUI for a new app | `go -C tools run ./gwc start -mode standalone` |
| `examples` | you want the example catalog or managed example-server lifecycle commands | `go -C tools run ./gwc examples` |

Practical distinction:

- use `bootstrap` when you want checks plus a guided next step
- use `start` when you already know you want scaffold generation
- use `examples` when the fastest path is studying a real maintained example

### Inner Loop

| Command | Use it when | Representative call |
| --- | --- | --- |
| `dev` | you want rebuild, serve, livereload, and optional hot reload for one app | `go -C tools run ./gwc dev -app .\main.go` |
| `serve` | you already have built assets and only need a narrow static server | `go -C tools run ./gwc serve -root .\static -wasm-file .\bin\app.wasm` |
| `dashboard` | you want a launcher-owned view of live dev status and provider config | `go -C tools run ./gwc dashboard -root .` |

Use `dev -dry-run` or `dev -json` when you want to inspect the resolved plan before you start the server.

#### Live Dev Server (`gwc dev`)

`gwc dev` is the live-reload server for the inner loop: it builds your app to `js/wasm`, serves it,
watches the source tree, and pushes rebuilds and reloads to the browser over a WebSocket. It runs until
you stop it (Ctrl-C).

```powershell
go -C tools run ./gwc dev                                   # run from a directory with main.go
go -C tools run ./gwc dev -app .\examples\public\counter    # or point at an app explicitly
go -C tools run ./gwc dev -port 9000 -host 0.0.0.0          # bind elsewhere
```

**Flags.** Run `go -C tools run ./gwc dev -h` for the always-current list. The ones you reach for most:

| Flag | Default | Purpose |
| --- | --- | --- |
| `-app` | auto-detect | App `main.go` file or its directory |
| `-root` | app directory | Project root to watch and serve |
| `-html` | `index.html` if present | HTML file to serve, relative to the root |
| `-host` / `-port` | `127.0.0.1` / `8080` | Bind address |
| `-hot` | `true` | Hot-reload the browser after each successful rebuild |
| `-tui` | `false` | Interactive terminal status view |
| `-agent` | `false` | Emit agent-native NDJSON dev-loop events |
| `-dry-run` / `-json` | `false` | Resolve and print the dev plan without starting the server |
| `-no-doctor` | `false` | Skip the automatic `gwc doctor` diagnosis on a failed start |

**How settings are resolved.** Each of host, port, app, root, and html is resolved in precedence order,
so you only set what you need:

1. an explicit flag (`-port 9000`)
2. `gwc-start.json` (`Tooling.DevHost` / `Tooling.DevPort`, plus scaffold metadata for app/root/html)
3. auto-detection — app entrypoint `./main.go` then `./cmd/web/main.go`; html `./index.html` if it exists
4. convention fallback — host `127.0.0.1`, port `8080`

`dev -dry-run -json` prints the resolved plan (and where each value came from) without binding a port.

**How live reload works.** A file watcher debounces changes (it waits ~0.5s after a single edit, longer
while you keep typing, and forces a build after at most 5s), then:

- a `.go` change triggers a wasm rebuild and a full browser reload;
- a `.css` change is **hot-swapped in place** — the stylesheet is replaced with no rebuild and no
  reload, so the app keeps its state;
- the browser is kept in sync over a WebSocket; if it drops, the client reconnects automatically.

If the app panics or the build fails, a full-screen **runtime error overlay** appears in the page (with
a copyable stack trace) instead of failing silently in the console.

**Endpoints** (host:port from above):

| Path | Purpose |
| --- | --- |
| `/` | Your HTML, with the livereload client script injected |
| `/ws` | WebSocket carrying `build_start` / `build_complete` / `build_error` / `reload` / `asset_swap` events |
| `/__gwc/status` | JSON build and server status (what `-tui` and `gwc dashboard` read) |

If the server fails to start, `gwc dev` automatically runs `gwc doctor` and prints the likely cause
(wrong Go version, missing `wasm_exec.js`, busy port). Suppress it with `-no-doctor`.

**Related servers.** Use `gwc serve` when you already have built assets and only need a narrow static
server (no watching). Use `gwc dashboard` to monitor a running dev server's `/__gwc/status` and provider
config from the terminal.

### Build, Release, And Validation

| Command | Use it when | Representative call |
| --- | --- | --- |
| `build` | you want one explicit `js/wasm` artifact | `go -C tools run ./gwc build -app .\main.go -profile development` |
| `release` | you want a packaged release directory plus manifest and compression sidecars | `go -C tools run ./gwc release -app .\main.go -out-dir .\bin\release` |
| `test` | you want explicit launcher-owned lanes such as `unit`, `wasm`, `hydration`, `browser`, or `release` | `go -C tools run ./gwc test -lane unit -lane wasm` |
| `verify` | you want the higher-signal pre-commit or CI path for one app | `go -C tools run ./gwc verify -app .\main.go -root . -audit` |
| `lint` | you want launcher-owned lint reporting instead of a raw tool invocation | `go -C tools run ./gwc lint -root . -json` |

Use `release -validate-smoke` when you want the post-build release smoke check, not only the artifact packaging.

Build profiles are intentionally distinct:

| Profile | Toolchain | Main flags | Use it when |
| --- | --- | --- | --- |
| `development` / `dev` | Go `js/wasm` | `-ldflags=-w` | you want the normal fast inner-loop artifact |
| `debug` / `source-debug` | Go `js/wasm` | `-gcflags=all=-N -l`, no `-trimpath`, no strip flags | you need symbol-stable browser debugging and readable local paths |
| `ci` / `verify` | Go `js/wasm` | `-trimpath -ldflags=-s -w -buildvcs=false -tags production` | you want release-shaped compile validation |
| `benchmark` / `bench` | Go `js/wasm` | same release-shaped flags as CI | you need repeatable measurement artifacts |
| `release` / `prod` | Go `js/wasm` | same release-shaped flags as CI | you want a deployable production artifact |
| `tinygo` | TinyGo `wasm` | `-target=wasm -opt=z -tags production` | you are proving a TinyGo-compatible leaf app |

### Code Generation

These commands turn stringly-typed surfaces into compile-checked Go, each with a `-check` CI staleness
gate (regenerate + diff, non-zero exit on drift):

| Command | Use it when | Representative call |
| --- | --- | --- |
| `routes gen` / `routes check` | you want typed `Link*` constructors from `router.MustDefineRoute` contracts so a bad path param is a compile error | `go -C tools run ./gwc routes gen -pkg .` |
| `i18n gen` / `i18n check` | you want typed message accessors (`i18n_keys_gen.go`) from a base-locale bundle so a missing key or `{param}` is a compile error | `go -C tools run ./gwc i18n gen -bundle .\messages.en.json -pkg .` |
| `server gen` / `server check` | you want browser stubs + server wiring generated from `//gwc:server` functions | `go -C tools run ./gwc server gen -pkg .` |
| `css gen` / `css check` | you want typed CSS-token constants from a theme JSON | `go -C tools run ./gwc css gen -theme .\theme.json -pkg .` |

`server gen` writes `serverfn_gen_client.go` (`js/wasm` stubs calling `serverfn.Call`) and
`serverfn_gen_server.go` (`RegisterServerFunctions(mux)`). See
[07 Data Loading And Mutations](07-data-loading-and-mutations.md).

### Security And Supply Chain

| Command | Use it when | Representative call |
| --- | --- | --- |
| `supplychain` | you want a zero-npm proof, Go module counts, checksum verification, and an optional direct-dependency budget | `go -C tools run ./gwc supplychain -root . -budget 25 -json` |
| `vuln` | you want a known-vulnerability scan that gates on reachable findings (`govulncheck`-backed) | `go -C tools run ./gwc vuln -strict` |

`vuln` classifies each finding as reachable (a call trace names the vulnerable function — fails the
build) or imported-only (passes unless `-strict`).

### Source-Debugging A Wasm App

Use the debug profile when a browser issue needs source-oriented inspection instead of a normal fast dev-loop artifact:

```powershell
go -C tools run ./gwc build -app .\main.go -root . -out .\bin\debug\app.wasm -profile debug -json
go -C tools run ./gwc serve -root .\static -wasm-file .\bin\debug\app.wasm
```

If the issue reproduces only in packaged output, keep the same profile visible in the release manifest:

```powershell
go -C tools run ./gwc release -app .\main.go -root . -out-dir .\bin\debug-release -profile debug -compression none -json
```

The debug profile keeps local source paths untrimmed and disables compiler optimization and inlining with `-gcflags=all=-N -l`. That makes wasm stack frames and DevTools' generated wasm view less surprising when you correlate a browser pause, crash report, or console stack back to Go source.

Current Go `js/wasm` artifacts do not emit browser source maps or `.debug_*` DWARF custom sections. Treat the Go-toolchain workflow as symbolized stack and generated-wasm debugging, not full Go-source stepping. When full DWARF-backed source stepping is required, validate a TinyGo-compatible app separately and keep that as an explicit toolchain experiment instead of changing the normal `development`, `ci`, or `release` profile.

### Inspection, Import, And Project Utilities

| Command | Use it when | Representative call |
| --- | --- | --- |
| `files` | you need a repeatable file inventory | `go -C tools run ./gwc files -root . -ext go -exclude-dir .git` |
| `import` | you want to convert static HTML or JSX into an inspectable GWC `main.go` | `go -C tools run ./gwc import -src .\design\landing.html -out .\bin\landing\main.go` |
| `upgrade` | you need to backfill lifecycle metadata, feature matrix defaults, and runtime assets | `go -C tools run ./gwc upgrade -root . -skip-runtime-assets` |
| `migrate` | you need an upgrade report and safe compatibility API rewrites between framework versions | `go -C tools run ./gwc migrate -root . -apply -json` |
| `seed` | the project exposes a seed package for local identities or fixture data | `go -C tools run ./gwc seed -root .` |
| `add` | you want to copy a headless, a11y-correct component into your repo (shadcn "own the code" model) or scaffold a component/route stub | `go -C tools run ./gwc add tabs -dir .\components` |
| `llms` | you want AI-native docs (`llms.txt` index + `llms-full.txt`) generated from the reference manual | `go -C tools run ./gwc llms -root . -check` |

`migrate` runs the same metadata/runtime upgrade path as `upgrade`, then writes `bin/gwc-migrate-report.json` with compatibility API findings. By default it is report-only. With `-apply`, it rewrites parsed router selector calls from `GoRegisterRoute` to `Register` and `GoGetRoute` to `Current`; quoted code and comments remain untouched and visible in the report so humans can decide what to do with them.

### Measurement

| Command | Use it when | Representative call |
| --- | --- | --- |
| `bench` | you want repeatable benchmark discovery, JSON output, or normalized scoring against a saved reference report | `go -C tools run ./gwc bench -root . -reference .\docs\benchmarks\reference.json` |
| `bench suite` and `bench suite compare` | you want one versioned report (schema `gwc.bench.suite/v1`) across native, wasm, size, and build lanes, and a significance-aware comparison of two reports | `go -C tools run ./gwc bench suite -lane native -count 5 -out bin/bench-suite.json` |
| `wasm measure` | you want a wasm artifact measurement pass with manifest output | `go -C tools run ./gwc wasm measure -package .\examples\public\ui-render` |
| `wasm compare` and related subcommands | you want wasm-specific diffing, compression, cache, or toolchain comparisons | `go -C tools run ./gwc wasm compare-toolchain -package .\examples\public\ui-render -baseline-go go1.25.4 -candidate-go go1.26.0` |
| `buildreport` | you want a "what rebuilt and why" report (rebuilt vs cached packages, ranked by compile time) from the build action graph | `go -C tools run ./gwc buildreport -pattern ./... -json` |

Use the subcommand help directly for the wasm tools:

```powershell
go -C tools run ./gwc wasm measure -h
go -C tools run ./gwc bench -h
```

### Benchmark Suite (`gwc bench suite`)

`gwc bench` runs and scores benchmarks; `gwc bench suite` runs a fixed set of lanes and writes ONE versioned report (schema `gwc.bench.suite/v1`, default `bin/bench-suite.json`) so two runs can be compared later. Run it from the repository root, because the size lane builds relative to the current directory.

```powershell
go -C tools run ./gwc bench suite -lane native -count 5 -shuffle -seed 6001 -out bin/bench-suite.json
go -C tools run ./gwc bench suite compare -max-regress-pct 10 bin/baseline.json bin/bench-suite.json
```

Lanes are `native`, `wasm` (micro benchmarks), `size` (release wasm bytes) and `build` (first build against rebuild); all four run by default. The browser, heap and health lanes are written as `not-run` placeholders so a report never silently omits them. Flags, from `tools/gwc/bench_suite.go`:

| Flag | Meaning |
| --- | --- |
| `-lane` | lanes to run; repeatable or comma-separated |
| `-count` | samples per benchmark (`go test -count`, default 5) |
| `-warmup` | discarded warm-up runs per package (default 1); applies to the native lane, and to wasm only when set explicitly |
| `-shuffle`, `-seed` | randomize package order with a seed; `-seed 0` picks one from the clock and records it, and the order is recorded in `run.order` |
| `-include` | root-relative directory to opt in even if skipped by default (for example `test/render`); repeatable |
| `-resume` | skip targets already finished in `<out>.partial.json`; refuses when the configuration differs |
| `-target-timeout` | kill one package's `go test` process tree after this long (default 15 minutes) and mark it `timeout`; `0` disables |
| `-quiet` | suppress the progress lines on stderr |
| `-out`, `-suite`, `-bench`, `-benchtime`, `-root`, `-json` | report path, suite name, benchmark regex, benchtime, root directory, JSON on stdout |

While a run is going, the progress lines (also written to `<out>.log`, even with `-quiet`) show the target index, status and an estimate of the time remaining, and `<out>.partial.json` is updated after each target so an interrupted run can be resumed. The report records a `machine` block (architecture, host architecture, CPU model, logical CPUs, RAM, Go version, `GOMAXPROCS`, `GOFLAGS`, build tags, git SHA and dirty flag), and comparison refuses mismatched machines.

`gwc bench suite compare` prints per-entry median deltas and per-layer geometric means. An entry is flagged only when its delta exceeds the larger of 2 percent and three times the combined relative MAD, which is why `-count` should be at least 5. `-max-regress-pct N` makes the command exit non-zero when a significant regression exceeds N percent, `-allow-machine-mismatch` compares reports from different architectures or CPUs, and `-json` emits the comparison as JSON.

Known limits: the `build` lane's `coldMs` is not a true cold build because the Go cache is not cleared, and the wasm lane needs the executor described in [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md). A baseline is only meaningful when the tree is frozen and nothing else is running, because the noise on a throttled machine is large.

## Runner Config Ownership

`gwc-runner.json` is the shared external configuration contract for launcher-owned paths and workspace behavior.

Use config for:

- stable workspace layout
- artifact roots
- browser workspace location
- livereload workspace location
- org-wide policy or security rules when the broader enterprise layer is in use

Use CLI flags for:

- one-off output destinations
- temporary experiments
- local debugging overrides
- short-lived operator intent

Practical rule:

- if everyone on the project should inherit it, put it in config
- if only one run needs it, keep it on the command line

## Starter Generator Contract

Use `gwc start` as a constrained scaffold, not as a second framework layer.
The maintained gallery in [docs/STARTERS.md](../STARTERS.md) lists the
supported presets and their intended app shapes.

The generated starter should stay:

- small enough to read end to end on day one
- aligned with `gwc dev`, `gwc test`, `gwc verify`, and `gwc release`
- explicit about browser mount, styles, and example validation commands
- ready to grow into route families, shared state, SSR, or PWA layers without hiding those decisions inside generation magic
- covered by generated `starter_test.go`, generated `.github/workflows/ci.yml`,
  and the repo starter-templates CI lane that scaffolds, tests, and builds every
  default preset

Practical rule:

- use the generated app as the first correct baseline
- keep app-specific architecture decisions in app code and repo docs after generation
- regenerate only when testing starter output itself, not as an everyday migration tool

## Design Notes And Boundaries

Keep these rules in mind when you use the launcher:

- `gwc` is the documented workflow surface; legacy wrappers are compatibility paths, not the preferred story
- not every command belongs in every app loop; pick the smallest command that matches the job
- `dev` and `serve` are different tools; `dev` owns rebuild orchestration, `serve` does not
- `build` produces one artifact; `release` produces a release directory and manifest-oriented output
- `test` is lane-oriented by design; do not hide the lane choice inside opaque scripts if the team needs explicit behavior
- the runner config should stay small and durable; do not turn it into a scratchpad for temporary experiments
- the launcher surface is broad, but this chapter is focused on the everyday app-authoring commands; use `go -C tools run ./gwc -h` for the full current command list

## Common Failure Modes

- using raw `go build` and custom scripts where a documented launcher command already exists
- putting temporary local overrides into `gwc-runner.json` and accidentally committing them
- using `serve` when you actually need the rebuild-and-watch behavior from `dev`
- assuming `build` and `release` are interchangeable
- running `verify` from the wrong root and then debugging missing test or build resolution
- forgetting that some tools, such as `wasm`, are subcommand-oriented and require `measure`, `compare`, or related subcommands
- treating `examples` as only a browser page and missing the managed server lifecycle operations

## Validation

Use the smallest checks that prove the launcher surface is healthy for the current workspace.

Command discovery:

```powershell
go -C tools run ./gwc -h
go -C tools run ./gwc dev -h
go -C tools run ./gwc release -h
```

Baseline launcher health:

```powershell
go -C tools run ./gwc doctor
go -C tools run ./gwc doctor -audit
```

Dry-run and machine-readable checks:

```powershell
go -C tools run ./gwc dev -app .\main.go -dry-run
go -C tools run ./gwc verify -app .\main.go -root . -json
go -C tools run ./gwc lint -root . -json
go -C tools run ./gwc lint -root . -fix    # apply golangci's verified autofixes in place
```

### Editor integration (VS Code)

The [`tools/vscode-gwc`](../../tools/vscode-gwc/) extension surfaces `gwc lint --json` as inline
diagnostics and offers **quick-fixes** for autofixable issues — the code-action delegates to
`gwc lint --fix` (golangci's own fixer), so the editor never computes an edit itself. Completion
for typed routes (`gwc routes gen`), message keys (`gwc i18n gen`), and theme tokens
(`gwc css gen`) needs no extension support: those generators emit ordinary typed Go symbols, so
gopls autocompletes them and a typo is a compile error. The mapping logic is host-independent and
unit-tested under plain Node.

Measurement help:

```powershell
go -C tools run ./gwc wasm measure -h
go -C tools run ./gwc bench -h
```

## Topic Pagination
Topic 2 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [01 Getting Started](01-getting-started.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [03 App Shapes](03-app-shapes.md)
