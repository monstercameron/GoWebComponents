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
WHERE created_at >= ?
  AND LOWER(TRIM(event_type)) IN ('invoice.payment_failed', 'payment_failed')
ORDER BY created_at DESC, id DESC
LIMIT ?;
