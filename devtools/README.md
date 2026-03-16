# Devtools Package

The `devtools` package provides a lightweight in-browser inspection surface for GoWebComponents applications.

It focuses on the minimum useful debugging view:

- component tree visibility
- hook state inspection
- current route inspection
- subtree-level profiling hotspots
- structured runtime diagnostics

## Core API

### Panel

Render an embeddable development overlay inside your app:

```go
import (
    "time"

    "github.com/monstercameron/GoWebComponents/devtools"
    "github.com/monstercameron/GoWebComponents/ui"
)

func App() ui.Node {
    return ui.Fragment(
        ui.CreateElement(MainUI),
        ui.CreateElement(devtools.Panel, devtools.PanelProps{
            Title:           "App Devtools",
            InitiallyOpen:   false,
            RefreshInterval: 750 * time.Millisecond,
            MaxDepth:        5,
        }),
    )
}
```

### SnapshotNow

Read the current inspection state programmatically:

```go
snapshot := devtools.SnapshotNow()
fmt.Println(snapshot.Route.Path)
fmt.Println(snapshot.Stats.TotalFibers)
fmt.Println(len(snapshot.Diagnostics))
```

### UseSnapshot

Subscribe to periodic inspection snapshots from inside a component:

```go
func InspectorSummary() ui.Node {
    snapshot := devtools.UseSnapshot(time.Second)

    return html.Div(html.Props{},
        html.P(html.Props{}, html.Text(snapshot.Route.Path)),
        html.P(html.Props{}, html.Text(fmt.Sprintf("Fibers: %d", snapshot.Stats.TotalFibers))),
    )
}
```

## What the Panel Shows

- Current route path, query params, route params, and route loader pending state
- Runtime totals for fibers, dirty nodes, hook entries, effects, and recent timing counters
- Hot branches ranked by subtree commit/effect/cleanup cost
- A committed component tree view with hook summaries per node
- Structured diagnostics reported by the runtime and router, including slow effect and cleanup paths

## Diagnostics Included Today

- invalid hook usage outside component context
- missing `RenderTo(...)` container selectors
- duplicate route registrations
- invalid route component registration

## Notes

- The panel is intended for development builds and is designed to stay small and embeddable.
- The first version is in-browser by design. It does not require a browser extension or VS Code integration.
- Runtime timings now include subtree-level commit/effect/cleanup hotspots so expensive branches can be located from the panel.

## Example

A smaller standalone example is available at [examples/16-devtools/README.md](examples/16-devtools/README.md).