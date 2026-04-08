# Example 100 Manual Bug-Hunt Checklist

Use this checklist when intentionally triggering failures to verify customer copy, operator evidence, and secret handling together.

For every failure below, confirm four things:
- the customer sees friendly copy
- the customer sees a visible support or request ID
- the operator log trail points at the same failure boundary
- no password, bearer token, API key, cookie, or upstream secret leaks into customer copy

## Trigger Matrix

| Failure to trigger | Suggested probe | What to verify |
| --- | --- | --- |
| Auth failure | bad credentials, expired session, or denied external-auth policy | calm auth copy, request ID, auth log trail |
| Chat send failure | provider timeout, provider unavailable, or usage-budget block | calm send copy, request ID, `rpc.Send` log trail, partial state preserved when appropriate |
| Settings save failure | storage unavailable or rejected mutation | calm settings copy, request ID, settings RPC log trail, no stale success state |
| Dashboard load failure | store unavailable or denied admin scope | calm dashboard copy, request ID, dashboard log trail, safe degraded slice state |
| Background-job failure | blocked job scope or handler failure | operator log trail, job key, failed status row |
| Webhook failure | delivery callback failure or retry update failure | operator log trail, delivery key, failed retry row |
| Provider failure | upstream timeout or unsupported model/provider | calm provider fallback copy, provider log trail, no raw upstream payload in the customer surface |

## Review Rule

Do not mark a failure path as verified until the customer copy, operator log, and persisted evidence line up with the same boundary and the same traceability ID.

