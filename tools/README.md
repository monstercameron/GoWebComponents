# Tools

Location: `tools/`

The supported user-facing tooling entrypoint in this repository is `gwc`.

Run it from the repo root:

```powershell
go run ./tools/gwc <command> [flags]
```

Start here:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc bootstrap
go run ./tools/gwc examples
go run ./tools/gwc dev -app .\examples\01-counter\main.go
go run ./tools/gwc build -app .\examples\01-counter\main.go -profile development
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc verify -app .\examples\01-counter\main.go -root .\examples\01-counter
go run ./tools/gwc lint -root . -out .\bin\lint-report.txt
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
- `examples`: serve the examples catalog or manage profile/path-backed example servers with `start`, `status`, `stop`, and `restart`
- `dev`: run the rebuild-and-serve inner loop for one app
- `serve`: serve a static root, `wasm_exec.js`, one wasm artifact, and optional JSON fixtures
- `build`: build one `js/wasm` target with an explicit profile
- `tailwind`: rebuild shared Tailwind CSS assets via the standalone CLI cached under `third_party/tailwindcss/bin`
- `release`: package one release build with manifest and compression sidecars
- `test`: run launcher-owned `unit`, `wasm`, `hydration`, `browser`, and `release` lanes
- `verify`: run app-local tests when present and then perform a CI-profile wasm build
- `lint`: run `golangci-lint`, capture structured findings, and render a text or JSON review report (`review` alias supported)
  if the default executable is missing, `gwc lint` installs it with `go install` before running
- `files`: list project files with repeatable extension and directory filters
- `import`: convert static `.html`, `.htm`, `.jsx`, or `.tsx` into an inspectable GWC `main.go`
- `bench`: run discovered repo benchmarks and write structured JSON reports
- `wasm`: run wasm-focused experiment helpers (`measure`, `compare`, `compare-compression`, `compare-cache`, `compare-toolchain`)
- `dashboard`: monitor live-reload clients and AI provider configuration
- `env`: print launcher-relevant environment variables and current values
- `seed`: run app-owned seed logic when a project exposes it

Common commands:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc start
go run ./tools/gwc bootstrap -examples
go run ./tools/gwc examples
go run ./tools/gwc examples start
go run ./tools/gwc examples restart
go run ./tools/gwc examples status
go run ./tools/gwc examples stop
go run ./tools/gwc .\examples\100-ai-chat-wizard\cmd\server start -json
go run ./tools/gwc .\examples\100-ai-chat-wizard\cmd\server restart -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server status -json
go run ./tools/gwc dev -app .\path\to\main.go
go run ./tools/gwc serve -root .\static -wasm-file .\bin\app\main.wasm
go run ./tools/gwc build -app .\path\to\main.go -profile release
go run ./tools/gwc tailwind
go run ./tools/gwc release -app .\path\to\main.go -out-dir .\bin\release
go run ./tools/gwc test -lane unit -lane wasm -lane browser
go run ./tools/gwc verify -app .\path\to\main.go -root . -audit
go run ./tools/gwc lint -root . -json
go run ./tools/gwc env
go run ./tools/gwc env -json
go run ./tools/gwc files -root . -ext go -exclude-dir .git
go run ./tools/gwc import -src .\design\landing.html -out .\bin\landing\main.go
go run ./tools/gwc bench -root .
go run ./tools/gwc bench compare -baseline .\docs\benchmarks\reference.json -candidate .\docs\benchmarks\latest.json
go run ./tools/gwc wasm measure -package .\examples\21-ui-render -out-dir .\bin\wasm-build-experiment
go run ./tools/gwc wasm compare -baseline .\bin\wasm-build-experiment\baseline.json -candidate .\bin\wasm-build-experiment\candidate.json
go run ./tools/gwc wasm compare-compression -package .\examples\21-ui-render
go run ./tools/gwc wasm compare-cache -package .\examples\21-ui-render
go run ./tools/gwc wasm compare-toolchain -package .\examples\21-ui-render -baseline-go go1.25.4 -candidate-go go1.26.0
```

Use `go run ./tools/gwc <command> -h` for the exact flag surface of a command.

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
- Legacy `tools/*.ps1` and `tools/*.sh` helper wrappers are deprecated compatibility shims. Use `go run ./tools/gwc ...` directly for new workflows.
