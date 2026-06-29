INSERT INTO billing_events (
    customer_id,
    subscription_id,
    invoice_id,
    event_type,
    event_source,
    event_summary,
    event_payload_json,
    actor_user_id,
    created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM billing_customers WHERE id = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM billing_subscriptions WHERE id = ? AND customer_id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM billing_invoices WHERE id = ? AND customer_id = ?));
