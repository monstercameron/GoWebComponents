# Observability And Failure Handling

Example 100 treats observability as part of the architecture. The app should be explainable when it fails, not only impressive when everything is online.

## Traceability Fields

Important flows should carry or log:

| Field | Purpose |
|---|---|
| Request ID | Stable lookup id for one RPC, HTTP request, or customer-visible failure. |
| Correlation ID | Links browser, bridge, server, and provider events across a longer journey. |
| Route | Shows whether the failure happened on public, app, thread, settings, dashboard, or canvas routes. |
| User/workspace/admin scope | Explains authorization and row-filter decisions without exposing secrets. |
| Provider/model | Explains runtime selection, fallback, timeout, and cost behavior. |
| Admin slice/target | Explains dashboard and mutation decisions. |

Provider HTTP adapters also apply traceability headers such as `x-request-id` and `x-correlation-id` where upstream calls support them.

## Logs

Client logs cover route guards, boot-shell teardown, gRPC readiness, reconnect state, model-catalog loads, conversation route resolution, stream failures, dashboard loads, and worker/render failures.

Server logs cover auth role resolution, admin slice denial, RPC bootstrap/list/detail loads, provider resolution and streaming errors, customer-safe error wrapping, mutation confirmation, mutation success, audit-write failure, and rollback-required operational failures.

Logs should be structured enough that an operator can answer "what failed, for whom, on which route, under which request id, and what should I check next?"

## Audit Events

Audit rows are for durable operator actions and security-relevant state changes. Admin mutations write audit entries with actor, workspace, target type/id, summary, payload JSON, request id, and correlation id where available. Auth, workspace, billing, support, incident, provider, and server-tool actions should use the same principle when they become durable product events.

Audit logs are not a replacement for runtime logs. Logs explain transient failures. Audit rows preserve intentional actions and decisions.

## Support IDs And Customer-safe Errors

Customer-facing errors should use friendly copy plus a stable support/request reference. They should never include raw provider payloads, stack traces, bearer tokens, cookies, passwords, API keys, webhook secrets, or upstream secret material.

Server-side customer-safe error helpers wrap chat/auth/settings/dashboard failures into typed gRPC details so the client can render safe copy while operators still get a lookup path. The customer sees enough to report the problem; the operator gets enough context to diagnose it.

## Operator-safe Diagnostics

Operator diagnostics should include specific next-action hints without leaking secrets. Examples:

| Event | Useful next action |
|---|---|
| Auth role denied | Check session, workspace membership, superuser role, or workspace status. |
| Dashboard slice denied | Verify role scope and whether the surface requires superuser access. |
| Provider unavailable | Check configured provider key, rate limits, provider health, and fallback policy. |
| Bridge reconnect churn | Compare client bridge logs with `/socket` connect/disconnect and request ids. |
| Mutation rollback required | Check operational side effects and repair audit/state before retrying. |

## Outage And Stale-state Behavior

The UI should preserve work when infrastructure flickers. A transient bridge loss should not erase the authenticated shell, open settings modal, current route, draft, or dashboard context. Route-owned data may show stale/loading/error state until the RPC succeeds again.

Fail closed for security and durable writes. If auth cannot be validated, private surfaces must clear. If a settings/admin mutation cannot persist, the UI should report failure and avoid claiming success. If a provider stream fails, the thread should get one coherent assistant error instead of hanging forever or duplicating partial failures.

## Failure Handling Rules

1. Keep customer copy safe and stable.
2. Keep operator diagnostics structured and correlated.
3. Preserve local user work during transient transport failure.
4. Fail closed for auth, admin scope, entitlement, billing, and durable writes.
5. Avoid silent no-ops for product-critical actions.
6. Pair every new failure branch with a test or documented manual verification path.
