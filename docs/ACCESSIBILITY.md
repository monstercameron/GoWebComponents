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
		router.GetOutlet(),
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

## Accessibility Audit Workflow

Treat accessibility review as a repeatable release gate instead of a one-time design pass.

Recommended loop for teams and CI:

1. Start with semantic structure review.
   Confirm the route or screen still has one clear page heading, real form labels, button or link semantics, and landmark structure before checking more subtle keyboard or live-region behavior.
2. Run focused keyboard-only review.
   Tab through the route, trigger overlays, exercise composite widgets, submit forms, and confirm that focus order, visible focus state, escape dismissal, and focus restoration remain coherent without a mouse.
3. Verify announcement paths deliberately.
   Check route-change announcements, validation errors, async pending or completion messages, and any success or recovery notices driven by `ui.UseAnnouncer()` or inline live regions.
4. Verify form and overlay state changes.
   Confirm `aria-invalid`, `aria-describedby`, `aria-busy`, dialog labeling, and inert-background behavior still match the actual UI state and do not drift during refactors.
5. Run automated browser coverage.
   Use the focused example Playwright specs or app-owned browser specs for the route families that carry accessibility risk, then promote repeated manual regressions into dedicated specs.
6. Keep one manual spot checklist in the repo.
   Use [examples/MANUAL_TESTING.md](../examples/MANUAL_TESTING.md) or an app-local equivalent for the cases that are still difficult to automate, and keep it updated when route or overlay behavior changes.

Recommended CI posture:

- keep at least one browser-runner lane for keyboard and focus behavior
- fail CI on deterministic accessibility regressions such as broken focus restore, missing route announcements, or inaccessible query paths
- reserve manual review for screen-reader nuance, visual focus polish, and complex announcement timing that the current automated suite cannot assert reliably yet

The practical goal is to catch drift in the same places accessibility usually regresses first: keyboard flow, route transitions, form validation, async status, and modal behavior.

Current repo anchors for that workflow:

- `examples/77-accessible-overlay` for modal focus trap, inert background, and dismiss behavior
- `examples/78-composite-navigation` for roving tabindex and keyboard movement
- `examples/79-form-accessibility` for validation announcements and focus-to-error behavior
- `examples/80-routed-accessibility` for route announcements and heading focus after navigation
- [examples/MANUAL_TESTING.md](../examples/MANUAL_TESTING.md) for the current manual spot-check list
- [TESTING.md](TESTING.md) and [WORKFLOWS.md](WORKFLOWS.md#test-a-component-or-app-flow) for the current automated validation lanes

## Regression Recipes

When a routed app or form-heavy app starts carrying real accessibility risk, split the regression strategy into two layers instead of expecting one test style to catch everything.

### 1. `js/wasm` fixture tests for deterministic state and markup contracts

Use `test/render` or `test/router` when the risk is local, deterministic, and easy to assert without a real browser:

- focus the first invalid field after submit
- preserve `aria-invalid`, `aria-describedby`, and related field state
- verify announcer text or status-region text changed in the rendered tree
- verify route params, query state, and rendered route content stay aligned when the shell changes
- verify composite-widget active item state and related markup after key handling logic runs

Prefer this layer when you want fast feedback on:

- validation feedback text
- focus-target selection logic
- route-shell state transitions
- keyboard-state bookkeeping inside one component tree

### 2. Playwright browser tests for focus movement and announcement behavior

Use Playwright when the assertion depends on browser focus, real keyboard events, overlays, or route transitions:

- tab into the route and confirm the expected control receives focus
- submit an invalid form and assert the first invalid field is focused
- open a dialog and assert focus trap, escape dismissal, and trigger-focus restoration
- navigate between routes and assert the live region updated plus the route heading became the active element
- exercise listbox or tablist arrow-key movement and confirm visible state plus ARIA state stay in sync

Prefer this layer when the risk depends on:

- `document.activeElement`
- real keyboard traversal
- modal trapping and dismissal
- route-change timing
- announcement timing that is observable only after browser event sequencing

### Routed app recipe

For route-heavy apps, keep one focused browser regression that covers:

1. navigate through at least two real routes
2. assert the new page heading is visible
3. assert the route announcer text changed
4. assert focus moved to the intended route heading or primary landmark
5. assert URL, route shell, and heading all agree

Current automated coverage lives in:

- `test/playwrightgo/examples/examples_suite_test.go` (`TestBrowserCompat`)

### Form-heavy app recipe

For form-heavy apps, keep one focused browser regression that covers:

1. submit the form empty or invalid
2. assert the first invalid field receives focus
3. assert inline validation feedback is visible
4. assert the assertive or polite announcement changed
5. submit the corrected form
6. assert pending and success announcements both fire in the intended order

Current automated coverage lives in:

- `test/playwrightgo/examples/examples_suite_test.go` (`TestLinks`)

### Overlay recipe

For modal or confirmation flows, keep one browser regression that covers:

1. open the overlay from a real trigger
2. assert focus moved into the dialog
3. assert keyboard traversal stays inside the overlay
4. dismiss with escape or the documented close action
5. assert focus returns to the trigger

Current automated coverage lives in:

- `test/playwrightgo/examples/examples_suite_test.go`

### Composite-widget recipe

For tabs, listboxes, menus, or similar composites, keep:

- a `js/wasm` test for active-item bookkeeping and rendered attributes
- a browser test for arrow keys, Home or End, and typeahead under real keyboard events

Current automated coverage lives in:

- `test/playwrightgo/examples/examples_suite_test.go`

The practical rule is simple: use `js/wasm` tests for deterministic accessibility state, and use Playwright for real browser focus and keyboard behavior. Keep both when the feature is user-facing and timing-sensitive.
