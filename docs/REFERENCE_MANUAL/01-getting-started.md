# 01 Getting Started

Use this chapter when you want the shortest correct path from evaluation to a working GoWebComponents app.

It is the right starting point for:

- first-time evaluation
- greenfield browser apps
- teams moving from examples into a small real app
- readers who want the supported public surface before deeper architecture detail

Use a later chapter instead when:

- you already know you need routing, loaders, and guarded navigation: go to [08 Routing](08-routing.md)
- you already know you need request-time HTML and hydration: go to [09 SSR And Hydration](09-ssr-and-hydration.md)
- you are designing a larger app boundary from day one: go to [03 App Shapes](03-app-shapes.md) and [14 Scaling Large Codebases](14-scaling-large-codebases.md)

## Overview

The current starter path is intentionally incremental.

Start with:

- `ui` for components, rendering, and hooks
- `html` or `html/shorthand` for DOM authoring
- `gwc` for the dev loop, builds, and validation

Add later only when the app needs them:

- `router` for multiple pages, params, loaders, metadata, and guards
- `state` for shared state outside one component subtree
- `fetch` for typed async data and shared cache behavior
- SSR and hydration only when request-time HTML or resume semantics matter

## Stability Note

For the getting-started path, the default public surface is mostly `Stable` as summarized in [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md).

Safe default starter choices:

- `ui.Render`, `ui.CreateElement`, `ui.UseState`, `ui.UseEffect`, `ui.UseEvent`
- `html` typed builders
- the documented `gwc` workflow

Surfaces you should usually delay until you need them:

- `Experimental` scheduling helpers such as `ui.UseTransition` and `ui.UseDeferredValue`
- advanced cached-resource flows
- advanced router lifecycle features if the app is still single-screen

## The Shortest Path

From the repo root, the shortest practical path is:

1. confirm the toolchain with `go run ./tools/gwc doctor`
2. browse the examples with `go run ./tools/gwc examples`
3. start from the smallest app shape that fits the work
4. use `go run ./tools/gwc dev -app .\main.go` for the inner loop
5. use `go run ./tools/gwc build`, `test`, and `verify` once the app stabilizes

If you are evaluating from the examples catalog instead of building a new app immediately, start with [examples/public/counter](../../examples/public/counter), [examples/public/ui-render](../../examples/public/ui-render), [examples/public/use-state](../../examples/public/use-state), and [examples/public/use-effect](../../examples/public/use-effect).

For a generated app, run `go run ./tools/gwc start` and choose the smallest
starter that matches the product shape. The maintained starter gallery is
[docs/STARTERS.md](../STARTERS.md); it covers minimal client apps, routed SPAs,
SSR apps, dashboards, marketing sites, content blogs, authed app shells, and the
broader reference app.

## Minimal Example

The smallest useful GWC app is one browser-mounted component plus one stateful event path.

`main.go`:

```go
package main

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderHelloApp renders the smallest useful interactive app for a first-run check.
func renderHelloApp() ui.Node {
	storeCount := ui.UseState(0)
	handleUserIncrement := ui.UseEvent(func() {
		storeCount.Update(func(getPreviousCount int) int {
			return getPreviousCount + 1
		})
	})

	return Main(
		Class("mx-auto max-w-xl space-y-4 p-6"),
		H1("Hello from GoWebComponents"),
		P(Textf("Current count: %d", storeCount.Get())),
		Button(
			Type("button"),
			OnClick(handleUserIncrement),
			"Increment",
		),
	)
}

// main mounts the app into the browser DOM and keeps the Go program alive.
func main() {
	ui.Run("#app", renderHelloApp)
}
```

`ui.Run(selector, component, props...)` is the one-line browser entrypoint: it
builds the component with `ui.CreateElement`, mounts it at the selector via
`ui.Render`, then blocks forever (the same keep-alive as `utils.WaitForever`) so
the js/wasm program stays alive to handle events. It never returns.

Reach for the explicit pair instead when `main` must do work after mounting, or
for SSR/native/test entrypoints where you do not want to block:

```go
func main() {
	ui.Render(ui.CreateElement(renderHelloApp, nil), "#app")
	// ... start background work, register globals, etc. ...
	utils.WaitForever()
}
```

Run it with:

```powershell
go run ./tools/gwc dev -app .\main.go
```

Why this is the right minimal path:

- it proves the public imports are correct
- it exercises `ui.Render`, `ui.CreateElement`, `ui.UseState`, and `ui.UseEvent`
- it avoids bringing in routing, SSR, or shared state before the app needs them

## Production-Shaped Starter

Once the first render works, the next useful step is a slightly more structured app with named components, typed props, and a clearer local-state shape.

`main.go`:

