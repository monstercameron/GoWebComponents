# Atlas Commerce OS Project Todo

This page is the public documentation mirror for Atlas Commerce OS planning.

## Current Status

The older public checklist in this file was stale. Atlas planning has moved to
the source-side document at
`examples/server/atlas-commerce-os/docs/ATLAS_COMMERCE_OS_TODO.md`.

That source document remains the active Atlas rewrite and validation backlog.
This public mirror intentionally does not duplicate its checkbox list because a
second checklist drifts quickly and creates false first-party TODOs in the
public documentation set.

## Evidence

- The source Atlas TODO tracks the current React-reference rewrite plan and its
  active route, parity, test, accessibility, localization, and validation work.
- The shipped Atlas example now includes fetch-backed resource loading,
  SSR-bootstrap cache seeding, route loaders, sqlite-backed server state,
  mutation endpoints, structured logging, and many focused server/shared tests.
- Current Atlas implementation notes live in
  `examples/server/atlas-commerce-os/docs/README.md`.
- Current route, bootstrap, cache, and derived-state rules live in
  `examples/server/atlas-commerce-os/docs/ATLAS_ROUTING_DATA_BASELINE.md`.
- The public examples site should link readers to the Atlas example and source
  docs instead of maintaining an independent backlog.

## Refresh Rule

Do not update Atlas planning here first. Update
`examples/server/atlas-commerce-os/docs/ATLAS_COMMERCE_OS_TODO.md` when Atlas
scope changes, then refresh this public mirror only if the source status or
links change.
