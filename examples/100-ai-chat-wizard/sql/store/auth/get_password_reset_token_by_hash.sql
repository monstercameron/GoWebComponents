SELECT
    id,
    user_id,
    email,
    status,
    requested_by_ip,
    expires_at,
    consumed_at,
    created_at
FROM password_reset_tokens
WHERE token_hash = ?
LIMIT 1;
