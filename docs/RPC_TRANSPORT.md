# Typed RPC and Streaming Transport

This page defines where typed browser RPC belongs in GoWebComponents and what boundary it should respect before any unary or streaming client ships.

## At A Glance

- Typed browser RPC is not part of the core rendering contract.
- It should not expand the base `interop` package into a framework-owned RPC stack by default.
- If the repo ships a first-party browser RPC client, it should live in a dedicated companion package layered on documented public APIs.
- The first transport package should start as `Experimental`, not as stable core or an automatic app default.

## Decision

Typed browser RPC for Go/WASM clients belongs outside core `interop` and outside the base rendering packages.

The intended home is a dedicated first-party companion package that can layer on:

- browser transport primitives from `interop`
- request and caching composition from `fetch` where appropriate
- route and hydration boundaries from `router` and `ui`
- diagnostics and inspection hooks from `devtools` and the shared logging model

That keeps protobuf-backed unary and streaming RPC in the "transport substrate" category rather than treating it as a universal framework primitive every app must adopt.

## Why It Stays Out Of Core

Core should own reusable browser UI and runtime primitives. Typed RPC over WebSocket or a similar tunnel does not meet that bar yet.

It remains outside core because:

- transport choice is still open and materially affects the client contract
- protobuf and code-generation workflow are opinionated integration decisions, not required runtime primitives
- auth, session attachment, reconnect policy, and stream lifecycle rules are application-sensitive
- many GoWebComponents apps can rely on ordinary HTTP handlers, `fetch`, forms, route loaders, or bootstrap payloads without needing a persistent RPC channel
- the repo already prefers companion packages for optional, policy-heavy integrations that build on stable public APIs

## Core Versus Companion Ownership

Core and `interop` should continue to own:

- browser capability wrappers such as WebSocket, worker, storage, and event primitives when they are broadly reusable
- SSR and hydration ownership rules
- structured errors, logging, and diagnostics seams that multiple packages can reuse

A dedicated RPC companion package should own:

- protobuf client-binding workflow and generated-stub expectations
- unary and streaming call APIs
- connection ownership, reconnect, and handshake policy
- auth and metadata propagation rules
- cancellation, timeout, and backpressure semantics
- RPC-specific diagnostics, devtools views, and examples

## Stability Expectation

If a first-party RPC companion package lands, it should start as `Experimental`.

It should only graduate toward `Supported companion` after all of the following are documented and validated:

- browser transport contract
- code-generation and client-binding workflow
- auth and metadata propagation
- cancellation and backpressure behavior
- SSR and hydration boundary rules
- diagnostics and actionable error surface
- at least one production-shaped unary plus streaming example

## Browser Transport Contract

The first browser transport contract should be explicit, connection-oriented, and WebSocket-backed.

That means:

- one browser RPC client owns one explicit WebSocket connection at a time
- the connection is created by application code or an app-owned service, not hidden behind route registration or component mount side effects
- components and routes should consume a shared client handle or call factory rather than opening their own sockets opportunistically
- the transport owns framing, correlation ids, and connection state; application code owns retry policy and UI behavior

## Connection Ownership

Connection ownership should be long-lived relative to one route component.

Recommended default:

- open the transport from an app shell, service container, or feature-scoped provider
- reuse that transport across related unary and streaming calls while the owning shell remains mounted
- close it explicitly on logout, tenant switch, auth invalidation, or app teardown
- do not treat navigation between sibling routes as a reason to churn the socket unless the auth or tenant boundary changed

This keeps connection lifecycle aligned with auth and operator context rather than with whichever component happened to render first.

## Handshake Behavior

The first call on a fresh socket should be a transport handshake before normal RPC traffic starts.

The handshake should establish:

- protocol version
- declared service or method namespace compatibility
- supported encodings and streaming modes
- connection-scoped auth or session metadata
- server acceptance, rejection, or downgrade of optional capabilities

Rules:

- no unary or streaming RPC should start until the handshake succeeds
- handshake failure should close the transport and surface one typed connection error to the owner
- auth rejection must surface distinctly from transport unavailability or protocol mismatch
- reconnect attempts must perform the full handshake again rather than assuming prior connection state still applies

