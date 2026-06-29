SELECT
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
FROM billing_dunning_events
WHERE customer_id = ?
ORDER BY id DESC
LIMIT ?;
