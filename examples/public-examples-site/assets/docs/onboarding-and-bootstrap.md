# Onboarding and Bootstrap

This page defines the current onboarding path for new GoWebComponents adopters.

Use it when deciding how a team should start a new app, what prerequisites are required, how starter-based apps should evolve, and what the recommended local edit-build-refresh loop looks like.

## At A Glance

- the repo-standard entrypoint is now `go run ./tools/gwc ...`
- the root README contains the smallest current starter-shaped app example
- `go run ./tools/gwc start` ships maintained starter variants for minimal client, routed SPA, SSR, dashboard, marketing, blog, authed shell, and reference-app shapes
- larger adoption paths should choose client-rendered, routed, SSR, forms-heavy, or prerender-first architecture intentionally instead of cloning random examples

## Quick Start Path

For a new evaluation or greenfield app, the current path is:

1. confirm prerequisites with `go run ./tools/gwc doctor`
2. choose the smallest matching starter from the `gwc start` gallery
3. run `go run ./tools/gwc examples` when browsing the example catalog is the fastest way to compare patterns
4. run `go run ./tools/gwc dev -app .\path\to\main.go` for the standalone wasm inner loop
5. move onto `gwc build`, `gwc test`, and `gwc verify` once the app shape stabilizes

## Current Bootstrap References

The practical bootstrap references today are:

- `go run ./tools/gwc start` for generated starter apps with `gwc-start.json`, `FEATURE_MATRIX.md`, generated CI, and baseline tests
- the root `README.md` starter example for the smallest current public-package shape
- slug-named examples for focused feature adoption
- `examples/server/atlas-commerce-os` for a larger integrated reference app
- `docs/START_HERE.md` when a team wants the broader documentation entrypoint before choosing an architecture path

## Official Starting Path

The current official way to start is a documented, explicit flow backed by the public runner surface and maintained starter templates.

The recommended starting path is:

1. confirm the local prerequisites for Go, Node, and browser support
2. choose the adoption mode that matches the app shape
3. start from the closest maintained starter or example, not from unrelated repo internals
4. keep the app on the documented public package surface: `ui`, `html`, `state`, `fetch`, `router`, `interop`, and `devtools`
5. adopt the documented `gwc` build, dev, and validation workflow instead of assembling commands from commit history

The starter gallery is:

- `minimal-client`: smallest browser-mounted wasm app
- `routed-spa`: client-rendered routes and browser smoke tests
- `ssr-app`: request-time HTML plus hydration
- `dashboard-app`: internal tools and reporting surfaces
- `marketing-site`: launch pages and SEO-sensitive public surfaces
- `content-blog`: docs, changelogs, and editorial content
- `authed-app-shell`: SaaS-style app shell with forms, async data, and shared state
- `reference-app`: broad baseline across the main public packages

The current production-shaped reference app is `examples/server/atlas-commerce-os`, which demonstrates the integrated SSR, hydration, routing, forms, state, devtools, and deployment shape for a larger app.

## Environment Prerequisites And Platform Expectations

The current prerequisite baseline is:

- Go 1.25 or newer
- Node.js and npm for the example dev server, browser suites, and any repo-local frontend tooling
- a browser with WebAssembly support that matches the documented support matrix in [BROWSER_SUPPORT.md](browser-support.md)

Platform notes:

- Windows workflows should use PowerShell environment assignment or the documented helper scripts
- macOS and Linux workflows may use shell environment prefixes such as `GOOS=js GOARCH=wasm`
- `wasm_exec.js` must come from the same Go toolchain version that produced the wasm binary
- browser-only code must still be kept behind browser execution paths because native `go test` does not provide `syscall/js`

Optional tooling remains application-owned unless a doc says otherwise.

## Choose Your Path

New adopters should choose an architecture path before cloning patterns from random examples.

The recommended path chooser is:

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

This "choose your path" model remains the official onboarding answer even with maintained starter variants: choose the closest app shape first, then add capabilities deliberately.

## Smallest Starter Shape Today

The smallest current public-package starter shape is the one documented in the root README:

- `ui` for component composition and rendering
- `html` or `html/shorthand` for element authoring
- `go run ./tools/gwc dev -app .\main.go` for the standalone wasm inner loop
- `gwc build`, `gwc test`, and `gwc verify` as the app matures

That starter is intentionally small. Teams should add `router`, `fetch`, `state`, `interop`, `i18n`, `head`, or `hotreload` only when the target architecture actually needs them.

## Starter Upgrade And Template-Sync Guidance

Starter-based apps should not be expected to stay in lockstep with the repo forever.

The current guidance is:

- official starters publish their intended shape through `gwc-start.json`, `FEATURE_MATRIX.md`, generated CI, and generated starter tests
- framework upgrades should follow `CHANGELOG.md`, `docs/MIGRATIONS.md`, and package-level API changes first
- starter-specific build or tooling improvements should be documented as starter release notes, not discovered by diffing the monorepo blindly
- applications should treat starters as an initial scaffold plus documented upgrade path, not as a permanent mirror target

The repo starter-templates CI lane scaffolds, tests, and builds every default preset so the gallery stays copyable.

## Recommended Inner-Loop Workflow

The recommended inner loop is explicit and fast enough to reason about.

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

## Current Boundary

This document defines the current onboarding and workflow contract only.

It does not yet claim:

- shipped starter applications
- a one-command project bootstrap command for new apps

Project-local hot reload is now documented, but repo-wide scaffolding and app generation remain separate backlog work.

## Review Checklist

- does the page point new users to `gwc` first instead of ad hoc scripts
- does it separate the smallest starter shape from the larger integrated reference-app path
- does it tell users how to choose a path before copying patterns from examples
- does the upgrade guidance point to `CHANGELOG.md` and `docs/MIGRATIONS.md`
- does it avoid claiming a shipped scaffold generator that the repo does not actually provide
