UPDATE billing_invoices
SET
    status = ?,
    paid_at = ?,
    updated_at = ?
WHERE id = ?
  AND customer_id = ?;
