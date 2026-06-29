# Design Parity Bucket

Future specs:

- `public-storefront-reference.spec.go`: checks public header, hero, featured-card band, catalog grid rhythm, and product-detail composition against `examples/server/atlas-commerce-os/design/homepage_store.tsx`.
- `internal-warehouse-reference.spec.go`: checks internal header, dashboard summary band, action cluster, dense list rhythm, and warehouse-ops hierarchy against `examples/server/atlas-commerce-os/design/homepage_warehouse.tsx`.

Run this bucket after buyer-flow and operator-flow functionality passes. Design parity should catch layout drift, not mask broken workflows.

Required helpers:

- `parity-landmarks`
- `route-shell-stability`
- `screenshot-capture`

Pair checkpoints:

- public start, midpoint, and end: landing, catalog, product detail
- operator start, midpoint, and end: dashboard, inventory, warehouse detail
