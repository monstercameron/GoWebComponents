INSERT INTO password_reset_tokens (
    user_id,
    email,
    token_hash,
    status,
    requested_by_ip,
    expires_at,
    consumed_at,
    created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);
