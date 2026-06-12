# Accessibility Statement

This statement covers the GoWebComponents **documentation site** (the pure-GWC
app under `examples/site`) and the framework's built-in accessibility support.

## Commitment

The documentation site aims to conform to **WCAG 2.1 Level AA**. Accessibility
is treated as a correctness property, not a cosmetic one: the framework ships
accessibility primitives, and the docs site is expected to dogfood them.

## What the framework provides

Apps built with GoWebComponents have first-class accessibility primitives in the
`ui` package, so accessible behavior is the default path rather than bespoke
wiring:

- **Focus management** — `ui.AccessibleOverlay`, `ui.UseFocusTrap`,
  `ui.UseFocusManager` keep focus inside open dialogs and restore it on close.
- **Live announcements** — `ui.UseAnnouncer` for polite/assertive screen-reader
  updates (route changes, validation results, async status).
- **Keyboard navigation** — `ui.UseCompositeNavigation` for arrow-key / Home /
  End / typeahead movement across tabs, listboxes, and menus.
- **User preferences** — `ui.UsePrefersReducedMotion()` and
  `ui.UsePrefersColorScheme()` so apps honor OS/browser settings.
- **Routed accessibility** — route-change announcements and heading focus after
  navigation (see the `routed-accessibility` example).

See [Forms, Accessibility, and i18n](REFERENCE_MANUAL/11-forms-accessibility-and-i18n.md)
for the full surface.

## Known gaps (tracked)

These are open items being worked toward AA conformance:

- An **automated axe-core audit lane** across the docs-site routes and the public
  examples is not yet wired into CI, so contrast/label/name regressions are not
  yet caught automatically.
- The docs-site **search modal** does not yet use `UseFocusTrap` / `UseAnnouncer`
  for focus containment and result-count announcements, and the gallery filters
  do not yet expose composite keyboard navigation.

Until those land, accessibility is verified manually against the rules in
[CONTRIBUTING.md](../CONTRIBUTING.md) and the production checklist in
[docs/PRODUCTION_READINESS.md](PRODUCTION_READINESS.md).

## Feedback

If you encounter an accessibility barrier on the documentation site, please open
an issue describing the page, the assistive technology and browser used, and the
barrier. Accessibility defects are treated as correctness bugs.
