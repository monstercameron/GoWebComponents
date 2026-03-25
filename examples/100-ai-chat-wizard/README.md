# 100 — AI Chat Wizard

A streaming AI chat interface built with **Go WASM + gRPC + GoGRPCBridge + OpenAI**.

The client and server are both written in Go. The browser-side code compiles to WASM. Messages travel
over a gRPC stream tunnelled through a WebSocket connection, so there is no separate HTTP polling—tokens
from OpenAI stream straight to the browser as they are generated.

---

## Architecture

```
Browser (Go WASM)                         Go Server (port 8095)
┌─────────────────────────────┐           ┌──────────────────────────────────┐
│  App() — React-like UI      │           │  HTTP mux                        │
│                             │  WS /grpc │  ├── GET /           static HTML  │
│  grpctunnel.Dial("/grpc")   │◄─────────►│  ├── /grpc          gRPC tunnel   │
│  ChatServiceClient.Send()   │  gRPC     │  │    (GoGRPCBridge.Wrap)         │
│                             │  stream   │  └── GET /healthz   probe         │
│  stream.Recv() → delta      │           │                                   │
│  appendDelta → re-render    │           │  gRPC server (in-process)         │
└─────────────────────────────┘           │  ChatService.Send()               │
                                          │   ↓ calls OpenAI /chat API        │
                                          │   ↓ maps SSE chunks → gRPC stream │
                                          └──────────────────────────────────┘
```

**Data flow**

1. User submits a message in the browser.
2. WASM calls `ChatService.Send(SendRequest{history, message})` over the gRPC tunnel.
3. The GoGRPCBridge WebSocket handler forwards the gRPC call (HTTP/2-over-WebSocket) to the in-process gRPC server.
4. `chatServer.Send()` calls the OpenAI Responses API streaming endpoint.
5. Each SSE `data:` line from OpenAI becomes one `ChatChunk{delta}` gRPC stream message.
6. The WASM client receives each chunk via `stream.Recv()`, appends the token to the UI, and triggers a re-render.
7. When OpenAI finishes, a `ChatChunk{done: true}` sentinel closes the stream.

---

## Prerequisites

| Tool | Purpose |
|---|---|
| Go ≥ 1.25 | Build server and WASM client |
| `protoc` | Proto compiler (generate Go stubs) |
| `protoc-gen-go` | Go protobuf plugin |
| `protoc-gen-go-grpc` | Go gRPC plugin |
| Tailwind CSS (pre-built) | Served from `examples/static/css/tailwind.css` |

The `third_party/GoGRPCBridge` git submodule must be initialised:

```powershell
git submodule update --init --recursive
```

---

## Quick start

### 1. Set at least one provider API key

```powershell
$env:OPENAI_API_KEY = "sk-..."
$env:CEREBRAS_API_KEY = "csk-..."
```

Anthropic is also supported:

```powershell
$env:ANTHROPIC_API_KEY = "sk-ant-..."
```

### 2. Build the WASM client

```powershell
.\examples\100-ai-chat-wizard\scripts\build-client.ps1
```

On macOS/Linux:

```bash
./examples/100-ai-chat-wizard/scripts/build-client.sh
```

That command builds both raw WASM artifacts and their Brotli sidecars:

- `examples/100-ai-chat-wizard/bin/client/app/chat.wasm`
- `examples/100-ai-chat-wizard/bin/client/app/chat.wasm.br`
- `examples/100-ai-chat-wizard/bin/client/worker/background-worker.wasm`
- `examples/100-ai-chat-wizard/bin/client/worker/background-worker.wasm.br`

The browser requests `app/chat.wasm` and `worker/background-worker.wasm`. The server transparently serves the `.br`
sidecars with `Content-Encoding: br` when the browser advertises Brotli support, and falls back to the raw WASM
files otherwise. For explicit testing, open `http://127.0.0.1:8095/?br=true` to force the Brotli sidecars for
the shell-loaded WASM assets.

### 3. Copy `wasm_exec.js` (first time only)

```powershell
Copy-Item "$(go env GOROOT)/misc/wasm/wasm_exec.js" examples/static/js/wasm_exec.js
```

