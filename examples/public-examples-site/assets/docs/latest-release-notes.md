# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.4.9 - 2026-06-24

The current latest changelog section is `v3.4.9 - 2026-06-24`. **Fixed:** a
controlled `<textarea value="…">` rendered empty in SSR. HTML ignores a `value`
attribute on `<textarea>` (its value is the element's text content), but the SSR
serializer emitted `<textarea value="x">`, so the field rendered blank
server-side and mismatched on hydration. The serializer now renders a textarea's
string `value` as escaped text content (`<textarea>x</textarea>`), in both the
buffered and streaming renderers — matching React's
`ReactDOMServerIntegrationTextarea`. Found while porting React's controlled-input
SSR tests. It is checked by the blocking `tools/changelogcheck` release gate
before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
