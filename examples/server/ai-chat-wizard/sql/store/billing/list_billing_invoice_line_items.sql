SELECT
    li.id,
    li.invoice_id,
    li.usage_event_id,
    li.line_type,
    li.description,
    li.quantity,
    li.unit_amount_cents,
    li.amount_cents,
    li.currency,
    li.period_start,
    li.period_end,
    li.created_at
FROM billing_invoice_line_items li
JOIN billing_invoices bi ON bi.id = li.invoice_id
WHERE li.invoice_id = ?
  AND bi.customer_id = ?
ORDER BY li.id ASC;
