# Parallel Region Authoring

Last updated: 2026-03-27

This guide explains the current public `ui.ParallelRegion(...)` authoring surface.

Use it when you want:

- the smallest correct registration pattern
- the current first-slice rules for region shape
- guidance on when to use `ui.ParallelRegion(...)` versus `ui.ReactiveRegion(...)`

## Status

Shipped today:

- `ui.RegisterParallelRegion(...)`
- `ui.ParallelRegion(...)`
- `ui.BuildParallelRegionSourceIDs(...)`
- local-first shell rendering with stable runtime2 shell markers

Not shipped today:

- worker-owned patch commit as a public default runtime path
- interactive event-slot transport
- hook or effect execution inside worker-rendered output

Rule: author regions today as worker-safe display surfaces even though the public shell still renders locally first.

## Public API

```go
type ParallelRegionSpec[T any] struct {
    RendererID       string
    RegionInstanceID string
    Props            T
    SourceIDs        []string
}

func RegisterParallelRegion[T any](rendererID string, render func(T) ui.Node) error

func ParallelRegion[T any](spec ParallelRegionSpec[T]) ui.Node

func BuildParallelRegionSourceIDs(sources ...ui.ReactiveSource) ([]string, error)
```

## Recommended Pattern

1. Register one stable renderer ID once during startup.
2. Keep region props fully serializable.
3. Bind explicit shared sources with `ui.BuildParallelRegionSourceIDs(...)`.
4. Give each mounted instance one stable `RegionInstanceID`.
5. Keep the renderer output display-only.

Example:

```go
type dashboardSummaryProps struct {
    Count  int
    Status string
}

func registerDashboardSummaryRenderer() {
    if err := ui.RegisterParallelRegion("dashboard.summary", renderDashboardSummary); err != nil {
        panic(err)
    }
}

func renderDashboardSummary(props dashboardSummaryProps) ui.Node {
    return html.Div(
        html.Props{Class: "rounded-3xl border border-cyan-300/20 bg-cyan-400/10 p-6"},
        html.P(html.Props{Class: "text-xs uppercase tracking-[0.24em] text-cyan-100"}, html.Text("Parallel Region")),
        html.Div(html.Props{Class: "mt-4 text-5xl font-black text-white font-mono"}, html.Textf("%d", props.Count)),
        html.P(html.Props{Class: "mt-3 text-sm text-cyan-50/90"}, html.Text(props.Status)),
    )
}

func renderDashboard() ui.Node {
    count := state.UseAtom("dashboard.count", 0)
    sourceIDs, err := ui.BuildParallelRegionSourceIDs(count)
    if err != nil {
        panic(err)
    }
    return ui.ParallelRegion(ui.ParallelRegionSpec[dashboardSummaryProps]{
        RendererID:       "dashboard.summary",
        RegionInstanceID: "dashboard.summary.primary",
        Props: dashboardSummaryProps{
            Count:  count.Get(),
            Status: "Healthy",
        },
        SourceIDs: sourceIDs,
    })
}
```

## First-Slice Safe Region Shapes

Keep public renderers inside these boundaries:

- display-oriented host and text output
- serializable props only
- stable instance IDs
- explicit shared-source IDs

Avoid these patterns in registered region renderers:

- refs
- portals
- direct DOM handles or interop objects
- event closure props
- assuming worker-side hooks or effects exist

The runtime2 validation layer already rejects ref-like markers, direct DOM markers, and event-closure props in region inputs.

## Choosing Between Parallel Region And Reactive Region

Use `ui.ParallelRegion(...)` when:

- you want an explicit renderer ID and instance ID
- you are authoring toward the future worker-backed render path
- you want a stable shell marker around the region boundary

Use `ui.ReactiveRegion(...)` when:

- you only need narrow subscribed rerender behavior today
- you do not need explicit renderer registration
- you are not shaping the subtree as a future worker-safe display surface

Short version:

- `ui.ReactiveRegion(...)` is the current narrow-update primitive.
- `ui.ParallelRegion(...)` is the explicit public shell for the future worker-backed path.

## Source Binding Guidance

Prefer:

```go
sourceIDs, err := ui.BuildParallelRegionSourceIDs(atomA, derivedB)
```

This keeps source IDs validated and deduplicated before they reach runtime2.

Do not manually stitch together string IDs unless the source does not already expose `ReactiveRegionSourceIDs()`.

## Identity Guidance

`RendererID`:

- names the implementation
- should be stable across app restarts
- should not encode per-instance data

`RegionInstanceID`:

- names one mounted region instance
- should stay stable across ordinary rerenders
- should change when the owning structure is intentionally remounted

## Current Runtime Behavior

Today `ui.ParallelRegion(...)`:

- renders a local-first shell immediately
- adds the runtime2 shell marker attribute
- mounts runtime2 host-lifecycle state on browser builds
- stays deterministic and local-only on native builds

That means you can start authoring the public shape now without waiting for the full worker commit path to be enabled.
