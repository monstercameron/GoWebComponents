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
go run ./tools/gwc <command> [flags]
```

Use `go run ./tools/gwc <command> -h` for exact flag details. This chapter explains when to use each command and how the commands fit together.

## Stability Note

The `gwc` workflow is the documented launcher contract for this repo.

Practical rule:

- prefer `gwc` when a documented command already owns the workflow
- prefer direct low-level `go build` or `go test` only when you intentionally need compiler-level control outside the launcher contract

The exact flag surface can grow over time, but the high-level workflow categories documented here are the supported path for app authors.

## Minimal Workflow

If you only need the shortest path from repo checkout to a running example, use this sequence:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc examples
go run ./tools/gwc dev -app .\examples\01-counter\main.go
go run ./tools/gwc verify -app .\examples\01-counter\main.go -root .\examples\01-counter
```

What each command proves:

- `doctor` checks the local machine and runtime prerequisites
- `examples` gives you the fastest entrypoint into the example catalog
- `dev` runs the active inner loop for one app
- `verify` proves the app can pass launcher-owned validation and a CI-style wasm build

## Production-Shaped Workflow

For a real app, the workflow usually becomes: scaffold, configure, run, validate, release.

`main.go` with hot reload opt-in:

```go
package main

import (
	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// renderRootApp returns the root application node for the standalone dev loop.
func renderRootApp() ui.Node {
	return ui.Text("hello from gwc dev")
}

// main enables state-preserving reload for gwc dev and mounts the app.
func main() {
	hotreload.Enable()
	ui.Render(ui.CreateElement(renderRootApp, nil), "#app")
	utils.WaitForever()
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
go run ./tools/gwc doctor
go run ./tools/gwc start
go run ./tools/gwc dev -app .\main.go
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc verify -app .\main.go -root .
go run ./tools/gwc release -app .\main.go -out-dir .\bin\release -compression gzip+brotli
```

Why this is the normal production-shaped path:

- the app stays on the documented launcher contract
- the dev loop, validation, and release artifacts all resolve through one tool surface
- runner config captures stable workspace rules while short-lived overrides remain on flags

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
go run ./tools/gwc doctor -audit
go run ./tools/gwc test -lane unit -lane wasm -lane browser
go run ./tools/gwc verify -app .\main.go -root . -audit
go run ./tools/gwc release -app .\main.go -out-dir .\bin\release -compression gzip+brotli -validate-smoke
go run ./tools/gwc bench -root .
go run ./tools/gwc wasm measure -package .\examples\21-ui-render
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
| `doctor` | you need prerequisite checks, runtime-asset checks, or port/audit signals | `go run ./tools/gwc doctor -audit` |
| `bootstrap` | you want prerequisite checks and then either scaffold or examples bootstrap flow | `go run ./tools/gwc bootstrap -examples` |
| `start` | you want the scaffold TUI for a new app | `go run ./tools/gwc start -mode standalone` |
| `examples` | you want the example catalog or managed example-server lifecycle commands | `go run ./tools/gwc examples` |

Practical distinction:

- use `bootstrap` when you want checks plus a guided next step
- use `start` when you already know you want scaffold generation
- use `examples` when the fastest path is studying a real maintained example

### Inner Loop

| Command | Use it when | Representative call |
| --- | --- | --- |
| `dev` | you want rebuild, serve, livereload, and optional hot reload for one app | `go run ./tools/gwc dev -app .\main.go` |
| `serve` | you already have built assets and only need a narrow static server | `go run ./tools/gwc serve -root .\static -wasm-file .\bin\app.wasm` |
| `dashboard` | you want a launcher-owned view of live dev status and provider config | `go run ./tools/gwc dashboard -root .` |

Use `dev -dry-run` or `dev -json` when you want to inspect the resolved plan before you start the server.

### Build, Release, And Validation

| Command | Use it when | Representative call |
| --- | --- | --- |
| `build` | you want one explicit `js/wasm` artifact | `go run ./tools/gwc build -app .\main.go -profile development` |
| `release` | you want a packaged release directory plus manifest and compression sidecars | `go run ./tools/gwc release -app .\main.go -out-dir .\bin\release` |
| `test` | you want explicit launcher-owned lanes such as `unit`, `wasm`, `hydration`, `browser`, or `release` | `go run ./tools/gwc test -lane unit -lane wasm` |
| `verify` | you want the higher-signal pre-commit or CI path for one app | `go run ./tools/gwc verify -app .\main.go -root . -audit` |
| `lint` | you want launcher-owned lint reporting instead of a raw tool invocation | `go run ./tools/gwc lint -root . -json` |

Use `release -validate-smoke` when you want the post-build release smoke check, not only the artifact packaging.

### Inspection, Import, And Project Utilities

| Command | Use it when | Representative call |
| --- | --- | --- |
| `files` | you need a repeatable file inventory | `go run ./tools/gwc files -root . -ext go -exclude-dir .git` |
| `import` | you want to convert static HTML or JSX into an inspectable GWC `main.go` | `go run ./tools/gwc import -src .\design\landing.html -out .\bin\landing\main.go` |
| `seed` | the project exposes a seed package for local identities or fixture data | `go run ./tools/gwc seed -root .` |

### Measurement

| Command | Use it when | Representative call |
| --- | --- | --- |
| `bench` | you want repeatable benchmark discovery, JSON output, or normalized scoring against a saved reference report | `go run ./tools/gwc bench -root . -reference .\docs\benchmarks\reference.json` |
| `wasm measure` | you want a wasm artifact measurement pass with manifest output | `go run ./tools/gwc wasm measure -package .\examples\21-ui-render` |
| `wasm compare` and related subcommands | you want wasm-specific diffing, compression, cache, or toolchain comparisons | `go run ./tools/gwc wasm compare-toolchain -package .\examples\21-ui-render -baseline-go go1.25.4 -candidate-go go1.26.0` |

Use the subcommand help directly for the wasm tools:

```powershell
go run ./tools/gwc wasm measure -h
go run ./tools/gwc bench -h
```

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

The generated starter should stay:

- small enough to read end to end on day one
- aligned with `gwc dev`, `gwc test`, `gwc verify`, and `gwc release`
- explicit about browser mount, styles, and example validation commands
- ready to grow into route families, shared state, SSR, or PWA layers without hiding those decisions inside generation magic

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
- the launcher surface is broad, but this chapter is focused on the everyday app-authoring commands; use `go run ./tools/gwc -h` for the full current command list

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
go run ./tools/gwc -h
go run ./tools/gwc dev -h
go run ./tools/gwc release -h
```

Baseline launcher health:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc doctor -audit
```

Dry-run and machine-readable checks:

```powershell
go run ./tools/gwc dev -app .\main.go -dry-run
go run ./tools/gwc verify -app .\main.go -root . -json
go run ./tools/gwc lint -root . -json
```

Measurement help:

```powershell
go run ./tools/gwc wasm measure -h
go run ./tools/gwc bench -h
```

## Topic Pagination
Topic 2 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [01 Getting Started](01-getting-started.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [03 App Shapes](03-app-shapes.md)
