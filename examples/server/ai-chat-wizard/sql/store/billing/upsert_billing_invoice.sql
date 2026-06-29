INSERT INTO billing_invoices (
    customer_id,
    subscription_id,
    provider_id,
    provider_invoice_id,
    status,
    currency,
    subtotal_cents,
    tax_cents,
    discount_cents,
    total_cents,
    amount_due_cents,
    amount_paid_cents,
    period_start,
    period_end,
    due_at,
    paid_at,
    hosted_invoice_url,
    external_metadata_json,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM billing_customers WHERE id = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM billing_subscriptions WHERE id = ? AND customer_id = ?))
ON CONFLICT(provider_invoice_id) DO UPDATE SET
    customer_id = excluded.customer_id,
    subscription_id = excluded.subscription_id,
    provider_id = excluded.provider_id,
    status = excluded.status,
    currency = excluded.currency,
    subtotal_cents = excluded.subtotal_cents,
    tax_cents = excluded.tax_cents,
    discount_cents = excluded.discount_cents,
    total_cents = excluded.total_cents,
    amount_due_cents = excluded.amount_due_cents,
    amount_paid_cents = excluded.amount_paid_cents,
    period_start = excluded.period_start,
    period_end = excluded.period_end,
    due_at = excluded.due_at,
    paid_at = excluded.paid_at,
    hosted_invoice_url = excluded.hosted_invoice_url,
    external_metadata_json = excluded.external_metadata_json,
    updated_at = excluded.updated_at;
