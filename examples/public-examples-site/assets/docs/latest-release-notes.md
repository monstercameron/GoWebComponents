# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v5.0.4 - 2026-09-03

Version 5.0.4 restores the browser release gate after v5.0.3 stopped before
publication while trying to download its Playwright driver from a retired CDN.
It does not change framework behavior.

Playwright-Go now uses its maintained module path and npm-based driver installer
instead of the retired Azure CDN.

See the repository root `CHANGELOG.md` for the full entry, methodology, and
measurements.
