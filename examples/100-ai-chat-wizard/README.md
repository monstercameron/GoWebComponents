# 100 - AI Chat Wizard

A streaming AI chat workspace built with **Go WASM + gRPC + GoGRPCBridge + provider runtime**.

The browser client and server are both written in Go. The chat UI and background render worker compile to
WASM, and chat messages stream over a gRPC tunnel carried by WebSocket (`/socket`). The server can resolve
models from OpenAI, Anthropic, and Cerebras catalogs through one runtime provider registry.

---

## Architecture

```
Browser (Go WASM, port 8095)                     Go Server (in-process gRPC)
+--------------------------------------------+    +---------------------------------------------+
| App shell routes: / /home /pricing /app... |    | HTTP mux                                     |
|  - Auth + chat UI state                     |    |  - GET /, /home, /pricing, /app/* -> shell  |
|  - Composer -> ChatService.Send() stream    |<-->|  - GET /app/chat.wasm, /worker/*.wasm       |
|                                            WS/gRPC| - GET /chat-bootstrap.js, /healthz           |
| Background worker (WASM)                     |    |  - /socket -> GoGRPCBridge tunnel            |
|  - Markdown/render metadata tasks            |    |                                             |
+--------------------------------------------+    | gRPC ChatService                             |
                                                  |  - Auth/session RPCs (Login/GetSession/...)  |
                                                  |  - Chat RPCs (Send/List/Load/Resolve)        |
                                                  |  - Admin/superuser RPCs                      |
                                                  |                                             |
                                                  | Store + provider runtime                      |
                                                  |  - SQLite tables (auth, chat, usage, ops)    |
                                                  |  - Provider registry + model catalog          |
                                                  +---------------------------------------------+
```

**Data flow**

1. Visitor lands on a public route (`/`, `/home`, `/pricing`, `/signup`) and bootstraps the shell.
2. User authenticates via `Login`/`Signup`; client persists token and validates with `GetSession`.
3. Client opens the gRPC tunnel at `/socket`; runtime marks `grpc ready` and worker readiness.
4. Authenticated shell hydrates profile/settings/catalog/conversation state from typed RPCs.
5. User sends a prompt; WASM client calls `ChatService.Send` with history, model, tone, and thinking settings.
6. Server resolves provider/model, streams `ChatChunk` deltas, and persists usage and conversation updates.
7. Client renders streamed output incrementally, normalizes canonical thread route, and keeps state resumable.

---
## Prerequisites

| Tool | Purpose |
|---|---|
| Go ≥ 1.25 | Build server and WASM client |
| `protoc` | Proto compiler (generate Go stubs) |
| `protoc-gen-go` | Go protobuf plugin |
| `protoc-gen-go-grpc` | Go gRPC plugin |
| Tailwind CSS CLI | Built via `go run ./tools/gwc tailwind` (cached under `third_party/tailwindcss/bin`) |

The `third_party/GoGRPCBridge` git submodule must be initialised:

```powershell
git submodule update --init --recursive
```
## Quick start

### 1. Initialize submodules (first clone only)

```powershell
git submodule update --init --recursive
```

### 2. Pick one shared local DB path

The server and `cmd/seed-test-db` must use the same `CHAT_DB_PATH`.

```powershell
$env:CHAT_DB_PATH = "examples/100-ai-chat-wizard/bin/runtime/test_chat.db"
```

### 3. Build both WASM artifacts

```powershell
go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\main.go -root .\examples\100-ai-chat-wizard\client -out .\examples\100-ai-chat-wizard\bin\client\app\chat.wasm -json
go run ./tools/gwc build -app .\examples\100-ai-chat-wizard\client\backgroundworker\main.go -root .\examples\100-ai-chat-wizard\client\backgroundworker -out .\examples\100-ai-chat-wizard\bin\client\worker\background-worker.wasm -json
```

The server serves Brotli sidecars when present, but raw `.wasm` artifacts are enough for local development.

### 4. Seed local login accounts

```powershell
go run ./examples/100-ai-chat-wizard/cmd/seed-test-db
```

Seeded credentials:

- `demo@example.com / password123`
- `admin@example.com / password`

### 5. Choose provider mode

Use real provider keys:

```powershell
$env:OPENAI_API_KEY = "sk-..."
$env:ANTHROPIC_API_KEY = "sk-ant-..."
$env:CEREBRAS_API_KEY = "csk-..."
```

