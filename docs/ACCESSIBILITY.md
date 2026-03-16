# Accessibility Guidance

This document defines the accessibility support that GoWebComponents currently ships today.

It is intentionally narrower than a full accessibility framework, but it now includes first-class public primitives for focus management, composite widget keyboard flow, live-region announcements, and modal-style overlay behavior.

## Current Baseline

The supported baseline for public applications is:

- Semantic host elements through the `html` package, including `Label`, `Input`, `Textarea`, `Select`, `Button`, `Dialog`, `Nav`, `Main`, `Section`, and other standard HTML tags.
- Typed accessibility-related props through `html.Props`, including `For`, `Role`, `Aria`, `AutoFocus`, focus handlers, and `Raw` for attributes such as `tabIndex` when no dedicated field exists.
- Stable generated IDs through `ui.UseId()` for wiring labels, helper text, and described-by relationships without manual string coordination.
- `ui.UseFocusManager()` for explicit focus movement, trigger restoration, and focusing the first invalid field.
- `ui.UseFocusTrap(...)` for keeping keyboard focus inside modal-style overlays and restoring the previous active element on cleanup.
- `ui.UseCompositeNavigation(...)` for roving tabindex, arrow-key movement, Home or End behavior, typeahead, and active-descendant driven composites.
- `ui.UseAnnouncer()` for polite and assertive live-region announcements that render as ordinary HTML.
- `ui.AccessibleOverlay(...)` for portal-backed dialog semantics with focus trapping, escape dismissal, background inerting, and scroll locking.
- Normal SSR and hydration behavior for semantic markup because the rendered output is ordinary HTML, not an opaque accessibility abstraction.

This means GoWebComponents now covers the common interaction building blocks that previously required bespoke application code.

## Current Scope Boundary

The accessibility surface still has clear boundaries:

- The current overlay primitive is modal-first. It does not yet define a full stacked overlay manager for nested dialogs, popovers, menus, and sheets.
- Composite navigation focuses the keyboard model for one collection at a time. Higher-level widget authoring patterns such as tree views, grid navigation, or full combobox behaviors still need application composition.
- Live-region helpers are local to the component tree that owns them. There is no global announcement bus.

Those remaining gaps are narrower than the previous baseline-only support and no longer block accessible forms, route announcements, or modal dialogs.

## Forms

For forms, the current recommended pattern is:

- Use `html.Label(html.Props{For: fieldID}, ...)` and `html.Input(html.Props{ID: fieldID, ...})` or the corresponding `Textarea` or `Select` builder.
- Generate repeated or component-local IDs with `ui.UseId()` instead of hand-managed string literals.
- Attach `aria-describedby`, `aria-invalid`, `aria-errormessage`, and similar attributes through `html.Props{Aria: map[string]string{...}}`.
- Use `Raw` for attributes that are valid but not yet modeled directly, such as `tabIndex`.
- Use `ui.UseAnnouncer()` to speak validation results and `ui.UseFocusManager().FocusFirstError(...)` to move focus to the first invalid field after validation.

Example references in this repository:

- `examples/35-use-id` shows stable generated IDs for form labeling.
- `examples/53-html-forms` shows typed form controls with explicit label/input wiring.
- `examples/79-form-accessibility` shows validation announcements, `aria-invalid`, `aria-describedby`, pending-state semantics, and shared focus-to-error behavior.

The framework does not automatically infer validation rules, but it now ships the primitives needed to announce and focus validation results consistently.

## Routed Apps

Router metadata support currently manages document title, description, and canonical URL. That helps document identity and SEO, but it is not a route-announcement feature for assistive technology.

For routed applications today:

- Keep page structure semantic, including one clear page-level heading.
- Manage focus explicitly after route-driven UI transitions when the UX requires it, typically by focusing the new page heading.
- Announce route changes through `ui.UseAnnouncer()` from a stable layout or shell component.

Example reference:

- `examples/80-routed-accessibility` shows routed page announcements and heading focus after navigation.

## Async UI

Hooks such as `ui.UseTransition`, async boundaries, fetch helpers, and form helpers can represent loading and pending state, but accessibility semantics for those states are still the application's responsibility.

Current guidance:

- Reflect pending work in visible text, not just spinners.
- Add `aria-busy`, status text, or `ui.UseAnnouncer()` updates when async work needs announcement.
- Keep controls disabled only when the disabled state is still understandable from surrounding text or status messaging.

The framework now includes a reusable live-region primitive, but applications still decide which async transitions deserve spoken feedback.

## Overlays And Portals

`html.Dialog(...)`, `Role: "dialog"`, `ui.Portal(...)`, `ui.UseFocusTrap(...)`, and `ui.AccessibleOverlay(...)` let applications render semantic overlay markup and move it to a dedicated DOM mount.

The current overlay primitive covers the most common modal responsibilities directly:

- portal-backed rendering into a separate mount
- initial focus placement
- focus trapping and focus restoration
- escape-key dismissal
- optional backdrop click dismissal
- app-shell `aria-hidden` and inert behavior while a modal is open
- optional body scroll locking
- `aria-labelledby` and `aria-describedby` wiring through the overlay props

Applications still need to own:

- visual styling and layout
- non-modal stacked overlay coordination across several simultaneous surfaces
- trigger-specific policies such as nested popover dismissal ordering

Example references:

- `examples/77-accessible-overlay` shows the modal overlay primitive, focus trap, backdrop dismissal, inert background behavior, and trigger-focus restoration.
- `examples/20-portals` still shows lower-level portal rendering patterns when you do not want the higher-level modal contract.

## Composite Widgets

`ui.UseCompositeNavigation(...)` is the recommended starting point for tabs, listboxes, menus, and similar keyboard-driven composites.

It provides:

- roving tabindex for focusable item collections
- arrow-key movement in horizontal, vertical, or both-axis composites
- Home and End behavior
- simple typeahead matching by item text
- `aria-activedescendant` support through `ActiveDescendant()`

Example reference:

- `examples/78-composite-navigation` shows the same hook applied to both a tablist and a listbox.

## Testing Expectations

The current repo validates the baseline rather than claiming a complete accessibility system.

The most useful checks today are:

- Unit tests that verify typed props preserve `role`, `aria-*`, `htmlFor`, `autofocus`, and similar markup.
- Tests that verify `ui.UseId()` remains stable enough to wire related elements together.
- Unit tests that verify `ui.UseCompositeNavigation(...)` and `ui.UseAnnouncer()` preserve their public contracts.
- Browser tests for modal focus trap behavior, keyboard composite navigation, validation announcements, and routed page announcements.

The examples Playwright suite now includes focused coverage for the new accessibility examples in addition to the lower-level hook tests.