SELECT
    id,
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
FROM billing_invoices
WHERE customer_id = ?
ORDER BY id DESC
LIMIT ?;