Or run fully local provider stubs:

```powershell
$env:CHAT_PROVIDER_STUBS = "all"
```

### 6. Start the managed example server

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server start -json
```

Useful lifecycle commands:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server status -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server restart -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server stop -json
```

Open `http://127.0.0.1:8095/`.

### 7. Verify auth and first chat

1. Log in with `demo@example.com / password123`.
2. Create a new thread and send one prompt.
3. Refresh and confirm the thread reopens at `/app/thread/:publicID`.

### 8. Run focused browser verification

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100StartupBoot -v
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100AuthenticatedHappyPath -v
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample100RouteSmokePricingAuthDashboard -v
```

---

## Route model

The server serves one shell-first SPA surface, and the client router decides which public/auth/chat view to render.

| Route class | Paths | Behavior |
|---|---|---|
| Public landing routes | `/`, `/home`, `/capabilities`, `/pricing`, `/signup` | Server returns the chat shell bootstrap HTML; client renders marketing/auth-facing views for unauthenticated users. |
| Auth entry | `/` (login default), `/signup` (signup mode) | Auth form submits to gRPC (`Login`/`Signup`), then authenticated sessions pivot into `/app`. |
| App root and thread routes | `/app`, `/app/thread/:publicID`, `/app/thread/:publicID/canvas/:canvasID` | Authenticated shell routes for conversation list, active thread replay, streaming replies, and canvas artifacts. |
| Settings routes | `/app/settings?panel=settings-*` | Still part of the SPA app shell; query `panel` selects profile/tone/prompt/intelligence/speech/memories/language section. |
| Legacy entry aliases | `/login`, `/logout`, `/thread/:legacyID` | Server still serves shell for compatibility; client normalizes into current `/app` route model. |
| Admin/dashboard surfaces | No dedicated browser route yet in this example | Admin visibility is currently RPC-gated (superuser checks on admin/superuser RPCs), with route-level dashboard IA still pending. |
| Static/runtime assets | `/chat-bootstrap.js`, `/app/chat.wasm`, `/worker/background-worker.wasm`, `/static/*`, `/healthz`, `/socket` | Not SPA routes; served directly by HTTP mux or gRPC bridge (`/socket`). |

### SPA vs server-shell behavior

- For route-like paths (public routes plus `/app*` paths without file extensions), the server returns the same shell document.
- The client-side router selects the active view and can normalize thread routes after conversation resolution.
- Asset paths (WASM, JS, CSS, images, static files) bypass SPA shelling and are served directly.
- Legacy `/chat.wasm` and `/background-worker.wasm` requests are rewritten to `/app/chat.wasm` and `/worker/background-worker.wasm`.

---

## Runtime pieces

This is the current runtime chain from first paint to streamed reply.

| Runtime piece | Responsibility | Depends on |
|---|---|---|
| Boot shell (`/chat-bootstrap.js`) | Shows startup state, loads `chat.wasm`, then hides once the app mounts. | HTTP shell route, WASM artifacts, `wasm_exec.js`. |
| WASM client (`client/main.go`, `client/app/*`) | Owns router state, auth/session UX, thread state, composer send, stream rendering, and settings panels. | gRPC tunnel readiness, model/profile/conversation RPCs. |
| Background worker (`client/backgroundworker/main.go`) | Offloads markdown/render metadata tasks and async render helpers. | Worker WASM artifact and worker message bridge. |
| gRPC tunnel (`/socket`) | Carries unary + streaming RPCs over WebSocket between browser and in-process gRPC server. | GoGRPCBridge handler, client dial/reconnect loop. |
| Server HTTP handlers (`server/app/server.go`) | Serves shell routes, static assets, health probe, wasm_exec, and tunnel endpoint. | Runtime config, static dirs, bridge handler. |
| Chat/auth/admin RPC handlers | Implement auth/session, chat send/list/load, model/profile preferences, and admin/superuser surfaces. | Store layer, auth manager, provider registry. |
| Store layer (SQLite) | Persists auth sessions, conversations/messages, usage events, and control-plane records. | SQL query files under `sql/store`, DB path/config. |
| Provider layer (`server/provider/*`) | Resolves provider/model runtime, streams completions, and reports health/rate-limit metadata. | API keys or `CHAT_PROVIDER_STUBS`, model catalog rows. |

### Interaction sequence

1. Shell HTML and bootstrap JS load, then fetch `app/chat.wasm`.
2. Client mounts, starts worker lifecycle, and opens gRPC tunnel to `/socket`.
3. Auth/session bootstrap resolves (`GetSession`, optional `RefreshSession`), then app state hydrates (`ListModelOptions`, profile/settings, conversations).
4. Composer submit calls `Send`; server resolves provider/model, streams `ChatChunk` deltas, and writes usage + conversation state.
5. Client applies streamed deltas, updates canonical thread route, and keeps session/thread state resumable across reloads.

---

## SQL-backed model catalog pattern

RelayDesk keeps provider and model discovery in SQLite instead of hard-coding model enums into the WASM client.

- [`sql/store/schema.sql`](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/sql/store/schema.sql) defines `model_catalog` as the source of truth for provider ID, display name, pricing, throughput, onboarding state, and capability flags such as thinking and speech support.
- [`server/app/model_catalog_store.go`](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/server/app/model_catalog_store.go) loads those rows into per-provider catalogs plus the global default, title-generation, and memory-extraction model picks during server startup.
- [`server/app/server.go`](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/server/app/server.go) exposes the runtime query contract through `ListModelOptions`, so the authenticated shell can ask the server which providers and models are currently live.

That pattern means a provider rollout is usually a data change, not a WASM rebuild: add or update rows in `model_catalog`, restart the server, and the next `ListModelOptions` refresh advertises the new catalog to connected clients.

For apps that want a zero-RPC first paint, use `ui.SSRBootstrap.Data` to ship the same catalog shape the RPC already returns:

```json
{
  "modelCatalog": {
    "defaultModel": "gpt-5.4-mini",
    "generatedAt": "2026-03-25T02:28:00-04:00",
    "models": [
      {
        "id": "gpt-5.4-mini",
        "label": "GPT-5.4 mini",
        "providerId": "openai",
        "providerLabel": "OpenAI",
        "supportsThinking": true,
        "supportsSpeech": true,
        "pricing": {
          "inputCostPerMillionUsd": 0.25,
          "outputCostPerMillionUsd": 2.0,
          "currency": "USD"
        }
      }
    ]
  }
}
```

RelayDesk treats that bootstrap payload, or the cached `chat-wizard:model-catalog` local-storage entry, as last-known-good UI state only. The authenticated shell still revalidates through `ListModelOptions` when the gRPC session comes up so long-lived tabs converge back to the server-owned catalog without a full page reload.

### Optional: one-command managed server start

Start the default managed profile directly:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server start -json
```

Then inspect or restart as needed:

```powershell
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server status -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server restart -json
go run ./tools/gwc examples .\examples\100-ai-chat-wizard\cmd\server stop -json
```

---

## File layout

```text
examples/100-ai-chat-wizard/
+-- docs/
|   +-- BUG_REPORT_TEMPLATES.md
|   +-- PERFORMANCE.md
+-- bin/
|   +-- client/
|   |   +-- app/chat.wasm
|   |   +-- worker/background-worker.wasm
|   +-- runtime/
|   |   +-- chat_history.db
|   |   +-- logs/server.stderr.log
|   |   +-- logs/server.stdout.log
|   |   +-- legacy-artifacts/
|   +-- server/
|       +-- chat-wizard.exe
|       +-- chat-wizard-server.exe
+-- scripts/                    # reserved for example-local helpers
+-- client/
|   +-- main.go
|   +-- app/
|   +-- backgroundworker/
+-- cmd/
|   +-- server/
|   +-- seed-test-db/
+-- proto/
|   +-- chat.proto
|   +-- chat.pb.go
|   +-- chat_grpc.pb.go
+-- server/
|   +-- app/
|   +-- provider/
+-- sql/
+-- testdata/
+-- README.md
+-- FLOWS.md
+-- MANUAL_SMOKE.md
+-- SCHEMA_TABLES.md
+-- DESIGN.md
+-- DOCS_MAP.md
+-- OPERATOR_RUNBOOK.md
+-- TODO.md
+-- CHANGELOG.md
```

## Repo layout rules

- Keep primary entry docs at root (`README.md`, `FLOWS.md`, `MANUAL_SMOKE.md`, `SCHEMA_TABLES.md`, `TODO.md`, `CHANGELOG.md`).
- Keep secondary/supporting docs under `docs/` (for example `docs/BUG_REPORT_TEMPLATES.md`, `docs/PERFORMANCE.md`).
- Keep runtime state and logs under `bin/runtime/` (`chat_history.db`, `logs/`, managed state files).
- Keep generated binaries and build artifacts under `bin/` subfolders (`bin/client`, `bin/server`, `bin/runtime/legacy-artifacts`).
- Keep helper scripts under `scripts/` only; avoid scattering executable helpers at root.
- Keep source code under `client/`, `server/`, `cmd/`, `proto/`, `sql/`, and `internal/`.

---

## Regenerating proto stubs

```powershell
cd examples/100-ai-chat-wizard
protoc --go_out=. --go_opt=paths=source_relative `
       --go-grpc_out=. --go-grpc_opt=paths=source_relative `
       proto/chat.proto
```

RelayDesk is the repo's reference implementation for runtime AI provider switching. It demonstrates:

- a server-backed runtime model catalog loaded into the authenticated shell at startup
- provider and model switching without a page reload
- per-user selected-model persistence through the chat RPC surface
- live cross-tab synchronization of the active provider/model selection
- local stub-provider workflows so provider switching stays testable without real upstream keys

---

## Benchmarks

Use the server benchmark suite to get a quick read on store cost and synthetic active-chat pressure by clients-per-core:

```powershell
go test ./examples/100-ai-chat-wizard/server/app -run '^$' -bench 'Benchmark(StoreCorePaths|SendClientsPerCore)' -benchmem
```

`BenchmarkSendClientsPerCore` uses a synthetic provider and reports `clients/core` sub-benchmarks at `1`, `2`, and `4`.

For an SLA sweep that pins the server to `1..8` cores and increases concurrent clients until the p95 request latency breaches the target:

```powershell
$env:CHAT_WIZARD_BENCH_SLA_MS = "100"
go test ./examples/100-ai-chat-wizard/server/app -run TestSendSLASweep -v
```

Optional knobs:
- `CHAT_WIZARD_BENCH_MAX_CORES` default `8`
- `CHAT_WIZARD_BENCH_MAX_CLIENTS` default `64`
- `CHAT_WIZARD_BENCH_BURST_RUNS` default `3`
- `CHAT_WIZARD_BENCH_PREDICT_CORES` default `32`

The sweep logs:
- measured `max_clients_under_sla` for each core count
- `clients_per_core`
- `scaling_vs_1_core`
- cumulative rollup totals across core levels
- a linear-regression projection for the requested prediction core count using uncensored cumulative rollup points

For a fuller explanation of the output fields, interpretation, and latest measured sample data, see [PERFORMANCE.md](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/docs/PERFORMANCE.md).

---

## Key packages

| Package | Role |
|---|---|
| `github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel` | WebSocket↔gRPC tunnel (server + WASM client) |
| `github.com/monstercameron/GoWebComponents/ui` | Hooks-based WASM UI (state, effects, events) |
| `github.com/monstercameron/GoWebComponents/html` | Typed HTML node builders |
| `google.golang.org/grpc` | gRPC runtime |
| `google.golang.org/protobuf` | Protobuf serialisation |

The GoGRPCBridge module lives at `third_party/GoGRPCBridge` (git submodule) and is
referenced via a `replace` directive in the root `go.mod`.

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `OPENAI_API_KEY` | _(optional)_ | OpenAI secret key |
| `ANTHROPIC_API_KEY` | _(optional)_ | Anthropic secret key |
| `CEREBRAS_API_KEY` | _(optional)_ | Cerebras secret key |
| `LISTEN_ADDR` | `127.0.0.1:8095` | Server listen address |
| `CHAT_DB_PATH` | `examples/100-ai-chat-wizard/bin/runtime/chat_history.db` | SQLite database path for auth and conversation persistence |
| `CHAT_USAGE_PREMIUM_PERCENT` | `5` | Premium percentage added on top of usage-based costs for account total display |

---

## What this example demonstrates

- **gRPC over WebSocket in the browser** — no HTTP polling, no custom wire format
- **Server-streaming RPC** — tokens flow from OpenAI → gRPC server → browser in real time
- **Go WASM UI** — the entire frontend is Go; no JavaScript application code
- **Hooks pattern for async state** — `UseState`, `UseEffect`, `UseRef`, goroutines
- **Companion submodule pattern** — `GoGRPCBridge` consumed via `third_party/` + `go.mod replace`

