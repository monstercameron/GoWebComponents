UPDATE billing_dunning_events
SET
    status = ?,
    resolved_at = ?,
    updated_at = ?
WHERE id = ?
  AND customer_id = ?;
