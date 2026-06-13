# Buyer Flow Bucket

Future specs:

- `browse.spec.go`: landing, catalog, product detail, warehouse detail, warehouse availability, and recovery back to catalog.
- `form-submit.spec.go`: quote request, restock request, public comment validation, success, reload, and direct-entry SSR.
- `mobile-nav.spec.go`: public mobile menu open, close, route transition, viewport change, and active navigation state.
- `progressive-enhancement.spec.go`: repeat quote, restock, and comment actions after reload, back navigation, and direct-entry SSR.

Required helpers:

- `buyer-navigation`
- `route-shell-stability`
- `screenshot-capture`

Primary screenshot checkpoints:

- `public-landing-start`
- `public-catalog-browsing`
- `public-product-decision`
- `public-warehouse-availability`
- `public-post-submit-success`