## RPC Shape Semantics

The browser contract should support four logical RPC shapes:

- unary: one request message, one terminal response message
- server streaming: one request message, zero or more streamed response messages, then one terminal completion
- client streaming: one stream open, zero or more client messages, one explicit client half-close, then one terminal response
- bidirectional streaming: one stream open followed by independent client-send and server-send traffic until one side completes or cancels

Each in-flight RPC or stream should have:

- one stream or call id
- one declared method
- one terminal outcome only: success, application error, transport error, or cancellation
- ordered messages within that stream id

The contract should not promise ordering across different stream ids.

## Reconnect Expectations

Reconnect should be conservative by default.

The first supported rule set should be:

- if the socket drops, all in-flight unary, client-streaming, and bidirectional calls fail with a typed transport-closed error
- server streams also fail by default unless the application explicitly opts into a resumable higher-level pattern later
- the transport may reconnect for future calls, but it must not silently replay mutations or client-streamed writes
- safe retry remains application-owned and should be explicit at the call site or service layer

This avoids the worst failure mode: a framework transport that guesses replay safety for writes it does not understand.

## Multiplexed Tunnel Pattern

Applications that route auth, unary RPC, and long-lived streaming calls through one gRPC-web or WebSocket-backed tunnel should treat that tunnel as one shell-owned connection, not as a per-component resource.

Recommended lifecycle:

1. establish the physical tunnel once when the owning WASM shell or service boundary initializes
2. complete the auth or session handshake before any unary or streaming RPC starts
3. reuse the same connection for ordinary API calls, background session refresh, and long-lived streams such as chat token output or speech synthesis
4. close the tunnel explicitly on logout, tenant switch, terminal auth failure, or shell teardown

Rules:

- components should receive a shared client handle instead of opening their own tunnels opportunistically
- silent token refresh should update the tunnel's effective auth context without forcing every component to remount or re-open its stream manually
- if the server cannot preserve stream continuity across an auth refresh, the reconnection or resubscription policy must stay at the application or service layer rather than being guessed by the transport
- sign-out must cancel all active streams, release server-side subscriptions, and reject new calls until a fresh authenticated handshake succeeds
- tunnel reuse is for connection efficiency and coherent auth ownership, not for hiding business-level retry policy

The current production-shaped proof point is `examples/100-ai-chat-wizard`, where one reconnect-aware browser tunnel carries session-aware unary calls plus long-lived chat and speech streams without introducing a second bespoke browser protocol.

## Closure And Error Surfacing

Transport closure must surface in two places:

- connection-level state for shells, diagnostics views, and retry controls
- per-call or per-stream terminal errors for the specific consumer waiting on that RPC

Expected closure outcomes:

- normal server completion ends a unary call or stream without being treated as a transport failure
- server-declared application errors terminate only the affected call or stream
- handshake rejection closes the whole connection and surfaces a typed auth, protocol, or capability error
- browser disconnects, socket close, and protocol corruption close the connection and fail all active calls
- late messages received after cancellation, terminal completion, or connection teardown are ignored and counted as diagnostics noise rather than reviving the stream

## Protobuf Code Generation And Client Binding Workflow

The code-generation workflow should stay explicit and repo-owned.

Recommended layout:

- keep `.proto` definitions under an app-owned `proto/` directory
- generate ordinary Go files next to the source proto with source-relative paths
- import the generated message and service packages from both server code and browser `js/wasm` client code as normal Go packages
- keep transport-specific client binding helpers in the future RPC companion package or in generated companion-owned stubs, not in `ui`, `router`, or `interop`

The current reference shape in this repo is `examples/100-ai-chat-wizard/proto/`, which keeps:

- `chat.proto` as the source of truth
- `chat.pb.go` for generated message types
- `chat_grpc.pb.go` for generated service and client stubs

## Workflow Rules

The intended workflow is:

1. edit the `.proto` contract
2. run an explicit regeneration command or checked-in script
3. review the generated Go diff in version control
4. run the normal `gwc dev`, `gwc build`, `gwc test`, or `gwc verify` flow against the generated code as ordinary source input

