SELECT
    id,
    customer_id,
    subscription_id,
    invoice_id,
    event_type,
    event_source,
    event_summary,
    event_payload_json,
    actor_user_id,
    created_at
FROM billing_events
WHERE customer_id = ?
ORDER BY id DESC
LIMIT ?;