```go
package main

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type counterPanelProps struct {
	Title         string
	CurrentCount  int
	PreviousLabel string
	HandleUserAdd func()
}

// renderCounterPanel renders one focused feature panel so the root stays simple.
func renderCounterPanel(getProps counterPanelProps) ui.Node {
	return Section(
		Class("space-y-3 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"),
		H2(Class("text-xl font-semibold"), getProps.Title),
		P(Textf("Current count: %d", getProps.CurrentCount)),
		P(Textf("Previous count: %s", getProps.PreviousLabel)),
		Button(
			Type("button"),
			OnClick(getProps.HandleUserAdd),
			Class("rounded-xl bg-slate-900 px-4 py-2 text-white"),
			"Increment",
		),
	)
}

// renderStarterApp shows the first production-shaped split between root state and child presentation.
func renderStarterApp() ui.Node {
	storeCount := ui.UseState(0)
	storeTitle := ui.UseState("Starter App")
	getPreviousCount := ui.UsePrevious(storeCount.Get())

	handleUserAdd := ui.UseEvent(func() {
		storeCount.Update(func(getCurrentCount int) int {
			return getCurrentCount + 1
		})
	})

	getPreviousLabel := "none yet"
	if getPreviousCount.Ok() {
		// Keep the label derivation in render-time Go so the component tree stays deterministic.
		getPreviousLabel = fmt.Sprintf("%d", getPreviousCount.Get())
	}

	return Main(
		Class("mx-auto max-w-2xl space-y-6 p-6"),
		H1(Class("text-3xl font-black"), storeTitle.Get()),
		ui.CreateElement(renderCounterPanel, counterPanelProps{
			Title:         "Counter",
			CurrentCount:  storeCount.Get(),
			PreviousLabel: getPreviousLabel,
			HandleUserAdd: handleUserAdd,
		}),
	)
}

// main mounts the starter app into the browser DOM.
func main() {
	ui.Run("#app", renderStarterApp)
}
```

Why this is a better real starter than the minimal example:

- the root component owns local state and passes only what the child needs
- the feature component stays focused on presentation
- the state and event flow is still simple enough that you do not need `router`, `state`, or `fetch` yet

## First Scale-Up Step

When the app stops being a single file, scale by ownership instead of by dumping everything into one `main.go`.

One practical early layout is:

```text
my-app/
|-- cmd/
|   \-- web/
|       \-- main.go
|-- internal/
|   |-- app/
|   |   \-- renderRootApp.go
|   \-- feature/
|       \-- counter/
|           \-- renderCounterFeature.go
|-- static/
|   \-- index.html
\-- go.mod
```

`cmd/web/main.go`:

```go
package main

import (
	"my-app/internal/app"

	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// main mounts the application entrypoint owned by the app package.
func main() {
	ui.Render(ui.CreateElement(app.RenderRootApp, nil), "#app")
	utils.WaitForever()
}
```

`internal/app/renderRootApp.go`:

```go
package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"

	"my-app/internal/feature/counter"
)

// RenderRootApp composes feature packages without turning cmd/web into an application dump.
func RenderRootApp() ui.Node {
	return Main(
		Class("mx-auto max-w-4xl space-y-6 p-6"),
		H1("My App"),
		ui.CreateElement(counter.RenderCounterFeature, nil),
	)
}
```

Why this scales better:

- `cmd/web` stays a bootstrap boundary, not a feature bucket
- `internal/app` owns global composition
- feature packages own local components before you introduce shared state, routing, or async data

The next growth steps should be deliberate:

- add `router` when the app really becomes multi-screen
- add `state` when unrelated branches need the same source of truth
- add `fetch` when async data becomes more important than local demo state

## Starter Surface Reference

Use this table as the default starter map.

| Surface | Stability | Start with it when | Delay it until |
| --- | --- | --- | --- |
| `ui.Render` and `ui.CreateElement` | `Stable` | you need the first browser mount | never; this is the base render path |
| `ui.UseState` and `ui.UseEvent` | `Stable` | one component owns the state and event flow | shared ownership or cross-route coordination appears |
| `html` or `html/shorthand` | `Stable` | you want typed or sugar-based DOM authoring | never; pick one style and stay consistent |
| `gwc doctor`, `dev`, `build` | documented workflow | you want the normal inner loop | you have a very specific lower-level compiler reason |
| `router` | `Stable` plus some `Experimental` advanced features | the app has multiple screens, params, or route-owned data | the app is still one screen |
| `state` | `Stable` | multiple unrelated consumers need shared state | local hooks and reducers still fit |
| `fetch.UseResource[T]` | `Stable` | typed async data belongs to one panel or feature | the app has no real async read ownership yet |
| SSR and hydration | `Stable` core entrypoints with deeper operational rules | request-time HTML and resume behavior matter | the app can stay client-rendered |

The generated starter presets map these surfaces into copyable app shapes. Use
`minimal-client` first for evaluation, `routed-spa` when navigation is already
real, `dashboard-app` for internal tools, `marketing-site` or `content-blog` for
content-first SSR surfaces, `authed-app-shell` for SaaS-style app shells, and
`reference-app` when you want the broadest scaffolded example.

## Design Notes And Boundaries

Keep these rules in mind from the beginning:

- the recommended application surface is the documented public packages, not `internal/`
- the framework is incremental; it does not require routing, SSR, or companion packages on day one
- the `gwc` runner is the documented workflow entrypoint and should be preferred over ad hoc local scripts
- examples are the fastest proof points, but they are not a substitute for choosing an app shape intentionally
- starter code should stay small until the app has a real reason to widen its surface area

## Common Failure Modes

- importing repo internals instead of the public package surface
- skipping `gwc doctor` and chasing avoidable toolchain or `wasm_exec.js` issues later
- treating every new app as an SSR app before the product requires request-time HTML
- adding `state` atoms for everything instead of starting with local hooks
- copying patterns from large integrated examples before understanding the small starter path
- using browser-only APIs in code paths that also run under native `go test`

## Validation

Use the smallest commands that prove the starter path is working.

Environment and workflow:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc examples
```

Standalone app inner loop:

```powershell
go run ./tools/gwc dev -app .\main.go
go run ./tools/gwc build -app .\main.go -profile development
go run ./tools/gwc verify -app .\main.go -root .
```

If you are validating from the repo examples first:

```powershell
go run ./tools/gwc dev -app .\examples\public\counter\main.go
go run ./tools/gwc verify -app .\examples\public\counter\main.go -root .\examples\public\counter
```

## Topic Pagination
Topic 1 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [02 GWC Workflows](02-gwc-workflows.md)
