# Chat Request Lifecycle

The chat request lifecycle starts with a composer draft and ends with persisted messages, usage metadata, a canonical thread route, and replayable conversation state.

## Happy Path

```text
composer text
  |
  v
client/app/stream.go
  |-- ignore empty text
  |-- block concurrent send
  |-- append user message locally
  |-- call ChatService.Send
  v
server/app/server.go
  |-- require authenticated user
  |-- authorize entitlement and workspace state
  |-- create or resolve conversation
  |-- resolve provider/model
  |-- stream provider events as ChatChunk
  |-- persist messages and usage
  v
client thread
  |-- apply deltas
  |-- render assistant answer incrementally
  |-- normalize /app/thread/:publicID
  |-- refresh conversation summary
```

The client sends the current history, draft text, selected model, tone, and thinking settings. The server owns the durable conversation. A draft route becomes a public conversation route only after the server creates or resolves a conversation and returns the public id.

## Draft State

Draft state is local until a send succeeds. Starter prompts, quote insertion, fork-from-message, and composer edits prepare the next request but do not create a durable conversation by themselves. The app clears or preserves local draft state based on route actions so `/app` can represent a new unsaved chat while `/app/thread/:publicID` represents a replayable server conversation.

## Server Handling

The server receives `Send` through the typed `ChatService`. The handler validates the session, checks billing/entitlement gates, loads or creates conversation rows, and builds the provider request. SQL-backed rows in `conversations`, `messages`, and `usage_events` become the durable replay and reporting source.

Server policy is intentionally outside the raw SQL files. Entitlement decisions, provider capability checks, safe error wrapping, and route/public-id behavior live in Go handlers and helper services so focused tests can assert each branch.

## Provider Selection

The provider registry normalizes OpenAI, Anthropic, Cerebras, and stub-provider behavior behind provider interfaces. The model catalog exposes selectable options, but send-time resolution still checks whether the requested model exists, whether the provider is available, and whether the requested capabilities such as thinking or speech are supported.

Provider adapters convert upstream streaming formats into normalized provider events. That lets the client process one `ChatChunk` stream instead of handling provider-specific SSE or chat-completion payloads.

## Streaming

The server streams assistant deltas as `ChatChunk` events. The client applies deltas progressively, tracks visible response timing, and updates assistant message state without waiting for the final response. A completed stream carries final stats and enough route/conversation metadata for the client to normalize the URL and refresh summaries.

The background worker may help render markdown, thought/canvas metadata, or artifact previews after chunks arrive. Worker failure should affect rendering affordances, not the integrity of the persisted conversation.

## Route Normalization And Replay

Thread routes use a public id: `/app/thread/:publicID`. The server stores the internal numeric id and public id; the client resolves route ids through conversation summaries and route-resolution RPCs. When a send starts from `/app`, the route should normalize to the new public thread after the server response provides one.

Replay loads conversation history through typed RPCs and renders stored messages. Refreshing or reopening a thread should not depend on the previous in-memory stream state.

## Notable Failure Branches

| Branch | Intended behavior |
|---|---|
| Empty draft | No send and no durable conversation. |
| Already streaming | Block the second send to preserve ordering. |
| gRPC unavailable | Show reconnect-safe client error and keep draft/thread state. |
| Auth expired or revoked | Route through auth failure handling and clear private account state. |
| Entitlement or billing block | Return customer-safe denial copy with operator context server-side. |
| Provider unavailable or timeout | Surface one assistant error, log provider/runtime metadata, and avoid hanging the stream. |
| Route public id mismatch | Resolve requested thread route or suppress unsafe normalization with diagnostics. |
| Persistence failure | Fail closed where durable state is required; avoid silently claiming a replayable thread exists. |
