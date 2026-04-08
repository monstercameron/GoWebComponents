UPDATE notification_outbox
SET
    status = ?,
    sent_at = ?,
    failed_at = ?,
    error_message = ?,
    updated_at = ?
WHERE id = ?;
