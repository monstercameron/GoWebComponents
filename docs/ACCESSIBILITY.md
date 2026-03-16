# Accessibility Guidance

This document defines the accessibility support that GoWebComponents currently ships today.

It is intentionally narrower than a full accessibility framework. The current public surface gives you typed semantic HTML builders, stable generated IDs, typed events, and SSR-compatible markup output. It does not yet ship a first-class focus manager, live-region helper, keyboard-navigation toolkit, or overlay accessibility manager.

## Current Baseline

The supported baseline for public applications is:

- Semantic host elements through the `html` package, including `Label`, `Input`, `Textarea`, `Select`, `Button`, `Dialog`, `Nav`, `Main`, `Section`, and other standard HTML tags.
- Typed accessibility-related props through `html.Props`, including `For`, `Role`, `Aria`, `AutoFocus`, focus handlers, and `Raw` for attributes such as `tabIndex` when no dedicated field exists.
- Stable generated IDs through `ui.UseId()` for wiring labels, helper text, and described-by relationships without manual string coordination.
- Normal SSR and hydration behavior for semantic markup because the rendered output is ordinary HTML, not an opaque accessibility abstraction.

This means GoWebComponents can express accessible markup well, but it does not yet automate higher-level interaction behavior.

## Current Non-Goals

The following capabilities are not currently first-class framework features:

- Focus trapping, focus return, and route-transition focus restoration.
- Live-region announcement helpers for validation, route changes, loading states, or toasts.
- Reusable keyboard-navigation primitives for tabs, listboxes, menus, trees, or command palettes.
- Overlay coordination such as inert background behavior, escape routing across nested overlays, or managed aria relationships between triggers and floating surfaces.

Applications that need these behaviors must currently implement them explicitly in app code.

## Forms

For forms, the current recommended pattern is:

- Use `html.Label(html.Props{For: fieldID}, ...)` and `html.Input(html.Props{ID: fieldID, ...})` or the corresponding `Textarea` or `Select` builder.
- Generate repeated or component-local IDs with `ui.UseId()` instead of hand-managed string literals.
- Attach `aria-describedby`, `aria-invalid`, `aria-errormessage`, and similar attributes through `html.Props{Aria: map[string]string{...}}`.
- Use `Raw` for attributes that are valid but not yet modeled directly, such as `tabIndex`.

Example references in this repository:

- `examples/35-use-id` shows stable generated IDs for form labeling.
- `examples/53-html-forms` shows typed form controls with explicit label/input wiring.

The framework does not yet announce validation results automatically or move focus to the first invalid field. Those flows remain application-owned behavior.

## Routed Apps

Router metadata support currently manages document title, description, and canonical URL. That helps document identity and SEO, but it is not a route-announcement feature for assistive technology.

For routed applications today:

- Keep page structure semantic, including one clear page-level heading.
- Manage focus explicitly after route-driven UI transitions when the UX requires it.
- Add your own polite or assertive live region if route changes need spoken announcements.

GoWebComponents does not yet provide a built-in screen-reader announcement primitive for navigation changes.

## Async UI

Hooks such as `ui.UseTransition`, async boundaries, fetch helpers, and form helpers can represent loading and pending state, but accessibility semantics for those states are still the application's responsibility.

Current guidance:

- Reflect pending work in visible text, not just spinners.
- Add `aria-busy`, status text, or live-region markup explicitly when async updates need announcement.
- Keep controls disabled only when the disabled state is still understandable from surrounding text or status messaging.

The framework does not yet include a reusable announcement container or status primitive.

## Overlays And Portals

`html.Dialog(...)`, `Role: "dialog"`, and `ui.Portal(...)` let applications render semantic overlay markup and move it to a dedicated DOM mount.

That is only the structural baseline. Applications still need to own:

- Initial focus placement.
- Focus trapping.
- Focus return after close.
- Escape-key dismissal policy.
- Background inerting or scroll locking.
- `aria-labelledby` and `aria-describedby` relationships for the overlay surface.

The portal examples in this repository demonstrate rendering patterns, not a complete accessibility overlay contract.

## Testing Expectations

The current repo validates the baseline rather than claiming a complete accessibility system.

The most useful checks today are:

- Unit tests that verify typed props preserve `role`, `aria-*`, `htmlFor`, `autofocus`, and similar markup.
- Tests that verify `ui.UseId()` remains stable enough to wire related elements together.
- Browser tests in applications that exercise the app-specific behaviors GoWebComponents does not yet automate.

Future backlog items for focus management, live regions, keyboard navigation, and accessible overlays remain open until concrete public APIs exist.