# GWC Launcher

`gwc` is the canonical tooling surface for GoWebComponents projects and for this repository itself.

Run it from the repo root:

```powershell
go run ./tools/gwc <command> [flags]
```

If you are working in a separate app repository that vendors or depends on GoWebComponents, run the same command from that app workspace instead of this monorepo.

## Why `gwc`

`gwc` exists to keep the documented workflow on one stable command surface instead of splitting it across wrapper scripts, ad hoc shell snippets, or project-local toolchains.

Legacy `tools/*.ps1` and `tools/*.sh` wrappers remain as compatibility shims only and are deprecated for new workflows.

Use it for:

- environment and runtime checks
- scaffolding and bootstrap
- example serving
- app development loops
- wasm builds and release packaging
- validation lanes
- file inventory and import helpers
- benchmark sweeps and saved reports

## Quick Start

Repo-first commands:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc bootstrap
go run ./tools/gwc examples
go run ./tools/gwc dev -app .\examples\01-counter\main.go
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc verify -app .\examples\01-counter\main.go -root .\examples\01-counter
go run ./tools/gwc lint -root . -out .\bin\lint-report.txt
```

App-first commands:

```powershell
go run ./tools/gwc start
go run ./tools/gwc dev -app .\main.go
go run ./tools/gwc build -app .\main.go -profile development
go run ./tools/gwc release -app .\main.go -out-dir .\bin\release
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc verify -app .\main.go -root .
go run ./tools/gwc lint -root . -json
```

## Command Guide

### `doctor`

Use `doctor` first when a machine or workspace is not ready.

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc doctor -audit
go run ./tools/gwc doctor -json
```

What it covers:

- Go toolchain presence
- runtime assets such as `wasm_exec.js`
- browser-test prerequisites
- project-detection signals
- optional golden-path audit checks
- local port availability

### `env`

Use `env` to inspect launcher-relevant environment variables and their current values.

```powershell
go run ./tools/gwc env
go run ./tools/gwc env -json
go run ./tools/gwc env -set-only
```

By default, secret-shaped values (such as API keys and tokens) are redacted. Use `-show-secrets` only in trusted local contexts.

### `bootstrap` and `start`

Use these when starting a new project or when you want the examples catalog flow from one command.

```powershell
go run ./tools/gwc bootstrap
go run ./tools/gwc bootstrap -examples
go run ./tools/gwc start
```

`bootstrap` runs checks and then launches either the scaffold flow or the examples flow. `start` is the scaffold TUI itself.

### `examples`

Use this to browse the repo examples catalog.

```powershell
go run ./tools/gwc examples
```

Default URLs:

- `http://127.0.0.1:8090/examples`
- `http://127.0.0.1:8090/examples/list`
- `http://127.0.0.1:8090/healthz`

Managed example-server profiles are also launcher-owned through `gwc examples start|status|stop`.

```powershell
go run ./tools/gwc examples start
go run ./tools/gwc examples status
go run ./tools/gwc examples stop
```

Current managed profile contract:

- `profile`: stable launcher profile key (default `chat-wizard-local`)
- `command`: executable plus arguments used to start the profile
- `address`: host and port for `LISTEN_ADDR`
- `health endpoint`: path probed before `start` reports success
- `log location`: profile log file under runtime metadata
- `state file`: persisted pid and profile metadata under runtime metadata

Runtime metadata root:

- default: `bin/runtime/examples-servers`
- when `gwc-runner.json` sets `paths.artifactRoot`: `<artifactRoot>/runtime/examples-servers`

Managed lifecycle guarantees:

- `examples start` only reports success after a health probe passes; failed readiness tears down the launched process and removes stale state.
- `examples stop` terminates the managed process tree (including descendants) before clearing state.
- `examples status` removes stale PID state automatically when no live process exists for the recorded profile.

### `dev`

Use this as the normal inner loop for one wasm app.

```powershell
go run ./tools/gwc dev -app .\examples\01-counter\main.go
go run ./tools/gwc dev -app .\main.go -root . -html .\index.html -wasm .\bin\main.wasm
```

`dev` owns:

- rebuild-on-save
- static serving for the app root
- livereload client injection
- hot-reload handoff when the app opts into `hotreload`

Use `-dry-run` or `-json` when you need to inspect the resolved plan without starting the server.

### `serve`

Use `serve` when you already have built assets and need a narrow static server.

```powershell
go run ./tools/gwc serve -root .\test\testapp -wasm-file .\bin\test\testapp\main.wasm
go run ./tools/gwc serve -root .\static -fixture-json /api/user/123=.\fixtures\user-123.json
```

`serve` is intentionally smaller than `dev`: it serves files, `wasm_exec.js`, one optional wasm route, and fixture JSON routes.

### `build`

Use `build` for one explicit `js/wasm` artifact.

```powershell
go run ./tools/gwc build -app .\main.go -profile development
go run ./tools/gwc build -app .\main.go -profile ci -out .\bin\app\main.wasm
```

Profiles:

- `development`
- `ci`
- `benchmark`
- `release`

### `tailwind`

Use `tailwind` to rebuild the shared examples stylesheet and generated class manifest through the launcher-owned Tailwind path.

```powershell
go run ./tools/gwc tailwind
go run ./tools/gwc tailwind -json
```

`tailwind` downloads the standalone Tailwind CLI into `third_party/tailwindcss/bin` on first run and then reuses that cached binary.

