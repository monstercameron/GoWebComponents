# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.4.6 - 2026-06-24

The current latest changelog section is `v3.4.6 - 2026-06-24`. **Security:** it
fixes a NUL-byte bypass of the CSS `<style>`/comment-close hardening (an input
like `*\x00/` could reconstitute `*/` in emitted CSS), found by fuzzing. It is
checked by the blocking `tools/changelogcheck` release gate before a versioned
release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
