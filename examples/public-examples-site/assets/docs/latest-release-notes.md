# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.4.10 - 2026-06-24

The current latest changelog section is `v3.4.10 - 2026-06-24`. **Fixed:** a
controlled `<select value="…">` did not mark the selected option in SSR — the
serializer put a browser-ignored `value` attribute on the `<select>`, so a
server-rendered select showed no initial selection and mismatched on hydration.
The serializer now drops `value` from the `<select>` and renders the matching
`<option>` with `selected` (matching by option value, or by text when it has no
value, and recursing into `<optgroup>`), in both the buffered and streaming
renderers — matching React's `ReactDOMServerIntegrationSelect`. Completes the
controlled-input SSR work begun in v3.4.9 (textarea). It is checked by the
blocking `tools/changelogcheck` release gate
before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
