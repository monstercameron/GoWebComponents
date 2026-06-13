# Atlas Server

Go 1.26+ Atlas native server with SSR rendering, hydration bootstrap, and sqlite-backed mutation endpoints.

What it does now:

- opens sqlite using a pure-Go driver
- applies numbered startup migrations from `examples/server/atlas-commerce-os/server/data/migrations/`
- seeds public and internal Atlas data, including comments, transfers, and receiving sessions
- serves public read and write APIs for catalog, product questions, quote requests, and restock requests
- serves public JSON parity for product comments and related products
- serves internal read and write APIs for bootstrap, inventory, threshold history, transfer recommendations, warehouses, preferences, saved views, moderation, transfers, receiving, purchase orders, and SKU thresholds behind a mock session
- renders Atlas pages through the shared `shared/atlas` package on the server
- hydrates the same Atlas page tree in the browser through `examples/server/atlas-commerce-os/client`
- exposes Atlas static assets from the shared examples static directory
- issues a double-submit CSRF cookie during SSR/bootstrap responses and enforces same-origin plus token validation on all public and internal mutation routes

Run from the repo root:

```powershell
New-Item -ItemType Directory -Path ./bin/examples -Force | Out-Null
go build -o ./bin/examples/atlas-commerce-os.wasm ./examples/server/atlas-commerce-os/client
go run ./examples/server/atlas-commerce-os/server
```

Useful endpoints:

- `GET /healthz`
- `GET /auth/mock-sign-in`
- `GET /shop`, `GET /shop/{slug}`, `GET /warehouses`, `GET /app`, `GET /app/dashboard`, `GET /app/inventory`, `GET /app/inventory/{sku}`
- `GET /app/warehouses`, `GET /app/warehouses/{warehouseId}`
- `GET /app/transfers`, `GET /app/transfers/{id}`
- `GET /app/purchase-orders`, `GET /app/purchase-orders/{id}`
- `GET /app/receiving`, `GET /app/receiving/{id}`
- `GET /app/comments`, `GET /app/settings`
- `GET /api/public/products/{slug}/comments`
- `GET /api/public/products/{slug}/related-products`
- `GET /api/app/preferences`, `GET /api/app/saved-views`
- `GET /api/app/inventory/{sku}/threshold-history`, `GET /api/app/inventory/{sku}/transfer-recommendations`
- `GET /api/app/warehouses`, `GET /api/app/warehouses/{warehouseId}`
- `GET /api/app/transfers/{id}`, `GET /api/app/receiving/{id}`
- `GET /api/app/purchase-orders`, `GET /api/app/purchase-orders/{id}`
- `POST /api/public/products/{slug}/comments`
- `POST /api/public/products/{slug}/quote-requests`
- `POST /api/public/products/{slug}/restock-requests`
- `POST /api/app/preferences`
- `POST /api/app/saved-views`
- `POST /api/app/comments/{id}/moderate`
- `POST /api/app/inventory/{sku}/threshold`
- `POST /api/app/transfers`
- `POST /api/app/receiving/{id}/reconcile`
- `POST /api/app/purchase-orders/{id}/status`
- `POST /auth/mock-sign-in`
- `POST /auth/mock-sign-out`

Mutation notes:

- SSR forms now include a hidden `csrf_token` field sourced from the bootstrap payload.
- JSON or imperative mutation clients must echo the same token in the `X-CSRF-Token` header.
- Mutation requests without a matching CSRF cookie/token pair or without a same-origin `Origin` or `Referer` are rejected with `403 Forbidden`.
- Validation failures return JSON with `error`, `message`, and field-keyed `fields` entries so clients can map issues back to individual controls.

Mock auth notes:

- Internal SSR routes redirect missing browser sessions to `GET /auth/mock-sign-in?next=...`.
- Internal API routes without a mock session return `401` JSON with `error=mock_sign_in_required` and a recovery URL.
- Mock sign-in currently offers the seeded roles `inventory_manager`, `warehouse_supervisor`, and `ops_lead`.