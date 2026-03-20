# Devtools Package

The `devtools` package provides a lightweight in-browser inspection surface for GoWebComponents applications.

It focuses on the minimum useful debugging view:

- component tree visibility
- hook state inspection
- current route inspection
- shared cache inspection
- recent framework log buffering
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

### Export And Compare Snapshots

Use the helper functions when you want to save or diff inspection state during
hot-reload or optimization work:

```go
before := devtools.SnapshotNow()
// ... change something ...
after := devtools.SnapshotNow()

payload, _ := devtools.ExportSnapshotJSON(after)
comparison, _ := devtools.CompareSnapshots(before, after)
fmt.Println(string(payload))
fmt.Println(comparison.ChangedSections)
```

## What the Panel Shows

- Current route path, query params, route params, and route loader pending state
- Shared cache entries including key, ready or stale state, subscriber count, resume policy, and last error
- Runtime totals for fibers, dirty nodes, hook entries, effects, and recent timing counters
- Hot branches ranked by subtree commit/effect/cleanup cost
- Recent framework logs buffered in memory, including router navigation, route-loader, cache invalidation, and mutation replay lifecycle events
- A committed component tree view with hook summaries per node
- Structured diagnostics reported by the runtime and router, now including classification metadata for correctness, performance, recovered, unsupported-but-recovered, and informational notices
- Recovered error-boundary diagnostics now include subtree path and component-stack context when the runtime can attribute the failure

## Diagnostics Included Today

- invalid hook usage outside component context
- missing `RenderTo(...)` container selectors
- duplicate route registrations
- invalid route component registration
- recovered boundary failures with path and component-stack context

## Notes

- The panel is intended for development builds and is designed to stay small and embeddable.
- The first version is in-browser by design. It does not require a browser extension or VS Code integration.
- Runtime timings now include subtree-level commit/effect/cleanup hotspots so expensive branches can be located from the panel.

## Example

A smaller standalone example is available at [examples/16-devtools/README.md](examples/16-devtools/README.md).
