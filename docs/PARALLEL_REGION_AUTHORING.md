# Parallel Region Authoring

Last updated: 2026-04-05

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
- worker-attached runtime2 patch consume and commit after local-first mount
- bounded local-first click-slot bridging through `html.OnClickParallel(...)`

Not shipped today:

- worker-owned shell ownership as a public default runtime path
- generic event-slot transport beyond the bounded public click-slot bridge
- hook or effect execution inside worker-rendered output

Rule: author regions today as worker-safe display surfaces even though the public shell still renders locally first and browser builds only attach worker commit after that owner render. Click-slot bridging is still local-first: the browser event runs the local handler first and then forwards one bounded semantic click event into runtime2.

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

Bounded click-slot helper:

```go
func html.OnClickParallel(slotID string, callback interface{}) html.PropOption
```

Use it when one local-first region node needs to keep its local click handler and also forward a semantic click event into runtime2. The internal click-slot marker is stripped before DOM commit, so it does not leak into rendered output.

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

## Transition Semantics

`ui.ParallelRegion(...)` already distinguishes between urgent owner rerenders and transition-wrapped owner rerenders.

Current rule:

- ordinary prop or source changes dispatch runtime2 updates immediately
- owner updates wrapped in `ui.StartTransition(...)` or `ui.UseTransition().Start(...)` are published as deferred runtime2 snapshot work
- a later urgent rerender can supersede the deferred snapshot before it is dispatched

That means transitions currently affect runtime2 dispatch priority, not the local-first shell contract. The region shell still starts in the owner component today, while the runtime2 side receives deferred versus urgent update classification and can attach for worker patch commit after the local render.

Example:

```go
func renderDashboard() ui.Node {
    count := state.UseAtom("dashboard.count", 0)
    transition := ui.UseTransition()
    sourceIDs, err := ui.BuildParallelRegionSourceIDs(count)
    if err != nil {
        panic(err)
    }

    incrementDeferred := ui.UseEvent(func() {
        transition.Start(func() {
            count.Update(func(previous int) int {
                return previous + 1
            })
        })
    })

    return html.Div(
        html.Button(html.Props{OnClick: incrementDeferred}, html.Text("Increment In Transition")),
        ui.ParallelRegion(ui.ParallelRegionSpec[dashboardSummaryProps]{
            RendererID:       "dashboard.summary",
            RegionInstanceID: "dashboard.summary.primary",
            Props: dashboardSummaryProps{
                Count:  count.Get(),
                Status: "Healthy",
            },
            SourceIDs: sourceIDs,
        }),
    )
}
```

Use transitions when:

- the owner update is intentionally non-urgent
- you are comfortable with runtime2 dispatch being deferred
- a later urgent owner update should be allowed to supersede the deferred snapshot

Do not assume transitions mean:

- worker-owned DOM commit is already the default public path
- deferred snapshots bypass validation or epoch or version rules
- owner-local rendering disappears while the runtime2 path catches up

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
- publishes browser-side rerender snapshots into runtime2 with monotonic input versions for dispatch, diagnostics, and transport selection
- stays deterministic and local-only on native builds

That means you can author the public shape now with a precise boundary: local shell ownership is current default behavior, and browser builds can still attach runtime2 for worker snapshot and patch commit without making worker-owned shell ownership the default public runtime path.
