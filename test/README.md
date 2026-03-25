# Browser Test Suite

Location: `test/`

This directory contains the testing surface for GoWebComponents.

At the top level it is the single browser test workspace for the repository, including both the functional regression suites and the browser performance benchmark harness.

It also now hosts the preferred public Go testing helper import paths under:

- `test/render`
- `test/hooks`
- `test/router`
- `test/ssr`

The older `testkit/...` import paths remain supported as compatibility aliases.

## What Is Covered

### Main browser suite

`test/playwrightgo/testapp_suite_test.go`

Focus:

- component rendering and interaction flows (`TestComponents`)
- integration data and form flows (`TestIntegration`)
- state stress behaviors (`TestState`)
- browser benchmark harness checks (`TestBenchmark`)
- aggregate suite entrypoint (`TestMainSuite`)

### Playwright smoke bootstrap

`test/playwrightgo/smoke_test.go`

Focus:

- verifies Playwright-Go Chromium install and launch
- validates one deterministic browser page roundtrip

### Examples browser suite

`test/playwrightgo/examples/examples_suite_test.go`

Focus:

- examples catalog availability and navigation checks
- focused route checks for SSR, Atlas, startup, browser compat, virtualization, and chat wizard
- aggregate suite entrypoint (`TestExamplesAll`)

## Setup

```powershell
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium
```

## Run All Browser Tests

```powershell
go test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v
```

## Run The Entire Project Test Matrix

From the repo root, the canonical main harness is now:

```powershell
go run ./tools/gwc test -lane browser
```

That root command runs:

- native Go tests from the main module
- js/wasm Go tests discovered from `*_wasm_test.go`
- nested Go tests in `tools/livereload`
- this browser workspace under `test/` (Playwright-Go wrappers)
- the example browser suites under `examples/` (Playwright-Go wrappers)

## Run Focused Suites

```powershell
go test -tags playwrightgo ./test/playwrightgo -run TestComponents -v
go test -tags playwrightgo ./test/playwrightgo -run TestIntegration -v
go test -tags playwrightgo ./test/playwrightgo -run TestState -v
go test -tags playwrightgo ./test/playwrightgo -run TestBenchmark -v
```

## How The Test App Works

- `testapp/` contains the Go wasm app used by these tests
- the test scripts rebuild the wasm bundle under `bin/test/testapp/main.wasm` before the relevant suites
- the Playwright-Go workspace starts `go run ../tools/gwc serve ...` to host the static test app, serve the matching toolchain `wasm_exec.js`, and expose the JSON fixture routes used by fetch/integration flows
- `benchmark/` provides the standalone browser benchmark harness used by `go test -tags playwrightgo ./test/playwrightgo -run TestBenchmark -v`

## Notes

- Older docs referenced building unrelated example apps before running the test suite. That is stale. The test package builds its own `testapp/` target.
- Browser test residue is written under `bin/test-results/`; legacy `test-results/` paths remain ignored and should not be committed.
