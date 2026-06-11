# 15 Design Notes And Boundaries

Use this chapter when you need the repo's explicit answer to these questions:

- what is safe to depend on long term
- what is still experimental
- what belongs in core versus a companion package versus app code
- what the scheduler does and does not promise
- what counts as a breaking change beyond Go type signatures

It is the right chapter for:

- stability tiers and support expectations
- framework scope and non-goals
- extension and companion-package boundaries
- scheduling tradeoffs and current limits
- migration, deprecation, and production-correctness rules

Use another chapter instead when:

- you need task-oriented app-building guidance first: go to [02 GWC Workflows](02-gwc-workflows.md)
- you need route, SSR, or deployment details first: go to [08 Routing](08-routing.md), [09 SSR And Hydration](09-ssr-and-hydration.md), or [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)
- you need team-scale structure and release workflow first: go to [14 Scaling Large Codebases](14-scaling-large-codebases.md)

## Overview

The project direction is conservative on purpose:

1. keep the core small and Go-first
2. prefer explicit ownership over hidden framework magic
3. ship typed helper surfaces before broad automation or code generation
4. treat behavior and deployment contracts as seriously as exported signatures
5. let optional or policy-heavy capabilities grow in companion packages or app code before widening core

If a feature cannot yet meet those rules, it should stay experimental or application-owned.

## Stability Tiers

Use these tiers when deciding what your app can rely on:

| Tier | Meaning | Current examples | Change policy |
| --- | --- | --- | --- |
| `Stable` | long-term production surface in core | `ui`, `html`, `state`, `fetch`, `flags`, `router`, documented SSR and hydration entrypoints | breaking changes only in major releases with migration guidance |
| `Supported companion` | production-safe integration surface outside the smallest core | `devtools`, `head`, `plugin` | major-version breaking changes only; additive diagnostics or fields may grow in minor releases |
| `Experimental` | public and usable, but still proving shape or lifecycle | transitions, deferred values, async boundaries, `fetch.UseCachedResource`, advanced router loaders or guards, alternative bootstrap transports, compiler-assisted experiments | shape may change in minor releases with release notes and migration guidance |
| `Internal` | unsupported implementation detail or repo-only helper | `internal/*`, example glue, runtime internals, repo tooling harnesses | no semver promise for consumer apps |

Current practical reading:

- if you need long-lived production bets, prefer `Stable`
- if you need optional but supported integrations, `Supported companion` is acceptable
- if you use `Experimental`, hide it behind app-owned wrappers
- if it is `Internal`, do not build product code on it

## Public API Conventions

The public surface follows a few repo-wide rules on purpose:

- Go-first typed composition instead of directive syntax or template-specific magic
- explicit owners for routes, state, loaders, bootstrap data, and browser bridges
- small exported primitives that compose into workflows instead of one giant lifecycle object
- stability expectations documented alongside the API family, not inferred from examples alone

For app authors, the practical takeaway is simple:

- prefer the documented public package names over old aliases or example glue
- keep experimental or policy-heavy surfaces behind local wrappers
- treat examples as proof points, not as undocumented API expansion

## Scope And Non-Goals

GWC is intentionally not trying to be every framework shape at once.

The repo's current non-goals are explicit:

- no public generic plugin lifecycle in core; the current framework-owned kernel stays internal
- no directive model separate from normal Go component composition
- no framework-owned route-manifest generation or prerender enumeration in core
- no compiler-required authoring path for ordinary apps
- no turnkey auth, CSP, compliance, or secret-management framework
- no hidden concurrent renderer with a rich public scheduler ladder

That does not mean those areas are ignored. It means the project only widens the supported surface when the ownership model is clear enough to defend under semver.

## Minimal Example

When you choose to use an experimental API, keep it behind one app-owned wrapper so the rest of the codebase depends on your stable local contract instead of the raw experimental shape.

