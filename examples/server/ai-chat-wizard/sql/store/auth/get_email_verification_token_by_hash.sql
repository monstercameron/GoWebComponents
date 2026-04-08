SELECT
    id,
    user_id,
    email,
    status,
    expires_at,
    verified_at,
    created_at
FROM email_verification_tokens
WHERE token_hash = ?
LIMIT 1;
