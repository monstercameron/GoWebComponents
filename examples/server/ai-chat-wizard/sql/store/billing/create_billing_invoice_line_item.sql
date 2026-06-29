INSERT INTO billing_invoice_line_items (
    invoice_id,
    usage_event_id,
    line_type,
    description,
    quantity,
    unit_amount_cents,
    amount_cents,
    currency,
    period_start,
    period_end,
    created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM billing_invoices WHERE id = ? AND customer_id = ?);