```go
package search

import (
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type searchModel struct {
	Query            string
	DeferredQuery    string
	IsPending        bool
	QueueAtlasSearch ui.Handler
}

// useSearchModel hides the experimental scheduling APIs behind one app-owned hook.
func useSearchModel() searchModel {
	getQuery := ui.UseState("")
	getTransition := ui.UseTransition()
	getDeferredQuery := ui.UseDeferredValue(getQuery.Get())

	return searchModel{
		Query:         getQuery.Get(),
		DeferredQuery: getDeferredQuery,
		IsPending:     getTransition.Pending(),
		QueueAtlasSearch: ui.UseEvent(func() {
			// Keep the scheduling decision local so the rest of the app reads one stable workflow hook.
			getTransition.Start(func() {
				getQuery.Set("atlas warehouse")
			})
		}),
	}
}

// renderSearchSurface exposes the app-owned surface instead of scattering transition calls everywhere.
func renderSearchSurface() ui.Node {
	getModel := useSearchModel()
	getPendingLabel := "no"
	if getModel.IsPending {
		getPendingLabel = "yes"
	}

	return h.Main(
		h.Class("space-y-4 p-6"),
		h.H1("Search surface"),
		h.P("query=" + getModel.Query),
		h.P("deferred=" + getModel.DeferredQuery),
		h.P("pending=" + getPendingLabel),
		h.Button(h.Type("button"), h.OnClick(getModel.QueueAtlasSearch), "Queue search refresh"),
	)
}
```

Why this is the right boundary:

- the app can adopt experimental scheduling without leaking it across every feature
- migration later touches one local hook first, not every component
- the design intent stays explicit: experimental runtime primitive, app-owned workflow contract

## Production-Shaped Example

Keep runtime route contracts in one place, but leave metadata, prerender policy, and export decisions application-owned.

```go
package routes

import (
	"net/url"
	"sort"

	"github.com/monstercameron/GoWebComponents/router"
)

type routeRecord struct {
	Contract        router.RouteContract
	MetadataKey     string
	IsPrerenderable bool
}

// getRouteRegistry stores runtime route contracts together with app-owned policy.
func getRouteRegistry() map[string]routeRecord {
	return map[string]routeRecord{
		"home": {
			Contract:        router.MustDefineRoute("/"),
			MetadataKey:     "marketing-home",
			IsPrerenderable: true,
		},
		"pricing": {
			Contract:        router.MustDefineRoute("/pricing"),
			MetadataKey:     "marketing-pricing",
			IsPrerenderable: true,
		},
		"inventory": {
			Contract:        router.MustDefineRoute("/app/inventory/:sku"),
			MetadataKey:     "inventory-detail",
			IsPrerenderable: false,
		},
	}
}

// buildInventoryHref uses the runtime route contract without requiring core-owned manifest generation.
func buildInventoryHref(getSKU string, getTab string) string {
	getQuery := url.Values{}
	if getTab != "" {
		getQuery.Set("tab", getTab)
	}
	return getRouteRegistry()["inventory"].Contract.MustHref(map[string]string{"sku": getSKU}, getQuery)
}

// buildPrerenderPaths derives export inputs from the app-owned registry instead of expecting router to own export tooling.
func buildPrerenderPaths() []string {
	getRegistry := getRouteRegistry()
	getPaths := make([]string, 0, len(getRegistry))
	for _, getRecord := range getRegistry {
		if !getRecord.IsPrerenderable || len(getRecord.Contract.ParamNames()) > 0 {
			continue
		}
		getPaths = append(getPaths, getRecord.Contract.Pattern())
	}
	sort.Strings(getPaths)
	return getPaths
}
```

Why this is the production-shaped boundary:

- runtime reverse routing stays stable and typed
- metadata and prerender policy stay in app code, where deployment shape actually varies
- core does not need to own a second manifest system before the real shared need is proven

## Extension And Companion Boundaries

Use this decision order whenever a new capability is proposed:

1. application-owned composition on current public APIs
2. package-owned extension point where one subsystem clearly owns the concern
3. supported companion package for optional or policy-heavy reusable features
4. internal shared plugin lifecycle only if multiple real extension categories prove the same need, and only publish it after the SPI is stable

Current examples of that policy:

- `router.RouteContract` is runtime-first, while route registries, prerender lists, and metadata grouping stay app-owned
- `devtools`, `head`, and `plugin` are companion surfaces rather than core runtime internals
- the public `plugin` package stays the application-owned companion host, while `internal/pluginruntime` owns the framework's internal deep-plugin kernel
- virtualization is intentionally aimed at a supported companion package first
- compiler-assisted features remain opt-in experiments rather than a second default authoring model

Core also does not define a directive model today because typed Go composition, hooks, and helper functions already cover the intended authoring path.