Rules:

- `gwc dev`, `gwc build`, and `gwc test` should not hide protobuf generation as an implicit side effect
- the application or companion package should own the regeneration command explicitly through a script, `go:generate`, or a future `gwc generate rpc` command if one is added later
- generated code should be committed for reproducible builds and reviewable contract diffs unless a future launcher contract explicitly replaces that rule
- tool versions for `protoc` and the relevant Go plugins should be pinned in app docs, scripts, or CI rather than discovered ad hoc on one developer machine

## Browser-Safe Binding Rules

Generated protobuf message types are acceptable in browser `js/wasm` builds as ordinary Go code.

The browser-specific binding layer should add:

- connection bootstrap and handshake
- stream lifecycle and correlation ids
- context-driven cancellation and deadlines
- typed conversion from generated request or response types into the chosen transport framing

That binding layer should remain separate from the generated message package so the app can regenerate schema types without rewriting UI components around transport details.

## Auth, Session, And Metadata Propagation

The default RPC auth model should match the rest of the repo: prefer same-origin server authority and avoid pushing raw credentials into browser-owned code unless a deployment explicitly requires it.

Recommended default:

- same-origin cookie or session-backed auth should be the first supported mode
- the server should resolve user identity during the WebSocket handshake, not from ad hoc UI state
- reconnect should re-evaluate the current session or cookie state rather than assuming the old connection identity is still valid
- logout, tenant switch, or session-expiry events should close the transport and force a fresh handshake before more RPC traffic starts

Bearer tokens are an explicit escape hatch, not the baseline recommendation.

If an app uses bearer auth:

- token sourcing and refresh remain application-owned
- the token provider should attach credentials at connection bootstrap or per-call metadata through one explicit hook, not by letting components set arbitrary headers
- raw bearer values must not be copied into bootstrap payloads, persisted browser storage, offline queues, or framework-owned logs and diagnostics

## Mutation Protection And Metadata Rules

Cookie-backed RPC mutations need the same conservative posture as other browser write paths.

Recommended rules:

- validate `Origin` or equivalent server-side handshake policy for cookie-authenticated socket upgrades
- treat mutation-capable RPC methods as server-authoritative writes, not as trusted because the browser already has a socket
- when the server requires CSRF-adjacent mutation proof beyond cookie auth, attach that proof through explicit transport metadata instead of leaking it into generated request bodies
- refresh or replace any per-connection mutation token when the server rotates session state, and fail closed on `unauthenticated`, `permission denied`, or token-mismatch responses

Metadata propagation should be explicit and allowlisted.

Connection-scoped metadata may carry:

- auth mode or session context derived by the server
- tenant or workspace selection
- locale, correlation id, or client-version hints

Per-call metadata may carry:

- request-specific idempotency or retry keys
- per-call tenant overrides when the application explicitly supports them
- operation-scoped tracing or correlation ids

The important split is identity versus request variance:

- handshake-time or connection-scoped metadata should represent the current
  authenticated session identity and any default tenant or workspace context the
  whole tunnel is allowed to assume
- per-call metadata should be limited to values that may legitimately differ
  between two RPCs on the same authenticated tunnel, such as idempotency keys,
  tracing ids, or an explicit tenant override the server contract already allows
- do not cache per-call auth hints or role selections into connection-scoped
  metadata if those values may change after silent token refresh, tenant switch,
  or user-driven workspace changes
- when the authenticated identity itself changes, the transport should tear down
  the old tunnel and perform a fresh handshake instead of trying to mutate the
  connection into a new principal in place

The companion package should not expose a generic "forward every browser header" escape hatch.

## Redaction And Identity Boundaries

Auth and metadata propagation must respect the same redaction rules already documented for forms, SSR, logging, and observability.

Rules:

- never emit raw cookies, bearer tokens, CSRF-style mutation tokens, or full auth metadata into framework-owned logs, diagnostics, traces, or devtools snapshots
- keep correlation ids opaque and non-user-identifying
- treat per-stream identity as fixed once the stream starts; identity changes require a new stream or a new connection
- do not persist session secrets, bearer tokens, or mutation tokens in browser storage just to make reconnect easier

