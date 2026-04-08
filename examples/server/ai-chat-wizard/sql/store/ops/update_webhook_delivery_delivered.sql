UPDATE webhook_deliveries
SET
    response_status = ?,
    response_body = ?,
    attempt_count = ?,
    delivered_at = ?,
    failed_at = '',
    next_retry_at = '',
    updated_at = ?
WHERE delivery_key = ?;