## Internal Plugin Kernel

Deep framework plugins now use an internal kernel rather than the public `plugin` package.

That split is intentional:

- the public `plugin` package remains the supported application-owned companion host
- the framework-owned kernel lives under `internal/pluginruntime`
- `devtools` is the first shipped consumer of that internal kernel
- the kernel is not yet a public semver-stable extension contract

Use this boundary table when deciding which layer owns an extension:

| Layer | Ownership | Stability | Use it for | Do not use it for |
| --- | --- | --- | --- | --- |
| `plugin` | application-owned public companion | `Supported companion` | route guards, request observers, app-owned devtools sections or actions, head providers, bootstrap payloads, form validators | deep framework inspection, runtime internals, hidden lifecycle control |
| `internal/pluginruntime` | framework-owned internal kernel | `Internal` | first-party plugins, health tracking, guarded plugin startup, typed service lookup, contribution registration | app code, third-party plugin contracts, semver promises |
| interposers | framework-owned adapters | `Internal` | hiding `runtime` versus `runtime2`, DOM adapter, router, fetch, asset, and security implementation details behind stable kernel service contracts | exposing raw internal structs to plugins |

Current built-in internal service families are:

- runtime and diagnostics summaries
- route and fetch or cache inspection
- DOM, style, and event inspection
- asset, cache, and release inspection
- security and capture services
- runtime2 metadata for worker-backed and transport-backed regions

Current built-in internal contribution families are:

- devtools sections
- devtools actions
- devtools panels

The current shipped slice is deliberately narrow:

- `devtools` reads kernel-backed sections and actions through the internal kernel
- `ApplyHostExtensions(...)` remains the compatibility bridge for application-owned `plugin.Host` contributions
- app-owned devtools state, compatibility host state, and kernel-owned contributions compose additively instead of replacing one another

## Interposers, Safety, And Performance

The internal kernel is built around an interposer layer.

Interposers exist so that plugin-facing services can stay stable while implementation details keep moving:

- plugins talk to normalized service contracts
- the kernel talks to interposers
- interposers adapt `internal/runtime`, `internal/runtime2`, browser adapters, router state, fetch state, asset state, and security state

This avoids freezing internal runtime structures just because one plugin needs deep visibility.

Kernel safety and performance rules are also explicit:

- every plugin entrypoint is guarded and panic-isolated
- the kernel tracks plugin health and can quarantine a misbehaving plugin without crashing the framework
- plugin work is classified internally as `hot`, `warm`, or `background`
- activation is explicit through `boot`, `view`, `session`, and `opportunistic` policies
- deep plugins read bounded normalized snapshots and issue typed commands instead of holding raw mutable runtime pointers
- DOM, event, network, asset, and security inspection must stay auditable and bounded rather than becoming an arbitrary callback escape hatch

For app code, the main takeaway is simple:

- use the public `plugin` package when you need an application-owned companion host
- use the public `devtools` package when you need embeddable inspection surfaces
- do not depend on `internal/pluginruntime` or any interposer path directly

## Scheduling Boundaries

The current scheduler exposes one narrow distinction only:

- urgent work
- transition work

Current shipped limits matter:

- transitions defer work, but they do not time-slice one render pass mid-flight
- `ui.UseTransition()` already provides the current pending-state API
- `ui.UseDeferredValue(...)` is for lagging derived views, not for route-loader fallback
- transitions do not override route-loader pending UI or `ui.AsyncBoundary` fallback behavior
- production-tag builds must keep the same scheduling semantics as development builds

Treat the scheduler as a focused ergonomics tool, not as a hidden fully concurrent renderer.

## Production-Correctness Rules

The repo treats operational behavior as part of the public contract.

That means a change is breaking when a valid app must change code, markup, deployment, or operational assumptions to preserve documented behavior. Examples:

- route matching or redirect behavior changes
- hydration reuse changes for documented valid markup
- form validation or touched-state sequencing changes
- snapshot import or export semantics changes
- SSR bootstrap format changes on the documented default path

Other correctness rules are equally explicit:

- browser-visible bootstrap, storage, worker messages, logs, and diagnostics are public data channels
- secrets, bearer tokens, and server-only policy must not cross into those channels
- `internal/*` and example glue are not supported contracts even when they are informative
- examples do not prove support on their own unless the docs and tests also back the claim

