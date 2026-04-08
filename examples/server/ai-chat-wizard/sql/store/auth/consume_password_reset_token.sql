UPDATE password_reset_tokens
SET
    status = 'consumed',
    consumed_at = ?
WHERE id = ?
  AND status = 'pending';
