# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.4.8 - 2026-06-24

The current latest changelog section is `v3.4.8 - 2026-06-24`. **Fixed:** a
reconciler bug where duplicate `key`s in a list leaked host nodes and corrupted
later renders — the one-fiber-per-key lookup let a second same-key sibling
overwrite the first, so the overwritten fiber was never deleted (e.g. `[a,a,b]`
→ `[a,b]` left 3 DOM children instead of 2, and stale nodes bled into later
renders). Duplicate-keyed old fibers are now routed to the positionally-matched
fallback list so every old fiber is cleaned up; unique keys are unaffected. Found
via the native reconciler harness. It is checked by the blocking
`tools/changelogcheck` release gate before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
