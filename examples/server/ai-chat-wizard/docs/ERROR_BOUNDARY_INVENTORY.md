# Example 100 Error Boundary Inventory

This inventory tracks the top-level failure boundaries that matter for incident triage in example 100.

Legend:
- `yes` means the boundary currently emits a useful signal at that seam.
- `partial` means the boundary emits some signal, but not consistently or not in the same place as the failure.
- `no` means the boundary currently has no explicit signal at that seam.
- `n/a` means the row is not a customer-facing copy boundary.

## Client Boundaries

| Boundary | Where it lives | Logs? | Customer-safe copy? | Drops trace context? | Notes |
| --- | --- | --- | --- | --- | --- |
| Auth/bootstrap fallback | `client/app/runtime.go` | yes | yes | partial | Startup retries and fallback mode already log worker or gRPC failures, and the UI can show a safe degraded state. |
| Client fallback bridge | `client/app/runtime.go` and `client/app/app.go` | yes | yes | partial | Fallback to main-thread rendering or alternate route state is visible, but the browser side still depends on client-local identity rather than server trace propagation. |
| Background worker bootstrap | `client/backgroundworker/main.go` | yes | n/a | partial | Bootstrap panics and ignored worker-message decode failures now log before the worker stops. |

## Server Boundaries

| Boundary | Where it lives | Logs? | Customer-safe copy? | Drops trace context? | Notes |
| --- | --- | --- | --- | --- | --- |
| HTTP entry and auth gate | `server/app/auth_service.go`, `server/app/server.go`, `server/app/tunnel_handler.go` | yes | yes | no | Authenticated entry paths preserve request/correlation/trace metadata and emit safe redirects or unauthorized responses. |
| gRPC send / streaming reply loop | `server/app/server.go` | yes | yes | no | Send-path logs already include actionable RPC fields, provider info, and customer-safe status handling. |
| Admin and superuser RPCs | `server/app/admin_*.go`, `server/app/superuser_*.go` | yes | yes | no | Most admin mutations and reads log with scoped fields and keep traceability through the request context. |
| Server-tool runtime | `server/app/server_tool_stub.go`, `server/app/server_tool_runtime.go` | partial | yes | no | Runtime errors are surfaced back to the client, but not every branch emits the same operator log shape yet. |

## Job And Notification Boundaries

| Boundary | Where it lives | Logs? | Customer-safe copy? | Drops trace context? | Notes |
| --- | --- | --- | --- | --- | --- |
| Background-job dispatch | `server/app/job_flows.go` | yes | n/a | no | Dispatch now logs blocked jobs, handler failures, and failed persistence updates with traceability fields. |
| Notification dispatch | `server/app/notification_flow.go` | yes | n/a | no | Delivery failures and status-update failures now log with request, user, and workspace context. |

## Webhook Boundary

| Boundary | Where it lives | Logs? | Customer-safe copy? | Drops trace context? | Notes |
| --- | --- | --- | --- | --- | --- |
| Webhook delivery bookkeeping | `server/app/store_superuser.go` | partial | n/a | no | Delivery rows and retry rows carry traceability envelopes, but the repository does not yet have a dedicated runtime delivery loop with its own operator log hook. |

## Follow-Up Rule

If a new top-level boundary is added in client, server, worker, stream, job, or webhook code, it should be checked against this inventory before the change lands.

