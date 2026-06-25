# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v3.5.1 - 2026-06-24

The current latest changelog section is `v3.5.1 - 2026-06-24`. **Added:**
streaming SSR (`RenderToStream`) now threads context to hooks. v3.5.0 ran hooks
in the streaming path but passed no context, so a streamed component's
`GoUseContextValue` fell back to the descriptor default. The streaming state now
carries the inherited context map (derived at each `ContextProvider` boundary and
restored after its children, plus captured per pending async boundary so deferred
Suspense content resolves the same context), so streamed `useContext` resolves to
the nearest provider — including nested-provider override. Completes the SSR hooks
work begun in v3.5.0. It is checked by the blocking `tools/changelogcheck` release
gate before a versioned release can ship.

See the repository root `CHANGELOG.md` for the full entry body.
