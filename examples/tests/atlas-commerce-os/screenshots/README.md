# Screenshot And Visual Parity Conventions

Screenshot artifacts still live under `examples/server/atlas-commerce-os/docs/screenshots/`. This folder defines the test-bucket capture contract for future browser automation.

## Naming

Use:

```text
atlas-<surface>-<route>-<state>-<theme>-<viewport>.png
```

Allowed values:

- surface: `public`, `internal`, `recovery`, `parity`
- theme: `light`, `dark`
- viewport: `desktop`, `mobile`
- locale review values: `en`, `fr`, `ar`

## Public Checkpoints

- landing start state
- catalog browsing state
- product-detail decision state
- warehouse availability state
- quote, restock, or comment post-submit success state

## Operator Checkpoints

- dashboard start state
- product editor state
- inventory triage state
- warehouse detail state
- purchase-order or receiving workflow state
- settings persistence state

## Recovery And State Checkpoints

- empty state
- no-results state
- recovery page
- validation error
- success state
- overlay-open state

## Design-Parity Checkpoints

- storefront start, midpoint, and end against `design/homepage_store.tsx`
- operator start, midpoint, and end against `design/homepage_warehouse.tsx`

Capture buyer-flow and operator-flow functionality first. Refresh design-parity screenshots after those pass so visual drift is reviewed against working routes.