The safe default is to surface only coarse auth state in tooling, such as "connected", "unauthenticated", or "permission denied", plus non-secret tenant or correlation identifiers when the application already treats them as safe to expose.

## Cancellation, Timeouts, And Backpressure

The first RPC companion should be context-driven, like the rest of the repo's long-running async work.

Recommended rules:

- every unary call and stream should start with a caller-owned `context.Context`
- `context.WithCancel(...)` cancels local waiting immediately and sends a best-effort stream cancel or half-close frame when the socket is still available
- `context.WithTimeout(...)` or deadline contexts should propagate an explicit deadline into transport metadata and fail locally with a typed timeout when the deadline expires
- if the transport is already closed, local cancellation still wins and should not wait on remote acknowledgement

Late traffic should be ignored after the owner stops listening.

That means:

- once a unary call resolves, times out, or is cancelled, later response frames for that call id are dropped
- once a server-stream consumer unsubscribes or the owning route unmounts, later stream frames are ignored and counted only as diagnostics noise
- client-streaming and bidirectional calls should reject further sends after local cancellation, local timeout, or terminal completion

## Buffering And Flow Control Expectations

The transport must stay bounded by default.

The first supported contract should be:

- no unbounded outbound message queue per connection
- no unbounded per-stream replay buffer in the browser
- control traffic such as handshake, cancel, close, and terminal status must not be starved behind an unlimited data queue
- when a connection-level or stream-level buffer limit is reached, the transport should fail the affected call or stream with a typed backpressure error rather than silently dropping messages or growing memory without bound

Backpressure policy should stay explicit and conservative:

- unary calls may wait briefly for write capacity, but should still respect caller cancellation and deadlines
- client-streaming and bidirectional sends should surface backpressure to the sender rather than pretending every `Send(...)` can always enqueue
- server-stream consumers should process frames on a dedicated receive path and hand off only the needed state to UI updates, instead of doing expensive work inline on the socket reader
- if one stream becomes hopelessly slow, the preferred first response is to fail that stream, not to freeze unrelated traffic on the same connection

The framework should not claim a sophisticated adaptive flow-control algorithm in the first version. The important first guarantee is bounded memory plus clear typed failure when producers outrun consumers.

## SSR And Hydration Boundaries

Live tunneled RPC is a hydrated-browser concern, not an SSR transport.

The current recommended split is:

- SSR route handlers and route loaders own first paint
- `ui.SSRBootstrap` owns public resume data needed for hydration
- the RPC transport attaches after hydration commit or from a browser-only route that never had SSR in the first place

That means the server should render one stable initial answer first, then the browser may attach live RPC for revalidation, incremental updates, or long-lived streams after resume.

## Initial Data Ownership

The first hydrated screen should not have two competing authorities for the same initial snapshot.

Recommended rule:

- if a route already rendered with SSR loader data or bootstrap cache seeds, use that snapshot for first paint
- after hydration, the RPC client may either subscribe for later updates or explicitly revalidate, depending on the route's reuse policy
- do not automatically re-request the exact same initial payload during hydration just because a live RPC client exists
- do not serialize live stream state, auth credentials, or transport handles into bootstrap to make RPC feel "already connected"

The important boundary is: bootstrap carries public resumable data, while RPC carries post-hydration live interaction and updates.

## Attachment Timing

RPC attachment should happen after hydration has committed the owning subtree.

Recommended timing rules:

- open long-lived route-scoped streams from effects or service initialization that runs after hydration commit
- cancel route-scoped streams when the owning route or panel unmounts
- keep app-shell-wide connections alive across sibling route navigation when the auth and tenant boundary is unchanged
- when strict hydration fails and a subtree falls back to client render, attach RPC only after the fallback subtree becomes the active client-owned tree

This matches the existing hydration rule that effects and subscriptions attach after the hydrated commit rather than during DOM matching.

## Loader And RPC Composition

Route loaders and RPC streams should compose by responsibility rather than racing each other.

Use this split:

