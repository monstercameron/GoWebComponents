INSERT INTO billing_dunning_events (
    id,
    customer_id,
    subscription_id,
    invoice_id,
    status,
    attempt_count,
    failure_reason,
    next_attempt_at,
    resolved_at,
    created_at,
    updated_at
)
VALUES (
    NULLIF(?, 0),
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
ON CONFLICT(id) DO UPDATE SET
    customer_id = excluded.customer_id,
    subscription_id = excluded.subscription_id,
    invoice_id = excluded.invoice_id,
    status = excluded.status,
    attempt_count = excluded.attempt_count,
    failure_reason = excluded.failure_reason,
    next_attempt_at = excluded.next_attempt_at,
    resolved_at = excluded.resolved_at,
    updated_at = excluded.updated_at;
