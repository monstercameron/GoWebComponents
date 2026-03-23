# Error Boundaries

This page records the current `ui.ErrorBoundary` contract for route composition, SSR, and hydration.

It is intentionally about the behavior that ships today, not a future router-specific abstraction.

## At A Glance

`ui.ErrorBoundary` is a subtree recovery primitive, not a router-managed failure system.

The practical rules are:

- boundary placement in the UI tree decides what can fail independently
- server rendering can recover through the nearest mounted boundary
- hydration mismatches do not render through boundary fallback UI
- client render, effect, or event panics still recover through normal boundary behavior

This means boundaries are for code failure isolation, not for masking hydration correctness problems.

## Quick Placement Guide

Use this rule of thumb:

- wrap `router.Outlet()` when the shell should survive leaf-route failures
- wrap the layout plus outlet together when shell and leaf should recover as one unit
- use separate sibling boundaries when neighboring route branches should fail independently
- align `ResetKeys` with route identity when navigation should intentionally clear a recovered error state

If you cannot explain what subtree a boundary owns, it is probably placed too high or too low.

## Current Shipped Surface

The public boundary surface today is still just:

- `ui.ErrorBoundary`
- `ui.ErrorBoundaryProps`
- `Fallback`
- `ErrorFallback`
- `OnError`
- `ResetKeys`

There is no router-only boundary type and no automatic route-boundary injection layer.

## Nested Routes And Layouts

`ui.ErrorBoundary` composes like any other subtree wrapper.

That means:

- a boundary around `router.Outlet()` only isolates the leaf route subtree rendered through that outlet
- a boundary around a layout shell and its outlet isolates both the layout render path and its current leaf
- sibling layouts or sibling route branches need their own boundaries if they should fail independently
- the router does not currently decide whether "route boundaries" mean leaf-only or shell-plus-leaf; placement in the UI tree is the authority

Recommended placement rules:

- wrap `router.Outlet()` when you want a persistent shell to survive route-leaf failures
- wrap a larger layout subtree when the shell and leaf must recover together
- keep reset keys aligned with route identity when navigation should clear the recovered state intentionally

This is the current answer to nested route composition: boundaries follow ordinary subtree ownership, not special router metadata.

## SSR Behavior

Server rendering already honors error boundaries.

If a panic occurs while rendering inside a boundary during `ui.RenderToString(...)`:

- the nearest boundary runs `OnError` if provided
- `ErrorFallback` renders fallback HTML when provided
- plain `Fallback` also works
- the rest of the request can continue rendering instead of failing the whole response

If no boundary catches the panic, normal server-render failure rules still apply.

There is no server-only "client render this subtree later" escape hatch attached to error boundaries today.

## Hydration Behavior

Hydration does not give error boundaries a separate protocol.

Current behavior:

- hydration reuses matching DOM where possible
- if hydration cannot continue safely for a subtree, the runtime discards that subtree's server DOM and falls back to client rendering for that subtree
- those hydration fallbacks are reported through runtime diagnostics
- if the client render or later effect or event work then panics under a mounted boundary, normal boundary recovery applies on the client

In other words:

- hydration mismatches are not themselves rendered through `ErrorFallback`
- render or effect panics during the client pass still are
- boundaries are recovery for code failures, while hydration fallback is recovery for DOM continuity failures

## Recovery Checklist

Before relying on a boundary in production, verify these questions:

- does the chosen boundary align with a real ownership boundary in the UI tree?
- should route navigation reset the recovered state, and if so are `ResetKeys` wired accordingly?
- does `OnError` record enough application context for debugging without exposing raw panic noise to users?
- is hydration-safe rendering handled separately from boundary fallback behavior?

If the answer to the last question is no, the boundary is being asked to solve the wrong problem.

## Practical Guidance

- use route-leaf boundaries when data-rich routed content can fail independently from the surrounding shell
- use broader layout boundaries when the shell depends on the same risky render path
- do not rely on boundaries as a substitute for explicit hydration-safe rendering; hydration mismatches still degrade through subtree client rerendering instead of boundary fallback UI
- wire `OnError` when you need application logging, because recovered server or client failures otherwise only surface through the runtime diagnostics path
