# Tools

Location: `tools/`

This directory contains repo-level build, serve, and wasm test helpers.

## Current Tools

### `gwc`

Canonical Go launcher for in-repo tooling.

Examples:

```powershell
go run ./tools/gwc build -app .\examples\01-counter\main.go -profile ci
go run ./tools/gwc release -app .\examples\01-counter\main.go -out-dir .\bin\gwc-release
go run ./tools/gwc examples
go run ./tools/gwc dev -app .\test\testapp\main.go
go run ./tools/gwc doctor
go run ./tools/gwc bootstrap
go run ./tools/gwc bootstrap -examples
go run ./tools/gwc import -src .\design\landing.jsx -out .\bin\landing\main.go
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc verify -app .\examples\01-counter\main.go -root .\examples\01-counter -audit -audit-min-severity error
```

Current status:

- `build` now produces a js/wasm artifact from the resolved app package with explicit `development`, `ci`, `benchmark`, or `release` profiles and optional JSON summaries
- `release` now packages a release-profile wasm artifact into an output directory, emits `wasm-release-manifest.json`, supports explicit `-compression none|gzip|brotli|gzip+brotli`, defaults to gzip plus Brotli sidecars, enforces optional JSON budgets, and supports machine-readable JSON summaries
- `examples` is now a Go-native catalog server replacement for the older Node-only entrypoint
- `dev` is a compatibility wrapper over the existing Go livereload server while broader project detection is still being built
- `doctor` now checks toolchains, `wasm_exec.js`, browser-test prerequisites, scaffold metadata, project-detection signals, and port availability
- `bootstrap` now runs prerequisite checks through `doctor` and then launches either the starter scaffold flow (`gwc start`) or examples bootstrap mode in one command
- `import` now converts a static `.html`, `.htm`, `.jsx`, or `.tsx` file into a single inspectable `main.go` that uses the GWC `html` library builders
- `test` now exposes explicit launcher-owned `unit`, `wasm`, `hydration`, `browser`, and `release` lanes, defaults to `unit` plus `wasm`, and supports JSON summaries for automation
- `verify` now runs app-local `go test ./...` when `_test.go` files exist under the resolved project root, then performs a `ci`-profile js/wasm build through the same launcher path, and can carry the golden-path audit with configurable `-audit-min-severity` gating for CI and editor tasks
- `start` now opens a Bubble Tea wizard, runs start-time prerequisite checks (Go, runtime assets, and browser tooling when needed), generates a runnable scaffold in a user-owned workspace location by default, supports optional post-generation setup skips through `-skip-tidy` and `-skip-runtime-assets`, and asks whether to launch it in the dev server immediately
- `gwc-runner.json` or `%GWC_RUNNER_CONFIG%` can now provide enterprise-oriented path overrides such as `generatedProjectRoot`, `artifactRoot`, `wasmExecJS`, `goWasmExec`, `browserWorkspace`, `livereloadWorkspace`, and `livereloadClientScript`
- launcher-owned temp artifacts now resolve under `bin/tmp/` beneath the relevant project root instead of the OS temp directory

Two practical usage modes:

- repo contributor: run `go run ./tools/gwc ...` from this monorepo when editing framework packages, examples, docs, or launcher code
- framework consumer: run the same `gwc` commands from the app workspace, but target the app's `main.go` and app root instead of repo-only example paths

Runner config reference:

- Copy the baseline contract from `docs/examples/gwc-runner.example.json` into `gwc-runner.json` at the repo or project root, or point `%GWC_RUNNER_CONFIG%` at an equivalent file. See `docs/RUNNER_CONFIG.md` for the canonical cross-tool schema contract.
- Relative paths are resolved from the directory that contains the config file.
- `generatedProjectRoot`: default output location for `gwc start` generated apps when a command does not pass an explicit destination.
- `artifactRoot`: root directory for launcher-owned artifacts such as wasm builds, release outputs, and temporary import work directories.
- `wasmExecJS`: override path for `wasm_exec.js` discovery used by launcher flows that need the JS runtime helper.
- `goWasmExec`: override path for the js/wasm test executor used by launcher and Node-driven test flows.
- `browserWorkspace`: workspace directory that contains the Playwright `package.json` used by browser lanes.
- `livereloadWorkspace`: workspace directory used when `gwc dev` shells into the nested livereload server.
- `livereloadClientScript`: explicit client script path served by the livereload server when auto-discovery should not be used.
- Ownership guidance: keep enterprise security and org-wide path standards in shared `GWC_RUNNER_CONFIG` or home-level config, use checked-in `gwc-runner.json` for repository-wide layout decisions, and prefer explicit flags for temporary local overrides.

