# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.5.4 - 2026-06-25

The current latest changelog section is `v3.5.4 - 2026-06-25`. **Fixed:** a
regression in v3.5.3 where effects were dropped when a component used render-phase
updates. The convergence loop cleared the fiber's per-iteration effect list before
each re-run; combined with the deps-equality check (a stable-deps effect does not
re-register when its deps are unchanged), the effect ended up registered in no
iteration and never ran. The loop no longer clears the effect list — effects
accumulate across iterations and are then de-duplicated by cleanup-index (keeping
the final render's effect per slot), so each effect runs exactly once with the
converged state (for both stable and changing deps), and multiple effects run once
each in declaration order. Found by stress-testing the v3.5.3 fix. It is checked by
the blocking `tools/changelogcheck` release gate before a versioned release can
ship.

See the repository root `CHANGELOG.md` for the full entry body.
