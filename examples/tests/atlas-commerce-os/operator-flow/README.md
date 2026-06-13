# Operator Flow Bucket

Future specs:

- `dashboard-triage.spec.go`: sign-in, dashboard alerts, inventory detail review, threshold update, and dashboard-summary confirmation.
- `product-workflow.spec.go`: products list, product editor, validation correction, unsaved-change warning, save, success, reload.
- `inventory-warehouse-workflow.spec.go`: inventory triage, SKU detail, warehouse detail, warehouse item detail, item update, return context.
- `logistics-workflow.spec.go`: transfers, purchase orders, receiving, approval or hold, discrepancy classification, summary refresh.
- `moderation-workflow.spec.go`: pending comments, approve or reject, confirmation overlay, queue update, toast verification.
- `settings-persistence.spec.go`: theme, locale, density, default warehouse, saved-view import or export, direct-entry reload fidelity.

Required helpers:

- `operator-navigation`
- `route-shell-stability`
- `screenshot-capture`

Primary screenshot checkpoints:

- `operator-dashboard-start`
- `operator-product-editor`
- `operator-inventory-triage`
- `operator-warehouse-detail`
- `operator-po-receiving`
- `operator-settings-persistence`
