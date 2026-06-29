# Example 100 Bridge Churn Hardening

This document is the bridge-churn contract for Example 100. It covers the browser gRPC bridge used by chat, settings, dashboard, telemetry, diagnostics, and background maintenance.

## Bridge State Model

Client code exposes a typed bridge state in `client/app/bridge_policy.go` and stores it in `appState`:

| State | Meaning | User-facing behavior |
| --- | --- | --- |
| `booting` | WASM app is mounted but the bridge has not resolved. | Keep boot/auth UI stable; avoid side-channel RPC noise. |
| `ready` | Tunnel is usable for foreground RPCs. | Normal behavior. |
| `reconnecting` | Existing or recent bridge is trying to reconnect. | Keep current route/shell/modal state; defer best-effort work; show subtle retrying status. |
| `degraded` | Bridge is present or recently present but not healthy enough for reliable user work. | Preserve UI state; make saves/chat/dashboard show safe retry copy. |
| `sleeping` | Hidden/offline-idle tab intentionally paused the bridge. | Do not panic the user; resume on focus/visibility/online. |
| `offline` | Browser/runtime stopped or navigator reports offline. | Keep shell state local; queue only bounded telemetry-like work. |

`GRPCReady` remains as a compatibility boolean and is derived from `BridgeState == ready`.

## RPC Policy Table

The central policy table lives in `parseBridgeRPCPolicyFor`.

| RPC family | Traffic class | Criticality | Idempotent | Retry safe | Timeout | Reconnect behavior |
| --- | --- | --- | --- | --- | --- | --- |
| Chat send/stream | foreground stream | critical | no | no | stream-owned | Do not auto-retry; user can retry after a safe failed assistant message. |
| Conversation list refresh | foreground read | high | yes | yes | 6s | May run while reconnecting if client exists; otherwise wait for ready. |
| Settings writes | foreground mutation | high | not globally | no | 8s | Single-shot. Optimistic local close; late failure appears as toast/status. |
| Auth/session refresh | foreground read | critical | yes | yes | 6s | May retry because it is a read/refresh path. |
| Admin dashboard loads | foreground read | medium | yes | yes | 8s | May retry/defer; stale dashboard shell must remain mounted. |
| Telemetry relay | best effort | low | yes | yes | 3s | Bounded deferred queue during reconnect/offline; drop accounting required. |
| Diagnostics polling | background read | low | yes | yes | 4s | Skip/defer during churn; never block foreground UI. |
| Background maintenance | background maintenance | low | yes | yes | 4s | Stop worker ticker while bridge is unavailable; restart on ready. |

## Best-Effort Guard

Use `parseBridgeBestEffortDecision` for telemetry, diagnostics polling, dashboard refresh, catalog sync, and background maintenance-like work. The helper returns:

- `run`: bridge state and policy permit the RPC.
- `defer`: bounded queue may hold the work.
- `skip`: drop the work because it is not user-critical or not safe to replay.

Do not use this helper for foreground mutations until the mutation has explicit idempotency semantics.

## Deferred Queue

The first implemented queue is telemetry-only:

- Code: `client/app/bridge_churn_queue.go`.
- Target: client log relay.
- Depth: `64` entries.
- Max age: `30s`.
- Drain: up to `16` entries when the bridge becomes ready.
- Drops: overflow and expiry increment `telemetry_relay:dropped`.

The queue is intentionally not a general outbox. Chat sends, settings writes, and admin mutations are excluded until they have request IDs and server-side dedupe.

## Retry Policy

Use `parseBridgeRetryPolicyFor` and `parseBridgeRetryableStatus` for retry-safe families only.

Retryable status:

- `Unavailable`
- `DeadlineExceeded`
- `Canceled`
- `Unknown`
- `ResourceExhausted`
- non-gRPC transport errors

Not retryable:

- authz/authn failures
- validation errors
- failed preconditions
- mutation conflicts

Allowed retry families:

- conversation list refresh
- auth/session refresh
- admin dashboard reads
- telemetry relay
- diagnostics polling
- background maintenance

Single-shot until idempotency exists:

- chat send/stream
- settings writes
- profile mutations
- admin mutations
- any operation with billing, account, provider policy, role, or destructive side effects

## Write-Path Audit

Settings/profile writes:

