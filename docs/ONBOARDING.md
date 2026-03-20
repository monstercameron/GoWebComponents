# Onboarding and Bootstrap

This page defines the intended onboarding path for new GoWebComponents adopters.

Use it when deciding how a team should start a new app, what prerequisites are required, how starter-based apps should evolve, and what the recommended local edit-build-refresh loop looks like.

## Official Starting Path

The current official way to start is a documented manual flow, not a hidden bootstrap CLI.

The intended starting path is:

1. confirm the local prerequisites for Go, Node, and browser support
2. choose the adoption mode that matches the app shape
3. start from the closest maintained example or future starter, not from unrelated repo internals
4. keep the app on the documented public package surface: `ui`, `html`, `state`, `fetch`, `router`, `interop`, and `devtools`
5. adopt the documented wasm build and serve workflow explicitly instead of assembling commands from commit history

Until first-party starter apps exist, the repo examples are the official bootstrap reference points rather than disposable toys.

The current production-shaped reference app is `examples/86-atlas-commerce-os`, which demonstrates the integrated SSR, hydration, routing, forms, state, devtools, and deployment shape for a larger app.

## Environment Prerequisites And Platform Expectations

The intended prerequisite baseline is:

- Go 1.25 or newer
- Node.js and npm for the example dev server, browser suites, and any repo-local frontend tooling
- a browser with WebAssembly support that matches the documented support matrix in [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)

Platform notes:

- Windows workflows should use PowerShell environment assignment or the documented helper scripts
- macOS and Linux workflows may use shell environment prefixes such as `GOOS=js GOARCH=wasm`
- `wasm_exec.js` must come from the same Go toolchain version that produced the wasm binary
- browser-only code must still be kept behind browser execution paths because native `go test` does not provide `syscall/js`

Optional tooling remains application-owned unless a doc says otherwise.

## Choose Your Path

New adopters should choose an architecture path before cloning patterns from random examples.

The intended path chooser is:

- mostly client-rendered app:
  start with `ui`, `html`, local state, and a small rendered shell
- routed SPA or dashboard:
  add `router`, typed async resources, and shared state
- SSR or hydration-aware app:
  adopt the server integration, bootstrap, and hydration docs early
- forms-heavy app:
  start from the form docs and SSR secure forms example if server posts matter
- static or prerender-oriented app:
  use the prerender, assets, and wasm release docs as early constraints instead of retrofitting them later

This "choose your path" model should remain the official onboarding answer until maintained starter variants exist.

## Starter Upgrade And Template-Sync Guidance

Starter-based apps should not be expected to stay in lockstep with the repo forever.

The intended guidance is:

- official starters should version their own conventions instead of pretending every repo change must be copied verbatim
- framework upgrades should follow `CHANGELOG.md`, `docs/MIGRATIONS.md`, and package-level API changes first
- starter-specific build or tooling improvements should be documented as starter release notes, not discovered by diffing the monorepo blindly
- applications should treat starters as an initial scaffold plus documented upgrade path, not as a permanent mirror target

When first-party starters arrive, they should publish:

- their target adoption mode
- supported upgrade window
- any starter-specific breaking changes separate from core framework releases

## Recommended Inner-Loop Workflow

The intended inner loop is explicit and fast enough to reason about.

For repo evaluation and example work today:

1. build or rebuild the relevant wasm target
2. serve the example or app assets with the documented dev server or equivalent static server
3. refresh the browser when the wasm output changes
4. run focused native or `js/wasm` tests when the change touches framework behavior

Recommended commands today:

- example serving: `npm run dev:examples`
- focused native validation: `go test ./internal/runtime` or the package under change
- focused wasm validation on Windows: `go test -exec .\tools\go_js_wasm_exec.bat ./...` for the relevant package path
- release-style artifact validation: `.\tools\build-wasm-release.ps1 ...`

Reasoning rules:

- browser refresh is the current default feedback loop; state-preserving hot reload is not the documented baseline
- treat stale wasm output as the first suspect when a browser change seems missing
- keep local validation focused on the package or example being edited instead of rerunning every suite on every save

## Current Boundary

This document defines the intended onboarding and workflow contract only.

It does not yet claim:

- shipped starter applications
- a one-command project bootstrap command for new apps
- a watch-mode dev loop that rebuilds and refreshes automatically

Those remain separate backlog work.
