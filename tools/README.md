# Tools

Location: `tools/`

The supported user-facing tooling entrypoint in this repository is `gwc`.

Run it from the repo root:

```powershell
go -C tools run ./gwc <command> [flags]
```

Start here:

```powershell
go -C tools run ./gwc doctor
go -C tools run ./gwc bootstrap
go -C tools run ./gwc examples
go -C tools run ./gwc examples build-public-site
go -C tools run ./gwc dev -app .\examples\public\counter\main.go
go -C tools run ./gwc build -app .\examples\public\counter\main.go -profile development
go -C tools run ./gwc test -lane unit -lane wasm
go -C tools run ./gwc verify -app .\examples\public\counter\main.go -root .\examples\public\counter
go -C tools run ./gwc lint -root . -out .\bin\lint-report.txt
```

## `gwc`

`gwc` is the canonical launcher for:

- prerequisite checks
- scaffolding and bootstrap
- example serving
- standalone app dev loops
- wasm builds and release packaging
- launcher-owned test lanes
- file inventory and import helpers
- benchmark sweeps and report generation

Current command surface:

- `doctor`: check toolchains, runtime assets, browser-test prerequisites, and optional audit signals
- `bootstrap`: run checks and then launch either the starter flow or the examples catalog flow
- `start`: open the scaffold TUI for a new app
- `init`: write lifecycle metadata and starter defaults without the scaffold TUI
- `upgrade`: backfill lifecycle metadata, feature matrix defaults, and runtime assets for an existing app
- `migrate`: run lifecycle upgrade, write a compatibility report, and optionally apply safe parser-backed router method rewrites with `-apply`
- `examples`: serve the examples catalog or manage profile/path-backed example servers with `start`, `status`, `stop`, and `restart`
  `build-public-site` builds the public docs shell plus the first staged embedded example wasm binary for local example browsing
- `dev`: run the rebuild-and-serve inner loop for one app
- `desktop`: opt-in Windows `init`, `doctor`, `dev`, `build`, and unsigned `package` workflow; see [Windows desktop reference](../docs/REFERENCE_MANUAL/16-windows-desktop.md)
- `serve`: serve a static root, `wasm_exec.js`, one wasm artifact, and optional JSON fixtures
- `build`: build one `js/wasm` target with an explicit profile (`development`, `debug`, `ci`, `benchmark`, `release`, or `tinygo`)
- `tailwind`: rebuild shared Tailwind CSS assets via the standalone CLI cached under `third_party/tailwindcss/bin`
- `release`: package one release build with manifest and compression sidecars
- `test`: run launcher-owned `unit`, `wasm`, `hydration`, `browser`, and `release` lanes
- `verify`: run app-local tests when present and then perform a CI-profile wasm build
- `lint`: run `golangci-lint`, capture structured findings, and render a text or JSON review report (`review` alias supported)
  if the default executable is missing, `gwc lint` installs it with `go install` before running; the built-in `gwc-hooks` pass also reports hook calls inside conditionals, loops, nested functions, and goroutine launches unless `-skip-hook-rules` is set
- `files`: list project files with repeatable extension and directory filters
- `import`: convert static `.html`, `.htm`, `.jsx`, or `.tsx` into an inspectable GWC `main.go`
- `bench`: run discovered repo benchmarks and write structured JSON reports
- `wasm`: run wasm-focused experiment helpers (`measure`, `compare`, `compare-compression`, `compare-cache`, `compare-toolchain`)
- `dashboard`: monitor live-reload clients and AI provider configuration
- `env`: print launcher-relevant environment variables and current values
- `seed`: run app-owned seed logic when a project exposes it

Common commands:

```powershell
go -C tools run ./gwc doctor
go -C tools run ./gwc start
go -C tools run ./gwc init -root . -skip-runtime-assets
go -C tools run ./gwc upgrade -root . -skip-runtime-assets
go -C tools run ./gwc migrate -root . -skip-runtime-assets -apply -json
go -C tools run ./gwc bootstrap -examples
go -C tools run ./gwc examples
go -C tools run ./gwc examples start
go -C tools run ./gwc examples restart
go -C tools run ./gwc examples status
go -C tools run ./gwc examples stop
go -C tools run ./gwc .\examples\server\ai-chat-wizard\cmd\server start -json
go -C tools run ./gwc .\examples\server\ai-chat-wizard\cmd\server restart -json
go -C tools run ./gwc examples .\examples\server\ai-chat-wizard\cmd\server status -json
go -C tools run ./gwc dev -app .\path\to\main.go
go -C tools run ./gwc serve -root .\static -wasm-file .\bin\app\main.wasm
go -C tools run ./gwc build -app .\path\to\main.go -profile release
go -C tools run ./gwc build -app .\path\to\main.go -profile debug -out .\bin\debug\app.wasm
go -C tools run ./gwc build -app .\path\to\main.go -profile tinygo
go -C tools run ./gwc tailwind
go -C tools run ./gwc release -app .\path\to\main.go -out-dir .\bin\release
go -C tools run ./gwc test -lane unit -lane wasm -lane browser
go -C tools run ./gwc verify -app .\path\to\main.go -root . -audit
go -C tools run ./gwc lint -root . -json
go -C tools run ./gwc env
go -C tools run ./gwc env -json
go -C tools run ./gwc files -root . -ext go -exclude-dir .git
go -C tools run ./gwc import -src .\design\landing.html -out .\bin\landing\main.go
go -C tools run ./gwc bench -root .
go -C tools run ./gwc bench compare -baseline .\docs\benchmarks\reference.json -candidate .\docs\benchmarks\latest.json
go -C tools run ./gwc wasm measure -package .\examples\public\ui-render -out-dir .\bin\wasm-build-experiment
go -C tools run ./gwc wasm compare -baseline .\bin\wasm-build-experiment\baseline.json -candidate .\bin\wasm-build-experiment\candidate.json
go -C tools run ./gwc wasm compare-compression -package .\examples\public\ui-render
go -C tools run ./gwc wasm compare-cache -package .\examples\public\ui-render
go -C tools run ./gwc wasm compare-toolchain -package .\examples\public\ui-render -baseline-go go1.25.4 -candidate-go go1.26.0
```

Use `go -C tools run ./gwc <command> -h` for the exact flag surface of a command.

## Core Docs

- [docs/REFERENCE_MANUAL/02-gwc-workflows.md](../docs/REFERENCE_MANUAL/02-gwc-workflows.md): canonical launcher guide, starter rules, and common workflows
- [docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md](../docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md): testing surface and launcher-owned validation lanes
- [docs/REFERENCE_MANUAL/13-assets-deployment-and-pwa.md](../docs/REFERENCE_MANUAL/13-assets-deployment-and-pwa.md): release, wasm, and deployment guidance
- [docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md](../docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md): stability and policy boundaries behind launcher behavior
- [tools/gwc/docs/README.md](./gwc/docs/README.md): code layout for the launcher package itself

## Implementation Notes

- `tools/gwc/` contains the launcher source.
- `tools/livereload/` contains the live-reload implementation used by `gwc dev`.
- `tools/runnerconfig/` contains the config loader and resolver used by launcher-owned workflows.
- Other files under `tools/` are implementation details or compatibility artifacts, not the documented primary workflow.
- Legacy `tools/*.ps1` and `tools/*.sh` helper wrappers are deprecated compatibility shims. Use `go -C tools run ./gwc ...` directly for new workflows.