### `serve.ps1`

Starts the Node/Express example server.

```powershell
.\tools\serve.ps1
```

Behavior:

- installs `tools/dev-server` dependencies if needed
- serves examples on `http://127.0.0.1:8090`
- exposes `http://127.0.0.1:8090/examples`
- exposes `http://127.0.0.1:8090/healthz`
- serves `.wasm` with the correct MIME type
- disables caching for more stable dev iteration

Equivalent repo-root command:

```powershell
npm --prefix tools/devtools run dev:examples
```

### `dev-server/`

Express-based server used by `serve.ps1` and `npm --prefix tools/devtools run dev:examples`.

Key behavior:

- `/examples` is generated from the actual `examples/` directory structure
- example HTML and shared static assets are served from the repo
- useful for local manual testing and Playwright-oriented smoke checks

### `dev.ps1` / `dev.sh`

Starts the path-based hot-reload server for a standalone Go WASM app.

Example:

```powershell
.\tools\dev.ps1 -App .\test\testapp\main.go
```

Behavior:

- builds the Go WASM app from the directory that contains the provided `main.go`
- rebuilds on file changes
- injects the live-reload client and preserves state through the browser rehydration bridge
- expects the app to call `hotreload.Enable()` or `hotreload.Configure(...)` when state-preserving reload should be active
- accepts optional `-Root`, `-Html`, `-Wasm`, `-ListenHost`, `-Port`, and `-NoHotReload` overrides

Use `-Root` when the app serves HTML from a different directory than the Go package. Use `-Html` and `-Wasm` to point at non-default HTML or wasm filenames.

### `build.sh` / `build.ps1`

Convenience wrappers for building a Go wasm target.

These are still useful for standalone projects, but the repo�s browser tests build their own dedicated test app under `test/testapp/`.

### `build-wasm-release.ps1`

PowerShell-first helper for production-style wasm release builds.

Example:

```powershell
.\tools\build-wasm-release.ps1 `
  -Package ./examples/21-ui-render `
  -OutDir .\bin\ui-render `
  -BudgetsPath .\tools\wasm-size-budgets.sample.json
```

Behavior:

- builds a `js/wasm` artifact with `-trimpath`, `-ldflags="-s -w"`, and `-buildvcs=false`
- writes the raw `.wasm` file plus a `.gz` sidecar by default, and a `.br` sidecar when the host PowerShell runtime supports Brotli compression
- emits `wasm-release-manifest.json` with relative paths, sizes, and sha256 hashes
- optionally fails when raw, gzip, or available brotli sizes exceed configured budgets

### `measure-wasm-build.ps1`

PowerShell helper for phase-attributed wasm build experiments.

Example:

```powershell
.\tools\measure-wasm-build.ps1 `
  -Package ./examples/21-ui-render `
  -OutDir .\bin\ui-render-build-exp `
  -ReleaseProfile
```

Behavior:

- builds a `js/wasm` artifact for the requested package
- accepts an optional `-GoExecutable` override so the same measurement flow can be used across different Go toolchains
- records `go_build_ms`, compression timings, and total wall-clock timing in `wasm-build-experiment.json`
- writes artifact size and sha256 metadata beside the phase timings
- emits a Brotli sidecar when either the PowerShell runtime or the repo's Node-based helper can provide Brotli compression
- accepts an optional `-ServeReloadMs` value when a browser or dev-server probe measures reload latency separately

### `compare-wasm-compression.ps1`

PowerShell helper for comparing supported wasm compression and post-processing variants for one package.

Example:

```powershell
.\tools\compare-wasm-compression.ps1 `
  -Package ./examples/21-ui-render `
  -OutDir .\bin\ui-render-compression-exp
```

Behavior:

- runs plain raw, stripped raw, and stripped plus compression builds through `measure-wasm-build.ps1`
- adds `wasm-opt`-processed variants by resolving `wasm-opt` from `PATH` or from the `binaryen` npm package through `npx`
- writes one summary JSON file that includes each variant manifest
- records whether brotli delivery or `wasm-opt`-based optimized variants are still unavailable after the repo-local fallbacks are attempted

