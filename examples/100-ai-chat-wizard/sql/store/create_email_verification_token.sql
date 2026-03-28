INSERT INTO email_verification_tokens (
    user_id,
    email,
    token_hash,
    status,
    expires_at,
    verified_at,
    created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?);
