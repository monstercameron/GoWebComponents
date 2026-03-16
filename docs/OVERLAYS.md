# Overlay Layering And Portal Coordination

This document describes the current first-class overlay model for GoWebComponents.

## Current Model

Use `ui.Overlay` when a surface must participate in shared overlay stack behavior instead of letting each dialog, popover, tooltip, or menu invent its own z-index and dismissal policy.

The stack manager currently owns:

- registration order for active overlays through `ui.UseOverlayStack`
- derived z-index allocation for backdrop and surface layers
- escape-key routing to the topmost eligible layer
- outside-click routing to the topmost eligible layer
- focus-trap ownership for the topmost focus-trapping layer
- nested scroll-lock counting across multiple modal layers
- nested background inert and `aria-hidden` counting by app-root selector

The stack manager does not try to own anchor measurement or placement math. Applications still decide where floating surfaces should sit and when placement must be recomputed.

## Public APIs

- `ui.Overlay`: stack-aware surface primitive that can render inline or through `ui.PortalTarget`
- `ui.UseOverlayStack`: low-level registration hook for components that need stack state without using the full `ui.Overlay` renderer
- `ui.AccessibleOverlay`: modal-focused convenience wrapper now implemented on top of `ui.Overlay`

`ui.Overlay` is the right starting point for:

- dialogs and sheets that need modal focus coordination
- anchored popovers and menus that should dismiss above a parent dialog without destabilizing it
- tooltips or transient helper surfaces that should stack above a parent menu or popover
- overlay families that render into different portal roots but still need one shared stack order

## Layering Rules

The current stack model is intentionally simple:

1. each open overlay registers in activation order
2. each layer receives a derived backdrop and surface z-index from `BaseZIndex + depth * 2`
3. the topmost layer that opted into a behavior owns that behavior

In practice that means:

- the last opened escape-dismissible layer handles Escape first
- the last opened outside-dismissible layer handles outside dismissal first
- the last opened focus-trapping layer owns tab-loop enforcement
- modal side effects such as scroll lock and app-root inerting compose by reference count instead of replacing each other

This avoids the common failure mode where a child popover or nested dialog closes the parent because both layers attached ad hoc document listeners.

## Nested Dismissal Guidance

For nested overlays:

- let the child layer register `CloseOnEscape` or `CloseOnOutsideClick` if it should dismiss before its parent
- keep parent overlays mounted while the child closes so focus can restore into the parent surface instead of falling back to the page
- reserve modal behavior for layers that truly need background suppression; non-modal child layers can still stack above a modal parent without taking over scroll lock or inert ownership

The recommended pattern is a modal parent dialog plus non-modal popovers or menus inside it, or a modal child dialog when the task requires a full secondary confirmation step.

## Focus And Accessibility Coordination

`ui.Overlay` coordinates focus in two distinct ways:

- on open, the layer can remember the previously focused element and move focus to an initial target, fallback target, first focusable element, or the surface itself
- while active, the topmost focus-trapping layer enforces the tab loop

When a nested modal closes, focus restores to the element that was active in the parent surface before the child opened. The parent trap then resumes without forcibly refocusing the entire parent dialog.

For semantics:

- use `Kind`, `Role`, `LabelledBy`, and `DescribedBy` to keep the surface relationships explicit
- use `Modal`, `LockScroll`, and `BackgroundInert` together for true dialog or sheet flows
- use `Role: "menu"` or `Role: "tooltip"` when building non-dialog surfaces so assistive technology gets the correct contract

## Anchored Positioning Guidance

The framework intentionally separates stack ownership from placement ownership.

Applications should own:

- anchor measurement with `getBoundingClientRect`
- collision handling against the viewport or scroll container
- recomputing placement after resize, scroll, zoom, or content changes
- deciding whether the overlay should flip, slide, or resize when it would overflow

The framework should own:

- layer ordering
- dismissal routing
- focus and modal coordination
- portal targeting

Recommended practice:

1. keep one component or hook responsible for measurement and placement state
2. pass the resulting classes or style map into `ui.Overlay`
3. set `AnchorSelector` and `Positioning` for diagnostics and documentation clarity
4. avoid mixing ad hoc `z-index` classes with manager-owned layering unless you are deliberately overriding the stack

## Examples

See the current examples for end-to-end reference:

- `examples/81-overlay-stack`: nested dialog plus popover dismissal and focus routing
- `examples/82-overlay-anchor`: menu plus tooltip layering across selector and explicit portal roots