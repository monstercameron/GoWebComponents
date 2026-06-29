# Accessibility Statement

This statement covers the GoWebComponents documentation and examples site under
`examples/public-examples-site` and the framework's built-in accessibility
support.

## Commitment

The documentation site aims to conform to WCAG 2.1 Level AA. Accessibility is
treated as a correctness property: the framework ships accessibility primitives,
and the docs site dogfoods them in its own search and navigation surfaces.

## What the framework provides

Apps built with GoWebComponents have first-class accessibility primitives in the
`ui` package, so accessible behavior is the default path rather than bespoke
wiring:

- **Focus management** - `ui.AccessibleOverlay`, `ui.UseFocusTrap`, and
  `ui.UseFocusManager` keep focus inside open dialogs and restore it on close.
- **Live announcements** - `ui.UseAnnouncer` provides polite/assertive
  screen-reader updates for route changes, validation results, async status, and
  search result counts.
- **Keyboard navigation** - `ui.UseCompositeNavigation` covers arrow-key,
  Home/End, and typeahead movement across tabs, listboxes, and menus.
- **User preferences** - `ui.UsePrefersReducedMotion()` and
  `ui.UsePrefersColorScheme()` let apps honor OS/browser settings.
- **Routed accessibility** - route-change announcements and heading focus after
  navigation are demonstrated in the `routed-accessibility` example.

See [Forms, Accessibility, and i18n](REFERENCE_MANUAL/11-forms-accessibility-and-i18n.md)
for the full surface.

## Current Verification

- The docs-site search dialog uses `ui.AccessibleOverlay` for modal semantics,
  focus trapping, Escape/outside dismissal, scroll lock, and focus restoration.
- The docs-site search and filters use `ui.UseAnnouncer` plus polite live
  regions so result-count changes are exposed to assistive technology.
- `go test -tags playwrightgo ./test/playwrightgo/examples -run TestAccessibilityAudit`
  injects vendored axe-core and fails on serious or critical WCAG 2A/2AA
  violations for the docs-site shell and representative public examples.
- `go test ./examples/public-examples-site` (run as `js/wasm`) checks the
  search dialog overlay contract, labelled search input, live result summary,
  and source/preview interaction states.

## Feedback

If you encounter an accessibility barrier on the documentation site, please open
an issue describing the page, the assistive technology and browser used, and the
barrier. Accessibility defects are treated as correctness bugs.
