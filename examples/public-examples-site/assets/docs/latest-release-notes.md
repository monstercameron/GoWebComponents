# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.4.7 - 2026-06-24

The current latest changelog section is `v3.4.7 - 2026-06-24`. **Security:** it
blocks `javascript:`/`vbscript:` URL injection through the normal element API —
a user-controlled `href`/`src`/`action`/… value (e.g.
`html.A(html.Props{Href: userURL})`) could previously ship a clickable
`<a href="javascript:alert(1)">`. Such schemes are now neutralized to
`about:blank` at both the SSR serializer and the browser DOM adapter (tolerating
whitespace/control-char/case obfuscation), while `http(s)`/`data:`/`mailto`/
relative/fragment URLs are untouched. Vectors ported from React's
`ReactDOMServerIntegrationUntrustedURL` suite. It is checked by the blocking
`tools/changelogcheck` release gate before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
