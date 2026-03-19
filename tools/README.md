# Tools

Location: `tools/`

This directory contains repo-level build, serve, and wasm test helpers.

## Current Tools

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
npm run dev:examples
```

### `dev-server/`

Express-based server used by `serve.ps1` and `npm run dev:examples`.

Key behavior:

- `/examples` is generated from the actual `examples/` directory structure
- example HTML and shared static assets are served from the repo
- useful for local manual testing and Playwright-oriented smoke checks

### `build.sh` / `build.ps1`

Convenience wrappers for building a Go wasm target.

These are still useful for standalone projects, but the repo�s browser tests build their own dedicated test app under `test/testapp/`.

### `build-wasm-release.ps1`

PowerShell-first helper for production-style wasm release builds.

Example:

```powershell
.\tools\build-wasm-release.ps1 `
  -Package ./examples/21-ui-render `
  -OutDir .\dist\ui-render `
  -BudgetsPath .\tools\wasm-size-budgets.sample.json
```

Behavior:

- builds a `js/wasm` artifact with `-trimpath`, `-ldflags="-s -w"`, and `-buildvcs=false`
- writes the raw `.wasm` file plus a `.gz` sidecar by default, and a `.br` sidecar when the host PowerShell runtime supports Brotli compression
- emits `wasm-release-manifest.json` with relative paths, sizes, and sha256 hashes
- optionally fails when raw, gzip, or available brotli sizes exceed configured budgets

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

Legacy/experimental live reload tooling.

It has not been removed, but it should not be treated as the primary development path. The main documented dev server is the Express server in `tools/dev-server/`.

## Recommended Workflow

### Serve examples

```powershell
npm run dev:examples
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
