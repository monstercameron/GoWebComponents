# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.5.3 - 2026-06-25

The current latest changelog section is `v3.5.3 - 2026-06-25`. **Fixed:** a
render-phase state update (a component calling its own `setState` during render —
the derived-state pattern) left the DOM showing a value the component never
actually rendered and never converged. `renderFunctionComponent` now detects a
render-phase update (via a new `activeRenderFiber` flag, distinct from the ambient
current fiber so it never trips on the SSR path or hook unit tests) and re-runs
the component to convergence — bounded at 25 iterations with a diagnostic to guard
against an unconditional-setState infinite loop — matching React's "keeps
restarting until there are no more new updates." Found while bug-hunting GWC's
state management against React. It is checked by the blocking `tools/changelogcheck`
release gate before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
