# Accessibility Guidance

This document defines the accessibility support that GoWebComponents currently ships today.

Use it when you need to understand the current first-class accessibility primitives, where application code still owns semantics, and how to compose accessible forms, routed screens, overlays, and keyboard-driven widgets without dropping into ad hoc DOM wiring.

GoWebComponents is still intentionally narrower than a full accessibility framework. The difference from the earlier baseline, though, is that the framework now ships public primitives for focus management, composite widget keyboard flow, live-region announcements, and modal-style overlay behavior.

## At A Glance

The current first-class accessibility surface covers:

- semantic host elements through the `html` package
- typed accessibility-related props through `html.Props`
- stable generated ids through `ui.UseId()`
- explicit focus movement through `ui.UseFocusManager()`
- modal focus containment through `ui.UseFocusTrap(...)`
- composite keyboard navigation through `ui.UseCompositeNavigation(...)`
- polite and assertive announcements through `ui.UseAnnouncer()`
- accessible modal overlays through `ui.AccessibleOverlay(...)`
- normal SSR and hydration behavior because the output stays ordinary HTML

That means the framework now covers the common interaction building blocks that previously required bespoke application code.

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

Minimal example:

```go
func EmailField() ui.Node {
	fieldID := ui.UseId()
	hintID := fieldID + "-hint"
	errorID := fieldID + "-error"
	value := ui.UseState("")
	invalid := strings.TrimSpace(value.Get()) == ""

	return html.Div(html.Props{},
		html.Label(html.Props{For: fieldID}, ui.Text("Email")),
		html.Input(html.Props{
			ID:    fieldID,
			Type:  "email",
			Value: value.Get(),
			Aria: map[string]string{
				"describedby": hintID + " " + errorID,
				"invalid":     strconv.FormatBool(invalid),
			},
			OnInput: ui.UseEvent(func(event ui.InputEvent) {
				value.Set(event.Value())
			}),
		}),
		html.P(html.Props{ID: hintID}, ui.Text("Use your work email address.")),
		ui.If(invalid,
			html.P(html.Props{ID: errorID, Aria: map[string]string{"live": "polite"}}, ui.Text("Email is required.")),
		),
	)
}
```

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

Minimal pattern:

```go
func RouteShell() ui.Node {
	announce := ui.UseAnnouncer()
	focus := ui.UseFocusManager()
	currentPath := router.InspectCurrentRoute().Path

	ui.UseEffect(func() func() {
		announce.Polite("Navigated to " + currentPath)
		focus.FocusSelector("#route-page-heading")
		return nil
	}, currentPath)

	return html.Main(html.Props{},
		html.H1(html.Props{ID: "route-page-heading", TabIndex: -1}, ui.Text("Current page")),
		router.Outlet(),
	)
}
```

Example reference:

- `examples/80-routed-accessibility` shows routed page announcements and heading focus after navigation.

## Async UI

Hooks such as `ui.UseTransition`, async boundaries, fetch helpers, and form helpers can represent loading and pending state, but accessibility semantics for those states are still the application's responsibility.

Current guidance:

- Reflect pending work in visible text, not just spinners.
- Add `aria-busy`, status text, or `ui.UseAnnouncer()` updates when async work needs announcement.
- Keep controls disabled only when the disabled state is still understandable from surrounding text or status messaging.

Minimal pattern:

```go
func PendingRegion() ui.Node {
	announce := ui.UseAnnouncer()
	transition := ui.UseTransition()
	status := "Results ready."
	if transition.Pending() {
		status = "Loading results..."
	}

	ui.UseEffect(func() func() {
		if transition.Pending() {
			announce.Polite("Loading updated results")
		}
		return nil
	}, transition.Pending())

	return html.Section(html.Props{
		Aria: map[string]string{"busy": strconv.FormatBool(transition.Pending())},
	},
		html.P(html.Props{}, ui.Text(status)),
	)
}
```

The framework includes a reusable live-region primitive, but applications still decide which async transitions deserve spoken feedback.

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

Minimal pattern:

```go
func ConfirmDialog(open bool, dismiss func()) ui.Node {
	return ui.AccessibleOverlay(ui.AccessibleOverlayProps{
		Open:          open,
		LabelledBy:    "confirm-title",
		DescribedBy:   "confirm-body",
		CloseOnEscape: true,
		OnDismiss:     dismiss,
		Child: html.Div(html.Props{},
			html.H2(html.Props{ID: "confirm-title"}, ui.Text("Delete item")),
			html.P(html.Props{ID: "confirm-body"}, ui.Text("This action cannot be undone.")),
		),
	})
}
```

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