- Name, tone, prompt, thinking enabled, thinking effort, speech provider, memory, and locale changes apply optimistically in local UI state.
- The settings route/modal closes immediately after scheduling writes.
- If the bridge drops before the server write starts, the modal still closes and a customer-safe late error is shown.
- Late failures invalidate relevant caches and surface through `SettingsError` toast/status.

Admin mutations:

- Keep confirmation flows single-shot.
- Do not close over a synchronous WASM-thread bridge RPC.
- Add request IDs and server-side dedupe before enabling automatic retry.

## Idempotency Review

Safe to retry today:

- read refreshes
- diagnostics polling
- telemetry relay
- background maintenance reads/ticks

Not safe to retry today:

- chat send/stream because duplicate sends create duplicate user/assistant messages.
- settings writes as a group because memory delete/upsert and profile changes do not yet share one transaction/request ID.
- admin toggles because duplicate submissions can create duplicate audit/events or unexpected policy state.

Future retriable writes must add:

- client-generated request ID
- server-side dedupe table or unique operation key
- durable audit correlation
- clear duplicate-result semantics

## User Experience

Bridge churn should be visible but not alarming:

- Shell banner: `Retrying connection`, `Connection degraded`, `Connection paused while this tab sleeps`, or `Offline`.
- Settings save button changes to `Save locally` when the bridge is not cleanly ready.
- Settings late failures appear as a toast/status after the modal closes.
- Telemetry-only failures are suppressed from customer-facing UI and counted instead.
- Authenticated routes must not collapse back to boot/auth shells during transient bridge loss.

## Observability Seam

`parseBridgeChurnMetrics` counts:

- `telemetry_relay:deferred`
- `telemetry_relay:dropped`
- `telemetry_relay:retried`
- `telemetry_relay:retry_recovered`
- `telemetry_relay:failed_permanent`
- `settings_write:failed_permanent`

Operator correlation comes from request ID, correlation ID, client ID, traceparent, tracestate, support IDs, tunnel connect/disconnect logs, and bridge-state logs.

## Browser Regression Suite

Required browser flows:

- Settings save during reconnect: modal closes; toast/status appears on failure; route remains stable; metrics include the failed/deferred event.
- Chat stream plus reconnect: active thread remains mounted; one customer-safe failed assistant message appears if send cannot continue; no duplicate send occurs.
- Dashboard polling during reconnect: current dashboard route and stale data stay visible; polling is skipped/deferred without noisy console failure.
- Client-log relay during reconnect: logs enter bounded queue; overflow/expiry is counted; ready state drains without replay storm.
- Route/shell reconnect transition: active route, sidebar state, current modal state, and auth shell do not reset during transient loss.

## Troubleshooting Notes

When investigating bridge churn:

1. Check client bridge-state logs for `booting`, `ready`, `reconnecting`, `degraded`, `sleeping`, and `offline`.
2. Check tunnel logs around `/socket` connect/disconnect.
3. Match client request IDs, correlation IDs, traceparent/tracestate, and support IDs with server logs.
4. Separate foreground failures from telemetry-only relay failures.
5. Inspect `parseBridgeChurnMetrics` counters to see whether work was deferred, dropped, retried, recovered, or permanently failed.
6. For settings, verify which write failed: name, tone, prompt, thinking, memory, or locale.

## Secondary Bridge Decision

Decision: do not add a secondary telemetry bridge now.

Scope considered: telemetry isolation only, not traffic balancing.

Comparison:

- Startup cost: a second tunnel increases WASM startup and readiness work.
- Auth handling: a second bridge needs duplicate auth metadata, client ID, correlation ID, expiry, and revocation behavior.
- Reconnect complexity: two independent reconnect loops create ambiguous foreground/background health states.
- Server load: telemetry isolation shifts churn but adds socket and gRPC server load.
- User-facing benefit: bounded best-effort queue and suppression already remove the noisy user-facing failure mode.

Keep single-bridge hardening until telemetry volume or reconnect evidence proves isolation is needed.

## No Round-Robin Note

Do not round-robin arbitrary RPCs across multiple bridges.

Multiple bridges are only acceptable after formal ordering, idempotency, state-coherency, failover, and traffic-class separation guarantees exist. Future multi-bridge work must preserve class separation, for example foreground stream bridge versus telemetry bridge, rather than naive per-RPC spreading.
