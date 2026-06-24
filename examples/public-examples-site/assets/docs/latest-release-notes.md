# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.4.3 - 2026-06-24

The current latest changelog section is `v3.4.3 - 2026-06-24`. It hardens the
global-event hooks to degrade to a no-op when the host lacks `addEventListener`
(defense-in-depth; no change in real browsers/Web Workers), plus correctness edge
tests, and is checked by the blocking `tools/changelogcheck` release gate before
a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
