# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v6.0.0 - 2026-09-08

### Migration

The module path is now `github.com/monstercameron/GoWebComponents/v6`.
Update application imports and `go.mod` requirements from `/v5` to `/v6`,
including local `replace` directives, then run `go mod tidy` and rebuild both
native and Wasm targets. Do not mix the two major versions in one component tree.
The Windows Wails adapter remains a separate optional module: web applications
do not need to import or initialize Wails.

### Changes

- Added Windows-first desktop APIs, an isolated Wails adapter and pinned Wails
  submodule, build-mode gates, capability ceilings, and CLI build/dev/test support.
- Added a Windows API Lab and web-safe capability handling for native services.
- Fixed runtime in-flight update coalescing and preserved useful JavaScript
  rejection messages without invoking arbitrary message getters.
- Moved chat Markdown and plain code-copy controls onto sanitized GWC nodes;
  migrated the load-harness dashboard and server-interactive SSE view to GWC.
- Fixed server-interactive hydration duplicating server-rendered controls, and
  improved reproducibility of chat/Atlas browser fixtures and WebSocket tests.

### Verification scope and known limitations

Native, Wasm, hydration, core race, JavaScript and Windows desktop checks were
run during preparation; exact commands, skips and results are recorded in
`docs/plans/v6-regression-verification.md`. Counts are not coverage percentages.

The full example migration and browser acceptance work was paused at the
maintainer's request. Known chat-example browser failures include admin/sign-in
flows, settings routing, billing summary and boot/isolation checks. Chat
Mermaid/math ownership, the render-benchmark dashboard and other legacy surfaces
remain on the migration plan. This release does not claim every example passes
or that all example UI has been migrated. The Atlas gzip size diagnostic also
exceeds the historical M5 target. Native desktop support is Windows-first;
macOS and Linux native hosts are not certified by these results.