### `release`

Use `release` when you want a packaged output directory and manifest instead of just one wasm file.

```powershell
go run ./tools/gwc release -app .\main.go -out-dir .\bin\release
go run ./tools/gwc release -app .\main.go -out-dir .\bin\release -compression gzip+brotli -validate-smoke
```

`release` can emit:

- the primary wasm artifact
- compression sidecars
- `wasm-release-manifest.json`
- optional size attribution
- optional manifest comparison
- optional browser startup measurement

### `test`

Use `test` for launcher-owned validation lanes.

```powershell
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc test -lane hydration
go run ./tools/gwc test -lane browser
go run ./tools/gwc test -lane release -app .\examples\01-counter\main.go -root .\examples\01-counter
```

Current lanes:

- `unit`
- `wasm`
- `hydration`
- `browser`
- `release`

### `verify`

Use `verify` as the higher-signal pre-commit or CI path for one app.

```powershell
go run ./tools/gwc verify -app .\main.go -root .
go run ./tools/gwc verify -app .\main.go -root . -audit -audit-min-severity error
```

`verify` runs app-local tests when present and then performs a CI-profile wasm build.

### `lint`

Use `lint` when you want a launcher-owned `golangci-lint` wrapper that emits a stable text or JSON report with grouped findings and file metadata.

```powershell
go run ./tools/gwc lint -root .
go run ./tools/gwc lint -root . -out .\bin\lint-report.txt
go run ./tools/gwc lint -root . -json
```

`review` is an alias for `lint`.

### `files`

Use `files` when you need a project inventory that is stable across shells.

```powershell
go run ./tools/gwc files -root . -ext go
go run ./tools/gwc files -root . -ext js -exclude-dir node_modules -exclude-dir examples -json
```

### `import`

Use `import` to turn static HTML or JSX into an inspectable GWC `main.go`.

```powershell
go run ./tools/gwc import -src .\design\landing.html -out .\bin\landing\main.go
go run ./tools/gwc import -src .\design\landing.jsx
```

### `bench`

Use `bench` for repo-owned benchmark sweeps and saved reports.

```powershell
go run ./tools/gwc bench -root .
go run ./tools/gwc bench -root . -count 3 -parallel 1
go run ./tools/gwc bench compare -baseline .\docs\benchmarks\reference.json -candidate .\docs\benchmarks\latest.json
```

`bench` writes structured reports to `docs/benchmarks/latest.json` by default and can score against `docs/benchmarks/reference.json`.

### `wasm`

Use `wasm` for launcher-owned experiment helpers.

```powershell
go run ./tools/gwc wasm measure -package .\examples\21-ui-render -out-dir .\bin\wasm-build-experiment
go run ./tools/gwc wasm compare -baseline .\bin\wasm-build-experiment\baseline.json -candidate .\bin\wasm-build-experiment\candidate.json
go run ./tools/gwc wasm compare-compression -package .\examples\21-ui-render
go run ./tools/gwc wasm compare-cache -package .\examples\21-ui-render
go run ./tools/gwc wasm compare-toolchain -package .\examples\21-ui-render -baseline-go go1.25.4 -candidate-go go1.26.0
```

Current `wasm` subcommands:

- `measure`: build one package and emit a phase-attributed manifest
- `compare`: compare two manifests with timing/size/other thresholds
- `compare-compression`: measure plain, stripped, compressed, and optional optimized variants
- `compare-cache`: measure cold, warm, and small-edit rebuild behavior across cache topologies
- `compare-toolchain`: compare baseline and candidate Go executables and propagate regression status

### `dashboard` and `seed`

These remain specialized commands:

- `dashboard` monitors live-reload clients and provider configuration
- `seed` runs app-owned local seed logic when a project exposes it

## Common Workflows

### Start A New App

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc start
go run ./tools/gwc dev -app .\main.go
```

### Work On One Existing App

```powershell
go run ./tools/gwc dev -app .\main.go
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc verify -app .\main.go -root .
```

### Browse And Validate Repo Examples

```powershell
go run ./tools/gwc examples
go run ./tools/gwc test -lane browser
```

### Ship A Release Build

```powershell
go run ./tools/gwc release -app .\main.go -out-dir .\bin\release -validate-smoke
```

### Track Performance

```powershell
go run ./tools/gwc bench -root .
go run ./tools/gwc bench compare -baseline .\docs\benchmarks\reference.json -candidate .\docs\benchmarks\latest.json
```

## Runner Config

`gwc` reads `gwc-runner.json` or `GWC_RUNNER_CONFIG` for shared path overrides.

Common path fields:

- `generatedProjectRoot`
- `artifactRoot`
- `workspaceBuildRoot`
- `wasmExecJS`
- `goWasmExec`
- `browserWorkspace`
- `livereloadWorkspace`
- `livereloadClientScript`

See [RUNNER_CONFIG.md](RUNNER_CONFIG.md) for the full schema contract.

## Repo Layout

- [tools/README.md](../tools/README.md): top-level tooling entrypoint note
- [tools/gwc/docs/README.md](../tools/gwc/docs/README.md): launcher code layout
- [TESTING.md](TESTING.md): testing surface and launcher-owned lanes
- [PERFORMANCE.md](PERFORMANCE.md): benchmark usage and report interpretation
- [ONBOARDING.md](ONBOARDING.md): adoption and bootstrap guidance