- route loaders fetch the initial route snapshot needed for server render and hydration reuse
- RPC streams provide ongoing updates, push events, or operator actions after the page is interactive
- mutation RPCs invalidate or refresh the affected loader-owned data explicitly instead of assuming the route snapshot mutates itself magically
- if a route is browser-only and intentionally skips SSR, the RPC client may become the first authority for that screen, but that is a client-only route choice rather than an SSR feature

The framework should not treat live RPC as a replacement for route loaders, bootstrap, or hydration. It is a later layer on top of those existing ownership rules.

## Observability, Diagnostics, And Actionable Errors

Typed RPC transport should plug into the repo's existing logging, diagnostics, and devtools model instead of inventing a parallel debugging surface.

Recommended logging and event rules:

- use stable `rpc` domain events for connection open, handshake success, handshake failure, reconnect attempt, reconnect success, reconnect failure, stream open, stream close, backpressure, decode failure, and auth rejection
- carry opaque correlation ids so one route interaction, reconnect loop, or mutation failure can be joined back to surrounding SSR, hydration, router, or fetch events
- prefer safe structured fields such as method, stream id, connection state, close class, duration, message counts, queue-depth buckets, and retryable booleans over raw payload dumps
- keep auth mode, tenant id, or peer identity visible only when those values are already safe to expose under the application's existing policy

## Diagnostic Contract

The first RPC companion should emit stable diagnostic identifiers for at least:

- handshake failure
- auth rejection
- protocol or schema mismatch
- stream restart
- decode failure
- backpressure overflow
- transport-closed while work was in flight

Each diagnostic should carry:

- stable code
- short message
- recoverable flag
- remediation text
- docs link back to this transport guide or a future focused troubleshooting page

This follows the same model already used by runtime, router, hydration, and interop diagnostics.

## Devtools Surface

The in-browser inspection story should reuse `devtools` snapshots and panels.

The minimum useful RPC inspection state is:

- current connection state
- resolved transport and protocol version
- current auth state class such as connected, unauthenticated, or permission denied
- active stream count and per-stream method or lifecycle phase summaries
- recent diagnostics and restart counts
- bounded log history for the last connection, handshake, and stream failures

Devtools should not expose raw headers, cookies, bearer tokens, full metadata maps, or full protobuf payload bodies by default.

## Actionable Error Shape

Actionable RPC failures should tell the caller what to do next.

Expected guidance:

- handshake failure: check protocol version, route, upgrade path, and auth state
- auth rejection: refresh session, re-authenticate, or verify the same-origin cookie or token source
- schema mismatch: regenerate stubs or roll back the client or server version skew
- backpressure failure: reduce concurrent sends, batch updates, or split high-volume streams
- decode failure: inspect the method contract and generated types before retrying blindly

The goal is not just transport visibility. The goal is to make RPC failures debuggable through the same structured surfaces already used elsewhere in the repo.

## Production-Shaped Reference Example

The current proof point for unary plus streaming browser RPC is `examples/100-ai-chat-wizard`.

It is the reference example because it already combines:

- checked-in protobuf contracts under `examples/100-ai-chat-wizard/proto/`
- unary RPC calls for session, profile, memory, and conversation management
- server-streaming chat output over a WebSocket-backed gRPC tunnel
- additional streaming output for text-to-speech chunks
- browser-side reconnect handling, auth-aware RPC flows, and a production-shaped Go server instead of a toy echo endpoint

That makes it good evidence for the transport value proposition:

- one contract format for both unary and streaming work
- browser-delivered interactive updates without inventing a separate bespoke wire protocol
- a realistic example where auth, persistence, streaming UI, and operational concerns all matter at once

This example does not mean a first-party RPC companion package already exists.

It does mean the repo now has one coherent reference application that proves the product value of typed unary plus streaming RPC for Go/WASM clients before a companion package is formalized.

## Candidate Transport Note

The backlog currently points at a `grpc-tunnel`-style unary plus streaming transport as a candidate implementation substrate.

That candidate may inform the first package shape, but it does not change the ownership decision above: browser RPC should prove itself as a companion package before any part of it is treated as core framework surface.
