# IDE And Editor Integration

This page defines the current first-party IDE and editor integration boundary for GoWebComponents.

Use it when deciding how `gwc` tasks, `gopls`-friendly conventions, snippets, diagnostics, route navigation, and editor-facing framework help should fit around the current Go-first workflow.

## At A Glance

- First-party editor support is VS Code-oriented first, because it is the most direct path for tasks, problem matchers, snippets, and workspace guidance around the existing `gwc` launcher.
- The repo should still keep the underlying conventions `gopls`-friendly and ordinary-Go-compatible so teams are not forced onto one editor to use the framework correctly.
- Broader editor or IDE integrations may come later, but the public DevX story should not imply a richer multi-IDE extension ecosystem than the repo currently ships.

## Current Support Boundary

The intended first-party support boundary is:

- document one VS Code-focused workflow first for launcher tasks, problem matching, snippets, and workspace diagnostics
- keep public package APIs, comments, examples, and generated scaffolds ordinary enough that `gopls` works without editor-specific shims
- treat any future JetBrains, Neovim, Zed, or broader LSP-specific work as follow-on integrations layered on the same documented command and diagnostic contracts

This means the project should not promise a full custom language server or broad editor parity yet.

The current editor story should be read as:

- `gwc` is the command substrate
- Go files, package docs, and examples remain the source of symbol truth
- VS Code is the first target for launcher-owned tasks and problem wiring
- `gopls` compatibility is a non-negotiable baseline even when richer editor affordances are added later

## Launcher Tasks And Problem Matchers

The supported editor workflow should start from checked-in `gwc` task definitions instead of asking every team to invent local shell wrappers.

The current reference task bundle is:

- `docs/examples/gwc-vscode.tasks.json`

It defines the baseline VS Code task set for:

- `gwc: dev`
- `gwc: test`
- `gwc: verify`
- `gwc: doctor`

Current matcher scope:

- `gwc: test` and `gwc: verify` use a Go-file matcher that catches ordinary `file.go:line:column: message` compiler and test output; `gwc: verify` now runs `gwc verify -audit -audit-min-severity warning` so the editor-facing CI task also carries the golden-path audit gate
- `gwc: dev` and `gwc: doctor` currently run as explicit terminal tasks without richer editor problem projection yet; `gwc: doctor` now uses advisory audit mode so architecture findings stay visible without turning the manual diagnostic task into a warning-gated failure path

That split is intentional:

- task ownership belongs in the launcher workflow now
- richer doctor, scaffold, budget, and auditor diagnostics projected into editor problems are follow-up work, not part of the first task-bundle claim

## Snippets And Public-Surface Discoverability

The first discoverability layer should be lightweight: checked-in snippets plus ordinary Go doc comments and example links, not a custom editor runtime.

The current reference snippet bundle is:

- `docs/examples/gwc-vscode.code-snippets.json`

Its baseline entries cover:

- stateful component setup with `ui.UseState(...)`
- browser entrypoint mounting through `ui.Render(...)`
- minimal path-based routing through `router.NewRouter(...)`
- focused component-test setup through the public testing helpers

The goal of this first bundle is not to automate architecture decisions. It is to make the current public surface easier to find from the editor without forcing a developer to start from grep alone.

## Focused Workflow Snippets

The same snippet bundle now also covers the higher-signal workflows the docs keep recommending:

- loader-backed route registration through `router.Options{Loader: ...}`
- manual route refresh through `router.UseRevalidator(...)`
- SSR bootstrap registration and inline script emission through `ui.RegisterRouteBootstrapData(...)` and `ui.RenderBootstrapScript(...)`
- browser hydration from a transferred bootstrap payload through `ui.ReadBootstrapScript(...)` and `ui.Hydrate(...)`
- typed form state wiring through `ui.UseForm(...)`
- shared async cache setup through `fetch.UseCachedResource(...)`

These focused snippets are meant to reinforce the recommended architecture paths, not generic boilerplate generation.

## Hover Usage Guidance

Symbol help should explain recommended usage, not only repeat the Go signature.

The intended hover shape is:

- what problem the symbol solves
- when it is the preferred choice over nearby alternatives
- one short warning about the most common misuse
- one example or doc pointer when the symbol participates in a larger workflow

High-value symbol groups for this treatment:

