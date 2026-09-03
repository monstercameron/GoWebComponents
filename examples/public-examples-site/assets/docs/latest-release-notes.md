# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v5.0.3 - 2026-09-03

Version 5.0.3 completes the v5 promotion path after earlier tagged builds
stopped in CI before GitHub could publish them. It does not change framework
behavior.

The docs-site release-note mirror now tracks the latest changelog section, the
release gate builds its generated wasm fixtures, and the release uses patched
gRPC and Goldmark dependencies. Playwright-Go now uses its maintained module
path and npm-based driver installer instead of the retired Azure CDN.

See the repository root `CHANGELOG.md` for the full entry, methodology, and
measurements.
