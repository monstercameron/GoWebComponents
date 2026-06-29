# Contributing to GoWebComponents

This guide is for people working **on the framework** (its packages, runtime,
and tooling). If you are building an app *with* GoWebComponents, start with the
[Getting Started chapter](docs/REFERENCE_MANUAL/01-getting-started.md) and the
[gwc workflow guide](docs/REFERENCE_MANUAL/02-gwc-workflows.md) instead.

## Prerequisites

- **Go 1.26+** (`go version`; matches the `go` directive in `go.mod`).
- A browser with WebAssembly support for the browser test lanes.
- The `gwc` launcher is run straight from source: `go run ./tools/gwc <cmd>`.

Run `go run ./tools/gwc doctor` first — it checks the toolchain, the
`wasm_exec.js` runtime asset, and browser-test prerequisites and prints fix
hints for anything missing.

## Editor setup (read this first)

Most of the framework's browser code is in files guarded by
`//go:build js && wasm`. A Go language server analyzing for your host OS will
**hide every wasm-tagged file** and show phantom errors across the whole
browser half of the codebase.

This repo commits `.vscode/settings.json`, which pins `gopls` to
`GOOS=js GOARCH=wasm` so the wasm files resolve. The trade-off is that
native-only files (SSR, the `gwc` tool, native stubs) are then the excluded
set — flip the env back to host values temporarily when working on those, or
open that package in a separate window. JetBrains users: set the same
`GOOS=js`/`GOARCH=wasm` in the Go build tags / environment for the project.

## Build

```powershell
# Native (tooling, SSR, native stubs)
go build ./...

# Browser (the wasm build the framework actually ships)
$env:GOOS='js'; $env:GOARCH='wasm'; go build ./...; Remove-Item Env:GOOS; Remove-Item Env:GOARCH
```

Bash equivalent for the wasm build: `GOOS=js GOARCH=wasm go build ./...`.

A change to public API should build under **both** targets. Many packages have a
`*_wasm.go` / `*_native.go` split; keep the exported surface identical across
the two so callers compile on either target.

## Testing

### Native tests

```powershell
go test ./...
```

### WASM unit tests (the "wasm dance")

`go test` cannot run a `js/wasm` binary directly. Compile the test to a `.wasm`
and run it under the Node wasm shim that ships with the Go toolchain:

```powershell
$env:GOOS='js'; $env:GOARCH='wasm'
go test -c -o pkg.wasm ./ui
node "$(go env GOROOT)/lib/wasm/wasm_exec_node.js" pkg.wasm -test.run=TestName
Remove-Item Env:GOOS; Remove-Item Env:GOARCH
```

The launcher wraps this for you:

```powershell
go run ./tools/gwc test -lane unit -lane wasm -lane hydration -lane browser
```

### Browser (Playwright-Go) tests

Browser suites live under `test/playwrightgo/` and example suites under
`test/playwrightgo/examples/`. They are gated behind the `playwrightgo` build
tag and run real Chromium:

```powershell
go test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExamplesAll -v
```

### Nested modules

Some tools are their own Go modules and must be tested from inside them. Most
notably the live-reload server:

```powershell
go -C tools/livereload test .
```

If `go test ./tools/livereload` reports *"main module does not contain
package"*, that is the nested-module signal — use `go -C` instead.

## Coverage

```powershell
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out
```

For wasm coverage, add `-coverprofile` to the `go test -c` step and run the
compiled binary with `-test.coverprofile`.

## Conventions

- **`parse`-prefixed locals.** Every local variable is prefixed `parse`
  (`parseKey`, `parseValue`, `parseErr`). See
  [docs/CONVENTIONS.md](docs/CONVENTIONS.md) for the full rationale and the
  function-naming rules (`verbSubject[Object]`).
- **GoDoc on every exported symbol**, first word = the symbol name.
- **No silent panics.** Browser code recovers at every goroutine and
  `js.FuncOf` boundary and emits a structured, agent-readable console report
  (crash containment) instead of killing the page. Use `ui.SafeGo(...)` for
  goroutines and do not swallow errors just to make a test pass.
- **Keep the public API stable.** `tools/api_compat_guard` pins exported
  symbols; additive changes pass, removals fail unless the baseline is
  intentionally updated.
- **Docs paths must resolve.** `docs/doclint` fails CI if a fenced command in
  any doc references a repo path that no longer exists — fix the doc when you
  move an example or package.

## Windows hazards (the framework is developed on Windows)

- **CRLF / BOM.** Several assets are embedded via `go:embed` and are
  line-ending sensitive. When scripting edits, preserve the file's existing
  EOL (the committed `.vscode/settings.json` sets `files.eol` to `\n`). Avoid
  PowerShell `Out-File` without `-Encoding utf8` for files other tools read —
  it writes UTF-16 with a BOM.
- **`go:embed` and `.gitignore`.** A broad ignore rule can exclude an embedded
  asset and break the build only in CI (a clean checkout). If you add an
  embedded asset under an ignored path, add a `!` negation and `git add -f` it.
- **Clear stale env before cross-compiling.** If a wasm build behaves oddly,
  confirm `GOOS`/`GOARCH` are what you expect — a leftover `GOOS=js` from a
  previous shell command silently breaks native builds, and vice versa.

## Before you open a PR

```powershell
go build ./...
GOOS=js GOARCH=wasm go build ./...
go vet ./...
go test ./...
go run ./tools/gwc test -lane unit -lane wasm
```

Run the relevant browser lane for UI changes, and add a focused Playwright spec
when a regression would otherwise only be caught by eye.

## Where to ask vs. file

- **Questions, ideas, "is this a bug?"** → [GitHub Discussions](https://github.com/monstercameron/GoWebComponents/discussions).
  Use *Q&A* for help, *Ideas* for proposals, *Show and tell* for what you built.
- **A concrete, reproducible defect** → open an **Issue** with a minimal repro
  (the smallest `main.go` + the `gwc` command that reproduces it).
- **A security report** → follow [SECURITY.md](SECURITY.md); do **not** open a
  public issue.

## Triage SLA

These are the maintainers' good-faith response targets (business days, not a
contractual guarantee — this is a volunteer project):

| What | First response | Notes |
| --- | --- | --- |
| Security report | **2 business days** | Acknowledgement + triage start. See SECURITY.md. |
| Bug with a repro | **5 business days** | Labeled (`bug`, severity) and routed. |
| Feature / idea | **10 business days** | Labeled and, if accepted, linked to the [roadmap](ROADMAP.md). |
| Pull request | **5 business days** | First review pass; CI must be green first. |

"First response" means a human has read it and labeled/replied — not that it is
resolved. Stale items with no maintainer reply past these windows may be bumped
by commenting `@maintainers triage`. The label taxonomy and escalation path live
in [GOVERNANCE.md](GOVERNANCE.md).

The public direction lives in [ROADMAP.md](ROADMAP.md); a feature request that
aligns with a roadmap theme is far more likely to be accepted quickly.