- hooks such as `ui.UseState(...)`, `ui.UseEffect(...)`, `ui.UseForm(...)`, and `ui.UseTransition(...)`
- shared-state helpers such as `state.UseAtom(...)`, `state.UseComputed(...)`, and `state.UseDerived(...)`
- router entrypoints such as route loaders, `router.UseNavigate(...)`, `router.UseRevalidator(...)`, and `router.Outlet()`
- async helpers such as `fetch.UseResource[T](...)`, `fetch.UseCachedResource[T](...)`, and `fetch.Fetch(...)`
- SSR and hydration helpers such as `ui.RenderToString(...)`, `ui.RenderBootstrapScript(...)`, and `ui.Hydrate(...)`
- devtools and diagnostics helpers where the main question is when the tool belongs in product code versus debugging code

The key rule is that hover text should reinforce the recommended path through the framework instead of leaving readers with a flat signature dump.

## Editor-Visible Project Diagnostics

The editor workflow should surface the highest-friction project checks without forcing developers to leave the editing surface for every issue.

The current intended diagnostic sources are:

- `gwc doctor` for toolchain, `wasm_exec.js`, browser-test workspace, scaffold metadata, project detection, and port checks
- `gwc verify` for app-local test plus CI-profile build status
- scaffold metadata validation around `gwc-start.json`
- dev-server resolution checks around app entrypoint, HTML shell, and wasm output ownership

The high-priority conditions to surface are:

- missing or unresolved `wasm_exec.js`
- broken or incomplete `gwc-start.json`
- unresolved `gwc dev` app or HTML entrypoints
- missing browser-test prerequisites when a project claims browser coverage
- launcher status that blocks the documented inner loop from starting

The point of this editor-visible layer is to make the current launcher checks more reachable, not to replace the launcher with editor-only logic.

## Editor Problem Projection

Launcher, scaffold, and future auditor findings should project into editor problems with the same discipline used by runtime diagnostics.

Recommended problem shape:

- stable code such as `GWC-DOCTOR-*`, `GWC-SCAFFOLD-*`, `GWC-VERIFY-*`, or a future auditor namespace
- severity mapped intentionally to `error`, `warning`, or `information`
- one short summary line
- remediation text that tells the developer what to fix next
- source attribution that points back to `gwc doctor`, `gwc verify`, scaffold generation, or a named audit pass

Recommended first projected sources:

- `gwc doctor` failures and warnings
- scaffold metadata validation failures
- future golden-path auditor findings
- release-budget and startup-budget failures once those audits exist

The important rule is consistency: the editor problem should carry the same stable identifier and remediation wording the terminal report uses so the two surfaces do not drift apart.

## Navigation And Refactor Scope

The project should improve navigation and refactor support where Go-first frontend code becomes grep-heavy, but it should not promise whole-app magic refactors that ignore ordinary Go ownership boundaries.

Recommended support boundary:

- rely on normal `gopls` go-to-definition and rename support for exported symbols, package APIs, and scaffolded Go code first
- add framework-specific help where route paths, loader ownership, and scaffold entrypoints are otherwise stringly and hard to follow
- avoid promising automated starter-template upgrades or global route rewrites until the route and scaffold metadata contracts are strong enough to make those edits safe

The intended value is targeted navigation help for known framework pain points, not a custom refactor engine for every GoWebComponents pattern.

## Route Symbol Indexing

Route-heavy apps need one extra navigation layer beyond ordinary Go symbol lookup: route-path ownership and route-chain discovery.

Recommended route index coverage:

- path patterns registered through `router.Register(...)`
- layout versus leaf route relationships
- loader, loading, and error renderer ownership for one route
- scaffolded entrypoints that mount or hydrate the route tree

Recommended editor affordances:

- go-to-definition from a route path reference to the route registration site
- find-references for a registered route path or route metadata key
- a route detail view that shows the registered component, loader, and surrounding layout chain without requiring full-text search

This is the most valuable framework-specific navigation addition because route ownership is where larger apps otherwise fall back to grep-driven archaeology.

## Lightweight Code Actions

Editor integration should offer a small set of corrective actions for common framework setup failures instead of only passive warnings.

Recommended first code actions:

- add or repair missing launcher tasks from the checked-in `gwc` task bundle
- add missing scaffold metadata keys or point to `gwc-start.json` fields that need repair
- generate a baseline component or test stub when the supported workflow expects one
- insert or update known auditor suppressions only when the audit contract documents that suppression explicitly

Recommended rule:

- keep code actions narrow, reviewable, and tied to existing documented contracts
- prefer generating or repairing one local file over mutating many files opaquely
- do not offer a quick fix when the framework does not have a documented golden-path answer yet

## Current Boundary

This document defines the intended editor-integration direction.

It does not yet claim:

- a shipped VS Code extension
- a custom language server
- cross-editor parity for snippets, code actions, or route indexing

Those remain follow-up work on top of the documented command, task, and diagnostic contracts.
