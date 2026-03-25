# Onboarding and Bootstrap

This page defines the current onboarding path for new GoWebComponents adopters.

Use it when deciding how a team should start a new app, what prerequisites are required, how starter-based apps should evolve, and what the recommended local edit-build-refresh loop looks like.

## At A Glance

- the repo-standard entrypoint is now `go run ./tools/gwc ...`
- one-command bootstrap is available through `go run ./tools/gwc bootstrap`
- the root README contains the smallest current starter-shaped app example
- `gwc start` now ships starter variants for client-only, routed/dashboard, SSR, and reference-app adoption modes
- larger adoption paths should choose client-rendered, routed, SSR, forms-heavy, or prerender-first architecture intentionally instead of cloning random examples

## Quick Start Path

For a new evaluation or greenfield app, the current path is:

1. run `go run ./tools/gwc bootstrap` for the starter bootstrap flow (doctor checks plus scaffold start path)
2. run `go run ./tools/gwc bootstrap -examples` when you want to bootstrap from the examples catalog instead of generating a starter
3. inspect the generated `FEATURE_MATRIX.md` and `README.md` in the starter output to confirm the selected adoption mode
4. move onto `gwc build`, `gwc test`, and `gwc verify` once the app shape stabilizes

## Current Bootstrap References

The practical bootstrap references today are:

- the root `README.md` starter example for the smallest current public-package shape
- numbered examples for focused feature adoption
- `examples/86-atlas-commerce-os` for a larger integrated reference app
- `docs/START_HERE.md` when a team wants the broader documentation entrypoint before choosing an architecture path

## Official Starting Path

The current official way to start is a documented, explicit flow backed by the public runner surface rather than a hidden scaffold generator.

The intended starting path is:

1. confirm the local prerequisites for Go, Node, and browser support
2. choose the adoption mode that matches the app shape
3. start from the closest maintained example or future starter, not from unrelated repo internals
4. keep the app on the documented public package surface: `ui`, `html`, `state`, `fetch`, `router`, `interop`, and `devtools`
5. adopt the documented `gwc` build, dev, and validation workflow instead of assembling commands from commit history

`gwc start` is now the first-party starter entrypoint. The current preset map is:

- `minimal-client` for client-only app bootstraps
- `routed-spa` for dashboard/content-style routed apps
- `ssr-app` for SSR plus hydration ownership from day one
- `reference-app` for a broader baseline with forms, async data, shared state, browser-test stubs, and release defaults

The production-shaped reference implementation remains `examples/86-atlas-commerce-os` for teams that want a larger integrated example beyond the generated starters.

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

This "choose your path" model is now mapped directly to maintained `gwc start` presets so teams can pick an architecture without reverse-engineering the examples tree.

## Smallest Starter Shape Today

The smallest current public-package starter shape is the one documented in the root README:

- `ui` for component composition and rendering
- `html` or `html/shorthand` for element authoring
- `go run ./tools/gwc dev -app .\main.go` for the standalone wasm inner loop
- `gwc build`, `gwc test`, and `gwc verify` as the app matures

That starter is intentionally small. Teams should add `router`, `fetch`, `state`, `interop`, `i18n`, `head`, or `hotreload` only when the target architecture actually needs them.

## Starter Upgrade And Template-Sync Guidance

Starter-based apps should not be expected to stay in lockstep with the repo forever.

The intended guidance is:

- official starters should version their own conventions instead of pretending every repo change must be copied verbatim
- framework upgrades should follow `CHANGELOG.md`, `docs/MIGRATIONS.md`, and package-level API changes first
- starter-specific build or tooling improvements should be documented as starter release notes, not discovered by diffing the monorepo blindly
- applications should treat starters as an initial scaffold plus documented upgrade path, not as a permanent mirror target
- starter output should follow the disposable ownership rules in `docs/STARTER_OUTPUT_RULES.md`

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

- runner health check: `go run ./tools/gwc doctor`
- example serving: `go run ./tools/gwc examples`
- standalone wasm inner loop: `go run ./tools/gwc dev -app .\path\to\main.go`
- release-style build: `go run ./tools/gwc build -app .\path\to\main.go -profile development`
- focused native validation: `go test ./internal/runtime` or the package under change
- focused wasm validation on Windows: `go test -exec .\tools\go_js_wasm_exec.bat ./...` for the relevant package path
- release-style artifact validation: `.\tools\build-wasm-release.ps1 ...`

Reasoning rules:

- browser refresh remains a reliable fallback, but state-preserving hot reload is now the documented dev loop for standalone apps that call `hotreload.Enable()` and run through `gwc dev`
- treat stale wasm output as the first suspect when a browser change seems missing
- keep local validation focused on the package or example being edited instead of rerunning every suite on every save

## Contributor Versus Consumer Workflows

The commands differ slightly depending on whether you are editing this repository itself or building an app on top of the public packages.

### Repo Contributor Workflow

Use this path when changing framework packages, examples, launcher code, or docs inside this monorepo.

1. run `go run ./tools/gwc doctor`
2. use `go run ./tools/gwc examples` when you need the examples catalog, or `go run ./tools/gwc dev -app .\examples\NN-name\main.go` when you want a single-example inner loop
3. validate only the package or example you changed first, then widen if needed
4. treat `examples/` and `docs/` as maintained repo surfaces, not disposable starter output

Typical commands:

- framework package edit: `go test ./ui ./html`
- example edit: `go run ./tools/gwc dev -app .\examples\01-counter\main.go`
- launcher/tooling edit: `go test ./tools/gwc/...` or the touched tool directory

### Framework Consumer Workflow

Use this path when you are building a separate application that depends on the public GoWebComponents packages.

1. scaffold or bootstrap the app with `go run ./tools/gwc bootstrap` or `go run ./tools/gwc start`
2. run `go run ./tools/gwc dev -app .\main.go` from the app workspace
3. use `gwc build`, `gwc test`, and `gwc verify` against the app root instead of copying monorepo-only example commands
4. treat the generated app as user-owned code that upgrades through package releases and documented migrations, not by diffing random repo internals

Typical commands:

- app dev loop: `go run ./tools/gwc dev -app .\main.go`
- app validation: `go run ./tools/gwc verify -app .\main.go -root .`
- release-style build: `go run ./tools/gwc build -app .\main.go -profile development`

## Current Boundary

This document defines the intended onboarding and workflow contract only.

Project-local hot reload is now documented, but repo-wide scaffolding and app generation remain separate backlog work.

## Review Checklist

- does the page point new users to `gwc` first instead of ad hoc scripts
- does it separate the smallest starter shape from the larger integrated reference-app path
- does it tell users how to choose a path before copying patterns from examples
- does the upgrade guidance point to `CHANGELOG.md` and `docs/MIGRATIONS.md`
- does it avoid claiming a shipped scaffold generator that the repo does not actually provide
