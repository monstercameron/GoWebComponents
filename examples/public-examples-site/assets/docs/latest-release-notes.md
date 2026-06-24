# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.4.5 - 2026-06-24

The current latest changelog section is `v3.4.5 - 2026-06-24`. It fixes the
`diagnostics`/`internal/diagnostics` test binaries failing to compile under
`GOOS=js GOARCH=wasm` (build-neutral tests referenced the native-only
`WriteHTTPError`), and is checked by the blocking `tools/changelogcheck` release
gate before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
