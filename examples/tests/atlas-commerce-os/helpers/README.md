# Atlas Playwright Helper Contracts

Future executable specs should use these helper contracts instead of recreating traversal logic per file.

## buyer-navigation

- starts at `/` or `/shop`
- follows public shell links and buyer CTAs
- opens product detail, warehouse detail, and warehouse availability routes
- asserts active public nav state and a visible route heading after every step
- returns to `/shop` from recovery states

## operator-navigation

- seeds the mock operator session cookie before internal entry
- opens dashboard, products, inventory, warehouses, logistics, moderation, comments, and settings routes
- follows nested item/detail links while preserving origin context
- asserts the internal shell header and main landmark remain mounted

## parity-landmarks

- resolves public landmarks: header, hero, featured-card band, catalog grid, product action rail, secondary panels
- resolves internal landmarks: header, dashboard summary band, quick actions, dense list or table, action cluster, warehouse ops hierarchy
- fails when a landmark is absent instead of silently skipping parity assertions

## screenshot-capture

- writes named checkpoints using the `atlas-<surface>-<route>-<state>-<theme>-<viewport>.png` convention
- records viewport, theme, locale, route, and checkpoint id beside the image when the future runner supports metadata
- runs after functionality checks pass

## route-shell-stability

- asserts the bootstrap marker, page title, main landmark, route heading, and persistent shell region
- compares shell stability after navigation, mutation, revalidation, overlay open or close, and back or forward navigation
- avoids full DOM snapshot assertions for dense tables unless the spec is explicitly a visual parity check
