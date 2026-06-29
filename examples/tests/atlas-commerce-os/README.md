# Atlas Commerce OS Browser Flow Tests

This directory owns the Atlas browser-flow planning layer before the executable Playwright specs are split out of `test/playwrightgo/examples`.

The goal is to keep buyer journeys, operator journeys, design-parity checks, screenshots, and recovery edge cases grouped by product intent instead of scattering route lists through ad hoc tests.

## Buckets

| Bucket | Directory | Purpose |
| --- | --- | --- |
| buyer-flow | `buyer-flow/` | Public storefront navigation, quote, restock, comment, mobile-nav, recovery, and progressive-enhancement stories. |
| operator-flow | `operator-flow/` | Internal sign-in, dashboard, products, inventory, warehouses, logistics, moderation, comments, and settings stories. |
| design-parity | `design-parity/` | React reference parity checks keyed to `design/homepage_store.tsx` and `design/homepage_warehouse.tsx`. |
| recovery-edge-cases | `recovery-edge-cases/` | 404, server-error, failed-write, network-interruption, duplicate-submit, invalid-query, and async-navigation edge cases. |
| e2e | `e2e/` | End-to-end user-flow tracks that mirror the Playwright buckets at acceptance-story level. |
| screenshots | `screenshots/` | Screenshot naming, capture points, viewports, themes, locale, and reference-pair conventions. |
| helpers | `helpers/` | Shared helper contracts for navigation, landmarks, screenshots, and route-shell stability. |

## Manifest

`manifest.json` is the source of truth for:

- bucket names and intended future spec files
- helper contracts each executable spec should use
- manual and automated Playwright story IDs
- E2E tracks
- failure and recovery edge-case stories
- screenshot checkpoint IDs and file-name rules
- run order: buyer-flow and operator-flow functionality first, design-parity after core behavior passes

Run the manifest guard from the repo root:

```powershell
go test ./examples/tests/atlas-commerce-os
```

This test does not replace executable Playwright coverage. It prevents the bucket map from drifting while the browser suites are being implemented.

## Future Playwright Layout

When executable specs land, keep the same bucket names:

```text
examples/tests/atlas-commerce-os/
  buyer-flow/
    browse.spec.go
    form-submit.spec.go
    mobile-nav.spec.go
    progressive-enhancement.spec.go
  operator-flow/
    dashboard-triage.spec.go
    product-workflow.spec.go
    inventory-warehouse-workflow.spec.go
    logistics-workflow.spec.go
    moderation-workflow.spec.go
    settings-persistence.spec.go
  design-parity/
    public-storefront-reference.spec.go
    internal-warehouse-reference.spec.go
```

The current executable Atlas smoke coverage remains in `test/playwrightgo/examples` until those specs are promoted.
