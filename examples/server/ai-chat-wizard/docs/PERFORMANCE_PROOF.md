# Example 100 Performance Proof

This document states the runtime claims Example 100 is allowed to make and how maintainers verify them locally. It complements `docs/PERFORMANCE.md`, which covers server-side synthetic benchmarks.

## Runtime Claims

Example 100 claims:

- Fast public first paint for `/`, `/home`, `/pricing`, and `/signup`.
- Stable authenticated shell startup for `/app` and `/app/thread/:publicID`.
- Measurable streamed first token for the first chat send.
- Worker-backed markdown and metadata isolation with a main-thread fallback.
- Paged sidebar behavior that remains usable under deterministic seeded conversation counts.
- Dashboard slice responsiveness for Business, Customers, Chats, Providers, and Ops.

These are local demo claims, not production SLOs. Record machine, browser, commit, database seed, and provider mode with every run.

## Build And Serve

From the repo root:

```powershell
go run ./tools/gwc build -app .\examples\server\ai-chat-wizard\client\main.go -root .\examples\server\ai-chat-wizard\client -out .\examples\server\ai-chat-wizard\bin\client\chat.wasm -json
go run ./tools/gwc build -app .\examples\server\ai-chat-wizard\client\backgroundworker\main.go -root .\examples\server\ai-chat-wizard\client\backgroundworker -out .\examples\server\ai-chat-wizard\bin\client\worker\background-worker.wasm -json
$env:CHAT_DB_PATH = Join-Path $env:TEMP "gwc-chatwizard-perf-proof.db"
go run ./examples/server/ai-chat-wizard/cmd/seed-test-db
go run ./examples/server/ai-chat-wizard/cmd/server
```

Use the local stub provider for deterministic first-pass measurements. Repeat one first-chat trace with a real-provider-compatible configuration before claiming upstream behavior.

## Cold-Start Checklist

Measure `/`, `/login`, `/app`, and `/app/dashboard/ops`.

For each route record:

- `shell_render_ms`: navigation start to first stable app shell content.
- `wasm_ready_ms`: navigation start to WASM app mounted.
- `tunnel_ready_ms`: navigation start to bridge state `ready`.
- `first_useful_interaction_ms`: navigation start to first usable action: public CTA, login submit, composer focus, or dashboard filter.

Browser steps:

1. Open Chrome DevTools Performance panel.
2. Disable cache for the first run; keep cache enabled for the warm run.
3. Start recording, navigate directly to the route, stop after the first useful interaction is possible.
4. Save the trace with route, cache mode, and commit in the file name.

Command-line smoke:

```powershell
go test ./examples/server/ai-chat-wizard/server/app -run TestRuntimeRouteHelpers -v
go test ./examples/server/ai-chat-wizard/cmd/server -run Test -v
```

Initial guardrails:

- Public first paint: warn above `1200 ms`, fail demo-ready above `2000 ms`.
- Authenticated shell render: warn above `1800 ms`, fail above `3000 ms`.
- Bridge/tunnel ready: warn above `2500 ms`, fail above `5000 ms`.
- First useful interaction: warn above `3000 ms`, fail above `6000 ms`.

## First-Chat Trace

Measure composer submit to first chunk and completed reply.

Paths:

- Stub provider: deterministic local provider for CI/manual smoke.
- Real-provider-compatible: same code path with live credentials or a provider shim that preserves upstream network/model timing categories.

Record:

- `submit_to_rpc_start_ms`
- `rpc_start_to_server_handler_ms`
- `server_handler_to_model_start_ms`
- `model_start_to_first_chunk_ms`
- `first_chunk_to_complete_ms`
- `client_render_apply_ms`

Local commands:

```powershell
go test ./examples/server/ai-chat-wizard/server/provider -run 'TestStubProvider|TestProviderStreaming' -v
go test ./examples/server/ai-chat-wizard/server/app -run 'Test.*Send|TestSendSLASweep' -v
```

Guardrail:

- Stub first chunk: warn above `800 ms`, fail above `1500 ms`.
- Stub completed reply: warn above `2500 ms`, fail above `5000 ms`.
- Real-provider-compatible first chunk must be recorded with provider, region, model, and network notes instead of compared blindly to stub thresholds.

## Dashboard Slice Responsiveness

Routes:

- `/app/dashboard/business`
- `/app/dashboard/customers`
- `/app/dashboard/chats`
- `/app/dashboard/providers`
- `/app/dashboard/ops`

Smoke steps:

1. Seed the perf database.
2. Sign in as `admin@email.com / password`.
3. Navigate directly to each route.
4. Apply one filter or drill-down action where available.
5. Record shell stability, load state duration, table render duration, denied/empty behavior if role-scoped, and whether interactions remain keyboard reachable.

Go proof:

```powershell
go test ./examples/server/ai-chat-wizard/server/app -run 'Test.*Admin.*|Test.*Dashboard.*|Test.*Superuser.*' -v
```

Guardrail:

- Initial seeded dashboard slice load: warn above `1500 ms`, fail above `3000 ms`.
- Filter/drill-down response under seeded state: warn above `500 ms`, fail above `1000 ms`.

## Sidebar Scale Proof

Use deterministic seeded conversation counts. The first pass target is `250` conversation rows.

Verify:

- Conversation list pagination or grouping remains bounded.
- Scroll restoration returns to the active thread after route changes.
- Route switching between `/app`, `/app/thread/:publicID`, and `/app/settings` does not rebuild the whole workspace shell.

Go proof:

```powershell
GOOS=js GOARCH=wasm go test ./examples/server/ai-chat-wizard/client/app -run 'Test.*Sidebar|Test.*Route|Test.*Scroll' -v
```

Guardrail:

- Sidebar route switch under seeded count: warn above `300 ms`, fail above `750 ms`.
- Scroll restoration must be deterministic; a missed restore blocks demo-ready even if timing passes.

## Worker Value Proof

Current offloaded work:

- Markdown render batches.
- Assistant thought/canvas metadata derivation.
- Thread cost summary derivation.
- Background maintenance tick/batch messages.

Fallback:

- Main-thread synchronous markdown render and metadata derivation remain available when the worker fails to start or a worker request fails.

Observe:

```powershell
GOOS=js GOARCH=wasm go test ./examples/server/ai-chat-wizard/client/app -run 'TestParse.*Worker|TestParse.*Metadata|TestParse.*ThreadCost' -v
GOOS=js GOARCH=wasm go test ./examples/server/ai-chat-wizard/client/app -run '^$' -bench 'BenchmarkRenderWorkerExample100|BenchmarkBuildAssistantMessageMetadata|BenchmarkBuildRenderWorker' -benchmem
```

Proof rule:

- Every worker claim must name the fallback behavior and one benchmark or smoke test that exercises the worker request shape.
- A worker failure may degrade responsiveness, but it must not break chat rendering.

## Performance Regression Checklist

Block demo-ready status when any of these fail:

- Public first paint exceeds `2000 ms`.
- Authenticated shell or deep-linked dashboard first useful interaction exceeds `6000 ms`.
- Stub first chat first chunk exceeds `1500 ms`.
- Dashboard seeded slice load exceeds `3000 ms`.
- Sidebar route switch loses scroll restoration or exceeds `750 ms`.
- Worker fallback breaks markdown render or assistant metadata.
- Any trace lacks route, commit, machine, browser, cache mode, seed, and provider mode.
