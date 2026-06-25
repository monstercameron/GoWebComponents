# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.5.2 - 2026-06-25

The current latest changelog section is `v3.5.2 - 2026-06-25`. **Fixed:**
`defaultValue`/`defaultChecked` rendered as inert, browser-ignored attributes in
SSR, so an uncontrolled form field set via `defaultValue` rendered empty
server-side. The SSR serializer (buffered + streaming) now maps `defaultValue` →
the element's controlled value and `defaultChecked` → `checked` (unless the
controlled prop is already set, so an explicit `value` wins). Because it runs
before the controlled-value logic, it flows through every form element: `<input>`
gets a `value`, `<textarea>` gets text content, and `<select>` marks the matching
`<option selected>` — matching React and completing the controlled-input SSR work
(v3.4.9 textarea, v3.4.10 select). It is checked by the blocking
`tools/changelogcheck` release gate before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
