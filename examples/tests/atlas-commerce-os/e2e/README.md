# E2E User Flow Tracks

These tracks mirror the Playwright buckets at acceptance-story level.

## buyer-flow-e2e

- `buyer-browse-and-convert`: landing discovery, catalog filtering, product evaluation, warehouse availability review, quote request completion.
- `buyer-restock`: landing discovery, product detail, restock request, confirmation-state recovery after reload.
- `buyer-feedback`: product detail, comment submission, pending-state visibility, moderation expectation messaging.
- `buyer-recovery`: back navigation, direct-entry SSR, post-submit return paths across catalog, product detail, and warehouse availability routes.
- `buyer-parity-review`: storefront-reference hierarchy, CTA emphasis, card rhythm, and secondary-panel behavior.

## operator-flow-e2e

- `operator-dashboard-triage`: sign-in, dashboard triage, inventory detail review, threshold adjustment, dashboard-summary confirmation.
- `operator-product-workflow`: products list, product creation or edit, validation correction, save, warehouse-context follow-up work.
- `operator-warehouse-workflow`: warehouse detail, item detail, replenishment or inventory update, return-to-warehouse continuity.
- `operator-logistics-workflow`: transfer creation, approval or cancelation, downstream route-summary verification.
- `operator-purchase-order-and-receiving`: purchase-order decision through receiving or warehouse follow-up.
- `operator-receiving-discrepancy`: discrepancy review, classification, reconciliation, activity-log confirmation.
- `operator-moderation`: comments backlog, approve or reject decisions, refreshed public visibility expectations.
- `operator-settings-and-persistence`: theme, locale, density, default warehouse, session persistence across route families.
- `operator-parity-review`: warehouse-reference dashboard hierarchy, data-density rhythm, quick-action emphasis, and navigation feel.
