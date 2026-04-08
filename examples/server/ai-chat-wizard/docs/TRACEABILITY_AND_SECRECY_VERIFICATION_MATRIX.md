# Example 100 Traceability And Secrecy Verification Matrix

This matrix maps the main customer-visible failure paths to the operator evidence that should exist for triage.

Use it to verify three things together:
- the customer sees calm copy plus a visible support or request ID
- operators can find the matching log trail and persisted record
- secret-bearing data stays out of customer copy and out of logs

## Verification Matrix

| Scenario | Customer-visible ID or copy | Operator log trail | Persisted evidence | Runtime log trail | Secrets check |
| --- | --- | --- | --- | --- | --- |
| Auth login or signup failure | safe auth error copy with request or support ID | `rpc.Login`, `rpc.Signup`, or auth bootstrap logs with request/correlation fields | auth/session rows and auth audit rows when applicable | client auth/session logs | no password, bearer token, cookie, or provider secret in the error copy |
| Chat send or stream failure | safe send error copy with request or support ID | `rpc.Send` logs with route, rpc, actor, workspace, model, and provider fields | conversation/message rows plus usage rows for successful partial work | client send/stream logs and server stream logs | no raw upstream provider payload, token, or secret route identifier in the user copy |
| Settings save failure | safe settings error copy with request or support ID | settings RPC logs such as `SetSelectedModel`, `SetSelectedTone`, `SetCustomSystemPrompt`, or memory mutation logs | profile, settings, or memory rows when the mutation succeeds | client settings logs | no raw secret, prompt payload, or token in the customer-facing copy |
| Dashboard load failure | safe dashboard error copy with request or support ID | dashboard RPC logs such as `GetAdminDashboard`, `GetAdminUserDetail`, or `GetWorkspaceAdminSlices` | dashboard reads may not persist anything, so log trail is the primary evidence | client dashboard logs and server slice logs | no internal SQL, stack trace, or hidden scope identifiers in the customer copy |
| Background-job failure | operator log plus job key and traceability fields | `background job dispatch failed`, `background job handler failed`, or `background job retry update failed` | `background_jobs` status rows with failure details | server job-flow logs | no secret-bearing payload fragments in persisted error text |
| Notification dispatch failure | operator log plus notification key and traceability fields | `notification dispatch failed` or `notification delivery failed` | `notification_outbox` rows with failed status and error text | server notification logs | no raw secrets in delivery payloads, headers, or customer copy |
| Webhook bookkeeping failure | operator log plus delivery key and traceability fields when the runtime loop exists | webhook delivery bookkeeping rows and retry rows | `webhook_deliveries` rows and retry status rows | store and superuser logs when writes fail | no webhook secret, bearer token, or signature value in customer-facing copy |

## Review Checklist

Before merging a change that affects a customer-visible failure path, confirm:

1. the customer-safe message points to the visible request or support ID
2. the operator log includes the traceability fields needed to find the record
3. any audit or status row still stores only scrubbed or operationally safe text
4. raw tokens, cookies, passwords, API keys, and upstream secret material never appear in the user copy

