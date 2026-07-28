# Atlas Server

Go 1.26+ Atlas native server with server-rendered route head metadata, a hydration bootstrap payload, and sqlite-backed mutation endpoints.

What it does now:

- opens sqlite using a pure-Go driver
- applies numbered startup migrations from `examples/server/atlas-commerce-os/server/data/migrations/`
- seeds public and internal Atlas data, including comments, transfers, and receiving sessions
- serves public read and write APIs for catalog, product questions, quote requests, and restock requests
- serves public JSON parity for product comments and related products
- serves internal read and write APIs for bootstrap, inventory, threshold history, transfer recommendations, warehouses, preferences, saved views, moderation, transfers, receiving, purchase orders, and SKU thresholds behind a mock session
- builds each route payload through the shared `shared/atlas` package and serves it as a `__ATLAS_BOOTSTRAP__` JSON payload (route, preferences, i18n, theme, page data, CSRF token) alongside server-rendered `<title>`, `<meta name=description>`, and `<link rel=canonical>` head metadata
- serves an empty `<div id="app"></div>` document body; the Atlas page tree itself is not rendered on the server, so every visible element comes from browser hydration and `/shop` is blank with scripting unavailable
- hydrates that same Atlas page tree in the browser through `examples/server/atlas-commerce-os/client`
- exposes Atlas static assets from the shared examples static directory
- issues a double-submit CSRF cookie with each HTML/bootstrap response and enforces same-origin plus token validation on all public and internal mutation routes

Run from the repo root. The client is a `js/wasm` program, so the build step needs `GOOS=js`/`GOARCH=wasm`; the server step must run natively, so the wasm environment is scoped to the build command only.

PowerShell:

```powershell
New-Item -ItemType Directory -Path .\examples\static\bin -Force | Out-Null
$env:GOOS = 'js'; $env:GOARCH = 'wasm'; go build -o .\examples\static\bin\atlas-commerce-os.wasm ./examples/server/atlas-commerce-os/client; Remove-Item Env:\GOOS, Env:\GOARCH
go run ./examples/server/atlas-commerce-os/server
```

Git Bash:

```bash
mkdir -p examples/static/bin
GOOS=js GOARCH=wasm go build -o examples/static/bin/atlas-commerce-os.wasm ./examples/server/atlas-commerce-os/client
go run ./examples/server/atlas-commerce-os/server
```

The server looks for the wasm at `examples/static/bin/atlas-commerce-os.wasm` and serves it as `/assets/bin/atlas-commerce-os.wasm`, so the output path has to be that directory.

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

- Hydrated Atlas forms include a hidden `csrf_token` field sourced from the bootstrap payload.
- JSON or imperative mutation clients must echo the same token in the `X-CSRF-Token` header.
- Mutation requests without a matching CSRF cookie/token pair or without a same-origin `Origin` or `Referer` are rejected with `403 Forbidden`.
- Validation failures return JSON with `error`, `message`, and field-keyed `fields` entries so clients can map issues back to individual controls.

Mock auth notes:

- Internal HTML routes redirect missing browser sessions to `GET /auth/mock-sign-in?next=...`.
- Internal API routes without a mock session return `401` JSON with `error=mock_sign_in_required` and a recovery URL.
- Mock sign-in currently offers the seeded roles `inventory_manager`, `warehouse_supervisor`, and `ops_lead`.