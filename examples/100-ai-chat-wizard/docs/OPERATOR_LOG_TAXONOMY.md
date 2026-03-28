# Example 100 Operator Log Taxonomy

This note defines the logging shape example 100 should use at server, worker, and transport boundaries.

## Severity Rules

| Level | Use for | Do not use for |
| --- | --- | --- |
| `info` | successful state changes, expected lifecycle transitions, safe recovery, and ordinary operator breadcrumbs | failures, retries, policy denials, and boundary exits that indicate something went wrong |
| `warn` | recoverable failures, fallback activation, policy blocks that still leave the system healthy, and user-actionable input problems | unrecoverable server faults or storage/provider outages |
| `error` | boundary failures that prevent the intended operation, storage failures, provider failures, bootstrap failures, and dispatch failures that need operator follow-up | normal branch choices or expected denials that already have a safe outcome |

## Required Fields

When a boundary log has traceability context, prefer these fields in addition to the message:

- `request.id`
- `correlation.id`
- `trace.id`
- `span.id`
- `support.id`
- `route`
- `rpc`
- `action`
- `provider`
- `actor.scope`
- `actor.user_id`
- `workspace_id`
- `target.scope`
- `target.id`
- `target.user_id`
- `target.workspace_id`

Not every log needs every field. The rule is to attach the smallest set that makes the boundary searchable and explainable.

## Redaction Rules

- Never log raw passwords, bearer tokens, API keys, cookies, session values, or webhook secrets.
- Never log raw provider credentials or unfiltered upstream payload fragments when a shared scrubber exists.
- Never rely on customer-facing copy to double as operator detail.
- Use the shared scrubbing helpers before persisting or emitting any diagnostic payload that may contain secrets.

## Boundary Logging Rules

- Emit one boundary log at the seam where a failure is first observed, not only later when the response is already gone.
- Use `warn` when the system falls back safely and `error` when the intended operation failed or the boundary stopped early.
- Keep the boundary message short and put the actionable context in fields.
- Preserve traceability fields whenever the context already carries them.
- If the failure is customer-visible, pair the operator log with customer-safe copy that includes the visible support or request ID.

## Review Rule

Before landing a new boundary, ask:

1. does it log
2. does the log include the right traceability fields
3. is any secret-bearing data redacted
4. does the user-facing path show safe copy if the failure is visible to customers