### 4. Start the server

```powershell
.\examples\100-ai-chat-wizard\scripts\run-server.ps1
```

Open **http://127.0.0.1:8095/** in your browser.

The server now serves one GWC shell for `/`, `/app`, thread deep links, and old auth/marketing entry routes. There are no separate server-rendered login, signup, or marketing HTML pages in this example anymore.

### 5. Seed local dev accounts

The server and the seeder must use the same `CHAT_DB_PATH`.

By default, the server uses `examples/100-ai-chat-wizard/bin/runtime/chat_history.db`, while `cmd/seed-test-db` defaults to `examples/100-ai-chat-wizard/bin/runtime/test_chat.db`.

If you want known login credentials for local testing, export one shared path first and then run the seeder and server against that same file:

```powershell
$env:CHAT_DB_PATH = "examples/100-ai-chat-wizard/bin/runtime/test_chat.db"
go run ./examples/100-ai-chat-wizard/cmd/seed-test-db
.\examples\100-ai-chat-wizard\scripts\run-server.ps1
```

The seeder creates these local accounts:

- `demo@example.com / password123`
- `admin@example.com / password`

Use the email address exactly as shown above when signing in. The auth UI is email-based, so there is no separate username-only `admin` login.

### 6. Enable provider switching without real API keys

If you want to exercise provider and model switching locally without live upstream credentials, export `CHAT_PROVIDER_STUBS` before starting the server:

```powershell
$env:CHAT_PROVIDER_STUBS = "all"
.\examples\100-ai-chat-wizard\scripts\run-server.ps1
```

The local stub mode keeps the OpenAI, Anthropic, and Cerebras model catalog entries available inside the running shell and returns deterministic stub replies, so you can test provider switches and model-picker behavior without restarting the server or burning rate-limited API calls.

### Optional: one-command local dev

This builds the client and then starts the server:

```powershell
.\examples\100-ai-chat-wizard\scripts\dev.ps1
```

On macOS/Linux:

```bash
./examples/100-ai-chat-wizard/scripts/dev.sh
```

---

## File layout

```
examples/100-ai-chat-wizard/
├── scripts/
│   ├── build-client.ps1  # Windows wrapper for the client build
│   ├── run-server.ps1    # Windows wrapper for the example server
│   └── dev.ps1           # Windows build + run entrypoint
├── client/
│   ├── main.go       # Thin WASM launcher
│   ├── app/
│   │   ├── app.go    # Chat shell component tree
│   │   └── state.go  # Reducer-backed local app state
│   └── backgroundworker/
│       └── main.go   # Secondary Go WASM worker for markdown + background ticks
├── proto/
│   ├── chat.proto    # Service definition (source of truth)
│   ├── chat.pb.go    # Generated message types
│   └── chat_grpc.pb.go  # Generated service stubs
└── server/
    ├── main.go       # Thin server launcher
    ├── app/
    │   ├── server.go # HTTP + gRPC-over-WebSocket server + OpenAI proxy
    │   └── store.go  # SQLite-backed persistence layer
    └── provider/
        └── ...       # LLM provider adapters
```

---

## Regenerating proto stubs

```powershell
cd examples/100-ai-chat-wizard
protoc --go_out=. --go_opt=paths=source_relative `
       --go-grpc_out=. --go-grpc_opt=paths=source_relative `
       proto/chat.proto
```

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

For a fuller explanation of the output fields, interpretation, and latest measured sample data, see [BENCHMARKS.md](C:/Users/Cam/Desktop/GoWebComponents/examples/100-ai-chat-wizard/BENCHMARKS.md).

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

---

## What this example demonstrates

- **gRPC over WebSocket in the browser** — no HTTP polling, no custom wire format
- **Server-streaming RPC** — tokens flow from OpenAI → gRPC server → browser in real time
- **Go WASM UI** — the entire frontend is Go; no JavaScript application code
- **Hooks pattern for async state** — `UseState`, `UseEffect`, `UseRef`, goroutines
- **Companion submodule pattern** — `GoGRPCBridge` consumed via `third_party/` + `go.mod replace`
