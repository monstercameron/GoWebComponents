# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.4.4 - 2026-06-24

The current latest changelog section is `v3.4.4 - 2026-06-24`. It fixes a
duplicate `ExampleUseAtom` that broke the `state` package test build under
`GOOS=js GOARCH=wasm`, plus SSR correctness edge tests, and is checked by the
blocking `tools/changelogcheck` release gate before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