## Security Governance And Configuration Boundaries

The framework sets trust boundaries, not a full compliance program.

Current rules:

- bootstrap, storage, worker messages, snapshots, and browser-visible logs are public-data channels
- auth, session issuance, CSRF validation, CSP, secret lookup, and compliance mapping remain application-owned server concerns
- feature flags and runtime configuration may cross into the browser only when they are non-secret, typed, and intentionally client-visible
- governance artifacts such as audit trails, redaction policy, retention, and procurement review packets belong in app or org process, not in the core runtime

That is why the manual treats security as an ownership model:

- framework helpers can reduce accidental exposure
- they do not remove the need for application review, redaction, and deployment policy

## Evaluation And Comparison Guidance

The comparative read is intentionally narrow:

- GWC is strongest when a team wants typed Go-first ownership, explicit route and data boundaries, and one runtime story across browser, SSR, and companion tooling
- it is weaker than larger ecosystems on market depth, third-party integrations, and out-of-the-box maturity breadth
- React, Vue, Svelte, Solid, Blazor, and Qwik each win different tradeoffs; the point of GWC is not to imitate all of them at once

For adoption and enterprise review, evaluate:

- stability tier of the exact features your app needs
- browser and deployment assumptions
- migration discipline and release notes
- security and bootstrap boundaries
- reference-app evidence for your intended app shape

Use feature-area readiness, not one repo-wide slogan, when deciding whether the framework is ready for a specific product.

## Deferred Experimental Directions

Several areas remain intentionally bounded:

- compiler-assisted authoring stays opt-in and must never become the only path for ordinary apps
- virtualization is treated as companion-package territory first
- server-interactive UI and richer streaming SSR stay experimental until latency, backpressure, offline, and reconnect behavior are defensible
- multithreaded runtime and `runtime2` work remain performance and architecture experiments, not the default app model
- the internal plugin kernel may normalize `runtime` and `runtime2` for first-party plugins, but that does not upgrade `runtime2` itself into the default app model
- worker pools, shared buffers, and alternative transports should be earned by measured workload pressure

## Migration And Release Discipline

Stable and supported companion surfaces follow a stricter lifecycle:

1. replacement path first
2. deprecation notice in GoDoc
3. changelog and migration-guide entry
4. support window of at least two minor releases and at least 90 days
5. removal only in a major release

Release discipline also requires:

- behavior changes must be classified by stability tier before release
- experimental changes still need release notes and migration guidance when user action is required
- major releases must include migration coverage for runtime authoring, router behavior, SSR and hydration, forms and local state, shared state and fetch, and deployment expectations
- feature-area readiness matters more than one vague repo-wide "production ready" claim

## Common Failure Modes

- depending on `internal/*` because it seems convenient during one refactor
- spreading experimental APIs across the whole app instead of hiding them behind a wrapper
- expecting the public `plugin` package to be the deep framework kernel, or expecting the internal kernel to be a public compatibility promise already
- expecting core to own directives, route manifests, or compiler transforms that the docs still classify as app-owned or experimental
- treating transitions as if they were true time-sliced rendering
- shipping secrets or privileged policy details in bootstrap, storage, logs, or exported snapshots
- assuming examples alone upgrade a surface from experimental to stable
- changing operational behavior without treating it as a semver or migration concern

## Validation

Use the smallest commands that prove the boundary you changed or are relying on:

- scheduling and deferred-value examples:
  `go run ./tools/gwc serve -root .\\examples\\27-transition-hooks -port 8027`
- lagging derived-value behavior:
  `go run ./tools/gwc serve -root .\\examples\\26-use-deferred-value -port 8026`
- route, SSR, and deployment contract verification in a production-shaped app:
  `go run ./tools/gwc verify -app .\\examples\\86-atlas-commerce-os\\client\\main.go -root .\\examples\\86-atlas-commerce-os`
- migration and behavior-sensitive release lane:
  `go run ./tools/gwc test -lane unit -lane wasm -lane hydration`

When a change affects stability tier, deprecation status, or extension ownership, update the owning docs in the same change instead of leaving the policy only in code comments or PR discussion.

## Topic Pagination

Topic 15 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.

- Previous topic: [14 Scaling Large Codebases](14-scaling-large-codebases.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [16 API Browser](16-api-browser.md)
