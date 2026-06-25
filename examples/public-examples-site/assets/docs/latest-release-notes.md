# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.5.0 - 2026-06-24

The current latest changelog section is `v3.5.0 - 2026-06-24`. **Added:** hooks
now run during server rendering (`ui.RenderToString`). Previously the string
serializer was hook-less — a component calling any hook errored, so only
hook-free trees could be server-rendered. `RenderToString` now installs a
transient hook fiber per component: `GoUseState` returns its initial value,
`GoUseRef`/`GoUseMemo` compute, `GoUseContextValue` resolves to the nearest
provider value (context flows through host elements; nested providers override),
and `GoUseEffect` is queued but never run on the server — matching React's
`ReactDOMServerIntegrationHooks`. This lets GWC server-render real hook-using
components. The buffered path is fully supported; the streaming path runs hooks
but does not yet thread context (a documented follow-up). It is checked by the
blocking `tools/changelogcheck` release gate before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