### `compare-wasm-build-cache.ps1`

PowerShell helper for comparing wasm build timings across different build-cache and module-cache strategies.

Example:

```powershell
.\tools\compare-wasm-build-cache.ps1 `
  -Package ./examples/21-ui-render `
  -OutDir .\bin\ui-render-cache-exp `
  -ReleaseProfile
```

Behavior:

- runs cold and warm builds with a dedicated shared `GOCACHE`
- adds a small-edit rebuild after the shared cache is warm by applying and restoring a temporary comment change in one package source file
- compares that against a fresh isolated build-cache run that still reuses the normal module cache
- runs a CI-style isolated `GOCACHE` plus isolated `GOMODCACHE` pass, then repeats it warm, then measures one small-edit rebuild against those hydrated caches
- writes one summary JSON file that includes each variant's `measure-wasm-build.ps1` manifest plus any module-download timing

### `compare-wasm-experiment.ps1`

PowerShell helper for CI-friendly comparison of saved wasm build experiment manifests.

Example:

```powershell
.\tools\compare-wasm-experiment.ps1 `
  -Baseline .\bin\ui-render-cache-exp\wasm-build-cache-comparison.json `
  -Candidate .\bin\ui-render-cache-exp\wasm-build-cache-comparison.json `
  -OutFile .\bin\ui-render-cache-exp\comparison.json
```

Behavior:

- flattens numeric timing and size metrics from saved JSON experiment manifests
- applies separate regression thresholds for timing metrics and artifact-size metrics
- writes a comparison summary JSON file when requested
- exits non-zero when the candidate exceeds the configured regression thresholds, making it suitable for CI gating

### `compare-wasm-go-toolchain.ps1`

PowerShell helper for comparing one wasm target across a baseline Go toolchain and a candidate Go toolchain.

Example:

```powershell
.\tools\compare-wasm-go-toolchain.ps1 `
  -Package ./examples/21-ui-render `
  -BaselineGo go `
  -CandidateGo go `
  -OutDir .\bin\ui-render-toolchain-exp `
  -ReleaseProfile
```

Behavior:

- runs `measure-wasm-build.ps1` once per toolchain with the same package and profile
- stores one manifest per toolchain plus a saved comparison result
- exits non-zero when the candidate toolchain exceeds the configured regression thresholds
- writes `wasm-toolchain-comparison.json` so CI and docs can attribute the compared Go versions

### `go_js_wasm_exec.bat`

Windows helper for running `go test` in `js/wasm` mode.

Example:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/runtime
```

### `bench-runtime.ps1` / `bench-runtime.sh`

Runs repeated benchmark samples and writes the raw output to a timestamped file for later comparison.

Windows example:

```powershell
.\tools\bench-runtime.ps1 -Package ./internal/runtime -Count 5
```

Wasm adapter example on Windows:

```powershell
.\tools\bench-runtime.ps1 -Package ./internal/platform/jsdom -Count 5 -Exec .\tools\go_js_wasm_exec.bat
```

POSIX example:

```bash
./tools/bench-runtime.sh --package ./internal/runtime --count 5
```

### `bench-compare.ps1` / `bench-compare.sh`

Compares two saved benchmark files with `benchstat` when available.

Windows example:

```powershell
.\tools\bench-compare.ps1 -Baseline .\tools\bench-before.txt -Candidate .\tools\bench-after.txt
```

POSIX example:

```bash
./tools/bench-compare.sh ./tools/bench-before.txt ./tools/bench-after.txt
```

### `livereload/`

Implementation package behind the first-class `tools/dev.ps1` and `tools/dev.sh` hot-reload workflow.

Most users should run the wrapper scripts instead of calling the Go server directly.

## Recommended Workflow

### Serve examples

```powershell
npm --prefix tools/devtools run dev:examples
```

### Run runtime tests

```bash
go test ./internal/runtime
```

### Run repeated runtime benchmarks

```powershell
.\tools\bench-runtime.ps1 -Package ./internal/runtime -Count 5
```

```bash
./tools/bench-runtime.sh --package ./internal/runtime --count 5
```

### Run wasm runtime tests on Windows

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/runtime
```

### Run browser tests

```powershell
cd test
npm install
npm run install:browsers
npm test
```

## Notes

- Older docs referred to `scripts/` paths. Those references are stale in this repo.
- The repo now documents the Express server first because it is the stable path used for manual example serving.
