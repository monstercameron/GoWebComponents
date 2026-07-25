# 14 Scaling Large Codebases

Use this chapter when your app is no longer a small demo and you need durable conventions for route families, shared modules, team ownership, validation lanes, and release discipline.

It is the right chapter for:

- medium and large app package layout
- single-shell route-family boundaries
- shared UI versus domain versus platform ownership
- state and data ownership rules for larger teams
- CI, release, and review patterns that keep a large GWC app predictable

Use another chapter instead when:

- you need the raw router APIs first: go to [08 Routing](08-routing.md)
- you need the state ownership ladder first: go to [06 State And Reactivity](06-state-and-reactivity.md)
- you need async reads, cache, and offline replay semantics first: go to [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- you need release artifacts, deployment, or PWA wiring first: go to [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)

## Overview

Large GWC apps scale best when they follow five rules:

1. keep one runtime and one router tree when the product is one app
2. split code by route family and feature ownership, not by random helper type
3. share contracts deliberately between server, browser, and SSR instead of duplicating payload rules
4. choose the smallest believable owner for state and data
5. grow validation by lane and ownership boundary, not by one giant catch-all script

The strongest repo references are:

- `examples/server/atlas-commerce-os` for a production-shaped SSR and hydration app split into `client/`, `shared/`, `server/`, and `docs/`
- `examples/server/ai-chat-wizard` for a single-shell product app with route families, server APIs, worker-backed rendering, client caches, and an operator run flow
- `examples/public/static-islands` for deliberate selective activation when content-heavy pages should not hydrate one full-page runtime

## Stability Note

This chapter documents `app-owned conventions` built on stable public surfaces rather than introducing a new framework API.

The stable foundations underneath these conventions are:

- `router.NewHistoryRouter(...)`, `Register(...)`, loaders, guards, and `HydrateMount(...)`
- `ui.UseState`, `ui.UseReducer`, `ui.UseEvent`, and provider patterns
- `fetch` resources, shared cache, and mutation queue ownership
- `gwc test`, `gwc verify`, `gwc release`, and the lane-oriented validation model

What stays application-owned:

- package layout and module names
- route-family ownership maps
- shared contract types between browser, server, and SSR
- review checklists, release gates, and team onboarding docs

## Minimal Example

Start with one shell runtime, one router, and route-family registration functions that live in separate modules.

Recommended medium-size layout:

```text
cmd/
  web/main.go
internal/
  app/
    routes/
      marketing/
      workspace/
    features/
      billing/
      projects/
    shell/
  ui/
    design/
    components/
  domain/
  platform/
    api/
    storage/
test/
  browser/
  integration/
```

```go
package app

import (
	h "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type appEnv struct {
	BuildLabel string
}

type shellProps struct {
	BuildLabel string
	Surface    string
}

// buildAppRouter creates one router and delegates registration to route-family owners.
func buildAppRouter(getEnv appEnv) *router.Router {
	getRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: "/"})
	registerMarketingRoutes(getRouter, getEnv)
	registerWorkspaceRoutes(getRouter, getEnv)
	return getRouter
}

// registerMarketingRoutes keeps public-route ownership in one module instead of scattering it across the shell.
func registerMarketingRoutes(getRouter *router.Router, getEnv appEnv) {
	getRouter.Register("/", buildSurfaceComponent(shellProps{BuildLabel: getEnv.BuildLabel, Surface: "marketing-home"}))
	getRouter.Register("/pricing", buildSurfaceComponent(shellProps{BuildLabel: getEnv.BuildLabel, Surface: "marketing-pricing"}))
}

// registerWorkspaceRoutes keeps authenticated-route ownership in its own family module.
func registerWorkspaceRoutes(getRouter *router.Router, getEnv appEnv) {
	getRouter.Register("/app", buildSurfaceComponent(shellProps{BuildLabel: getEnv.BuildLabel, Surface: "workspace-home"}))
	getRouter.Register("/app/settings", buildSurfaceComponent(shellProps{BuildLabel: getEnv.BuildLabel, Surface: "workspace-settings"}))
}

// buildSurfaceComponent binds route registration to one shared shell renderer.
func buildSurfaceComponent(getProps shellProps) router.Component {
	return func(getAttrs router.Attrs) *router.Element {
		_ = getAttrs
		return ui.CreateElement(renderShellSurface, getProps)
	}
}

// renderShellSurface renders one route-family surface while keeping the runtime root shared.
func renderShellSurface(getProps shellProps) ui.Node {
	return h.Main(
		h.Class("mx-auto max-w-4xl space-y-4 p-6"),
		h.H1("Large-app shell"),
		h.P("surface=" + getProps.Surface),
		h.P("build=" + getProps.BuildLabel),
	)
}
```

Why this is the right first scaling step:

- one runtime owns public and private routes together
- route families can evolve independently without creating multiple app roots
- the shell stays the shared composition boundary instead of becoming a dumping ground

## Production-Shaped Example

For a real feature, keep the route loader, reducer-backed workflow hook, and route view in one feature module. Route data stays route-owned; local transition logic stays feature-owned.

Recommended module split:

```text
internal/app/routes/inventory/
  route.go
  workflow.go
  view.go
```

```go
package inventory

import (
	"context"
	"fmt"

	h "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type routeProps struct {
	FallbackData router.Attrs
}

type workflowState struct {
	Filter      string
	SelectedSKU string
}

type workflowAction struct {
	Kind  string
	Value string
}

type workflowModel struct {
	State       workflowState
	ApplyFilter ui.Handler
	SelectSKU   ui.Handler
}

// registerInventoryRoute keeps one route family's loader and view contract co-located.
func registerInventoryRoute(getRouter *router.Router) {
	getRouter.Register("/app/inventory", renderInventoryRoute, router.Options{
		Loader: loadInventoryRouteData,
	})
}

// loadInventoryRouteData returns route-owned data for first paint and explicit revalidation.
func loadInventoryRouteData(getCtx context.Context, getRouteCtx router.RouteContext) (router.Attrs, error) {
	_ = getCtx
	getWarehouse := getRouteCtx.Query.Get("warehouse")
	if getWarehouse == "" {
		getWarehouse = "all"
	}

	return router.Attrs{
		"title": "Inventory queue",
		"rows": []string{
			fmt.Sprintf("warehouse=%s sku=atlas-core", getWarehouse),
			fmt.Sprintf("warehouse=%s sku=atlas-edge", getWarehouse),
		},
	}, nil
}

// buildInitialWorkflowState returns the local UI workflow defaults for the inventory route.
func buildInitialWorkflowState() workflowState {
	return workflowState{
		Filter:      "all",
		SelectedSKU: "",
	}
}

// reduceWorkflowState applies local route-surface transitions without turning them into global state.
func reduceWorkflowState(getState workflowState, getAction workflowAction) workflowState {
	switch getAction.Kind {
	case "filter":
		getState.Filter = getAction.Value
	case "select":
		getState.SelectedSKU = getAction.Value
	}
	return getState
}

// useInventoryWorkflow wraps UseReducer behind semantic handlers for the route surface.
func useInventoryWorkflow() workflowModel {
	getWorkflow := ui.UseReducer(reduceWorkflowState, buildInitialWorkflowState())

	return workflowModel{
		State: getWorkflow.Get(),
		ApplyFilter: ui.UseEvent(func() {
			getWorkflow.Dispatch(workflowAction{Kind: "filter", Value: "needs-review"})
		}),
		SelectSKU: ui.UseEvent(func() {
			getWorkflow.Dispatch(workflowAction{Kind: "select", Value: "atlas-core"})
		}),
	}
}

// renderInventoryRouteView renders route data together with local workflow state.
func renderInventoryRouteView(getProps routeProps) ui.Node {
	getRouteData := router.UseRouteData()
	if getRouteData == nil {
		getRouteData = getProps.FallbackData
	}

	getTitle, _ := getRouteData["title"].(string)
	getRows, _ := getRouteData["rows"].([]string)
	getWorkflow := useInventoryWorkflow()
	getPrimaryRow := ""
	getSecondaryRow := ""
	if len(getRows) > 0 {
		getPrimaryRow = getRows[0]
	}
	if len(getRows) > 1 {
		getSecondaryRow = getRows[1]
	}

	return h.Main(
		h.Class("space-y-4 p-6"),
		h.H1(getTitle),
		h.P("filter=" + getWorkflow.State.Filter),
		h.P("selected=" + getWorkflow.State.SelectedSKU),
		h.Div(
			h.Class("flex gap-3"),
			h.Button(h.Type("button"), h.OnClick(getWorkflow.ApplyFilter), "Apply triage filter"),
			h.Button(h.Type("button"), h.OnClick(getWorkflow.SelectSKU), "Select atlas-core"),
		),
		h.Ul(
			h.Li(getPrimaryRow),
			h.Li(getSecondaryRow),
		),
	)
}

// renderInventoryRoute bridges router attrs into the route view component.
func renderInventoryRoute(getAttrs router.Attrs) *router.Element {
	return ui.CreateElement(renderInventoryRouteView, routeProps{FallbackData: getAttrs})
}
```

Why this scales:

- route data is owned by the route loader, not by an unrelated atom or shell singleton
- local transitions stay inside a reducer-backed workflow hook instead of spreading raw `Dispatch(...)` calls everywhere
- route-family code can grow to multiple files without leaking its internals into sibling route families

## Proven Repo Shapes

Use the repo's product-shaped examples as reference layouts, not just as demos:

```text
examples/server/atlas-commerce-os/
  client/
  shared/
  server/
  docs/

examples/server/ai-chat-wizard/
  client/app/
  client/cachecore/
  server/app/
  sql/store/
  cmd/
  docs/
```

Read those structures this way:

- `client/` or `client/app/` owns browser runtime, route registration, and client-only coordination
- `shared/` owns SSR-safe contracts, page trees, repository interfaces, route payload rules, and reusable helpers
- `server/` or `server/app/` owns HTTP, auth, SSR handlers, persistence, and server-only policies
- `sql/` or persistence-specific folders own database contracts and migration assets
- `docs/` owns route maps, design notes, runbooks, smoke checklists, and backlog checkpoints that should not live only in PR threads

When a route family is mostly static and content-heavy, `examples/public/static-islands` shows the other valid scaling path: keep the content shell inert and hydrate only explicit browser islands. That is still one intentional ownership model, not an accident.

## Ownership Model

Use this ownership split for larger teams:

| Layer | Preferred owner | Typical contents | Avoid putting here |
| --- | --- | --- | --- |
| Route family | `internal/app/routes/*` or equivalent | route registration, loaders, guards, metadata, route-specific shells | unrelated sibling feature logic |
| Feature module | `internal/app/features/*` | reducer-backed workflows, panel logic, route-local composition | browser boot, global auth, global deployment rules |
| Shared UI | `internal/ui/*` or `shared/ui/*` | tokens, primitives, a11y-safe components, layout helpers | route-specific business policy |
| Domain | `domain/*` or `shared/*` contracts | entities, repository interfaces, pure policy helpers, typed payloads | browser-only APIs |
| Platform | `server/`, `platform/`, deployment tooling | HTTP, auth, persistence, logging, release, cache headers | leaf UI composition |

Team ownership should mirror the same split:

- route-family owners: marketing, auth, workspace, admin, or ops
- shared-state owners: cache keys, snapshot policy, queue policy, cross-tab ownership
- platform owners: release pipeline, deployment, observability, and security headers

## State And Data Ownership At Scale

The scaling rule is still the same as the smaller chapters: widen ownership only when the app shape forces it.

For large apps, that usually means:

- local `ui.UseState` for toggles, drafts, and ephemeral selections
- `ui.UseReducer` for multi-step feature workflows
- context for route-shell or layout-scoped dependencies
- shared atoms for cross-route preferences and coordination
- route loaders for route-critical first reads
- shared cached resources only when multiple screens really reuse the same query
- mutation queues only for workflows that must survive reloads or offline periods

The two common mistakes are:

- promoting route-local workflow state into app-wide atoms too early
- turning atoms into an unofficial async cache

If a value belongs to one route family and one workflow, keep it there.

## Validation Strategy

Grow validation by risk and ownership boundary.

Minimum lane model for a large app:

- unit: pure helpers, reducers, route-policy helpers, repository contracts
- wasm: render logic, hook workflows, route helpers, interop wrappers
- hydration: SSR reuse, bootstrap contracts, route resume behavior
- browser: end-to-end route families, focus flows, auth transitions, offline or multi-tab behaviors where relevant
- release: build, verify, and smoke output for the deployable app

Recommended command baseline:

```powershell
go run ./tools/gwc test -lane unit -lane wasm -lane hydration
go run ./tools/gwc test -lane browser -root .\examples\server\atlas-commerce-os
go run ./tools/gwc verify -app .\examples\server\atlas-commerce-os\client\main.go -root .\examples\server\atlas-commerce-os
go run ./tools/gwc release -app .\examples\server\atlas-commerce-os\client\main.go -root .\examples\server\atlas-commerce-os -out-dir .\bin\atlas-release -validate-smoke
```

For example-scale product apps, keep one browser journey set per route family or role:

- public and auth entry
- authenticated workspace shell
- one high-risk mutation flow
- one hydration-sensitive deep link
- one offline or cross-tab flow when the product supports it

## CI And Release Patterns

Keep CI and release work aligned with ownership rather than one opaque job name.

Recommended pattern:

1. run fast native and wasm checks on every PR
2. run hydration and browser suites for route, shell, or auth-sensitive changes
3. run `gwc verify` before promotion candidates
4. run `gwc release` for the real artifact shape, not a separate ad hoc build script
5. keep route maps, migration notes, and rollback steps in repo docs beside the code

Use a release-friendly output policy:

- dedicated output directories under `./bin`
- emitted wasm release manifests checked during smoke or deploy validation
- static assets, bootstrap rules, and cache policy documented per app
- changelog or checkpoint notes for large coordinated changes

Example 100 is the best repo reference for operator-oriented runbooks, seeded local data, server lifecycle commands, and journey-based smoke validation. Atlas is the best repo reference for SSR, hydration, shared contracts, and medium-size route-family structure.

## Project Structure And Team Conventions

Large teams need a few repo rules to stay coherent:

- shared UI packages expose reusable primitives, not route-family business logic
- domain and shared-contract packages stay portable across browser, server, and SSR code
- platform packages own auth, persistence, logging, release, and deployment behavior
- route maps, runbooks, smoke paths, and rollback notes live in repo docs beside the app, not only in tickets or PR threads

Keep internal component-library work equally explicit:

- publish tokens, primitives, and accessibility-safe composition points first
- avoid packaging route-specific policy as if it were a shared design-system primitive
- version internal component libraries with the same migration discipline used for the app itself

## Adoption And Evaluation Guidance

For a serious rollout, judge maturity by feature area instead of one vague repo-wide label.

Review at least these questions:

- does the chosen app shape stay inside `Stable` or consciously accepted advanced surfaces
- does one reference app in the repo prove the same route, data, SSR, or deployment pattern
- do CI lanes cover the real risk classes for this product
- are release, rollback, and browser-support expectations documented beside the app
- are security, auth, and bootstrap boundaries explicit enough for the team that will own production

`examples/server/atlas-commerce-os` is the best medium-size reference for SSR, hydration, auth-aware shells, and shared contracts. `examples/server/ai-chat-wizard` is the best operational reference for seeded local data, route-family growth, and journey-based smoke validation.

## Review And Onboarding Rules

Large apps need explicit architectural review, not just passing tests.

Review checklist:

- route changes update params, loaders, guards, and metadata together
- state changes keep ownership at the smallest believable scope
- shared UI changes stay SSR-safe and accessibility-safe
- async changes include clear load, error, retry, and invalidation behavior
- deployment-sensitive changes update release, cache, or bootstrap docs when needed

Onboarding checklist:

1. read the route map and architecture overview first
2. run the local `gwc` dev, test, and verify commands
3. trace one feature from route registration to loader to UI workflow to test
4. learn the release and rollback path before touching deployment-sensitive code

## Common Failure Modes

- route families import each other's internals instead of promoting a real shared abstraction
- the shell becomes a giant singleton because every feature wants global access
- atoms become a shadow async cache with no clear invalidation rules
- route loaders and component reads fetch the same data with different keys and drift apart
- server, shared, and browser payload contracts are duplicated instead of shared
- browser-only interop leaks into domain packages that should stay portable
- CI only runs unit tests, so hydration, browser, or release regressions escape
- ownership decisions live only in PR comments and are not recorded in repo docs or runbooks

## Validation

Use the smallest commands that prove the scaled boundary you changed:

- route, reducer, and SSR medium-size app checks:
  `go run ./tools/gwc test -lane unit -lane wasm -lane hydration -root .\examples\server\atlas-commerce-os`
- large single-shell product app browser journeys:
  `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v`
- route-family release verification:
  `go run ./tools/gwc verify -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard\client`
- deployable artifact smoke:
  `go run ./tools/gwc release -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard\client -out-dir .\bin\chat-wizard-release -validate-smoke`

When the change affects ownership or release process itself, update the app-local route map, runbook, or changelog in the same PR.

## Topic Pagination
Topic 14 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md)
