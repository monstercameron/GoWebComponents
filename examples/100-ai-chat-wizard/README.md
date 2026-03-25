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

### 1. Set your OpenAI API key

```powershell
$env:OPENAI_API_KEY = "sk-..."
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
| `OPENAI_API_KEY` | _(required)_ | OpenAI secret key |
| `LISTEN_ADDR` | `127.0.0.1:8095` | Server listen address |
| `CHAT_DB_PATH` | `examples/100-ai-chat-wizard/bin/runtime/chat_history.db` | SQLite database path for auth and conversation persistence |

---

## What this example demonstrates

- **gRPC over WebSocket in the browser** — no HTTP polling, no custom wire format
- **Server-streaming RPC** — tokens flow from OpenAI → gRPC server → browser in real time
- **Go WASM UI** — the entire frontend is Go; no JavaScript application code
- **Hooks pattern for async state** — `UseState`, `UseEffect`, `UseRef`, goroutines
- **Companion submodule pattern** — `GoGRPCBridge` consumed via `third_party/` + `go.mod replace`
